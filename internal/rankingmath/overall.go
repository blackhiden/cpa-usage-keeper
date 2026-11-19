package rankingmath

import (
	"math"
	"math/big"
	"sort"
)

type Metrics struct {
	RequestCount       int64
	SuccessCount       int64
	FailureCount       int64
	InputTokens        int64
	CacheReadTokens    int64
	TotalTokens        int64
	TTFTSumMS          int64
	TTFTSampleCount    int64
	LatencySumMS       int64
	LatencySampleCount int64
	Peak5MRequestCount int64
	Peak5MTotalTokens  int64
}

func Eligible(row Metrics) bool {
	return row.RequestCount > 0 && row.SuccessCount+row.FailureCount > 0 && row.TotalTokens > 0 && row.InputTokens > 0 && row.TTFTSampleCount > 0 && row.TTFTSumMS > 0 && row.LatencySampleCount > 0 && row.LatencySumMS > 0 && row.Peak5MRequestCount > 0 && row.Peak5MTotalTokens > 0
}

func Score(rows []Metrics) []int64 {
	if len(rows) == 0 {
		return []int64{}
	}
	tokens, requests, cache, ttft, latency := make([]float64, len(rows)), make([]float64, len(rows)), make([]float64, len(rows)), make([]float64, len(rows)), make([]float64, len(rows))
	peakTokens, peakRequests := make([]float64, len(rows)), make([]float64, len(rows))
	cacheSamples, ttftSamples, latencySamples := make([]int64, len(rows)), make([]int64, len(rows)), make([]int64, len(rows))
	for i, row := range rows {
		tokens[i] = math.Log1p(float64(row.TotalTokens))
		requests[i] = math.Log1p(float64(row.RequestCount))
		cache[i] = float64(ScaledRatio(row.CacheReadTokens, row.InputTokens, 1_000_000))
		ttft[i] = float64(ScaledRatio(row.TTFTSumMS, row.TTFTSampleCount, 1_000))
		latency[i] = float64(ScaledRatio(row.LatencySumMS, row.LatencySampleCount, 1_000))
		peakTokens[i] = math.Log1p(float64(row.Peak5MTotalTokens) / 5)
		peakRequests[i] = math.Log1p(float64(row.Peak5MRequestCount) / 5)
		cacheSamples[i] = row.RequestCount
		ttftSamples[i] = row.TTFTSampleCount
		latencySamples[i] = row.LatencySampleCount
	}
	tokenUtility := robust(tokens, true)
	requestUtility := robust(requests, true)
	cacheUtility := shrink(robust(cache, true), cacheSamples)
	ttftUtility := shrink(robust(ttft, false), ttftSamples)
	latencyUtility := shrink(robust(latency, false), latencySamples)
	peakTokenUtility := robust(peakTokens, true)
	peakRequestUtility := robust(peakRequests, true)
	scores := make([]int64, len(rows))
	for i, row := range rows {
		workload := (20*tokenUtility[i] + 10*requestUtility[i]) / 30
		response := (15*ttftUtility[i] + 10*latencyUtility[i]) / 25
		peak := (3*peakTokenUtility[i] + 2*peakRequestUtility[i]) / 5
		scores[i] = overall([5]float64{workload, wilson(row.SuccessCount, row.FailureCount), response, cacheUtility[i], peak})
	}
	return scores
}

func ScaledRatio(numerator, denominator, scale int64) int64 {
	if denominator <= 0 || numerator <= 0 || scale <= 0 {
		return 0
	}
	value := new(big.Int).Mul(big.NewInt(numerator), big.NewInt(scale))
	value.Div(value, big.NewInt(denominator))
	if !value.IsInt64() {
		return math.MaxInt64
	}
	return value.Int64()
}
func shrink(values []float64, samples []int64) []float64 {
	out := make([]float64, len(values))
	for i, value := range values {
		n := math.Max(0, float64(samples[i]))
		w := n / (n + 100)
		out[i] = .5 + w*(value-.5)
	}
	return out
}
func robust(values []float64, higher bool) []float64 {
	center := median(values)
	dev := make([]float64, len(values))
	for i, value := range values {
		dev[i] = math.Abs(value - center)
	}
	scale := 1.4826 * median(dev)
	if scale <= 1e-12 {
		return percentiles(values, higher)
	}
	out := make([]float64, len(values))
	for i, value := range values {
		z := (value - center) / scale
		if !higher {
			z = -z
		}
		z = math.Max(-8, math.Min(8, z))
		out[i] = clamp(.5 * (1 + math.Erf(z/math.Sqrt2)))
	}
	return out
}
func percentiles(values []float64, higher bool) []float64 {
	type item struct {
		index int
		value float64
	}
	ordered := make([]item, len(values))
	for i, v := range values {
		ordered[i] = item{i, v}
	}
	sort.Slice(ordered, func(a, b int) bool {
		if ordered[a].value != ordered[b].value {
			return ordered[a].value < ordered[b].value
		}
		return ordered[a].index < ordered[b].index
	})
	out := make([]float64, len(values))
	for start := 0; start < len(ordered); {
		end := start + 1
		for end < len(ordered) && ordered[end].value == ordered[start].value {
			end++
		}
		p := float64(start+end) / (2 * float64(len(ordered)))
		if !higher {
			p = 1 - p
		}
		for i := start; i < end; i++ {
			out[ordered[i].index] = p
		}
		start = end
	}
	return out
}
func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	middle := len(ordered) / 2
	if len(ordered)%2 == 1 {
		return ordered[middle]
	}
	return (ordered[middle-1] + ordered[middle]) / 2
}
func wilson(successes, failures int64) float64 {
	n := float64(successes) + float64(failures)
	if n <= 0 {
		return 0
	}
	p := float64(successes) / n
	z := 1.959963984540054
	z2 := z * z
	center := p + z2/(2*n)
	margin := z * math.Sqrt((p*(1-p)+z2/(4*n))/n)
	return clamp((center - margin) / (1 + z2/n))
}
func overall(values [5]float64) int64 {
	weights := [5]float64{30, 30, 25, 10, 5}
	sum, weight := 0.0, 0.0
	for i, value := range values {
		q := math.Round(clamp(value)*1e6) / 1e6
		sum += weights[i] * math.Log(.1+.9*q)
		weight += weights[i]
	}
	geo := math.Exp(sum / weight)
	return int64(math.Round(clamp((geo-.1)/.9) * 100))
}
func clamp(value float64) float64 {
	if math.IsNaN(value) || value <= 0 {
		return 0
	}
	if value >= 1 {
		return 1
	}
	return value
}
