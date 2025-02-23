# Configuration Management

A thread-safe configuration management system for Go applications.

## Features

- YAML and JSON configuration files
- Environment variable overrides
- Type-safe access
- Thread-safe operations

## Component Overview

![Config Component](https://www.plantuml.com/plantuml/png/bLEzRjim4Dxv55SFhGF4Ue0EGG2roM8ZWOpQGzlX8fua0XGfy2WdwTEND4L5WOC2tGZf-_dutV6MFJe_zbfyyXtr18D6POHNNXEiIciQrOuElR86TcYm3HZZeRJzO4qyVLFOEknX_MEGw4bUhOJucNW9xtu3CfGx8Rx0exCd9SanR43R6ZLO1uvwwqdKi-Hg6tybZSnOHP5j-RY4LMVY1xWgu8BR4NtT_OzP8cIluwNN9QmAi63r_SMJCwXXfh08TU0JybmZt2bDi2xurRmKhzZhgxD2UISSrHvD6ni_g65IFYm_RsqRceJr7noAT4xaxSEzgBKTPKu8Ut8dLHF_Ccigsk8QWZcF-Xh8rpAHgdsCN94-ZvKxDR0eTx3PtCI6uIkCJ0nhIGsEsiCmLNkLGTK2f9gfqZm0gAUT8Pa9iSiBrUTKuaBq-0_HpmaFPF19H-KOj2XkGdk1vGWbGwCqpTyFKSpqwBX3pLCe4OFalxqrobkNU51teMbYKKtyQXyW3SMCF8N98jL75fUWuvlEOFKRWDE0vcuUxWD2svQ1JkUz4IR2d6ex3xQ9BmVWtOCAIjh66zVldgSrqosAG4ZRi5X7G4UnNWdF7Hd072TF5nGlHbS8CX9Y474RJcEl_m80)

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

![](https://www.plantuml.com/plantuml/png/pLPDR_Cs3BxxLt2vl7QN6EZEHT71ROS2oHQasteOTb1i9X4YIuP4fnzP__jiqtOSfMYxxlBwq9eVeiY73-bSEHAMobm5Fz06SuH22Qa3jvMw45Raa2hXtAtHT2zV4Cv_yaq_4rcvB0aFFkS1IL88eyJebLoNLf0q6cP2YpNcg0cI-YHSIx6kuuH_59aWpA9H47o3EqreLo955yZsjGyr0k7WZjzX7m3yE3KY2oD0QusjvL-GmYq-WoChzJg2FiJ-jJNVDvOZ9_xVsTCA1n6U77qGb6x2b9uWDPNbYUA4_u_1w6GZz1fXLUeqZBfqNeFJ2kRMx6I6bYlff64jxnnkKkZEjWBilvpSDwYSKek4t11qGTFIxZfk65-Np9gB9h2J1LhedxD6Zl-i_pPsPTRLcOFzHHJnjDQnkUWgvkS85FPuREjgds7bxE2Q1dLshqqJo70bIaMkDUUY-8lx-xVlYNetjxYIJ-p9Net5Ocu8--wSBOvaBiGerL1r9nG0aCmnlcwfVgZZHekbmWm0biOe1b0eMTEz1v1bKu7OMZY-e0rx39EhUms_ucFOc5avRZ4VOZq6Kv23E8v_A-gC8ZWxwcaJnvyT-61uuAFfWNV61_uxHG7ssX2-jaSZI8LIhgTGtEPF1YnEIfqBwpP2GSdR159U4qjS6OiWzSvigppxoyAece6Ey5DJnNvZGgV9tEUzJtbkbK-XdeQVP20xU0HvWnlU1FWWCoG-VWiKcUlmM4c5O-ZXSdKCquOSzvUx0JZC_ZVGMRoBZZ_n_XXzfp3nxBTe3O7omF6OuwtdQV9m3Cq9BgT3-t_7P6Qq96DTqs986ry7GcT0LjONk6Q2bYBTWj6jmqcVJsjP-BLyOlLxTNdxqjkMtdV18yfNSV4BEwRk_9vicH9_Fk7tvmA7Y-n6PuMHceQw-M7b3cBpVXt1nGM4jsDZqutC8hYyvC2Sql7kZVZRkq3LbEysid11zwFcuf_9199PqFyqO4srXtpLebPnctgd1q-pg3H1CWDJlVV7UmMxTd8FIITpPS4LwgpCrRy0)

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
