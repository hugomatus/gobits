# Configuration Management

A thread-safe configuration management system supporting multiple sources and dynamic updates.

## Features

- File-based configuration (YAML, JSON)
- Environment variable overrides
- Remote configuration providers (Consul, etcd)
- Runtime configuration updates
- Schema validation
- Type-safe access
- Thread-safe operations

## Component Overview

![Config Component](../../docs/diagrams/config-component.puml)

## Usage

### Basic Configuration

```go
logger, _ := zap.NewProduction()
cfg := config.New("config.yaml", logger,
    config.WithEnvPrefix("APP"),
    config.WithDefaults(map[string]interface{}{
        "server.port": "8080",
    }),
)

if err := cfg.Load(); err != nil {
    log.Fatal(err)
}
```

### Schema Validation

```go
type AppConfig struct {
    Server struct {
        Port string `validate:"required"`
    }
}

cfg := config.New("config.yaml", logger,
    config.WithSchema(&AppConfig{}),
)
```

### Dynamic Updates

```go
ctx := context.Background()
cfg.Watch(ctx, func() {
    if appCfg, ok := cfg.GetSchema().(*AppConfig); ok {
        // Handle updated configuration
    }
})
```

### Remote Configuration

```go
cfg := config.New("config.yaml", logger,
    config.WithRemoteProvider(&config.RemoteProvider{
        Type:     "consul",
        Endpoint: "localhost:8500",
        Path:     "myapp/config",
    }),
)
```

## Available Options

| Option               | Description              |
| -------------------- | ------------------------ |
| `WithSchema`         | Adds schema validation   |
| `WithWatcher`        | Enables config watching  |
| `WithEnvPrefix`      | Sets environment prefix  |
| `WithDefaults`       | Sets default values      |
| `WithRemoteProvider` | Configures remote source |

## Configuration Priority

1. Environment variables (highest)
2. Remote provider
3. Local config file
4. Default values (lowest)

## Implementation Details

### Core Interfaces

```go
type Config interface {
    Load() error
    Get(key string) interface{}
    GetString(key string) string
    Watch(ctx context.Context, onChange func()) error
}

type ConfigLoader interface {
    Load() error
}

type ConfigWatcher interface {
    Watch(ctx context.Context, onChange func()) error
}
```

### Class Structure

![Class Diagram](../../docs/diagrams/config-class.puml)

## Best Practices

1. **Environment Variables**

   ```
   APP_SERVER_PORT=8080
   APP_DB_HOST=localhost
   ```

2. **YAML Configuration**

   ```yaml
   server:
     port: 8080
   database:
     host: localhost
   ```

3. **Schema Definition**

   ```go
   type Config struct {
       Server struct {
           Port string `validate:"required"`
       }
       Database struct {
           Host string `validate:"required"`
       }
   }
   ```

## Thread Safety

The configuration system is safe for concurrent:

- Value retrieval
- Configuration updates
- Watch operations

## Example

See [main.go](../../examples/run_config.go) for a complete implementation example.
