package engine

import (
	"benz-sniper/models"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Engine 分析引擎
type Engine struct {
	db    *gorm.DB
	state *AtomicState
}

// New 创建引擎实例
func New(db *gorm.DB, state *AtomicState) *Engine {
	return &Engine{
		db:    db,
		state: state,
	}
}

// Run 后台运行（单goroutine，无并发）
func (e *Engine) Run() {
	log.Println("🚀 策略引擎启动（无锁模式）")

	for {
		e.tick()
		time.Sleep(2 * time.Second)
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
	current := e.state.Get()
	if current != nil && current.RoundID == latest.RoundID {
		return
	}

	log.Printf("💰 新期号: %s", latest.RoundID)

	// 3. 查询历史数据（无锁操作）
	var rounds []models.GameRound
	e.db.Order("round_id DESC").Limit(50).Find(&rounds)

	// 反转顺序（从旧到新）
	for i := 0; i < len(rounds)/2; i++ {
		rounds[i], rounds[len(rounds)-1-i] = rounds[len(rounds)-1-i], rounds[i]
	}

	// 4. 计算热度（无锁操作）
	scores := e.calcHeatScores(rounds, 30)

	// 5. 计算两个策略（无锁操作）
	hot3 := StratHot3(scores)
	balanced4 := StratBalanced4(scores)

	log.Printf("  🎯 热门3码: %v", hot3)
	log.Printf("  🎯 均衡4码: %v", balanced4)

	// 6. 原子更新（唯一同步点）
	newState := &State{
		RoundID:   latest.RoundID,
		UpdatedAt: time.Now(),
		Strategies: []StrategyResult{
			{Name: "热门3码", Predictions: hot3},
			{Name: "均衡4码", Predictions: balanced4},
		},
	}

	e.state.Set(newState)
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
