package engine

import (
	"sync/atomic"
	"time"
)

// State 状态快照（不可变）
type State struct {
	RoundID    string           `json:"round_id"`
	UpdatedAt  time.Time        `json:"updated_at"`
	Strategies []StrategyResult `json:"strategies"`
}

// StrategyResult 策略结果
type StrategyResult struct {
	Name        string   `json:"name"`
	Predictions []string `json:"predictions"`
}

// AtomicState 原子状态容器
type AtomicState struct {
	ptr atomic.Pointer[State]
}

// Get 无锁读取
func (a *AtomicState) Get() *State {
	return a.ptr.Load()
}

// Set 原子更新
func (a *AtomicState) Set(s *State) {
	a.ptr.Store(s)
}
