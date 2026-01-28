package engine

import (
	"benz-sniper/models"
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"
)

// 状态常量
const (
	StatusVirtual = 0 // 虚盘/观望
	StatusReal    = 1 // 实盘/下注
)

// StrategyConfig 策略配置（可动态修改）
type StrategyConfig struct {
	EntryCondition     int     `json:"entry_condition"`      // 连赢几把进场
	ExitCondition      int     `json:"exit_condition"`       // 连输几把离场
	Hot3BetAmount      float64 `json:"hot3_bet_amount"`      // 热门3码下注金额
	Balanced4BetAmount float64 `json:"balanced4_bet_amount"` // 均衡4码下注金额
	Hot3Enabled        bool    `json:"hot3_enabled"`         // 热门3码启用（用于预测接口）
	Balanced4Enabled   bool    `json:"balanced4_enabled"`    // 均衡4码启用（用于预测接口）
}

// DefaultStrategyConfig 返回默认配置
func DefaultStrategyConfig() StrategyConfig {
	return StrategyConfig{
		EntryCondition:     2,     // 连赢2把进场
		ExitCondition:      1,     // 连输1把离场
		Hot3BetAmount:      100.0, // 热门3码默认100元
		Balanced4BetAmount: 100.0, // 均衡4码默认100元
		Hot3Enabled:        true,  // 默认启用热门3码
		Balanced4Enabled:   true,  // 默认启用均衡4码
	}
}

// StrategyState 策略状态
type StrategyState struct {
	Name              string              // 策略名称
	Status            int                 // 0=虚盘, 1=实盘
	Predictions       []string            // 当前预测
	VirtualStreak     int                 // 虚盘连赢次数
	RealProfit        float64             // 实盘累计盈利
	RoundPredictions  map[string][]string // 每期的预测（期号 -> 预测列表）
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
	startTime  time.Time      // 系统启动时间
	config     StrategyConfig // 策略配置
}

// NewStrategyManager 创建策略管理器实例
func NewStrategyManager(db *gorm.DB) *StrategyManager {
	now := time.Now()
	m := &StrategyManager{
		db:         db,
		strategies: make(map[string]*StrategyState),
		updatedAt:  now,
		startTime:  now,                    // 记录启动时间
		config:     DefaultStrategyConfig(), // 使用默认配置
	}
	
	// 从数据库加载配置
	m.loadConfigFromDB()
	
	return m
}

// loadConfigFromDB 从数据库加载配置
func (m *StrategyManager) loadConfigFromDB() {
	var dbConfig models.SystemConfig
	result := m.db.First(&dbConfig)
	
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// 数据库中没有配置记录，创建默认配置
			dbConfig = models.SystemConfig{
				EntryCondition:     m.config.EntryCondition,
				ExitCondition:      m.config.ExitCondition,
				Hot3BetAmount:      m.config.Hot3BetAmount,
				Balanced4BetAmount: m.config.Balanced4BetAmount,
				Hot3Enabled:        m.config.Hot3Enabled,
				Balanced4Enabled:   m.config.Balanced4Enabled,
			}
			if err := m.db.Create(&dbConfig).Error; err != nil {
				log.Printf("❌ 创建默认配置失败: %v", err)
			} else {
				log.Println("✅ 已创建并加载默认配置")
			}
		} else {
			log.Printf("❌ 加载配置失败: %v", result.Error)
		}
		return
	}
	
	// 从数据库加载配置
	m.config = StrategyConfig{
		EntryCondition:     dbConfig.EntryCondition,
		ExitCondition:      dbConfig.ExitCondition,
		Hot3BetAmount:      dbConfig.Hot3BetAmount,
		Balanced4BetAmount: dbConfig.Balanced4BetAmount,
		Hot3Enabled:        dbConfig.Hot3Enabled,
		Balanced4Enabled:   dbConfig.Balanced4Enabled,
	}
	
	log.Println("✅ 已从数据库加载配置")
}

// getStrategyBetAmount 根据策略名称获取下注金额
func (m *StrategyManager) getStrategyBetAmount(strategyName string) float64 {
	if strategyName == "热门3码" {
		return m.config.Hot3BetAmount
	}
	return m.config.Balanced4BetAmount
}

