package types

// ConfigManager defines the interface for configuration management
type ConfigManager interface {
	IsMaster() bool
	GetAuthConfig() AuthConfig
	GetCORSConfig() CORSConfig
	GetPerformanceConfig() PerformanceConfig
	GetLogConfig() LogConfig
	GetDatabaseConfig() DatabaseConfig
	GetEffectiveServerConfig() ServerConfig
	GetRedisDSN() string
	Validate() error
	DisplayServerConfig()
	ReloadConfig() error
}

// SystemSettings 定義所有系統配置項
type SystemSettings struct {
	// 基礎參數
	AppUrl                         string `json:"app_url" default:"http://localhost:3001" name:"專案位址" category:"基礎參數" desc:"專案的基礎 URL，用於拼接分組終端節點位址。系統配置優先於環境變數 APP_URL。" validate:"required"`
	RequestLogRetentionDays        int    `json:"request_log_retention_days" default:"7" name:"日誌保留時長（天）" category:"基礎參數" desc:"請求日誌在資料庫中的保留天數，0為不清理日誌。" validate:"required,min=0"`
	RequestLogWriteIntervalMinutes int    `json:"request_log_write_interval_minutes" default:"1" name:"日誌延遲寫入週期（分鐘）" category:"基礎參數" desc:"請求日誌從快取寫入資料庫的週期（分鐘），0為即時寫入資料。" validate:"required,min=0"`
	ProxyKeys                      string `json:"proxy_keys" name:"全域代理密鑰" category:"基礎參數" desc:"全域代理密鑰，用於存取所有分組的代理端點。多個密鑰請用逗號分隔。" validate:"required"`

	// 請求設定
	RequestTimeout        int    `json:"request_timeout" default:"600" name:"請求逾時（秒）" category:"請求設定" desc:"轉發請求的完整生命週期逾時（秒）等。" validate:"required,min=1"`
	ConnectTimeout        int    `json:"connect_timeout" default:"15" name:"連線逾時（秒）" category:"請求設定" desc:"與上游服務建立新連線的逾時時間（秒）。" validate:"required,min=1"`
	IdleConnTimeout       int    `json:"idle_conn_timeout" default:"120" name:"閒置連線逾時（秒）" category:"請求設定" desc:"HTTP 用戶端中閒置連線的逾時時間（秒）。" validate:"required,min=1"`
	ResponseHeaderTimeout int    `json:"response_header_timeout" default:"600" name:"回應標頭逾時（秒）" category:"請求設定" desc:"等待上游服務回應標頭的最長時間（秒）。" validate:"required,min=1"`
	MaxIdleConns          int    `json:"max_idle_conns" default:"100" name:"最大閒置連線數" category:"請求設定" desc:"HTTP 用戶端連線池中允許的最大閒置連線總數。" validate:"required,min=1"`
	MaxIdleConnsPerHost   int    `json:"max_idle_conns_per_host" default:"50" name:"每主機最大閒置連線數" category:"請求設定" desc:"HTTP 用戶端連線池對每個上游主機允許的最大閒置連線數。" validate:"required,min=1"`
	ProxyURL              string `json:"proxy_url" name:"代理伺服器位址" category:"請求設定" desc:"全域 HTTP/HTTPS 代理伺服器位址，例如：http://user:pass@host:port。如果為空，則使用環境變數配置。"`

	// 密鑰配置
	MaxRetries                   int `json:"max_retries" default:"3" name:"最大重試次數" category:"密鑰配置" desc:"單個請求使用不同 Key 的最大重試次數，0為不重試。" validate:"required,min=0"`
	BlacklistThreshold           int `json:"blacklist_threshold" default:"3" name:"黑名單閾值" category:"密鑰配置" desc:"一個 Key 連續失敗多少次後進入黑名單，0為不拉黑。" validate:"required,min=0"`
	KeyValidationIntervalMinutes int `json:"key_validation_interval_minutes" default:"60" name:"密鑰驗證間隔（分鐘）" category:"密鑰配置" desc:"後台驗證密鑰的預設間隔（分鐘）。" validate:"required,min=1"`
	KeyValidationConcurrency     int `json:"key_validation_concurrency" default:"10" name:"密鑰驗證併發數" category:"密鑰配置" desc:"後台定時驗證無效 Key 時的併發數，如果使用SQLite或者執行環境效能不佳，請盡量保證20以下，避免過高的併發導致資料不一致問題。" validate:"required,min=1"`
	KeyValidationTimeoutSeconds  int `json:"key_validation_timeout_seconds" default:"20" name:"密鑰驗證逾時（秒）" category:"密鑰配置" desc:"後台定時驗證單個 Key 時的 API 請求逾時時間（秒）。" validate:"required,min=1"`

	// For cache
	ProxyKeysMap map[string]struct{} `json:"-"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port                    int    `json:"port"`
	Host                    string `json:"host"`
	IsMaster                bool   `json:"is_master"`
	ReadTimeout             int    `json:"read_timeout"`
	WriteTimeout            int    `json:"write_timeout"`
	IdleTimeout             int    `json:"idle_timeout"`
	GracefulShutdownTimeout int    `json:"graceful_shutdown_timeout"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Key string `json:"key"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	Enabled          bool     `json:"enabled"`
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
}

// PerformanceConfig represents performance configuration
type PerformanceConfig struct {
	MaxConcurrentRequests int `json:"max_concurrent_requests"`
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	EnableFile bool   `json:"enable_file"`
	FilePath   string `json:"file_path"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	DSN string `json:"dsn"`
}

type RetryError struct {
	StatusCode         int    `json:"status_code"`
	ErrorMessage       string `json:"error_message"`
	ParsedErrorMessage string `json:"-"`
	KeyValue           string `json:"key_value"`
	Attempt            int    `json:"attempt"`
	UpstreamAddr       string `json:"-"`
}
