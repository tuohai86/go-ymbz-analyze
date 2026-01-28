package api

import (
	"benz-sniper/engine"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler API处理器
type Handler struct {
	state *engine.AtomicState
}

// New 创建API处理器实例
func New(state *engine.AtomicState) *Handler {
	return &Handler{state: state}
}

// StatusResponse 状态响应
type StatusResponse struct {
	RoundID    string                   `json:"round_id"`
	NextRound  string                   `json:"next_round"`
	UpdatedAt  string                   `json:"updated_at"`
	TimePassed int                      `json:"time_passed"`
	Countdown  int                      `json:"countdown"`
	Strategies []engine.StrategyResult  `json:"strategies"`
}

// GetStatus 获取当前状态（无锁读取）
func (h *Handler) GetStatus(c *gin.Context) {
	// 直接读取，无锁
	s := h.state.Get()
	
	if s == nil {
		c.JSON(http.StatusOK, StatusResponse{
			RoundID:    "",
			NextRound:  "",
			UpdatedAt:  "",
			TimePassed: 0,
			Countdown:  0,
			Strategies: []engine.StrategyResult{},
		})
		return
	}

	// 计算时间相关
	timePassed := int(time.Since(s.UpdatedAt).Seconds())
	countdown := 34 - timePassed
	if countdown < 0 {
		countdown = 0
	}

	// 计算下一期号
	nextRound := ""
	if s.RoundID != "" {
		if num, err := strconv.Atoi(s.RoundID); err == nil {
			nextRound = strconv.Itoa(num + 1)
		}
	}

	c.JSON(http.StatusOK, StatusResponse{
		RoundID:    s.RoundID,
		NextRound:  nextRound,
		UpdatedAt:  s.UpdatedAt.Format("15:04:05"),
		TimePassed: timePassed,
		Countdown:  countdown,
		Strategies: s.Strategies,
	})
}

// PredictionsResponse 预测响应
type PredictionsResponse struct {
	Round       string         `json:"round"`
	Predictions map[string]int `json:"predictions"`
}

// GetPredictions 获取预测（无锁读取）
func (h *Handler) GetPredictions(c *gin.Context) {
	// 直接读取，无锁
	s := h.state.Get()

	if s == nil {
		c.JSON(http.StatusOK, PredictionsResponse{
			Round:       "",
			Predictions: make(map[string]int),
		})
		return
	}

	// 使用 map 去重所有策略的预测
	allItems := make(map[string]bool)
	for _, strategy := range s.Strategies {
		for _, item := range strategy.Predictions {
			allItems[item] = true
		}
	}

	// 转换为 map 格式，每项金额固定为 100
	predictions := make(map[string]int)
	for item := range allItems {
		predictions[item] = 100
	}

	// 计算下注期号
	nextRound := ""
	if s.RoundID != "" {
		if num, err := strconv.Atoi(s.RoundID); err == nil {
			nextRound = strconv.Itoa(num + 1)
		}
	}

	c.JSON(http.StatusOK, PredictionsResponse{
		Round:       nextRound,
		Predictions: predictions,
	})
}

// SetupRoutes 设置路由
func (h *Handler) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		api.GET("/status", h.GetStatus)
		api.GET("/predictions", h.GetPredictions)
	}
}
