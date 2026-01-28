package models

import "time"

// GameRound 游戏期数表
type GameRound struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Timestamp   int64      `gorm:"column:timestamp" json:"timestamp"`
	RoundID     string     `gorm:"column:round_id;type:varchar(50);uniqueIndex" json:"round_id"`
	ResultType  int        `gorm:"column:result_type" json:"result_type"`
	ResultName  string     `gorm:"column:result_name" json:"result_name"`
	TotalInput  float64    `gorm:"column:total_input" json:"total_input"`
	TotalOutput float64    `gorm:"column:total_output" json:"total_output"`
	HouseNet    float64    `gorm:"column:house_net" json:"house_net"`
	CreatedAt   *time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (GameRound) TableName() string {
	return "game_rounds"
}

// GameWinner 获胜项表
type GameWinner struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	RoundID    string     `gorm:"column:round_id;type:varchar(50);index" json:"round_id"`
	WinnerID   int        `gorm:"column:winner_id" json:"winner_id"`
	WinnerName string     `gorm:"column:winner_name" json:"winner_name"`
	Position   int        `gorm:"column:position" json:"position"`
	CreatedAt  *time.Time `gorm:"column:created_at" json:"created_at"`
}

func (GameWinner) TableName() string {
	return "game_winners"
}

// BetDistribution 投注分布表
type BetDistribution struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	RoundID    string     `gorm:"column:round_id;type:varchar(50);index" json:"round_id"`
	OptionID   int        `gorm:"column:option_id" json:"option_id"`
	OptionName string     `gorm:"column:option_name" json:"option_name"`
	Odds       float64    `gorm:"column:odds" json:"odds"`
	Amount     float64    `gorm:"column:amount" json:"amount"`
	CreatedAt  *time.Time `gorm:"column:created_at" json:"created_at"`
}

func (BetDistribution) TableName() string {
	return "bet_distribution"
}

// StrategyHistory 策略下注历史表
type StrategyHistory struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	RoundID       string     `gorm:"column:round_id;type:varchar(50);index" json:"round_id"`
	Strategy      string     `gorm:"column:strategy;type:varchar(50)" json:"strategy"`
	Status        int        `gorm:"column:status" json:"status"`                            // 0=虚盘, 1=实盘
	Predictions   string     `gorm:"column:predictions;type:text" json:"predictions"`        // JSON 格式
	Winners       string     `gorm:"column:winners;type:text" json:"winners"`                // JSON 格式
	SpecialReward string     `gorm:"column:special_reward;type:varchar(50)" json:"special_reward"` // 特殊奖项
	Result        string     `gorm:"column:result;type:varchar(10)" json:"result"`           // 赢/输
	BetAmount     float64    `gorm:"column:bet_amount" json:"bet_amount"`                    // 下注金额
	Profit        float64    `gorm:"column:profit" json:"profit"`                            // 本期盈亏
	TotalProfit   float64    `gorm:"column:total_profit" json:"total_profit"`                // 累计盈利
	CreatedAt     *time.Time `gorm:"column:created_at" json:"created_at"`
}

func (StrategyHistory) TableName() string {
	return "strategy_history"
}
