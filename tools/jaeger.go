package tools

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	service     = "goflylivechat"
	environment = "development"
	id          = 1
)

var Tracer = otel.Tracer("goflylivechat")

// InitJaeger 输入 Jaeger 上报地址，输出为关闭函数与错误信息，目的在于初始化全局 OpenTelemetry tracing。
func InitJaeger(endpoint string) (func(context.Context) error, error) {
	// 创建Jaeger exporter
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
	if err != nil {
		return nil, err
	}

	// 创建资源
	r := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(service),
		attribute.String("environment", environment),
		attribute.Int64("ID", id),
	)

	// 创建trace provider
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(r),
		tracesdk.WithSampler(tracesdk.AlwaysSample()), // 生产环境可使用更合理的采样策略
	)

	// 设置全局tracer provider
	otel.SetTracerProvider(tp)

	// 设置全局propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp.Shutdown, nil
}
