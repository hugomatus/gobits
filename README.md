# Gobits

A collection of Go packages providing common building blocks for Go applications.

## Packages

### Available

- `config`: Type-safe configuration management built on Viper

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

### Roadmap (Priority Order)

1. `worker`: Generic worker pool for concurrent task processing

   ```go
   pool := worker.New[MyTask](10)
   pool.Submit(task)
   ```

2. `queue`: Message queue abstractions

   - In-memory and Redis implementations
   - Pub/sub patterns
   - Dead letter support

3. `log`: Structured logging with context awareness

4. `http`: Enhanced HTTP client

   - Retries and circuit breaking
   - Response caching

5. `metrics`: Prometheus-based metrics collection

Future Considerations:

- `cache`: Generic caching interface
- `health`: Health check system
- `retry`: Backoff strategies

## Design Principles

- Standard Go idioms and interfaces
- Context awareness
- Minimal external dependencies
- Complete test coverage
- Clear documentation and examples

## Installation

```bash
go get github.com/hugomatus/gobits
```

## Requirements

- Go 1.20+

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0 - see [LICENSE](LICENSE)
