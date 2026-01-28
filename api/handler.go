package api

import (
	"benz-sniper/engine"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler API处理器
type Handler struct {
	manager *engine.StrategyManager
}

// New 创建API处理器实例
func New(manager *engine.StrategyManager) *Handler {
	return &Handler{manager: manager}
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

// GetStatus 获取当前状态（读锁）
func (h *Handler) GetStatus(c *gin.Context) {
	// 读取状态（自动加读锁）
	s := h.manager.GetState()
	
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

// GetPredictions 获取预测（读锁，只返回实盘策略）
func (h *Handler) GetPredictions(c *gin.Context) {
	// 读取状态（自动加读锁）
	s := h.manager.GetState()

	if s == nil {
		c.JSON(http.StatusOK, PredictionsResponse{
			Round:       "",
			Predictions: make(map[string]int),
		})
		return
	}

	// 只获取实盘策略的预测
	realStrategies := h.manager.GetRealPredictions()

	// 使用 map 去重所有实盘策略的预测
	allItems := make(map[string]bool)
	for _, strategy := range realStrategies {
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

// HistoryResponse 历史记录响应
type HistoryResponse struct {
	Records []engine.HistoryRecord `json:"records"`
	Total   int                    `json:"total"`
}

// GetHistory 获取历史记录（读锁）
func (h *Handler) GetHistory(c *gin.Context) {
	// 获取查询参数
	limitStr := c.DefaultQuery("limit", "50")
	limit := 50
	if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil {
		if limit > 100 {
			limit = 100
		}
	}

	// 读取历史记录
	records := h.manager.GetHistory(limit)

	c.JSON(http.StatusOK, HistoryResponse{
		Records: records,
		Total:   len(records),
	})
}

// ClearHistory 清空历史记录（写锁）
func (h *Handler) ClearHistory(c *gin.Context) {
	h.manager.ClearHistory()
	c.JSON(http.StatusOK, gin.H{
		"message": "历史记录已清空",
		"success": true,
	})
}

// SetupRoutes 设置路由
func (h *Handler) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		api.GET("/status", h.GetStatus)
		api.GET("/predictions", h.GetPredictions)
		api.GET("/history", h.GetHistory)
		api.POST("/history/clear", h.ClearHistory)
	}
}
