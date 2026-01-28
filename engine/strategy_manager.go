package engine

import (
	"benz-sniper/models"
	"encoding/json"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
)

// 状态常量
const (
	StatusVirtual = 0 // 虚盘/观望
	StatusReal    = 1 // 实盘/下注
)

// 进场/离场配置
const (
	EntryCondition = 2 // 连赢2把进场
	ExitCondition  = 1 // 连输1把离场
)

// StrategyState 策略状态
type StrategyState struct {
	Name           string   // 策略名称
	Status         int      // 0=虚盘, 1=实盘
	Predictions    []string // 当前预测
	VirtualStreak  int      // 虚盘连赢次数
	RealProfit     float64  // 实盘累计盈利
	LastPrediction []string // 上期预测（用于结算）
}

// HistoryRecord 历史记录
type HistoryRecord struct {
	RoundID       string    `json:"round_id"`       // 期号
	Strategy      string    `json:"strategy"`       // 策略名称
	Status        int       `json:"status"`         // 状态：0=虚盘, 1=实盘
	StatusText    string    `json:"status_text"`    // 状态文字
	Predictions   []string  `json:"predictions"`    // 预测内容
	Winners       []string  `json:"winners"`        // 获胜车型
	SpecialReward string    `json:"special_reward"` // 特殊奖项
	Result        string    `json:"result"`         // 结果：赢/输
	BetAmount     float64   `json:"bet_amount"`     // 下注金额
	Profit        float64   `json:"profit"`         // 本期盈亏
	TotalProfit   float64   `json:"total_profit"`   // 累计盈利
	Timestamp     time.Time `json:"timestamp"`      // 时间戳
}

// StrategyManager 策略管理器（带读写锁）
type StrategyManager struct {
	mu         sync.RWMutex
	db         *gorm.DB // 数据库连接
	strategies map[string]*StrategyState
	roundID    string
	updatedAt  time.Time
	betAmount  float64 // 下注金额配置
}

// NewStrategyManager 创建策略管理器实例
func NewStrategyManager(db *gorm.DB, betAmount float64) *StrategyManager {
	if betAmount <= 0 {
		betAmount = 100 // 默认100元
	}
	return &StrategyManager{
		db:         db,
		strategies: make(map[string]*StrategyState),
		updatedAt:  time.Now(),
		betAmount:  betAmount,
	}
}

// UpdatePredictions 更新策略预测（写锁）
func (m *StrategyManager) UpdatePredictions(roundID string, name string, predictions []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 获取或创建策略状态
	state, exists := m.strategies[name]
	if !exists {
		state = &StrategyState{
			Name:          name,
			Status:        StatusVirtual, // 初始为虚盘
			VirtualStreak: 0,
			RealProfit:    0.0,
		}
		m.strategies[name] = state
		log.Printf("🎯 初始化策略: %s (虚盘模式)", name)
	}

	// 保存上期预测用于结算
	state.LastPrediction = state.Predictions
	// 更新当前预测
	state.Predictions = predictions

	// 更新全局期号
	m.roundID = roundID
	m.updatedAt = time.Now()
}

// SettleRound 结算上一期盈亏（写锁）
func (m *StrategyManager) SettleRound(roundID string, winners []string, specialReward string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 遍历所有策略进行结算
	for _, state := range m.strategies {
		// 只有当上期有预测时才需要结算
		if len(state.LastPrediction) == 0 {
			continue
		}

		// 判断是否命中：预测中是否有获胜车型
		hitWinner := m.checkWin(state.LastPrediction, winners)

		// 记录本期盈亏（在状态更新前）
		profit := 0.0
		statusBeforeUpdate := state.Status
		betAmount := float64(len(state.LastPrediction)) * m.betAmount

		// 计算盈利（虚盘和实盘都需要计算，用于判定胜负）
		var won bool
		if hitWinner {
			// 计算真实盈利：(命中车型赔率 - 1) * 单注金额 - (未命中车型数量 * 单注金额)
			profit = m.calculateProfit(state.LastPrediction, winners)
			// 只有盈利 > 0 才算真正的赢，打平也算输
			won = profit > 0
		} else {
			// 没有命中，直接判定为输
			won = false
			profit = -betAmount
		}

		// 实盘状态需要记录实际盈亏
		if state.Status == StatusVirtual {
			// 虚盘不记录盈亏，但需要判定胜负
			profit = 0.0
		}

		// 根据当前状态执行流转逻辑
		m.updateStatus(state, won, profit)

		// 保存历史记录到数据库
		result := "输"
		if won {
			result = "赢"
		}

		// 序列化预测和获胜车型
		predictionsJSON, _ := json.Marshal(state.LastPrediction)
		winnersJSON, _ := json.Marshal(winners)

		history := models.StrategyHistory{
			RoundID:       roundID,
			Strategy:      state.Name,
			Status:        statusBeforeUpdate,
			Predictions:   string(predictionsJSON),
			Winners:       string(winnersJSON),
			SpecialReward: specialReward,
			Result:        result,
			BetAmount:     betAmount,
			Profit:        profit,
			TotalProfit:   state.RealProfit,
		}

		// 写入数据库
		if err := m.db.Create(&history).Error; err != nil {
			log.Printf("❌ 保存历史记录失败: %v", err)
		}

		// 清空上期预测
		state.LastPrediction = nil
	}
}

