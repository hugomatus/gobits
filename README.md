# Gobits

A collection of production-ready Go components built on SOLID principles and cloud-native best practices.

## Design Philosophy

- **S**ingle Responsibility: Each component does one thing well
- **O**pen/Closed: Extensible designs using interfaces and options
- **L**iskov Substitution: Interchangeable implementations
- **I**nterface Segregation: Focused, minimal interfaces
- **D**ependency Inversion: High-level modules independent of details

## Best Practices

- 🔍 Interface-driven design
- 🧪 Test-driven development
- 📏 Clear boundaries between components
- 🛡️ Proper error handling
- 🔄 Context propagation
- 📊 Observability built-in
- 🔒 Thread-safe by default
- 📝 Comprehensive documentation

## Components

### Configuration Management ([pkg/config](pkg/config))

A flexible configuration system supporting:

- Multiple sources (files, env vars, remote providers)
- Dynamic updates
- Schema validation
- Type-safe access

```go
cfg := config.New("config.yaml", logger,
    config.WithSchema(&AppConfig{}),
    config.WithWatcher(),
    config.WithEnvPrefix("APP"),
)
```

[Learn more about config component →](pkg/config/README.md)

### Coming Soon

- **Circuit Breaker**: Fault tolerance for distributed systems
- **Rate Limiter**: Traffic control and resource protection
- **Service Discovery**: Dynamic service registration and discovery
- **Metrics**: Application metrics collection and export
- **Caching**: Multi-level caching with various backends

## Quality Standards

Each component must:

1. **Design**

   - Follow SOLID principles
   - Use idiomatic Go patterns
   - Implement proper context handling
   - Handle errors appropriately

2. **Testing**

   - Unit tests (>80% coverage)
   - Integration tests
   - Benchmarks
   - Examples

3. **Documentation**

   - GoDoc comments
   - UML diagrams
   - Usage examples
   - Performance characteristics

4. **Observability**
   - Structured logging
   - Metrics exposure
   - Tracing support
   - Health checks

## Project Structure

```
gobits/
├── pkg/                    # Components
│   └── config/            # Configuration component
├── internal/              # Shared internal code
├── docs/                  # Documentation
│   └── diagrams/         # UML diagrams
├── examples/              # Usage examples
└── tests/                # Integration tests
```

## Getting Started

```bash
go get github.com/hugomatus/gobits
```

## Development

### Prerequisites

- Go 1.20+
- Docker (for integration tests)
- PlantUML (for diagrams)

### Guidelines

- Use Go modules
- Follow [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- Write comprehensive tests
- Document public APIs
- Include benchmarks

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests
4. Update documentation
5. Submit a pull request

## License

Apache License - see [LICENSE](LICENSE)
