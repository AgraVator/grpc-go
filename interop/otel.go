package interop

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/grpclog"
)

// SetupOpenTelemetry configures OpenTelemetry tracing for interop tests.
func SetupOpenTelemetry(enableOpenTelemetry bool, otelCollectorAddress string, logger grpclog.DepthLoggerV2) (*sdktrace.TracerProvider, propagation.TextMapPropagator, func()) {
	fmt.Printf("[OTEL_DEBUG] >>> SetupOpenTelemetry called with: enableOpenTelemetry=%v, otelCollectorAddress=%q\n", enableOpenTelemetry, otelCollectorAddress)
	logger.Infof("[OTEL_DEBUG] >>> SetupOpenTelemetry called with: enableOpenTelemetry=%v, otelCollectorAddress=%q", enableOpenTelemetry, otelCollectorAddress)

	if !enableOpenTelemetry && otelCollectorAddress == "" {
		fmt.Printf("[OTEL_DEBUG] Branch: DISABLED (enableOpenTelemetry is false and otelCollectorAddress is empty)\n")
		logger.Infof("[OTEL_DEBUG] Branch: DISABLED (enableOpenTelemetry is false and otelCollectorAddress is empty)")
		return nil, propagation.TraceContext{}, func() {}
	}

	fmt.Printf("[OTEL_DEBUG] Branch: ENABLED! Initializing OpenTelemetry exporter...\n")
	logger.Infof("[OTEL_DEBUG] Branch: ENABLED! Initializing OpenTelemetry exporter...")

	ctx := context.Background()
	var exporterOpts []otlptracegrpc.Option
	if otelCollectorAddress != "" {
		addr := otelCollectorAddress
		fmt.Printf("[OTEL_DEBUG] Flag otelCollectorAddress provided: %q\n", addr)
		if strings.HasPrefix(addr, "https://") {
			addr = strings.TrimPrefix(addr, "https://")
			fmt.Printf("[OTEL_DEBUG] Detected https:// prefix. Trimmed to: %q\n", addr)
		} else if strings.HasPrefix(addr, "http://") {
			addr = strings.TrimPrefix(addr, "http://")
			fmt.Printf("[OTEL_DEBUG] Detected http:// prefix. Trimmed to: %q\n", addr)
		} else {
			fmt.Printf("[OTEL_DEBUG] No scheme prefix detected. Using addr as-is: %q\n", addr)
		}
		fmt.Printf("[OTEL_DEBUG] Appending WithInsecure() and WithEndpoint(%q)\n", addr)
		exporterOpts = append(exporterOpts, otlptracegrpc.WithInsecure())
		exporterOpts = append(exporterOpts, otlptracegrpc.WithEndpoint(addr))
	} else {
		fmt.Printf("[OTEL_DEBUG] Flag otelCollectorAddress was empty, relying on defaults / OTEL_EXPORTER_OTLP_ENDPOINT env\n")
		fmt.Printf("[OTEL_DEBUG] Current OTEL_EXPORTER_OTLP_ENDPOINT env = %q\n", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
		fmt.Printf("[OTEL_DEBUG] Appending WithInsecure()\n")
		exporterOpts = append(exporterOpts, otlptracegrpc.WithInsecure())
	}

	fmt.Printf("[OTEL_DEBUG] Calling otlptracegrpc.New(ctx, exporterOpts...)\n")
	exp, err := otlptracegrpc.New(ctx, exporterOpts...)
	if err != nil {
		fmt.Printf("[OTEL_DEBUG] ERROR: Failed to create OTLP trace exporter: %v\n", err)
		logger.Fatalf("Failed to create OTLP trace exporter: %v", err)
	}
	fmt.Printf("[OTEL_DEBUG] Successfully created OTLP trace exporter!\n")

	fmt.Printf("[OTEL_DEBUG] Creating TracerProvider with WithBatcher(exp) and AlwaysSample()...\n")
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	propagator := propagation.TraceContext{}

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		fmt.Printf("[OTEL_DEBUG] OpenTelemetry internal error: %v\n", err)
		logger.Errorf("OpenTelemetry error: %v", err)
	}))

	shutdownFunc := func() {
		fmt.Printf("[OTEL_DEBUG] >>> shutdownFunc called! Flushing and shutting down TracerProvider...\n")
		logger.Infof("[OTEL_DEBUG] >>> shutdownFunc called! Flushing and shutting down TracerProvider...")
		if err := tp.Shutdown(context.Background()); err != nil {
			fmt.Printf("[OTEL_DEBUG] ERROR: Failed to shutdown TracerProvider: %v\n", err)
			logger.Errorf("Failed to shutdown TracerProvider: %v", err)
		} else {
			fmt.Printf("[OTEL_DEBUG] TracerProvider shutdown completed successfully!\n")
			logger.Infof("[OTEL_DEBUG] TracerProvider shutdown completed successfully!")
		}
	}
	fmt.Printf("[OTEL_DEBUG] SetupOpenTelemetry complete, returning TracerProvider\n")
	return tp, propagator, shutdownFunc
}
