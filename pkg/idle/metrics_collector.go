package idle

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// MetricsCollector collects and analyzes CloudWatch metrics for idle detection
type MetricsCollector struct {
	cwClient *cloudwatch.Client
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(awsConfig aws.Config) *MetricsCollector {
	return &MetricsCollector{
		cwClient: cloudwatch.NewFromConfig(awsConfig),
	}
}

// IsInstanceIdle checks if an instance has been idle for the specified duration
// based on CPU, network, and other metrics
func (mc *MetricsCollector) IsInstanceIdle(ctx context.Context, instanceID string, schedule *Schedule) (bool, error) {
	now := time.Now()
	idleDuration := time.Duration(schedule.IdleMinutes) * time.Minute

	// Check CPU usage
	cpuThreshold := schedule.CPUThreshold
	if cpuThreshold == 0 {
		cpuThreshold = 5.0 // Default 5% CPU threshold
	}

	avgCPU, err := mc.getAverageCPU(ctx, instanceID, idleDuration, now)
	if err != nil {
		return false, fmt.Errorf("failed to get CPU metrics: %w", err)
	}

	// If CPU is above threshold, not idle
	if avgCPU > cpuThreshold {
		return false, nil
	}

	// Check network activity
	networkThreshold := schedule.NetworkThreshold
	if networkThreshold == 0 {
		networkThreshold = 1000.0 // Default 1KB/s threshold
	}

	avgNetwork, err := mc.getAverageNetworkBytes(ctx, instanceID, idleDuration, now)
	if err != nil {
		return false, fmt.Errorf("failed to get network metrics: %w", err)
	}

	// If network is above threshold, not idle
	if avgNetwork > networkThreshold {
		return false, nil
	}

	// Both CPU and network are below thresholds for the idle duration
	return true, nil
}

// getAverageCPU gets the average CPU utilization over a period
func (mc *MetricsCollector) getAverageCPU(ctx context.Context, instanceID string, duration time.Duration, endTime time.Time) (float64, error) {
	startTime := endTime.Add(-duration)

	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String("CPUUtilization"),
		Dimensions: []types.Dimension{
			{
				Name:  aws.String("InstanceId"),
				Value: aws.String(instanceID),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(300), // 5-minute periods
		Statistics: []types.Statistic{types.StatisticAverage},
	}

	result, err := mc.cwClient.GetMetricStatistics(ctx, input)
	if err != nil {
		return 0, err
	}

	if len(result.Datapoints) == 0 {
		// No datapoints means instance might be stopped or just launched
		return 0, nil
	}

	// Calculate average across all datapoints
	var sum float64
	for _, dp := range result.Datapoints {
		if dp.Average != nil {
			sum += *dp.Average
		}
	}

	return sum / float64(len(result.Datapoints)), nil
}

// getAverageNetworkBytes gets the average network bytes (in+out) per second over a period
func (mc *MetricsCollector) getAverageNetworkBytes(ctx context.Context, instanceID string, duration time.Duration, endTime time.Time) (float64, error) {
	startTime := endTime.Add(-duration)

	// Get NetworkIn
	networkIn, err := mc.getNetworkMetric(ctx, instanceID, "NetworkIn", startTime, endTime)
	if err != nil {
		return 0, err
	}

	// Get NetworkOut
	networkOut, err := mc.getNetworkMetric(ctx, instanceID, "NetworkOut", startTime, endTime)
	if err != nil {
		return 0, err
	}

	// Return average bytes per second
	return (networkIn + networkOut) / duration.Seconds(), nil
}

// getNetworkMetric gets a specific network metric
func (mc *MetricsCollector) getNetworkMetric(ctx context.Context, instanceID, metricName string, startTime, endTime time.Time) (float64, error) {
	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String(metricName),
		Dimensions: []types.Dimension{
			{
				Name:  aws.String("InstanceId"),
				Value: aws.String(instanceID),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(300), // 5-minute periods
		Statistics: []types.Statistic{types.StatisticSum},
	}

	result, err := mc.cwClient.GetMetricStatistics(ctx, input)
	if err != nil {
		return 0, err
	}

	if len(result.Datapoints) == 0 {
		return 0, nil
	}

	// Sum all datapoints
	var total float64
	for _, dp := range result.Datapoints {
		if dp.Sum != nil {
			total += *dp.Sum
		}
	}

	return total, nil
}

// GetInstanceMetrics retrieves comprehensive metrics for an instance
func (mc *MetricsCollector) GetInstanceMetrics(ctx context.Context, instanceID string, duration time.Duration) (*InstanceMetrics, error) {
	now := time.Now()

	// Get CPU
	avgCPU, err := mc.getAverageCPU(ctx, instanceID, duration, now)
	if err != nil {
		return nil, err
	}

	// Get Network
	avgNetwork, err := mc.getAverageNetworkBytes(ctx, instanceID, duration, now)
	if err != nil {
		return nil, err
	}

	return &InstanceMetrics{
		InstanceID:        instanceID,
		AverageCPU:        avgCPU,
		AverageNetworkBPS: avgNetwork,
		Period:            duration,
		CollectedAt:       now,
	}, nil
}

// InstanceMetrics represents collected instance metrics
type InstanceMetrics struct {
	InstanceID        string
	AverageCPU        float64
	AverageNetworkBPS float64
	Period            time.Duration
	CollectedAt       time.Time
}
