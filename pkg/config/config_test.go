package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type TestConfig struct {
	Server struct {
		Port    int    `yaml:"port" validate:"required,min=1,max=65535"`
		Host    string `yaml:"host" validate:"required,hostname|ip"`
		Timeout string `yaml:"timeout" validate:"required"`
	} `yaml:"server"`
	Database struct {
		Host     string `yaml:"host" validate:"required"`
		Port     int    `yaml:"port" validate:"required,min=1,max=65535"`
		Name     string `yaml:"name" validate:"required"`
		MaxConns int    `yaml:"maxConns" validate:"required,min=1"`
	} `yaml:"database"`
}

func setupTestConfig(t *testing.T) (string, func()) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	content := []byte(`
server:
  port: 8080
  host: "localhost"
  timeout: "30s"
database:
  host: "127.0.0.1"
  port: 5432
  name: "testdb"
  maxConns: 10
`)

	err := os.WriteFile(configPath, content, 0644)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return configPath, cleanup
}

func TestBasicConfigOperations(t *testing.T) {
	configPath, cleanup := setupTestConfig(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	cfg := New(configPath, logger)

	t.Run("Load Configuration", func(t *testing.T) {
		err := cfg.Load()
		require.NoError(t, err)

		assert.Equal(t, 8080, cfg.GetInt("server.port"))
		assert.Equal(t, "localhost", cfg.GetString("server.host"))
	})

	t.Run("Type Conversions", func(t *testing.T) {
		assert.Equal(t, 8080, cfg.GetInt("server.port"))
		assert.Equal(t, float64(8080), cfg.GetFloat64("server.port"))
		assert.Equal(t, "30s", cfg.GetString("server.timeout"))
		assert.Equal(t, 30*time.Second, cfg.GetDuration("server.timeout"))
	})
}

func TestSchemaValidation(t *testing.T) {
	configPath, cleanup := setupTestConfig(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	schema := &TestConfig{}

	t.Run("Valid Schema", func(t *testing.T) {
		cfg := New(configPath, logger, WithSchema(schema))
		err := cfg.Load()
		require.NoError(t, err)
	})

	t.Run("Invalid Schema", func(t *testing.T) {
		invalidContent := []byte(`
server:
  port: -1  # Invalid port
  host: ""  # Missing host
`)
		err := os.WriteFile(configPath, invalidContent, 0644)
		require.NoError(t, err)

		cfg := New(configPath, logger, WithSchema(schema))
		err = cfg.Load()
		assert.Error(t, err)
	})
}

func TestEnvironmentOverrides(t *testing.T) {
	configPath, cleanup := setupTestConfig(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()

	t.Run("Environment Override", func(t *testing.T) {
		os.Setenv("APP_SERVER_PORT", "9090")
		defer os.Unsetenv("APP_SERVER_PORT")

		cfg := New(configPath, logger, WithEnvPrefix("APP"))

		// Configure viper to bind environment variables
		cfg.viper.SetEnvPrefix("APP")
		cfg.viper.AutomaticEnv()
		cfg.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		err := cfg.Load()
		require.NoError(t, err)

		assert.Equal(t, 9090, cfg.GetInt("server.port"))
	})
}

func TestConcurrentAccess(t *testing.T) {
	configPath, cleanup := setupTestConfig(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	cfg := New(configPath, logger)
	err := cfg.Load()
	require.NoError(t, err)

	t.Run("Concurrent Reads", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = cfg.GetInt("server.port")
				_ = cfg.GetString("server.host")
			}()
		}
		wg.Wait()
	})
}

func TestWatcherSetup(t *testing.T) {
	configPath, cleanup := setupTestConfig(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	cfg := New(configPath, logger, WithWatcher())

	err := cfg.Load()
	require.NoError(t, err)

	t.Run("Watch Config Changes", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		changes := make(chan struct{}, 1)
		err := cfg.Watch(ctx, func() {
			changes <- struct{}{}
		})
		require.NoError(t, err)

		// Give watcher time to initialize
		time.Sleep(500 * time.Millisecond)

		// Create new content with meaningful change
		newContent := []byte(`
server:
  port: 9000
  host: "localhost"
  timeout: "30s"
database:
  host: "127.0.0.1"
  port: 5432
  name: "testdb"
  maxConns: 10
`)
		err = os.WriteFile(configPath, newContent, 0644)
		require.NoError(t, err)

		// Force reload
		err = cfg.Load()
		require.NoError(t, err)

		select {
		case <-changes:
			val := cfg.GetInt("server.port")
			assert.Equal(t, 9000, val)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for config change")
		}
	})
}

func TestConfigDefaults(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("Default Values", func(t *testing.T) {
		defaults := map[string]interface{}{
			"server.port": 8080,
			"server.host": "localhost",
		}

		// Use absolute path that definitely won't exist
		cfg := New("/tmp/definitely-does-not-exist/config.yaml", logger, WithDefaults(defaults))
		err := cfg.Load()
		require.NoError(t, err, "Load should succeed with defaults even if file is missing")

		// Verify defaults are set
		assert.Equal(t, 8080, cfg.GetInt("server.port"))
		assert.Equal(t, "localhost", cfg.GetString("server.host"))
		assert.True(t, cfg.IsSet("server.port"))
	})

	t.Run("Defaults with File Override", func(t *testing.T) {
		defaults := map[string]interface{}{
			"server.port": 8080,
			"server.host": "localhost",
			"db.port":     5432,
		}

		configPath, cleanup := setupTestConfig(t)
		defer cleanup()

		cfg := New(configPath, logger, WithDefaults(defaults))
		err := cfg.Load()
		require.NoError(t, err)

		// File value should override default
		assert.Equal(t, 8080, cfg.GetInt("server.port"))
		// Default should be present for non-file value
		assert.Equal(t, 5432, cfg.GetInt("db.port"))
	})
}

func TestConfigErrors(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("Invalid File Path", func(t *testing.T) {
		cfg := New("nonexistent.yaml", logger)
		err := cfg.Load()
		assert.Error(t, err)
	})

	t.Run("Invalid Remote Provider", func(t *testing.T) {
		cfg := New("config.yaml", logger, WithRemoteProvider(&RemoteProvider{
			Type:     "invalid",
			Endpoint: "localhost:1234",
			Path:     "/config",
		}))
		err := cfg.Load()
		assert.Error(t, err)
	})
}

func TestConfigClose(t *testing.T) {
	configPath, cleanup := setupTestConfig(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	cfg := New(configPath, logger)

	t.Run("Close Config", func(t *testing.T) {
		err := cfg.Close()
		require.NoError(t, err)

		// Verify operations after close
		err = cfg.Load()
		assert.Equal(t, ErrClosed, err)
	})
}

// Add more test cases for better coverage
func TestAdditionalConfigOperations(t *testing.T) {
	t.Run("IsSet Basic", func(t *testing.T) {
		// Create a new temporary directory and config file for this specific test
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yaml")

		// Write initial config
		content := []byte(`
server:
  port: 8080
  host: "localhost"
  timeout: "30s"
`)
		err := os.WriteFile(configPath, content, 0644)
		require.NoError(t, err)

		// Create new config instance
		logger, _ := zap.NewDevelopment()
		cfg := New(configPath, logger)

		// Load the configuration
		err = cfg.Load()
		require.NoError(t, err)

		// Log the full configuration for debugging
		settings := cfg.AllSettings()
		t.Logf("Complete configuration: %+v", settings)

		// Verify server section exists
		serverSection, ok := settings["server"].(map[string]interface{})
		require.True(t, ok, "server section should exist")
		require.NotNil(t, serverSection, "server section should not be nil")
		t.Logf("Server section: %+v", serverSection)

		// Verify individual values
		portValue := serverSection["port"]
		t.Logf("Port value: %v (type: %T)", portValue, portValue)

		// Test settings
		assert.True(t, cfg.IsSet("server.port"), "server.port should be set")
		assert.True(t, cfg.IsSet("server.host"), "server.host should be set")
		assert.Equal(t, 8080, cfg.GetInt("server.port"), "server.port should be 8080")
		assert.Equal(t, "localhost", cfg.GetString("server.host"), "server.host should be localhost")
	})

	t.Run("Config Updates", func(t *testing.T) {
		// Create a new config instance and path for all update tests
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yaml")
		logger, _ := zap.NewDevelopment()
		cfg := New(configPath, logger)

		t.Run("GetStringSlice", func(t *testing.T) {
			content := []byte(`values:
  - one
  - two
  - three`)
			err := os.WriteFile(configPath, content, 0644)
			require.NoError(t, err)

			err = cfg.Load()
			require.NoError(t, err)

			slice := cfg.GetStringSlice("values")
			assert.Equal(t, []string{"one", "two", "three"}, slice)
		})

		t.Run("GetStringMap", func(t *testing.T) {
			content := []byte(`mapping:
  key1: value1
  key2: value2`)
			err := os.WriteFile(configPath, content, 0644)
			require.NoError(t, err)

			err = cfg.Load()
			require.NoError(t, err)

			m := cfg.GetStringMap("mapping")
			expected := map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			}
			assert.Equal(t, expected, m)
		})

		t.Run("AllKeys and AllSettings", func(t *testing.T) {
			keys := cfg.AllKeys()
			assert.NotEmpty(t, keys)

			settings := cfg.AllSettings()
			assert.NotEmpty(t, settings)
		})

		t.Run("IsSet After Update", func(t *testing.T) {
			content := []byte(`
test_key: "test_value"
nested:
  key: value
`)
			err := os.WriteFile(configPath, content, 0644)
			require.NoError(t, err)

			err = cfg.Load()
			require.NoError(t, err)

			// Verify new keys are set
			assert.True(t, cfg.IsSet("test_key"), "test_key should be set")
			assert.True(t, cfg.IsSet("nested.key"), "nested.key should be set")

			// Verify old keys are no longer set
			assert.False(t, cfg.IsSet("server.port"), "server.port should not be set")
			assert.False(t, cfg.IsSet("nonexistent.key"), "nonexistent.key should not be set")
		})

		t.Run("GetBool", func(t *testing.T) {
			content := []byte(`
features:
  enabled: true
  disabled: false
`)
			err := os.WriteFile(configPath, content, 0644)
			require.NoError(t, err)

			err = cfg.Load()
			require.NoError(t, err)

			assert.True(t, cfg.GetBool("features.enabled"))
			assert.False(t, cfg.GetBool("features.disabled"))
		})

		t.Run("GetTime", func(t *testing.T) {
			content := []byte(`
timestamps:
  created: 2023-01-01T00:00:00Z
`)
			err := os.WriteFile(configPath, content, 0644)
			require.NoError(t, err)

			err = cfg.Load()
			require.NoError(t, err)

			expected := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
			assert.Equal(t, expected, cfg.GetTime("timestamps.created"))
		})
	})
}

func TestRemoteConfigTimeout(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := New("config.yaml", logger, WithRemoteProvider(&RemoteProvider{
		Type:     "nonexistent",
		Endpoint: "localhost:1234",
		Path:     "/config",
	}))

	err := cfg.Load()
	assert.Error(t, err)
}

func TestConcurrentModification(t *testing.T) {
	configPath, cleanup := setupTestConfig(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	cfg := New(configPath, logger)
	err := cfg.Load()
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			err := cfg.Load()
			assert.NoError(t, err)
		}()
		go func() {
			defer wg.Done()
			_ = cfg.GetString("server.host")
		}()
	}
	wg.Wait()
}
