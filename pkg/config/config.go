// Copyright 2023 Hugo Matus
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package config provides a flexible configuration management system
package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Config is the unified interface for reading and watching configuration.
type Config interface {
	Load() error
	Get(key string) interface{}
	GetString(key string) string
	GetInt(key string) int
	GetFloat64(key string) float64
	GetBool(key string) bool
	GetStringSlice(key string) []string
	GetStringMap(key string) map[string]interface{}
	GetDuration(key string) time.Duration
	GetTime(key string) time.Time
	IsSet(key string) bool
	GetSchema() interface{}
	Watch(ctx context.Context, onChange func()) error
	AllKeys() []string
	AllSettings() map[string]interface{}
}

// RemoteProvider holds parameters for an external config source.
// E.g., "etcd", "etcd3", "consul", "firestore", "nats"
type RemoteProvider struct {
	Type     string
	Endpoint string
	Path     string
}

// Option is a function that applies a configuration to the ConfigManager.
type Option func(*ConfigManager)

// ConfigProvider abstracts the configuration-providing responsibility.
type ConfigProvider interface {
	Load() error
}

// ConfigWatcher abstracts the config watching responsibility.
type ConfigWatcher interface {
	Watch(ctx context.Context, onChange func()) error
}

// ConfigManager is the main facade that delegates to a provider and watcher.
type ConfigManager struct {
	viper          *viper.Viper
	logger         *zap.Logger
	provider       ConfigProvider
	watcher        ConfigWatcher
	schema         interface{}
	defaults       map[string]interface{}
	envPrefix      string
	remoteProvider *RemoteProvider
	pollInterval   time.Duration
	watchEnabled   bool
	validate       *validator.Validate
	path           string
	mu             sync.RWMutex
	closed         bool
	done           chan struct{}
}

var (
	ErrTimeout = errors.New("operation timed out")
	ErrClosed  = errors.New("config manager is closed")
)

// New creates a new ConfigManager using the provided file path, logger, and options.
func New(path string, logger *zap.Logger, opts ...Option) *ConfigManager {
	cm := &ConfigManager{
		viper:        viper.New(),
		logger:       logger,
		path:         path,
		defaults:     make(map[string]interface{}),
		pollInterval: 10 * time.Second, // default poll interval
		watchEnabled: false,
		validate:     validator.New(),
		done:         make(chan struct{}),
	}

	// Apply provided options first so that schema, envPrefix, etc. are set.
	for _, opt := range opts {
		opt(cm)
	}

	// Now that options have been applied, initialize provider and watcher.
	if cm.remoteProvider != nil {
		cm.provider = &RemoteConfigProvider{
			viper:     cm.viper,
			logger:    logger,
			provider:  cm.remoteProvider,
			defaults:  cm.defaults,
			envPrefix: cm.envPrefix,
			validate:  cm.validate,
			schema:    cm.schema,
		}
		if cm.watchEnabled {
			cm.watcher = &RemoteConfigWatcher{
				viper:        cm.viper,
				logger:       logger,
				pollInterval: cm.pollInterval,
				provider:     cm.remoteProvider,
			}
		}
	} else {
		cm.provider = &LocalConfigProvider{
			viper:     cm.viper,
			logger:    logger,
			path:      cm.path,
			defaults:  cm.defaults,
			envPrefix: cm.envPrefix,
			validate:  cm.validate,
			schema:    cm.schema,
		}
		cm.watcher = &LocalConfigWatcher{
			viper:  cm.viper,
			logger: logger,
		}
	}

	return cm
}

// Close gracefully shuts down the config manager and its watchers
func (cm *ConfigManager) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.closed {
		return nil
	}

	cm.closed = true
	close(cm.done)
	return nil
}

// Load delegates to the underlying config provider.
func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.closed {
		return ErrClosed
	}
	return cm.provider.Load()
}

// Get returns a value for the given key.
func (cm *ConfigManager) Get(key string) interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.Get(key)
}

// GetString returns a string value for the given key.
func (cm *ConfigManager) GetString(key string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.GetString(key)
}

// GetInt returns an integer value for the given key.
func (cm *ConfigManager) GetInt(key string) int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.GetInt(key)
}

// GetFloat64 returns a float64 value for the given key.
func (cm *ConfigManager) GetFloat64(key string) float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.GetFloat64(key)
}

