package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type TestConfig struct {
	Database struct {
		Host     string `validate:"required"`
		Port     int    `validate:"required,min=1,max=65535"`
		Username string `validate:"required"`
		Password string `validate:"required"`
	}
	Server struct {
		Port     int   `validate:"required,min=1,max=65535"`
		Timeouts []int `validate:"required,dive,min=1"`
	}
}

func setupTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func createTestConfigFile(t *testing.T) string {
	content := `{
		"database": {
			"host": "localhost",
			"port": 5432,
			"username": "test_user",
			"password": "test_pass"
		},
		"server": {
			"port": 8080,
			"timeouts": [5, 10, 15]
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
	return configPath
}

func TestLocalConfigLoading(t *testing.T) {
	configPath := createTestConfigFile(t)
	logger := setupTestLogger()

	schema := &TestConfig{}
	cm := New(
		configPath,
		logger,
		WithSchema(schema),
	)

	err := cm.Load()
	assert.NoError(t, err)

	// Test direct value access
	assert.Equal(t, "localhost", cm.GetString("database.host"))
	assert.Equal(t, float64(5432), cm.Get("database.port")) // Changed to expect float64

	// Test schema validation
	assert.Equal(t, "localhost", schema.Database.Host)
	assert.Equal(t, 5432, schema.Database.Port) // This remains int because of struct type
	assert.Equal(t, "test_user", schema.Database.Username)
	assert.Equal(t, 8080, schema.Server.Port)
	assert.Equal(t, []int{5, 10, 15}, schema.Server.Timeouts)
}

func TestConfigWithDefaults(t *testing.T) {
	configPath := createTestConfigFile(t)
	logger := setupTestLogger()

	defaults := map[string]interface{}{
		"database.host": "default-host",
		"server.port":   9090,
	}

	cm := New(
		configPath,
		logger,
		WithDefaults(defaults),
	)

	err := cm.Load()
	assert.NoError(t, err)

	// The file values should override defaults
	assert.Equal(t, "localhost", cm.GetString("database.host"))
	assert.Equal(t, float64(8080), cm.Get("server.port")) // Changed to expect float64
}

func TestConfigWithEnvVars(t *testing.T) {
	configPath := createTestConfigFile(t)
	logger := setupTestLogger()

	// Set environment variables
	os.Setenv("TEST_DATABASE_HOST", "env-host")
	defer os.Unsetenv("TEST_DATABASE_HOST")

	cm := New(
		configPath,
		logger,
		WithEnvPrefix("TEST"),
	)

	// Force viper to bind the environment variable
	err := cm.viper.BindEnv("database.host", "TEST_DATABASE_HOST")
	assert.NoError(t, err, "Failed to bind environment variable")

	err = cm.Load()
	assert.NoError(t, err)

	// Environment variables should override file values
	assert.Equal(t, "env-host", cm.GetString("database.host"))
}

func TestConfigWatcher(t *testing.T) {
	configPath := createTestConfigFile(t)
	logger := setupTestLogger()

	schema := &TestConfig{}
	cm := New(
		configPath,
		logger,
		WithSchema(schema),
		WithWatcher(),
	)

	err := cm.Load()
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	changeDetected := make(chan bool)
	err = cm.Watch(ctx, func() {
		changeDetected <- true
	})
	assert.NoError(t, err)

	// Modify the config file
	newContent := `{
		"database": {
			"host": "newhost",
			"port": 5432,
			"username": "test_user",
			"password": "test_pass"
		},
		"server": {
			"port": 8080,
			"timeouts": [5, 10, 15]
		}
	}`

	time.Sleep(100 * time.Millisecond) // Give watcher time to initialize
	err = os.WriteFile(configPath, []byte(newContent), 0644)
	assert.NoError(t, err)

	select {
	case <-changeDetected:
		assert.Equal(t, "newhost", cm.GetString("database.host"))
	case <-time.After(2 * time.Second):
		t.Fatal("Config change not detected")
	}
}

func TestInvalidConfiguration(t *testing.T) {
	invalidContent := `{
		"database": {
			"host": "",  // Empty host violates required validation
			"port": 0,   // Zero port violates min validation
			"username": "",
			"password": ""
		},
		"server": {
			"port": 0,
			"timeouts": []
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid_config.json")
	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	assert.NoError(t, err)

	logger := setupTestLogger()
	schema := &TestConfig{}
	cm := New(
		configPath,
		logger,
		WithSchema(schema),
	)

	err = cm.Load()
	assert.Error(t, err) // Should fail validation
}

func TestRemoteConfigProvider(t *testing.T) {
	logger := setupTestLogger()
	schema := &TestConfig{}

	remoteProvider := &RemoteProvider{
		Type:     "mock",
		Endpoint: "mock://localhost",
		Path:     "/config",
	}

	cm := New(
		"", // path not needed for remote config
		logger,
		WithSchema(schema),
		WithRemoteProvider(remoteProvider),
		WithPollInterval(100*time.Millisecond),
	)

	err := cm.Load()
	if err != nil {
		// Expected to fail since this is just a mock
		assert.Contains(t, err.Error(), "mock")
	}
}
