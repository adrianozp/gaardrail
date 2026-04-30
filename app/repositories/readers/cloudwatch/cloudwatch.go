package cloudwatch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/rs/zerolog/log"
)

type Reader struct {
	client     *cloudwatch.Client
	cfg        config.CloudWatch
	mappings   map[string]string
	dimensions []cwtypes.Dimension
}

func New(cfg config.Config) (*Reader, error) {
	awscfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.CloudWatch.Region),
	)
	if err != nil {
		return nil, err
	}
	dims := make([]cwtypes.Dimension, 0, len(cfg.CloudWatch.Dimensions))
	for k, v := range cfg.CloudWatch.Dimensions {
		dims = append(dims, cwtypes.Dimension{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}
	return &Reader{
		client:     cloudwatch.NewFromConfig(awscfg),
		cfg:        cfg.CloudWatch,
		mappings:   cfg.MetricsPoller.Mappings,
		dimensions: dims,
	}, nil
}

func (r *Reader) Read(ctx context.Context) (entities.Metrics, error) {
	metrics := make(map[string]float64)
	var measureTime time.Time

	for expr, name := range r.mappings {
		val, t, err := r.queryMetric(ctx, expr)
		if err != nil {
			return entities.Metrics{}, fmt.Errorf("cloudwatch: %s: %w", name, err)
		}
		metrics[name] = val
		measureTime = t
	}

	log.Debug().Msgf("cloudwatch: read metrics: %v", metrics)
	return entities.Metrics{
		MeasureTime: measureTime,
		Metrics:     metrics,
	}, nil
}

func (r *Reader) queryMetric(ctx context.Context, expr string) (float64, time.Time, error) {
	parts := strings.SplitN(expr, "/", 3)
	if len(parts) != 3 {
		return 0, time.Time{}, fmt.Errorf("invalid mapping %q: want Namespace/MetricName/Stat", expr)
	}
	namespace, metricName, stat := parts[0], parts[1], parts[2]

	now := time.Now()
	startTime := now.Add(-time.Duration(r.cfg.Period*2) * time.Second)

	out, err := r.client.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricName),
		Statistics: []cwtypes.Statistic{cwtypes.Statistic(stat)},
		Period:     aws.Int32(r.cfg.Period),
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(now),
		Dimensions: r.dimensions,
	})
	if err != nil {
		return 0, time.Time{}, err
	}
	if len(out.Datapoints) == 0 {
		return 0, time.Time{}, fmt.Errorf("no datapoints for %q", expr)
	}

	latest := out.Datapoints[0]
	for _, dp := range out.Datapoints[1:] {
		if dp.Timestamp.After(*latest.Timestamp) {
			latest = dp
		}
	}

	val := statValue(latest, stat)
	return val, *latest.Timestamp, nil
}

func statValue(dp cwtypes.Datapoint, stat string) float64 {
	switch stat {
	case "Average":
		if dp.Average != nil {
			return *dp.Average
		}
	case "Sum":
		if dp.Sum != nil {
			return *dp.Sum
		}
	case "Maximum":
		if dp.Maximum != nil {
			return *dp.Maximum
		}
	case "Minimum":
		if dp.Minimum != nil {
			return *dp.Minimum
		}
	case "SampleCount":
		if dp.SampleCount != nil {
			return *dp.SampleCount
		}
	}
	return 0
}