// GetConfig 获取当前配置（读锁）
func (m *StrategyManager) GetConfig() StrategyConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// UpdateConfig 更新配置（写锁，支持部分更新）
func (m *StrategyManager) UpdateConfig(newConfig StrategyConfig) StrategyConfig {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 只更新非零值字段（支持部分更新）
	if newConfig.EntryCondition > 0 {
		m.config.EntryCondition = newConfig.EntryCondition
	}
	if newConfig.ExitCondition > 0 {
		m.config.ExitCondition = newConfig.ExitCondition
	}
	if newConfig.Hot3BetAmount > 0 {
		m.config.Hot3BetAmount = newConfig.Hot3BetAmount
	}
	if newConfig.Balanced4BetAmount > 0 {
		m.config.Balanced4BetAmount = newConfig.Balanced4BetAmount
	}

	// 布尔值需要特殊处理（总是更新）
	m.config.Hot3Enabled = newConfig.Hot3Enabled
	m.config.Balanced4Enabled = newConfig.Balanced4Enabled

	log.Printf("📝 配置已更新: 进场条件=%d, 离场条件=%d, 热门3码金额=%.2f, 均衡4码金额=%.2f, 热门3码启用=%v, 均衡4码启用=%v",
		m.config.EntryCondition, m.config.ExitCondition,
		m.config.Hot3BetAmount, m.config.Balanced4BetAmount,
		m.config.Hot3Enabled, m.config.Balanced4Enabled)

	// 保存配置到数据库
	m.saveConfigToDB()

	return m.config
}

// saveConfigToDB 保存配置到数据库（调用前需要持有锁）
func (m *StrategyManager) saveConfigToDB() {
	// 更新数据库中的配置（ID=1）
	dbConfig := models.SystemConfig{
		ID:                 1,
		EntryCondition:     m.config.EntryCondition,
		ExitCondition:      m.config.ExitCondition,
		Hot3BetAmount:      m.config.Hot3BetAmount,
		Balanced4BetAmount: m.config.Balanced4BetAmount,
		Hot3Enabled:        m.config.Hot3Enabled,
		Balanced4Enabled:   m.config.Balanced4Enabled,
	}
	
	// 使用 Save 方法（存在则更新，不存在则创建）
	if err := m.db.Save(&dbConfig).Error; err != nil {
		log.Printf("❌ 保存配置到数据库失败: %v", err)
	} else {
		log.Println("✅ 配置已保存到数据库")
	}
}

// UpdatePredictions 更新策略预测（写锁）
// currentRoundID: 当前已开奖的期号（比如06）
// targetRoundID: 预测针对的期号（比如07）
// predictions: 预测内容
func (m *StrategyManager) UpdatePredictions(currentRoundID string, targetRoundID string, name string, predictions []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 获取或创建策略状态
	state, exists := m.strategies[name]
	if !exists {
		state = &StrategyState{
			Name:             name,
			Status:           StatusVirtual, // 初始为虚盘
			VirtualStreak:    0,
			RealProfit:       0.0,
			RoundPredictions: make(map[string][]string),
		}
		m.strategies[name] = state
		log.Printf("🎯 初始化策略: %s (虚盘模式)", name)
	}

	// 将预测保存到【目标期号】的 key 中
	// 例如：06期生成的预测是对07期的，所以存到 RoundPredictions["07"]
	state.RoundPredictions[targetRoundID] = predictions
	// 更新当前预测（用于显示）
	state.Predictions = predictions

	// 更新全局期号（显示的是当前已开奖的期号）
	m.roundID = currentRoundID
	m.updatedAt = time.Now()
}

