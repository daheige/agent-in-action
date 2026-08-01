package router

import (
	"math"
	"sort"
	"time"
)

// Priority 按配置顺序选择 Provider。
type Priority struct{}

// Name 返回策略名称 priority。
func (Priority) Name() string {
	return "priority"
}

// Order 保持候选 Provider 的原始顺序。
func (Priority) Order(candidates []Candidate, _ map[string]Stats) []Candidate {
	return append([]Candidate(nil), candidates...)
}

// CheapestFirst 按单价从低到高选择 Provider。
type CheapestFirst struct{}

// Name 返回策略名称 cheapest。
func (CheapestFirst) Name() string {
	return "cheapest"
}

// Order 按输入加输出单价对候选 Provider 排序。
func (CheapestFirst) Order(candidates []Candidate, _ map[string]Stats) []Candidate {
	ordered := append([]Candidate(nil), candidates...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return priceScore(ordered[i]) < priceScore(ordered[j])
	})
	return ordered
}

// priceScore 计算候选 Provider 的价格评分，未配置价格的评分为正无穷。
func priceScore(candidate Candidate) float64 {
	if !candidate.Pricing.Configured() {
		return math.Inf(1)
	}
	return candidate.Pricing.InputPer1M + candidate.Pricing.OutputPer1M
}

// LowestLatency 按历史延迟从低到高选择 Provider。
type LowestLatency struct{}

// Name 返回策略名称 latency。
func (LowestLatency) Name() string {
	return "latency"
}

// Order 按历史 P50 延迟或 LatencyHint 对候选 Provider 排序。
func (LowestLatency) Order(candidates []Candidate, stats map[string]Stats) []Candidate {
	ordered := append([]Candidate(nil), candidates...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return latencyScore(ordered[i], stats) < latencyScore(ordered[j], stats)
	})
	return ordered
}

// latencyScore 计算候选 Provider 的延迟评分，优先使用历史统计，否则使用 LatencyHint。
func latencyScore(candidate Candidate, stats map[string]Stats) time.Duration {
	if stat := stats[candidate.Provider.Name()]; stat.Count > 0 {
		return stat.P50
	}
	if candidate.LatencyHint > 0 {
		return candidate.LatencyHint
	}
	return time.Duration(math.MaxInt64)
}
