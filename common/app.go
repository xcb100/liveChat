package common

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type AppConfig struct {
	ServiceName              string
	BuildVersion             string
	BuildCommit              string
	BuildTime                string
	HTTPPort                 string
	GRPCPort                 string
	HTTPReadTimeout          time.Duration
	HTTPWriteTimeout         time.Duration
	HTTPIdleTimeout          time.Duration
	RequestTimeout           time.Duration
	ShutdownTimeout          time.Duration
	RateLimitPerSecond       float64
	RateLimitBurst           int
	AgentHeartbeatTTL        time.Duration
	AgentDialTimeout         time.Duration
	AgentRequestTimeout      time.Duration
	AgentDispatchMode        string
	AgentKafkaBrokers        []string
	AgentKafkaTopic          string
	AgentKafkaGroupID        string
	AgentKafkaClientID       string
	AgentKafkaDialTimeout    time.Duration
	AgentKafkaReadTimeout    time.Duration
	AgentKafkaWriteTimeout   time.Duration
	AgentKafkaEnqueueTimeout time.Duration
	AgentKafkaConsumeBackoff time.Duration
	CircuitBreakerTimeout    time.Duration
	CircuitBreakerMaxHalf    uint32
	JaegerEndpoint           string
	EnableTracing            bool
	EnableMetrics            bool
}

// GetAppConfig 输入环境变量，输出为应用运行配置，目的在于统一 HTTP、gRPC、限流与观测能力的默认参数。
func GetAppConfig() AppConfig {
	return AppConfig{
		ServiceName:              getEnvString("LIVECHAT_SERVICE_NAME", "goflylivechat"),
		BuildVersion:             getEnvString("LIVECHAT_BUILD_VERSION", Version),
		BuildCommit:              getEnvString("LIVECHAT_BUILD_COMMIT", "dev"),
		BuildTime:                getEnvString("LIVECHAT_BUILD_TIME", ""),
		HTTPPort:                 getEnvString("LIVECHAT_HTTP_PORT", "8081"),
		GRPCPort:                 getEnvString("LIVECHAT_GRPC_PORT", "9090"),
		HTTPReadTimeout:          getEnvDuration("LIVECHAT_HTTP_READ_TIMEOUT", 10*time.Second),
		HTTPWriteTimeout:         getEnvDuration("LIVECHAT_HTTP_WRITE_TIMEOUT", 15*time.Second),
		HTTPIdleTimeout:          getEnvDuration("LIVECHAT_HTTP_IDLE_TIMEOUT", 60*time.Second),
		RequestTimeout:           getEnvDuration("LIVECHAT_REQUEST_TIMEOUT", 12*time.Second),
		ShutdownTimeout:          getEnvDuration("LIVECHAT_SHUTDOWN_TIMEOUT", 15*time.Second),
		RateLimitPerSecond:       getEnvFloat("LIVECHAT_RATE_LIMIT_RPS", 8),
		RateLimitBurst:           getEnvInt("LIVECHAT_RATE_LIMIT_BURST", 16),
		AgentHeartbeatTTL:        getEnvDuration("LIVECHAT_AGENT_HEARTBEAT_TTL", 75*time.Second),
		AgentDialTimeout:         getEnvDuration("LIVECHAT_AGENT_DIAL_TIMEOUT", 2*time.Second),
		AgentRequestTimeout:      getEnvDuration("LIVECHAT_AGENT_REQUEST_TIMEOUT", 1200*time.Millisecond),
		AgentDispatchMode:        getEnvString("LIVECHAT_AGENT_DISPATCH_MODE", "direct"),
		AgentKafkaBrokers:        getEnvStringList("LIVECHAT_AGENT_KAFKA_BROKERS", []string{"127.0.0.1:9092"}),
		AgentKafkaTopic:          getEnvString("LIVECHAT_AGENT_KAFKA_TOPIC", "livechat.agent.session"),
		AgentKafkaGroupID:        getEnvString("LIVECHAT_AGENT_KAFKA_GROUP_ID", "livechat-agent-dispatcher"),
		AgentKafkaClientID:       getEnvString("LIVECHAT_AGENT_KAFKA_CLIENT_ID", "livechat"),
		AgentKafkaDialTimeout:    getEnvDuration("LIVECHAT_AGENT_KAFKA_DIAL_TIMEOUT", 2*time.Second),
		AgentKafkaReadTimeout:    getEnvDuration("LIVECHAT_AGENT_KAFKA_READ_TIMEOUT", 3*time.Second),
		AgentKafkaWriteTimeout:   getEnvDuration("LIVECHAT_AGENT_KAFKA_WRITE_TIMEOUT", 3*time.Second),
		AgentKafkaEnqueueTimeout: getEnvDuration("LIVECHAT_AGENT_KAFKA_ENQUEUE_TIMEOUT", 800*time.Millisecond),
		AgentKafkaConsumeBackoff: getEnvDuration("LIVECHAT_AGENT_KAFKA_CONSUME_BACKOFF", time.Second),
		CircuitBreakerTimeout:    getEnvDuration("LIVECHAT_BREAKER_TIMEOUT", 5*time.Second),
		CircuitBreakerMaxHalf:    uint32(getEnvInt("LIVECHAT_BREAKER_HALF_OPEN_MAX", 3)),
		JaegerEndpoint:           getEnvString("LIVECHAT_JAEGER_ENDPOINT", "http://localhost:14268/api/traces"),
		EnableTracing:            getEnvBool("LIVECHAT_ENABLE_TRACING", Debug),
		EnableMetrics:            getEnvBool("LIVECHAT_ENABLE_METRICS", true),
	}
}

// getEnvString 输入环境变量名称和默认值，输出为字符串配置，目的在于读取可覆盖的基础配置。
func getEnvString(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvStringList 输入环境变量名称和默认值，输出为字符串切片配置，目的在于统一读取 Kafka broker 列表等多值配置。
func getEnvStringList(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return append([]string(nil), defaultValue...)
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		values = append(values, trimmed)
	}
	if len(values) == 0 {
		return append([]string(nil), defaultValue...)
	}
	return values
}

// getEnvInt 输入环境变量名称和默认值，输出为整型配置，目的在于安全读取数值型环境变量。
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsedValue, parseError := strconv.Atoi(value)
	if parseError != nil {
		return defaultValue
	}
	return parsedValue
}

// getEnvFloat 输入环境变量名称和默认值，输出为浮点型配置，目的在于安全读取限流速率等配置。
func getEnvFloat(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsedValue, parseError := strconv.ParseFloat(value, 64)
	if parseError != nil {
		return defaultValue
	}
	return parsedValue
}

// getEnvBool 输入环境变量名称和默认值，输出为布尔配置，目的在于控制 tracing 与 metrics 等开关。
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsedValue, parseError := strconv.ParseBool(value)
	if parseError != nil {
		return defaultValue
	}
	return parsedValue
}

// getEnvDuration 输入环境变量名称和默认值，输出为时长配置，目的在于统一解析 timeout 与 TTL 类配置。
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsedValue, parseError := time.ParseDuration(value)
	if parseError != nil {
		return defaultValue
	}
	return parsedValue
}