// calculateProfit 计算真实盈利
func (m *StrategyManager) calculateProfit(predictions []string, winners []string) float64 {
	// 创建获胜车型集合
	winnerSet := make(map[string]bool)
	for _, w := range winners {
		winnerSet[w] = true
	}

	// 找出命中的车型
	hitCar := ""
	for _, pred := range predictions {
		if winnerSet[pred] {
			hitCar = pred
			break
		}
	}

	if hitCar == "" {
		// 没有命中，理论上不应该到这里
		return -float64(len(predictions)) * m.betAmount
	}

	// 获取赔率
	odds, exists := REAL_ODDS[hitCar]
	if !exists {
		log.Printf("⚠️ 未找到车型 %s 的赔率，使用默认赔率10", hitCar)
		odds = 10
	}

	// 计算盈利
	// 盈利 = (赔率 - 1) * 单注金额 - (未命中车型数量 * 单注金额)
	winAmount := float64(odds-1) * m.betAmount
	loseAmount := float64(len(predictions)-1) * m.betAmount
	profit := winAmount - loseAmount

	return profit
}

// checkWin 检查预测是否命中
func (m *StrategyManager) checkWin(predictions []string, winners []string) bool {
	winnerSet := make(map[string]bool)
	for _, w := range winners {
		winnerSet[w] = true
	}

	// 只要预测中有任意一个车型命中就算赢
	for _, pred := range predictions {
		if winnerSet[pred] {
			return true
		}
	}
	return false
}

// updateStatus 状态流转核心逻辑（内部方法，调用者需持有锁）
func (m *StrategyManager) updateStatus(state *StrategyState, won bool, profit float64) {
	if state.Status == StatusVirtual {
		// 场景 A：虚盘状态
		if won {
			// 赢了：连赢次数加1
			state.VirtualStreak++
			log.Printf("🎉 [%s] 虚盘赢 | 连赢: %d/%d", state.Name, state.VirtualStreak, EntryCondition)

			// 判断进场：达到进场条件
			if state.VirtualStreak >= EntryCondition {
				state.Status = StatusReal
				log.Printf("🚀 [%s] 表现优异，切换至实盘模式！", state.Name)
			}
		} else {
			// 输了：连赢次数归零
			if state.VirtualStreak > 0 {
				log.Printf("😔 [%s] 虚盘输 | 连赢归零: %d -> 0", state.Name, state.VirtualStreak)
			}
			state.VirtualStreak = 0
		}
	} else {
		// 场景 B：实盘状态
		if won {
			state.RealProfit += profit
			log.Printf("💰 [%s] 实盘赢 +%.2f | 累计盈利: %.2f", state.Name, profit, state.RealProfit)
		} else {
			state.RealProfit += profit
			log.Printf("⚠️ [%s] 实盘输 %.2f | 累计盈利: %.2f", state.Name, profit, state.RealProfit)

			// 触发止损：切换回虚盘
			state.Status = StatusVirtual
			state.VirtualStreak = 0
			log.Printf("🛑 [%s] 实盘止损，退回观望模式", state.Name)
		}
	}
}

