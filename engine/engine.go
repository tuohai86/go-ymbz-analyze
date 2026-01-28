package engine

import (
	"benz-sniper/models"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Engine 分析引擎
type Engine struct {
	db                *gorm.DB
	manager           *StrategyManager
	pendingSettlement []string // 待结算的期号列表
}

// New 创建引擎实例
func New(db *gorm.DB, manager *StrategyManager) *Engine {
	return &Engine{
		db:                db,
		manager:           manager,
		pendingSettlement: make([]string, 0),
	}
}

// Run 后台运行（单goroutine，无并发）
func (e *Engine) Run() {
	log.Println("🚀 策略引擎启动（虚实盘模式）")

	for {
		e.tick()
		time.Sleep(1 * time.Second)
	}
}

// tick 单次轮询处理
func (e *Engine) tick() {
	// 1. 查询最新期号
	var latest models.GameRound
	if err := e.db.Order("round_id DESC").First(&latest).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("查询最新期数失败: %v", err)
		}
		return
	}

	// 2. 检查是否已处理
	current := e.manager.GetState()
	isNewRound := current == nil || current.RoundID != latest.RoundID

	// 3. 如果不是新期号，只处理待结算列表
	if !isNewRound {
		e.processPendingSettlements()
		return
	}

	log.Printf("💰 新期号: %s", latest.RoundID)

	// 4. 将【当前新期号】加入待结算列表
	// 因为之前已经有对这一期的预测了（在上一期时生成的）
	// 例如：检测到07开奖 → 将07加入待结算 → 用07的结果验证之前对07的预测
	e.addPendingSettlement(latest.RoundID)

	// 5. 查询历史数据
	var rounds []models.GameRound
	e.db.Order("round_id DESC").Limit(50).Find(&rounds)

	// 反转顺序（从旧到新）
	for i := 0; i < len(rounds)/2; i++ {
		rounds[i], rounds[len(rounds)-1-i] = rounds[len(rounds)-1-i], rounds[i]
	}

	// 6. 计算热度
	scores := e.calcHeatScores(rounds, 30)

	// 7. 计算两个策略
	hot3 := StratHot3(scores)
	balanced4 := StratBalanced4(scores)

	// 8. 计算下一期期号（预测的目标期号）
	nextRoundID := calcNextRoundID(latest.RoundID)

	log.Printf("  🎯 预测目标: %s | 热门3码: %v | 均衡4码: %v", nextRoundID, hot3, balanced4)

	// 9. 更新策略预测
	// currentRoundID=当前已开奖期号, targetRoundID=预测目标期号
	e.manager.UpdatePredictions(latest.RoundID, nextRoundID, "热门3码", hot3)
	e.manager.UpdatePredictions(latest.RoundID, nextRoundID, "均衡4码", balanced4)

	// 10. 处理所有待结算的期号
	e.processPendingSettlements()
}

// calcNextRoundID 计算下一期期号
func calcNextRoundID(currentRoundID string) string {
	// 尝试将期号转换为数字并加1
	num := 0
	for _, c := range currentRoundID {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		}
	}
	if num > 0 {
		return fmt.Sprintf("%d", num+1)
	}
	return currentRoundID + "_next"
}

// addPendingSettlement 添加待结算期号
func (e *Engine) addPendingSettlement(roundID string) {
	// 检查是否已存在
	for _, pending := range e.pendingSettlement {
		if pending == roundID {
			return
		}
	}
	e.pendingSettlement = append(e.pendingSettlement, roundID)
	log.Printf("📋 添加待结算期号: %s", roundID)
}

