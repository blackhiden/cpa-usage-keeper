package ranking

import (
	"cpa-usage-keeper/internal/rankingmath"
)

const (
	localOverallGeometricEpsilon = 0.1
	localOverallShrinkagePrior   = 100.0
	localOverallScoreScale       = 100.0
	localOverallUtilityScale     = 1_000_000.0
)

func localOverallEligible(row localRankingPopulationRow) bool {
	return rankingmath.Eligible(rankingmath.Metrics{
		RequestCount: row.RequestCount, SuccessCount: row.SuccessCount, FailureCount: row.FailureCount,
		InputTokens: row.InputTokens, CacheReadTokens: row.CacheReadTokens, TotalTokens: row.TotalTokens,
		TTFTSumMS: row.TTFTSumMS, TTFTSampleCount: row.TTFTSampleCount,
		LatencySumMS: row.LatencySumMS, LatencySampleCount: row.LatencySampleCount,
		Peak5MRequestCount: row.Peak5MRequestCount, Peak5MTotalTokens: row.Peak5MTotalTokens,
	})
}

func scoreLocalOverallPopulation(rows []localRankingPopulationRow) []int64 {
	metrics := make([]rankingmath.Metrics, len(rows))
	for i, row := range rows {
		metrics[i] = rankingmath.Metrics{RequestCount: row.RequestCount, SuccessCount: row.SuccessCount, FailureCount: row.FailureCount, InputTokens: row.InputTokens, CacheReadTokens: row.CacheReadTokens, TotalTokens: row.TotalTokens, TTFTSumMS: row.TTFTSumMS, TTFTSampleCount: row.TTFTSampleCount, LatencySumMS: row.LatencySumMS, LatencySampleCount: row.LatencySampleCount, Peak5MRequestCount: row.Peak5MRequestCount, Peak5MTotalTokens: row.Peak5MTotalTokens}
	}
	return rankingmath.Score(metrics)
}

func localOverallScoreExplanation() *ScoreExplanation {
	return &ScoreExplanation{Version: 2, Texts: map[string]string{
		"en":    "Local overall score combines workload 30% (tokens 20%, requests 10%), reliability 30% (95% Wilson lower bound), response 25% (TTFT 15%, latency 10%), cache efficiency 10%, and peak capacity 5% (TPM 3%, RPM 2%). It is normalized only among API keys in this Keeper and period, and is shown on a 0–100 scale.",
		"zh":    "本地综合分由使用规模 30%（Token 20%、请求数 10%）、可靠性 30%（成功率的 95% Wilson 下界）、响应体验 25%（首字延迟 15%、总延迟 10%）、缓存效率 10% 和峰值能力 5%（TPM 3%、RPM 2%）组成。分数仅在当前 Keeper、当前周期的 API Key 之间归一化，范围为 0–100。",
		"zh-TW": "本地綜合分由使用規模 30%（Token 20%、請求數 10%）、可靠性 30%（成功率的 95% Wilson 下界）、回應體驗 25%（首字延遲 15%、總延遲 10%）、快取效率 10% 和峰值能力 5%（TPM 3%、RPM 2%）組成。分數僅在目前 Keeper、目前週期的 API Key 之間正規化，範圍為 0–100。",
	}}
}
