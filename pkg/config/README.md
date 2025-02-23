# Configuration Management

A thread-safe configuration management system for Go applications.

## Features

- YAML and JSON configuration files
- Environment variable overrides
- Type-safe access
- Thread-safe operations

## Component Overview

![Config Component](../../docs/diagrams/config-component.puml)

## Usage

### Basic Configuration

```go
cfg := config.New("config.yaml",
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

cfg := config.New("config.yaml",
    config.WithSchema(&AppConfig{}),
)
```

## Available Options

| Option          | Description             |
| --------------- | ----------------------- |
| `WithSchema`    | Adds schema validation  |
| `WithEnvPrefix` | Sets environment prefix |
| `WithDefaults`  | Sets default values     |

## Configuration Priority

1. Environment variables (highest)
2. Local config file
3. Default values (lowest)

## Class Structure

![Class Diagram](../../docs/diagrams/config-class.puml)

## Best Practices

1. **Environment Variables**

   ```
   APP_SERVER_PORT=8080
   APP_DB_HOST=localhost
   ```

2. **Configuration Structure**

   YAML configuration:

   ```yaml
   server:
     port: 8080
     host: '0.0.0.0'
     timeout: 30s
   database:
     host: localhost
     port: 5432
     name: myapp
     maxConns: 10
   ```

   Corresponding Go schema with validation:

   ```go
   type Config struct {
       Server struct {
           Port    string        `yaml:"port" validate:"required,numeric"`
           Host    string        `yaml:"host" validate:"required,ip"`
           Timeout time.Duration `yaml:"timeout" validate:"required"`
       } `yaml:"server"`
       Database struct {
           Host     string `yaml:"host" validate:"required"`
           Port     int    `yaml:"port" validate:"required,min=1,max=65535"`
           Name     string `yaml:"name" validate:"required"`
           MaxConns int    `yaml:"maxConns" validate:"required,min=1"`
       } `yaml:"database"`
   }
   ```

## Example

See [main.go](../../examples/run_config.go) for a complete implementation example.
