package cli

import (
	"context"
	"errors"
	"fmt"

	grpcZap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/krzko/otelgen/internal/metrics"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric/global"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var generateMetricsCounterCommand = &cli.Command{
	Name:        "counter",
	Usage:       "generate metrics of type counter",
	Description: "Counter demonstrates how to measure non-decreasing numbers",
	Aliases:     []string{"c"},
	Action: func(c *cli.Context) error {
		return generateMetricsCounterAction(c)
	},
}

func generateMetricsCounterAction(c *cli.Context) error {
	if c.String("otel-exporter-otlp-endpoint") == "" {
		return errors.New("'otel-exporter-otlp-endpoint' must be set")
	}

	metricsCfg := &metrics.Config{
		Endpoint:    c.String("otel-exporter-otlp-endpoint"),
		NumMetrics:  1,
		ServiceName: c.String("service-name"),
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to obtain logger: %v", err))
	}
	defer logger.Sync()

	grpcZap.ReplaceGrpcLoggerV2(logger.WithOptions(
		zap.AddCallerSkip(3),
	))

	grpcExpOpt := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(metricsCfg.Endpoint),
		otlpmetricgrpc.WithDialOption(
			grpc.WithBlock(),
		),
	}

	httpExpOpt := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(metricsCfg.Endpoint),
	}

	if c.Bool("insecure") {
		grpcExpOpt = append(grpcExpOpt, otlpmetricgrpc.WithInsecure())
		httpExpOpt = append(httpExpOpt, otlpmetrichttp.WithInsecure())
	}

	var exp *otlpmetric.Exporter
	if c.String("protocl") == "http" {
		logger.Info("starting HTTP exporter")
		exp, err = otlpmetrichttp.New(context.Background(), httpExpOpt...)
	} else {
		logger.Info("starting gRPC exporter")
		exp, err = otlpmetricgrpc.New(context.Background(), grpcExpOpt...)
	}

	if err != nil {
		logger.Error("failed to obtain OTLP exporter", zap.Error(err))
		return err
	}
	defer func() {
		logger.Info("stopping the exporter")
		if err = exp.Shutdown(context.Background()); err != nil {
			logger.Error("failed to stop the exporter", zap.Error(err))
			return
		}
	}()

	ctx := context.Background()
	logger.Info("Starting metrics generation")

	var meter = global.MeterProvider().Meter(c.String("service-name"))

	metrics.Run(ctx, exp, meter, metricsCfg, logger)

	metrics.Counter(ctx, meter, metricsCfg, logger)

	return nil
}