// processPendingSettlements 处理所有待结算的期号
func (e *Engine) processPendingSettlements() {
	if len(e.pendingSettlement) == 0 {
		return
	}

	toRemove := make([]string, 0)

	// 遍历所有待结算期号
	for _, roundID := range e.pendingSettlement {
		// 查询该期的开奖结果
		var winners []models.GameWinner
		e.db.Where("round_id = ?", roundID).Find(&winners)

		// 如果没有开奖结果，跳过（等待数据写入）
		if len(winners) == 0 {
			continue
		}

		// 获取获胜车型名称
		winnerNames := make([]string, 0, len(winners))
		for _, w := range winners {
			cleaned := cleanName(w.WinnerName)
			winnerNames = append(winnerNames, cleaned)
		}

		// 查询特殊奖项
		specialReward := ""
		var round models.GameRound
		if err := e.db.Where("round_id = ?", roundID).First(&round).Error; err == nil {
			for _, sr := range SPECIAL_REWARDS {
				if strings.Contains(round.ResultName, sr) {
					specialReward = sr
					break
				}
			}
		}

		// 执行结算
		hasSettled := e.manager.SettleRound(roundID, winnerNames, specialReward)
		
		if hasSettled {
			log.Printf("🏆 结算期号 %s: %v", roundID, winnerNames)
			if specialReward != "" {
				log.Printf("✨ 特殊奖项: %s", specialReward)
			}
		} else {
			// 没有预测可结算（比如系统刚启动的第一期），也要移除
			log.Printf("⏭️ 跳过期号 %s（无预测）", roundID)
		}

		// 只要开奖结果存在，就从待结算列表中移除（无论是否有预测）
		toRemove = append(toRemove, roundID)
	}

	// 移除已处理的期号
	if len(toRemove) > 0 {
		newPending := make([]string, 0)
		for _, roundID := range e.pendingSettlement {
			shouldRemove := false
			for _, r := range toRemove {
				if r == roundID {
					shouldRemove = true
					break
				}
			}
			if !shouldRemove {
				newPending = append(newPending, roundID)
			}
		}
		e.pendingSettlement = newPending
		if len(newPending) > 0 || len(toRemove) > 0 {
			log.Printf("✅ 已处理 %d 个期号，剩余待结算: %d", len(toRemove), len(newPending))
		}
	}
}

// calcHeatScores 计算热度评分
func (e *Engine) calcHeatScores(rounds []models.GameRound, limit int) map[string]float64 {
	scores := make(map[string]float64)

	// 初始化所有车型分数为0
	for _, label := range BET_LABELS {
		scores[label] = 0.0
	}

	// 限制分析数量
	if len(rounds) > limit {
		rounds = rounds[len(rounds)-limit:]
	}

	if len(rounds) == 0 {
		return scores
	}

	total := float64(len(rounds))

	// 批量查询所有期的获胜项
	roundIDs := make([]string, len(rounds))
	for i, round := range rounds {
		roundIDs[i] = round.RoundID
	}

	var allWinners []models.GameWinner
	e.db.Where("round_id IN ?", roundIDs).Find(&allWinners)

	// 按round_id分组
	winnersMap := make(map[string][]string)
	for _, w := range allWinners {
		cleaned := cleanName(w.WinnerName)
		winnersMap[w.RoundID] = append(winnersMap[w.RoundID], cleaned)
	}

	// 遍历每期，计算加权分数
	for idx, round := range rounds {
		winners := winnersMap[round.RoundID]

		// 计算时间加权：越近的期数权重越高（0.5 ~ 1.5）
		weight := 0.5 + float64(idx)/total

		// 为获胜车型累加分数
		for _, winner := range winners {
			for _, label := range BET_LABELS {
				if strings.Contains(winner, label) || label == winner {
					scores[label] += 1.0 * weight
				}
			}
		}
	}

	return scores
}

// cleanName 清理车型名称
func cleanName(name string) string {
	name = strings.TrimSpace(name)

	// 检查是否是标准车型
	for _, label := range BET_LABELS {
		if name == label {
			return label
		}
		// 模糊匹配：包含颜色和品牌
		if len(label) == 3 && strings.Contains(name, string([]rune(label)[0])) && strings.Contains(name, label[len(label)-2:]) {
			return label
		}
	}

	return name
}
