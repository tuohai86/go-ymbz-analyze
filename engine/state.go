package engine

import (
	"sync/atomic"
	"time"
)

// State 状态快照（不可变）
type State struct {
	RoundID       string           `json:"round_id"`
	UpdatedAt     time.Time        `json:"updated_at"`
	SystemUptime  int              `json:"system_uptime"` // 系统运行时长（秒）
	Strategies    []StrategyResult `json:"strategies"`
}

// StrategyResult 策略结果
type StrategyResult struct {
	Name          string   `json:"name"`
	Predictions   []string `json:"predictions"`
	Status        int      `json:"status"`          // 0=虚盘, 1=实盘
	StatusText    string   `json:"status_text"`     // 状态文字
	VirtualStreak int      `json:"virtual_streak"`  // 虚盘连赢次数
	RealProfit    float64  `json:"real_profit"`     // 实盘累计盈利
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
