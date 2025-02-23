package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/hugomatus/gobits/pkg/config"
	"go.uber.org/zap"
)

func main() {
	// Parse flags for the config file path
	configPath := flag.String("config", "examples/.config.yaml", "path to config file")
	flag.Parse()

	// Create a logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create the config manager instance with desired options
	cfg := config.New(*configPath, logger,
		// Provide a schema for validation (optional).
		config.WithSchema(&config.AppConfig{}),
		// Enable file (or remote) watching.
		config.WithWatcher(),
		// Use environment variable prefix for overrides (e.g., SYNX_SERVER_PORT).
		config.WithEnvPrefix("SYNX"),
		// Provide default values in case they’re absent in file or ENV.
		config.WithDefaults(map[string]interface{}{
			"server.port": "8080",
		}),
		// Uncomment to switch to a remote provider, e.g. Consul:
		// config.WithRemoteProvider(&config.RemoteProvider{
		//    Type:     "consul",
		//    Endpoint: "localhost:8500",
		//    Path:     "myapp/config",
		// }),
	)

	// Load configuration (from file, env, or remote)
	if err := cfg.Load(); err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Start watching for changes in a background context
	ctx := context.Background()
	if err := cfg.Watch(ctx, func() {
		logger.Info("Configuration changed!")
		// Handle updated config as needed...
		//if appCfg, ok := cfg.GetSchema().(*config.AppConfig); ok {
		//	logger.Info("Updated port", zap.String("server.port", appCfg.Server.Port))
		//}
	}); err != nil {
		logger.Error("Failed to watch configuration", zap.Error(err))
	}

	// Access typed configuration (cast from interface{})
	appCfg, ok := cfg.GetSchema().(*config.AppConfig)
	if !ok {
		logger.Fatal("Unable to cast to AppConfig")
	}

	logger.Info("Configuration loaded",
		zap.String("server.port", appCfg.Server.Port),
		zap.String("elasticsearch.endpoint", appCfg.Storage.Elasticsearch.Endpoint),
	)

	// Simulate your application's main loop or server
	fmt.Println("Server starting on port:", appCfg.Server.Port)
	select {} // Block forever to keep the process running
}