// GetState 获取状态快照（读锁）
func (m *StrategyManager) GetState() *State {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]StrategyResult, 0, len(m.strategies))
	for _, state := range m.strategies {
		statusText := "虚盘观望"
		if state.Status == StatusReal {
			statusText = "实盘下注"
		}

		// 从数据库计算该策略的实盘总盈利
		realProfit := m.GetStrategyRealProfit(state.Name)

		results = append(results, StrategyResult{
			Name:          state.Name,
			Predictions:   state.Predictions,
			Status:        state.Status,
			StatusText:    statusText,
			VirtualStreak: state.VirtualStreak,
			RealProfit:    realProfit, // 使用从数据库计算的值
		})
	}

	return &State{
		RoundID:    m.roundID,
		UpdatedAt:  m.updatedAt,
		Strategies: results,
	}
}

// GetRealPredictions 只返回实盘策略（读锁）
func (m *StrategyManager) GetRealPredictions() []StrategyResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]StrategyResult, 0)
	for _, state := range m.strategies {
		// 只返回实盘状态的策略
		if state.Status == StatusReal {
			statusText := "实盘下注"
			// 从数据库计算该策略的实盘总盈利
			realProfit := m.GetStrategyRealProfit(state.Name)
			
			results = append(results, StrategyResult{
				Name:          state.Name,
				Predictions:   state.Predictions,
				Status:        state.Status,
				StatusText:    statusText,
				VirtualStreak: state.VirtualStreak,
				RealProfit:    realProfit, // 使用从数据库计算的值
			})
		}
	}

	return results
}

// GetHistory 获取历史记录（从数据库）
func (m *StrategyManager) GetHistory(limit int) []HistoryRecord {
	if limit <= 0 {
		limit = 50
	}

	var dbRecords []models.StrategyHistory
	err := m.db.Order("created_at DESC, id DESC").Limit(limit).Find(&dbRecords).Error
	if err != nil {
		log.Printf("❌ 查询历史记录失败: %v", err)
		return []HistoryRecord{}
	}

	// 转换为 HistoryRecord 格式
	records := make([]HistoryRecord, 0, len(dbRecords))
	for _, dbRecord := range dbRecords {
		var predictions []string
		var winners []string
		json.Unmarshal([]byte(dbRecord.Predictions), &predictions)
		json.Unmarshal([]byte(dbRecord.Winners), &winners)

		statusText := "虚盘观望"
		if dbRecord.Status == StatusReal {
			statusText = "实盘下注"
		}

		timestamp := time.Now()
		if dbRecord.CreatedAt != nil {
			timestamp = *dbRecord.CreatedAt
		}

		records = append(records, HistoryRecord{
			RoundID:       dbRecord.RoundID,
			Strategy:      dbRecord.Strategy,
			Status:        dbRecord.Status,
			StatusText:    statusText,
			Predictions:   predictions,
			Winners:       winners,
			SpecialReward: dbRecord.SpecialReward,
			Result:        dbRecord.Result,
			BetAmount:     dbRecord.BetAmount,
			Profit:        dbRecord.Profit,
			TotalProfit:   dbRecord.TotalProfit,
			Timestamp:     timestamp,
		})
	}

	return records
}

// ClearHistory 清空历史记录（从数据库）
func (m *StrategyManager) ClearHistory() {
	err := m.db.Where("1 = 1").Delete(&models.StrategyHistory{}).Error
	if err != nil {
		log.Printf("❌ 清空历史记录失败: %v", err)
	} else {
		log.Println("📝 历史记录已清空")
	}
}

// GetTotalRealProfit 计算所有实盘注单的总盈利（从数据库）
func (m *StrategyManager) GetTotalRealProfit() float64 {
	var totalProfit float64
	
	// 查询所有实盘状态的历史记录，累计盈利
	err := m.db.Model(&models.StrategyHistory{}).
		Where("status = ?", StatusReal).
		Select("COALESCE(SUM(profit), 0)").
		Scan(&totalProfit).Error
	
	if err != nil {
		log.Printf("❌ 计算实盘总盈利失败: %v", err)
		return 0.0
	}
	
	return totalProfit
}

// GetStrategyRealProfit 计算单个策略的实盘总盈利（从数据库）
func (m *StrategyManager) GetStrategyRealProfit(strategyName string) float64 {
	var totalProfit float64
	
	// 查询指定策略的所有实盘状态的历史记录，累计盈利
	err := m.db.Model(&models.StrategyHistory{}).
		Where("strategy = ? AND status = ?", strategyName, StatusReal).
		Select("COALESCE(SUM(profit), 0)").
		Scan(&totalProfit).Error
	
	if err != nil {
		log.Printf("❌ 计算策略 %s 实盘总盈利失败: %v", strategyName, err)
		return 0.0
	}
	
	return totalProfit
}