// SettleRound 结算上一期盈亏（写锁）
// 返回值：是否有任何策略被结算
func (m *StrategyManager) SettleRound(roundID string, winners []string, specialReward string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	settled := false

	// 遍历所有策略进行结算
	for _, state := range m.strategies {
		// 从 map 中获取该期号的预测
		predictions, exists := state.RoundPredictions[roundID]
		if !exists || len(predictions) == 0 {
			// 如果该期号没有预测，跳过
			continue
		}

		settled = true

		// 判断是否命中：预测中是否有获胜车型
		hitWinner := m.checkWin(predictions, winners)

		// 获取该策略的单注金额
		unitBetAmount := m.getStrategyBetAmount(state.Name)

		// 记录本期盈亏（在状态更新前）
		profit := 0.0
		statusBeforeUpdate := state.Status
		betAmount := float64(len(predictions)) * unitBetAmount

		// 计算盈利（虚盘和实盘都需要计算，用于判定胜负）
		var won bool
		if hitWinner {
			// 计算真实盈利：(命中车型赔率 - 1) * 单注金额 - (未命中车型数量 * 单注金额)
			profit = m.calculateProfit(predictions, winners, unitBetAmount)
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
		predictionsJSON, _ := json.Marshal(predictions)
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

		// 从 map 中删除已结算的期号预测
		delete(state.RoundPredictions, roundID)
	}

	return settled
}

// calculateProfit 计算真实盈利
// 支持多个命中：下注多个车型，可能命中多个
// betAmount: 单注金额
func (m *StrategyManager) calculateProfit(predictions []string, winners []string, betAmount float64) float64 {
	// 创建获胜车型集合
	winnerSet := make(map[string]bool)
	for _, w := range winners {
		winnerSet[w] = true
	}

	// 找出所有命中的车型
	hitCars := make([]string, 0)
	missCount := 0
	for _, pred := range predictions {
		if winnerSet[pred] {
			hitCars = append(hitCars, pred)
		} else {
			missCount++
		}
	}

	if len(hitCars) == 0 {
		// 没有命中，理论上不应该到这里
		return -float64(len(predictions)) * betAmount
	}

	// 计算所有命中车型的盈利
	totalWinAmount := 0.0
	for _, hitCar := range hitCars {
		// 获取赔率
		odds, exists := REAL_ODDS[hitCar]
		if !exists {
			log.Printf("⚠️ 未找到车型 %s 的赔率，使用默认赔率10", hitCar)
			odds = 10
		}
		// 每个命中车型的盈利 = (赔率 - 1) * 单注金额
		totalWinAmount += float64(odds-1) * betAmount
	}

	// 计算未命中车型的损失
	loseAmount := float64(missCount) * betAmount

	// 总盈利 = 所有命中车型的盈利之和 - 未命中车型的损失
	profit := totalWinAmount - loseAmount

	log.Printf("💵 盈利计算: 命中 %d 个 %v, 未命中 %d 个, 盈利=%.2f-%.2f=%.2f", 
		len(hitCars), hitCars, missCount, totalWinAmount, loseAmount, profit)

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
			log.Printf("🎉 [%s] 虚盘赢 | 连赢: %d/%d", state.Name, state.VirtualStreak, m.config.EntryCondition)

			// 判断进场：达到进场条件
			if state.VirtualStreak >= m.config.EntryCondition {
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

			// 触发止损：切换回虚盘（连输达到离场条件）
			// 注：当前实现是连输1把即离场，可根据 ExitCondition 配置扩展
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

	// 计算系统运行时长（从启动到现在）
	systemUptime := int(time.Since(m.startTime).Seconds())

	return &State{
		RoundID:      m.roundID,
		UpdatedAt:    m.updatedAt,
		SystemUptime: systemUptime,
		Strategies:   results,
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

// NextPredictionItem 下一期预测项
type NextPredictionItem struct {
	Name        string   `json:"name"`        // 策略名称
	Predictions []string `json:"predictions"` // 预测内容
	BetAmount   float64  `json:"bet_amount"`  // 下注金额
}

// NextPredictionResult 下一期预测结果
type NextPredictionResult struct {
	Round      string               `json:"round"`      // 下一期期号
	Strategies []NextPredictionItem `json:"strategies"` // 启用的策略列表
}

// GetNextPrediction 获取下一期预测（只返回启用的策略）
func (m *StrategyManager) GetNextPrediction() NextPredictionResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	strategies := make([]NextPredictionItem, 0)

	// 遍历所有策略
	for _, state := range m.strategies {
		// 检查是否启用
		enabled := false
		betAmount := 0.0

		if state.Name == "热门3码" && m.config.Hot3Enabled {
			enabled = true
			betAmount = m.config.Hot3BetAmount
		} else if state.Name == "均衡4码" && m.config.Balanced4Enabled {
			enabled = true
			betAmount = m.config.Balanced4BetAmount
		}

		// 只返回实盘状态的预测（虚盘不返回）
		if enabled && len(state.Predictions) > 0 && state.Status == StatusReal {
			strategies = append(strategies, NextPredictionItem{
				Name:        state.Name,
				Predictions: state.Predictions,
				BetAmount:   betAmount,
			})
		}
	}

	// 计算下一期期号
	nextRound := ""
	if m.roundID != "" {
		if num, err := strconv.Atoi(m.roundID); err == nil {
			nextRound = strconv.Itoa(num + 1)
		}
	}

	return NextPredictionResult{
		Round:      nextRound,
		Strategies: strategies,
	}
}

// HistoryQueryParams 历史记录查询参数
type HistoryQueryParams struct {
	Page     int  // 页码（从1开始）
	PageSize int  // 每页大小
	RealOnly bool // 是否只查询实盘记录
}

// HistoryResult 历史记录查询结果
type HistoryResult struct {
	Records    []HistoryRecord // 记录列表
	Total      int64           // 总记录数
	TotalPages int             // 总页数
	Page       int             // 当前页码
	PageSize   int             // 每页大小
}

// GetHistory 获取历史记录（从数据库，支持分页和筛选）
func (m *StrategyManager) GetHistory(params HistoryQueryParams) HistoryResult {
	// 参数验证
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	// 构建查询
	query := m.db.Model(&models.StrategyHistory{})
	
	// 筛选实盘记录
	if params.RealOnly {
		query = query.Where("status = ?", StatusReal)
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("❌ 查询历史记录总数失败: %v", err)
		return HistoryResult{
			Records:    []HistoryRecord{},
			Total:      0,
			TotalPages: 0,
			Page:       params.Page,
			PageSize:   params.PageSize,
		}
	}

	// 计算总页数
	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	// 分页查询
	offset := (params.Page - 1) * params.PageSize
	var dbRecords []models.StrategyHistory
	err := query.Order("created_at DESC, id DESC").
		Limit(params.PageSize).
		Offset(offset).
		Find(&dbRecords).Error
	
	if err != nil {
		log.Printf("❌ 查询历史记录失败: %v", err)
		return HistoryResult{
			Records:    []HistoryRecord{},
			Total:      total,
			TotalPages: totalPages,
			Page:       params.Page,
			PageSize:   params.PageSize,
		}
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

	return HistoryResult{
		Records:    records,
		Total:      total,
		TotalPages: totalPages,
		Page:       params.Page,
		PageSize:   params.PageSize,
	}
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

// ReportSummary 总体报表统计
type ReportSummary struct {
	TotalBets   int64   `json:"total_bets"`   // 总下单次数
	TotalWins   int64   `json:"total_wins"`   // 总命中次数
	WinRate     float64 `json:"win_rate"`     // 命中率
	TotalProfit float64 `json:"total_profit"` // 总盈利
}

// DailyReportItem 每日报表统计
type DailyReportItem struct {
	Date        string  `json:"date"`         // 日期
	TotalBets   int64   `json:"bets"`         // 下单次数
	TotalWins   int64   `json:"wins"`         // 命中次数
	WinRate     float64 `json:"win_rate"`     // 命中率
	TotalProfit float64 `json:"profit"`       // 盈利
}

// StrategyReportItem 策略报表统计
type StrategyReportItem struct {
	Name               string   `json:"name"`                // 策略名称
	TotalBets          int64    `json:"total_bets"`          // 实盘下单次数
	TotalWins          int64    `json:"total_wins"`          // 实盘命中次数
	WinRate            float64  `json:"win_rate"`            // 实盘命中率
	TotalProfit        float64  `json:"total_profit"`        // 实盘总盈利
	Status             int      `json:"status"`              // 当前状态
	StatusText         string   `json:"status_text"`         // 状态描述
	CurrentPredictions []string `json:"current_predictions"` // 当前推荐
}

// GetReportSummary 获取总体统计报表（只统计实盘）
func (m *StrategyManager) GetReportSummary() ReportSummary {
	var result ReportSummary
	type Result struct {
		Bets   int64
		Wins   int64
		Profit float64
	}
	var dbResult Result

	// 统计实盘记录
	// 命中次数定义：result='赢'
	m.db.Model(&models.StrategyHistory{}).
		Where("status = ?", StatusReal).
		Select("COUNT(*) as bets, SUM(CASE WHEN result='赢' THEN 1 ELSE 0 END) as wins, COALESCE(SUM(profit), 0) as profit").
		Scan(&dbResult)

	result.TotalBets = dbResult.Bets
	result.TotalWins = dbResult.Wins
	result.TotalProfit = dbResult.Profit
	if result.TotalBets > 0 {
		result.WinRate = float64(result.TotalWins) / float64(result.TotalBets) * 100
	}
	return result
}

// GetDailyReport 获取每日统计报表（只统计实盘）
func (m *StrategyManager) GetDailyReport() []DailyReportItem {
	var results []DailyReportItem
	
	type DailyStat struct {
		DateStr string  `gorm:"column:date"`
		Bets    int64   `gorm:"column:bets"`
		Wins    int64   `gorm:"column:wins"`
		Profit  float64 `gorm:"column:profit"`
	}
	var stats []DailyStat

	// 按日期分组统计实盘数据
	// 使用 DATE_FORMAT 确保日期格式一致
	err := m.db.Model(&models.StrategyHistory{}).
		Where("status = ?", StatusReal).
		Select("DATE_FORMAT(created_at, '%Y-%m-%d') as date, COUNT(*) as bets, SUM(CASE WHEN result='赢' THEN 1 ELSE 0 END) as wins, COALESCE(SUM(profit), 0) as profit").
		Group("DATE_FORMAT(created_at, '%Y-%m-%d')").
		Order("date DESC").
		Scan(&stats).Error

	if err != nil {
		log.Printf("❌ 查询每日报表失败: %v", err)
		return []DailyReportItem{}
	}

	for _, stat := range stats {
		item := DailyReportItem{
			Date:        stat.DateStr,
			TotalBets:   stat.Bets,
			TotalWins:   stat.Wins,
			TotalProfit: stat.Profit,
		}
		if item.TotalBets > 0 {
			item.WinRate = float64(item.TotalWins) / float64(item.TotalBets) * 100
		}
		results = append(results, item)
	}

	return results
}

// GetStrategyReport 获取策略统计报表
func (m *StrategyManager) GetStrategyReport() []StrategyReportItem {
	// 1. 获取数据库统计数据（只统计实盘）
	type StatResult struct {
		Strategy string  `gorm:"column:strategy"`
		Bets     int64   `gorm:"column:bets"`
		Wins     int64   `gorm:"column:wins"`
		Profit   float64 `gorm:"column:profit"`
	}
	var stats []StatResult

	m.db.Model(&models.StrategyHistory{}).
		Where("status = ?", StatusReal).
		Select("strategy, COUNT(*) as bets, SUM(CASE WHEN result='赢' THEN 1 ELSE 0 END) as wins, COALESCE(SUM(profit), 0) as profit").
		Group("strategy").
		Scan(&stats)

	statsMap := make(map[string]StatResult)
	for _, s := range stats {
		statsMap[s.Strategy] = s
	}

	// 2. 结合内存中的当前状态
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []StrategyReportItem

	// 遍历当前管理的所有策略
	for name, state := range m.strategies {
		stat := statsMap[name]

		item := StrategyReportItem{
			Name:               name,
			TotalBets:          stat.Bets,
			TotalWins:          stat.Wins,
			TotalProfit:        stat.Profit,
			Status:             state.Status,
			CurrentPredictions: state.Predictions,
		}

		if item.TotalBets > 0 {
			item.WinRate = float64(item.TotalWins) / float64(item.TotalBets) * 100
		}

		if state.Status == StatusReal {
			item.StatusText = "实盘下注"
		} else {
			item.StatusText = "虚盘观望"
		}

		results = append(results, item)
	}
	
	// 对结果进行排序，热门3码放前面
	// 注意：这里需要自己实现简单的排序或直接依赖前端排序
	// 为了简单起见，这里先不排序，让前端处理
	
	return results
}
