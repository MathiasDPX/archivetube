package traces

import (
	"context"
	"os"

	"github.com/MathiasDPX/archivetube/internal/config"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

func Init(cfg config.ObservabilityConfig, ctx context.Context) (func(context.Context) error, error) {
	os.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.OTelExporter)

	exp, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, err
	}

	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("archivetube"),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}
