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
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

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
}

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
	}

	// Apply provided options first so that schema, envPrefix, etc. are set.
	for _, opt := range opts {
		opt(cm)
	}

	// Now that options have been applied, initialize provider and watcher.
	if cm.remoteProvider != nil {
		cm.provider = &RemoteConfigProvider{
			viper:     cm.viper,
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

// Load delegates to the underlying config provider.
func (cm *ConfigManager) Load() error {
	return cm.provider.Load()
}

// Get returns a value for the given key.
func (cm *ConfigManager) Get(key string) interface{} {
	return cm.viper.Get(key)
}

// GetString returns a string value for the given key.
func (cm *ConfigManager) GetString(key string) string {
	return cm.viper.GetString(key)
}

// GetInt returns an integer value for the given key.
func (cm *ConfigManager) GetInt(key string) int {
	return cm.viper.GetInt(key)
}

// GetFloat64 returns a float64 value for the given key.
func (cm *ConfigManager) GetFloat64(key string) float64 {
	return cm.viper.GetFloat64(key)
}

// GetBool returns a boolean value for the given key.
func (cm *ConfigManager) GetBool(key string) bool {
	return cm.viper.GetBool(key)
}

// GetStringSlice returns a string slice value for the given key.
func (cm *ConfigManager) GetStringSlice(key string) []string {
	return cm.viper.GetStringSlice(key)
}

// GetStringMap returns a map[string]interface{} value for the given key.
func (cm *ConfigManager) GetStringMap(key string) map[string]interface{} {
	return cm.viper.GetStringMap(key)
}

// GetDuration returns a duration value for the given key.
func (cm *ConfigManager) GetDuration(key string) time.Duration {
	return cm.viper.GetDuration(key)
}

// GetTime returns a time.Time value for the given key.
func (cm *ConfigManager) GetTime(key string) time.Time {
	return cm.viper.GetTime(key)
}

// IsSet returns true if the key is set in the configuration.
func (cm *ConfigManager) IsSet(key string) bool {
	return cm.viper.IsSet(key)
}

// GetSchema returns the configured schema (if any).
func (cm *ConfigManager) GetSchema() interface{} {
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
	return cm.viper.AllKeys()
}

// AllSettings returns all settings in the configuration.
func (cm *ConfigManager) AllSettings() map[string]interface{} {
	return cm.viper.AllSettings()
}

// LocalConfigProvider implements ConfigProvider for file-based + ENV configs.
type LocalConfigProvider struct {
	viper     *viper.Viper
	path      string
	defaults  map[string]interface{}
	envPrefix string
	validate  *validator.Validate
	schema    interface{}
}

func (l *LocalConfigProvider) Load() error {
	// Set defaults.
	for key, value := range l.defaults {
		l.viper.SetDefault(key, value)
	}
	// Set environment variable prefix and enable automatic env loading.
	if l.envPrefix != "" {
		l.viper.SetEnvPrefix(l.envPrefix)
		l.viper.AutomaticEnv()
	}
	// Configure the file path.
	l.viper.SetConfigFile(l.path)
	if err := l.viper.ReadInConfig(); err != nil {
		return err
	}
	// Unmarshal into the schema if provided.
	if l.schema != nil {
		if err := l.viper.Unmarshal(l.schema); err != nil {
			return err
		}
		// Validate the loaded configuration.
		if l.validate != nil {
			if err := l.validate.Struct(l.schema); err != nil {
				return err
			}
		}
	}
	return nil
}

// RemoteConfigProvider implements ConfigProvider for remote configs.
type RemoteConfigProvider struct {
	viper     *viper.Viper
	provider  *RemoteProvider
	defaults  map[string]interface{}
	envPrefix string
	validate  *validator.Validate
	schema    interface{}
}

func (r *RemoteConfigProvider) Load() error {
	// Configure the remote provider.
	if err := r.viper.AddRemoteProvider(
		r.provider.Type,
		r.provider.Endpoint,
		r.provider.Path,
	); err != nil {
		return err
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
		return err
	}
	// Unmarshal into the schema if provided.
	if r.schema != nil {
		if err := r.viper.Unmarshal(r.schema); err != nil {
			return err
		}
		// Validate the configuration.
		if r.validate != nil {
			if err := r.validate.Struct(r.schema); err != nil {
				return err
			}
		}
	}
	return nil
}

// LocalConfigWatcher implements ConfigWatcher using Viper's file-watching.
type LocalConfigWatcher struct {
	viper  *viper.Viper
	logger *zap.Logger
}

func (w *LocalConfigWatcher) Watch(ctx context.Context, onChange func()) error {
	w.viper.WatchConfig()
	w.viper.OnConfigChange(func(e fsnotify.Event) {
		w.logger.Info("Local configuration changed", zap.String("file", e.Name))
		onChange()
	})
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
	ticker := time.NewTicker(w.pollInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := w.viper.ReadRemoteConfig(); err == nil {
					w.logger.Info("Remote configuration updated")
					onChange()
				} else {
					w.logger.Error("Error reading remote config", zap.Error(err))
				}
			case <-ctx.Done():
				return
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