// GetBool returns a boolean value for the given key.
func (cm *ConfigManager) GetBool(key string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.GetBool(key)
}

// GetStringSlice returns a string slice value for the given key.
func (cm *ConfigManager) GetStringSlice(key string) []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.GetStringSlice(key)
}

// GetStringMap returns a map[string]interface{} value for the given key.
func (cm *ConfigManager) GetStringMap(key string) map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.GetStringMap(key)
}

// GetDuration returns a duration value for the given key.
func (cm *ConfigManager) GetDuration(key string) time.Duration {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.GetDuration(key)
}

// GetTime returns a time.Time value for the given key.
func (cm *ConfigManager) GetTime(key string) time.Time {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.GetTime(key)
}

// IsSet returns true if the key is set in the configuration.
func (cm *ConfigManager) IsSet(key string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.IsSet(key)
}

// GetSchema returns the configured schema (if any).
func (cm *ConfigManager) GetSchema() interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.schema
}

// Watch delegates to the underlying config watcher.
func (cm *ConfigManager) Watch(ctx context.Context, onChange func()) error {
	if cm.watcher != nil {
		return cm.watcher.Watch(ctx, func() {
			if err := cm.provider.Load(); err != nil {
				cm.logger.Error("Failed to reload configuration", zap.Error(err))
			}
			onChange()
		})
	}
	return nil
}

// AllKeys returns all keys holding a value in the configuration.
func (cm *ConfigManager) AllKeys() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.AllKeys()
}

// AllSettings returns all settings in the configuration.
func (cm *ConfigManager) AllSettings() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.AllSettings()
}

// LocalConfigProvider implements ConfigProvider for file-based + ENV configs.
type LocalConfigProvider struct {
	viper     *viper.Viper
	logger    *zap.Logger
	path      string
	defaults  map[string]interface{}
	envPrefix string
	validate  *validator.Validate
	schema    interface{}
}

func (l *LocalConfigProvider) Load() error {
	// Clear all settings but keep the instance
	l.viper.AllSettings()
	l.viper.AllKeys()
	for _, key := range l.viper.AllKeys() {
		l.viper.Set(key, nil)
	}

	// Set defaults
	for key, value := range l.defaults {
		l.viper.SetDefault(key, value)
		l.logger.Debug("Setting default value",
			zap.String("key", key),
			zap.Any("value", value))
	}

	// Configure environment variables
	if l.envPrefix != "" {
		l.viper.SetEnvPrefix(l.envPrefix)
		l.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		l.viper.AutomaticEnv()
	}

	// Set config type based on file extension
	if ext := filepath.Ext(l.path); ext != "" {
		l.viper.SetConfigType(strings.TrimPrefix(ext, "."))
	}

	// Load the config file if it exists
	if _, err := os.Stat(l.path); err == nil {
		l.viper.SetConfigFile(l.path)
		if err := l.viper.ReadInConfig(); err != nil {
			return fmt.Errorf("error reading config file: %w", err)
		}
	} else if os.IsNotExist(err) && len(l.defaults) == 0 {
		return fmt.Errorf("no configuration file found at %s and no defaults provided", l.path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking config file: %w", err)
	}

	// Log loaded configuration for debugging
	l.logger.Debug("Configuration loaded",
		zap.Any("settings", l.viper.AllSettings()))

	// Unmarshal into schema if provided
	if l.schema != nil {
		if err := l.viper.Unmarshal(l.schema); err != nil {
			return err
		}
		// Validate the loaded configuration
		if l.validate != nil {
			if err := l.validateConfig(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *LocalConfigProvider) validateConfig() error {
	if l.schema == nil || l.validate == nil {
		return nil
	}

	err := l.validate.StructCtx(context.Background(), l.schema)
	if err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			for _, e := range validationErrors {
				l.viper.Get(e.Namespace()) // Log the actual value
				return fmt.Errorf("validation failed for field '%s': %s",
					e.Namespace(), e.Tag())
			}
		}
		return err
	}
	return nil
}

// RemoteConfigProvider implements ConfigProvider for remote configs.
type RemoteConfigProvider struct {
	viper     *viper.Viper
	logger    *zap.Logger
	provider  *RemoteProvider
	defaults  map[string]interface{}
	envPrefix string
	validate  *validator.Validate
	schema    interface{}
}

func (r *RemoteConfigProvider) Load() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		// Configure the remote provider.
		if err := r.viper.AddRemoteProvider(
			r.provider.Type,
			r.provider.Endpoint,
			r.provider.Path,
		); err != nil {
			r.logger.Error("Failed to add remote provider",
				zap.String("type", r.provider.Type),
				zap.String("endpoint", r.provider.Endpoint),
				zap.Error(err))
			errCh <- err
			return
		}

		// Set the configuration type (e.g., JSON).
		r.viper.SetConfigType("json")

		// Set defaults and environment prefix.
		for key, value := range r.defaults {
			r.viper.SetDefault(key, value)
		}
		if r.envPrefix != "" {
			r.viper.SetEnvPrefix(r.envPrefix)
			r.viper.AutomaticEnv()
		}

		// Read remote configuration.
		if err := r.viper.ReadRemoteConfig(); err != nil {
			r.logger.Error("Failed to read remote config",
				zap.String("endpoint", r.provider.Endpoint),
				zap.Error(err))
			errCh <- err
			return
		}

		// Unmarshal into the schema if provided.
		if r.schema != nil {
			if err := r.viper.Unmarshal(r.schema); err != nil {
				r.logger.Error("Failed to unmarshal remote config", zap.Error(err))
				errCh <- err
				return
			}
			// Validate the configuration.
			if r.validate != nil {
				if err := r.validate.Struct(r.schema); err != nil {
					r.logger.Error("Remote config validation failed", zap.Error(err))
					errCh <- err
					return
				}
			}
		}

		r.logger.Debug("Successfully loaded remote configuration",
			zap.String("endpoint", r.provider.Endpoint))
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		r.logger.Error("Remote config operation timed out",
			zap.String("endpoint", r.provider.Endpoint))
		return ErrTimeout
	}
}

