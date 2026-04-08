package tools

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricsInitOnce sync.Once

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "livechat_http_requests_total",
			Help: "Total number of HTTP requests handled by the service.",
		},
		[]string{"method", "route", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "livechat_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)
	httpInflightRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "livechat_http_inflight_requests",
			Help: "Current number of inflight HTTP requests.",
		},
	)
	serviceReadiness = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "livechat_service_readiness",
			Help: "Readiness state for core dependencies.",
		},
		[]string{"component"},
	)
)

// InitMetrics 输入为空，输出为指标注册结果，目的在于初始化 Prometheus 指标并避免重复注册。
func InitMetrics() {
	metricsInitOnce.Do(func() {
		prometheus.MustRegister(httpRequestsTotal, httpRequestDuration, httpInflightRequests, serviceReadiness)
	})
}

// MetricsMiddleware 输入为空，输出为 Gin 中间件，目的在于记录请求计数、耗时与并发请求数。
func MetricsMiddleware() gin.HandlerFunc {
	InitMetrics()
	return func(c *gin.Context) {
		httpInflightRequests.Inc()
		startedAt := time.Now()
		c.Next()
		httpInflightRequests.Dec()

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		statusCode := strconv.Itoa(c.Writer.Status())
		durationSeconds := time.Since(startedAt).Seconds()
		httpRequestsTotal.WithLabelValues(c.Request.Method, route, statusCode).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, route, statusCode).Observe(durationSeconds)
	}
}

// MetricsHandler 输入为空，输出为 HTTP 指标处理器，目的在于向 Prometheus 暴露采集端点。
func MetricsHandler() http.Handler {
	InitMetrics()
	return promhttp.Handler()
}

// SetReadinessMetric 输入组件名和就绪状态，输出为指标更新结果，目的在于将依赖可用性写入监控。
func SetReadinessMetric(component string, ready bool) {
	InitMetrics()
	if ready {
		serviceReadiness.WithLabelValues(component).Set(1)
		return
	}
	serviceReadiness.WithLabelValues(component).Set(0)
}
