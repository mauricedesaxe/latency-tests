package latency_simulations

import (
	"time"

	"github.com/montanaflynn/stats"
)

const (
	productCount          = 1_000
	reviewCountPerProduct = 10
	queryCount            = 100
)

type Simulation struct {
	Read1  LatencyStats
	Read2  LatencyStats
	Write1 LatencyStats
}

type LatencyStats struct {
	MedianLatency float64
	P10Latency    float64
	P25Latency    float64
	P75Latency    float64
	P90Latency    float64
	P95Latency    float64
	Count         float64
}

func calculateLatencyStatsNs(latencies []time.Duration) (LatencyStats, error) {
	durations := make([]float64, len(latencies))
	for i, d := range latencies {
		durations[i] = float64(d.Nanoseconds())
	}

	medianLatency, err := stats.Median(stats.Float64Data(durations))
	if err != nil {
		return LatencyStats{
			MedianLatency: medianLatency,
		}, err
	}

	p10Latency, err := stats.Percentile(stats.Float64Data(durations), 10)
	if err != nil {
		return LatencyStats{
			MedianLatency: medianLatency,
			P10Latency:    p10Latency,
		}, err
	}

	p25Latency, err := stats.Percentile(stats.Float64Data(durations), 25)
	if err != nil {
		return LatencyStats{
			MedianLatency: medianLatency,
			P10Latency:    p10Latency,
			P25Latency:    p25Latency,
		}, err
	}

	p75Latency, err := stats.Percentile(stats.Float64Data(durations), 75)
	if err != nil {
		return LatencyStats{
			MedianLatency: medianLatency,
			P10Latency:    p10Latency,
			P25Latency:    p25Latency,
			P75Latency:    p75Latency,
		}, err
	}

	p90Latency, err := stats.Percentile(stats.Float64Data(durations), 90)
	if err != nil {
		return LatencyStats{
			MedianLatency: medianLatency,
			P10Latency:    p10Latency,
			P25Latency:    p25Latency,
			P75Latency:    p75Latency,
			P90Latency:    p90Latency,
		}, err
	}

	p95Latency, err := stats.Percentile(stats.Float64Data(durations), 95)
	if err != nil {
		return LatencyStats{
			MedianLatency: medianLatency,
			P10Latency:    p10Latency,
			P25Latency:    p25Latency,
			P75Latency:    p75Latency,
			P90Latency:    p90Latency,
		}, err
	}

	return LatencyStats{
		MedianLatency: medianLatency,
		P10Latency:    p10Latency,
		P25Latency:    p25Latency,
		P75Latency:    p75Latency,
		P90Latency:    p90Latency,
		P95Latency:    p95Latency,
		Count:         float64(len(latencies)),
	}, nil
}