// LocalConfigWatcher implements ConfigWatcher using Viper's file-watching.
type LocalConfigWatcher struct {
	viper  *viper.Viper
	logger *zap.Logger
}

func (w *LocalConfigWatcher) Watch(ctx context.Context, onChange func()) error {
	if ctx == nil {
		return errors.New("context cannot be nil")
	}

	// Create a channel for cleanup
	done := make(chan struct{})

	// Setup the watcher
	w.viper.OnConfigChange(func(e fsnotify.Event) {
		select {
		case <-ctx.Done():
			return
		default:
			w.logger.Info("Local configuration changed", zap.String("file", e.Name))
			onChange()
		}
	})
	w.viper.WatchConfig()

	// Handle context cancellation
	go func() {
		select {
		case <-ctx.Done():
			close(done)
		}
	}()

	return nil
}

// RemoteConfigWatcher implements ConfigWatcher by polling the remote source.
type RemoteConfigWatcher struct {
	viper        *viper.Viper
	logger       *zap.Logger
	pollInterval time.Duration
	provider     *RemoteProvider
}

func (w *RemoteConfigWatcher) Watch(ctx context.Context, onChange func()) error {
	if ctx == nil {
		return errors.New("context cannot be nil")
	}

	go func() {
		ticker := time.NewTicker(w.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := w.viper.WatchRemoteConfig(); err != nil {
					w.logger.Error("Error watching remote config",
						zap.Error(err),
						zap.Duration("backoff", w.pollInterval))
					continue
				}
				w.logger.Debug("Remote configuration check completed")
				onChange()
			}
		}
	}()
	return nil
}

// Option functions for configuring the ConfigManager.

func WithWatcher() Option {
	return func(cm *ConfigManager) {
		cm.watchEnabled = true
	}
}

func WithRemoteProvider(rp *RemoteProvider) Option {
	return func(cm *ConfigManager) {
		cm.remoteProvider = rp
	}
}

func WithSchema(schema interface{}) Option {
	return func(cm *ConfigManager) {
		cm.schema = schema
	}
}

func WithDefaults(defaults map[string]interface{}) Option {
	return func(cm *ConfigManager) {
		cm.defaults = defaults
	}
}

func WithEnvPrefix(prefix string) Option {
	return func(cm *ConfigManager) {
		cm.envPrefix = prefix
	}
}

func WithPollInterval(interval time.Duration) Option {
	return func(cm *ConfigManager) {
		cm.pollInterval = interval
	}
}
