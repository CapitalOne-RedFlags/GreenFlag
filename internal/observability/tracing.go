package observability

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/contrib/detectors/aws/ecs"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitTracer initializes the OpenTelemetry tracer provider with AWS Distro for OpenTelemetry
func InitTracer(serviceName string, serviceVersion string) func(context.Context) error {
	ctx := context.Background()

	// Create a new exporter
	exporter, err := newExporter(ctx)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	// Create a new tracer provider with a batch span processor and the given exporter
	tp := newTracerProvider(exporter, serviceName, serviceVersion)

	// Set the tracer provider and propagator
	otel.SetTracerProvider(tp)

	// Use the X-Ray propagator for AWS compatibility
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		xray.Propagator{},
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return a function to shutdown the tracer provider
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown TracerProvider: %w", err)
		}
		return nil
	}
}

// newExporter creates a new OTLP exporter configured for AWS X-Ray
func newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	// For Lambda, the X-Ray daemon is available at localhost:2000
	return otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint("localhost:2000"),
		otlptracegrpc.WithInsecure(),
	)
}

// newTracerProvider creates a new tracer provider with the given exporter
func newTracerProvider(exp sdktrace.SpanExporter, serviceName, serviceVersion string) *sdktrace.TracerProvider {
	// Create a resource detector for AWS
	ecsDetector := ecs.NewResourceDetector()
	ecsResource, _ := ecsDetector.Detect(context.Background())

	// Create a resource with service information
	baseResource := resource.Default()
	baseResource, _ = resource.Merge(baseResource, ecsResource)
	r, err := resource.Merge(
		baseResource,
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
	)
	if err != nil {
		panic(err)
	}

	// Create a tracer provider with the given exporter and resource
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
		// Use the X-Ray ID Generator for AWS compatibility
		sdktrace.WithIDGenerator(xray.NewIDGenerator()),
		// Sample all traces for simplicity
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
}
