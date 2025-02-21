package config

type AppConfig struct {
	Server struct {
		Port            string `mapstructure:"port" validate:"required,numeric"`
		ReadTimeout     string `mapstructure:"read_timeout" validate:"required,endswith=s"`
		WriteTimeout    string `mapstructure:"write_timeout" validate:"required,endswith=s"`
		IdleTimeout     string `mapstructure:"idle_timeout" validate:"required,endswith=s"`
		ShutdownTimeout string `mapstructure:"shutdown_timeout" validate:"required,endswith=s"`
	} `mapstructure:"server"`
	Crawler struct {
		MaxDepth   int    `mapstructure:"maxDepth"`
		UserAgent  string `mapstructure:"userAgent"`
		Async      bool   `mapstructure:"async"`
		Timeout    string `mapstructure:"timeout" validate:"required,endswith=s"`
		NumWorkers int    `mapstructure:"num_workers"`
	} `mapstructure:"crawler"`
	Checkpoint struct {
		Enabled  bool   `mapstructure:"enabled"`
		Type     string `mapstructure:"type"`
		FilePath string `mapstructure:"filePath"`
	} `mapstructure:"checkpoint"`
	Logging struct {
		Level  string `mapstructure:"level" validate:"required,oneof=debug info warn error"`
		Output string `mapstructure:"output" validate:"required"`
	} `mapstructure:"logging"`
	Storage struct {
		Elasticsearch struct {
			Endpoint   string `mapstructure:"endpoint" validate:"required,url"`
			Index      string `mapstructure:"index" validate:"required"`
			Timeout    string `mapstructure:"timeout" validate:"required,endswith=s"`
			RetryLimit int    `mapstructure:"retryLimit" validate:"required,min=1,max=10"`
		} `mapstructure:"elasticsearch"`
	} `mapstructure:"storage"`
	Redis struct {
		Endpoint   string `mapstructure:"endpoint"`
		DB         int    `mapstructure:"db"`
		Password   string `mapstructure:"password"`
		QueueKey   string `mapstructure:"queueKey"`
		VisitedKey string `mapstructure:"visitedKey"`
		RetryLimit int    `mapstructure:"retryLimit"`
	} `mapstructure:"redis"`
	Metrics struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"metrics"`
}
