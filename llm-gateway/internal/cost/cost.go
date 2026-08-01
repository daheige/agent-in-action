package cost

import (
	"fmt"
	"sync"

	"agent-in-action/llm-gateway/internal/llm"
)

// Pricing 表示每百万 token 的输入/输出单价。
type Pricing struct {
	InputPer1M  float64
	OutputPer1M float64
	Currency    string
}

// Validate 校验 Pricing 中的单价是否为非负数。
func (p Pricing) Validate() error {
	if p.InputPer1M < 0 || p.OutputPer1M < 0 {
		return fmt.Errorf("token 单价不能为负数")
	}
	return nil
}

// Configured 判断 Pricing 是否已配置（至少一个单价大于 0）。
func (p Pricing) Configured() bool {
	return p.InputPer1M > 0 || p.OutputPer1M > 0
}

// Estimate 根据使用量与单价估算一次调用的费用。
func Estimate(usage llm.Usage, pricing Pricing) float64 {
	return float64(usage.InputTokens)/1e6*pricing.InputPer1M +
		float64(usage.OutputTokens)/1e6*pricing.OutputPer1M
}

// Snapshot 表示某一时刻累计的使用量与费用快照。
type Snapshot struct {
	Usage llm.Usage
	Cost  float64
}

// Accumulator 线程安全地累计使用量与费用。
type Accumulator struct {
	mu    sync.Mutex
	usage llm.Usage
	cost  float64
}

// Add 将一次调用的使用量与对应单价累加到 Accumulator。
func (a *Accumulator) Add(usage llm.Usage, pricing Pricing) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.usage.InputTokens += usage.InputTokens
	a.usage.OutputTokens += usage.OutputTokens
	a.cost += Estimate(usage, pricing)
}

// Snapshot 返回当前累计使用量与费用的快照。
func (a *Accumulator) Snapshot() Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	return Snapshot{Usage: a.usage, Cost: a.cost}
}
