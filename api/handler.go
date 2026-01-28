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
	manager *engine.StrategyManager
}

// New 创建API处理器实例
func New(manager *engine.StrategyManager) *Handler {
	return &Handler{manager: manager}
}

// StatusResponse 状态响应
type StatusResponse struct {
	RoundID         string                   `json:"round_id"`
	NextRound       string                   `json:"next_round"`
	UpdatedAt       string                   `json:"updated_at"`
	TimePassed      int                      `json:"time_passed"`
	Countdown       int                      `json:"countdown"`
	Strategies      []engine.StrategyResult  `json:"strategies"`
	TotalRealProfit float64                  `json:"total_real_profit"` // 所有实盘注单总盈利
}

// GetStatus 获取当前状态（读锁）
func (h *Handler) GetStatus(c *gin.Context) {
	// 读取状态（自动加读锁）
	s := h.manager.GetState()
	
	if s == nil {
		c.JSON(http.StatusOK, StatusResponse{
			RoundID:         "",
			NextRound:       "",
			UpdatedAt:       "",
			TimePassed:      0,
			Countdown:       0,
			Strategies:      []engine.StrategyResult{},
			TotalRealProfit: 0.0,
		})
		return
	}

	// 计算时间相关
	timePassed := s.SystemUptime // 使用系统运行时长
	roundTimePassed := int(time.Since(s.UpdatedAt).Seconds())
	countdown := 24 - roundTimePassed // 减去10秒偏移量来同步实际游戏
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

	// 计算所有实盘注单的总盈利
	totalRealProfit := h.manager.GetTotalRealProfit()

	c.JSON(http.StatusOK, StatusResponse{
		RoundID:         s.RoundID,
		NextRound:       nextRound,
		UpdatedAt:       s.UpdatedAt.Format("15:04:05"),
		TimePassed:      timePassed,
		Countdown:       countdown,
		Strategies:      s.Strategies,
		TotalRealProfit: totalRealProfit,
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
	Records    []engine.HistoryRecord `json:"records"`
	Total      int64                  `json:"total"`       // 总记录数
	TotalPages int                    `json:"total_pages"` // 总页数
	Page       int                    `json:"page"`        // 当前页码
	PageSize   int                    `json:"page_size"`   // 每页大小
}

// GetHistory 获取历史记录（读锁，支持分页和筛选）
func (h *Handler) GetHistory(c *gin.Context) {
	// 获取查询参数
	page := 1
	pageSize := 20
	realOnly := false
	
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
			if pageSize > 100 {
				pageSize = 100
			}
		}
	}
	
	if realOnlyStr := c.Query("real_only"); realOnlyStr == "true" || realOnlyStr == "1" {
		realOnly = true
	}

	// 读取历史记录
	result := h.manager.GetHistory(engine.HistoryQueryParams{
		Page:     page,
		PageSize: pageSize,
		RealOnly: realOnly,
	})

	c.JSON(http.StatusOK, HistoryResponse{
		Records:    result.Records,
		Total:      result.Total,
		TotalPages: result.TotalPages,
		Page:       result.Page,
		PageSize:   result.PageSize,
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

// ReportResponse 财务报表响应
type ReportResponse struct {
	Summary    engine.ReportSummary        `json:"summary"`
	Daily      []engine.DailyReportItem    `json:"daily"`
	Strategies []engine.StrategyReportItem `json:"strategies"`
}

// GetReport 获取财务报表
func (h *Handler) GetReport(c *gin.Context) {
	summary := h.manager.GetReportSummary()
	daily := h.manager.GetDailyReport()
	strategies := h.manager.GetStrategyReport()

	c.JSON(http.StatusOK, ReportResponse{
		Summary:    summary,
		Daily:      daily,
		Strategies: strategies,
	})
}

// GetConfig 获取当前配置
func (h *Handler) GetConfig(c *gin.Context) {
	config := h.manager.GetConfig()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"config":  config,
	})
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	EntryCondition     *int     `json:"entry_condition"`      // 连赢几把进场
	ExitCondition      *int     `json:"exit_condition"`       // 连输几把离场
	Hot3BetAmount      *float64 `json:"hot3_bet_amount"`      // 热门3码下注金额
	Balanced4BetAmount *float64 `json:"balanced4_bet_amount"` // 均衡4码下注金额
	Hot3Enabled        *bool    `json:"hot3_enabled"`         // 热门3码启用
	Balanced4Enabled   *bool    `json:"balanced4_enabled"`    // 均衡4码启用
}

// UpdateConfig 更新配置
func (h *Handler) UpdateConfig(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前配置
	currentConfig := h.manager.GetConfig()

	// 构建新配置（支持部分更新）
	newConfig := engine.StrategyConfig{
		EntryCondition:     currentConfig.EntryCondition,
		ExitCondition:      currentConfig.ExitCondition,
		Hot3BetAmount:      currentConfig.Hot3BetAmount,
		Balanced4BetAmount: currentConfig.Balanced4BetAmount,
		Hot3Enabled:        currentConfig.Hot3Enabled,
		Balanced4Enabled:   currentConfig.Balanced4Enabled,
	}

	// 更新提供的字段
	if req.EntryCondition != nil && *req.EntryCondition > 0 {
		newConfig.EntryCondition = *req.EntryCondition
	}
	if req.ExitCondition != nil && *req.ExitCondition > 0 {
		newConfig.ExitCondition = *req.ExitCondition
	}
	if req.Hot3BetAmount != nil && *req.Hot3BetAmount > 0 {
		newConfig.Hot3BetAmount = *req.Hot3BetAmount
	}
	if req.Balanced4BetAmount != nil && *req.Balanced4BetAmount > 0 {
		newConfig.Balanced4BetAmount = *req.Balanced4BetAmount
	}
	if req.Hot3Enabled != nil {
		newConfig.Hot3Enabled = *req.Hot3Enabled
	}
	if req.Balanced4Enabled != nil {
		newConfig.Balanced4Enabled = *req.Balanced4Enabled
	}

	// 更新配置
	updatedConfig := h.manager.UpdateConfig(newConfig)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置已更新",
		"config":  updatedConfig,
	})
}

// GetNextPrediction 获取下一期预测（基于启用状态过滤）
func (h *Handler) GetNextPrediction(c *gin.Context) {
	result := h.manager.GetNextPrediction()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// UploadUserBetRequest 上传用户派彩请求
type UploadUserBetRequest struct {
	RoundID      string  `json:"round_id" binding:"required"`      // 期号
	UserAccount  string  `json:"user_account" binding:"required"`  // 用户账号
	BetAmount    float64 `json:"bet_amount" binding:"required"`    // 下注金额
	PayoutAmount float64 `json:"payout_amount"`                    // 派彩金额
	Balance      float64 `json:"balance"`                          // 剩余余额
}

// UploadUserBet 上传用户派彩记录
func (h *Handler) UploadUserBet(c *gin.Context) {
	var req UploadUserBetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 保存用户派彩记录
	err := h.manager.SaveUserBet(engine.UserBetRecord{
		RoundID:      req.RoundID,
		UserAccount:  req.UserAccount,
		BetAmount:    req.BetAmount,
		PayoutAmount: req.PayoutAmount,
		Balance:      req.Balance,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "保存失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "用户派彩记录已保存",
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
		api.GET("/report", h.GetReport)
		api.GET("/config", h.GetConfig)           // 获取配置
		api.POST("/config", h.UpdateConfig)       // 更新配置
		api.GET("/next-prediction", h.GetNextPrediction) // 获取下一期预测
		api.POST("/user-bets", h.UploadUserBet)   // 上传用户派彩记录
	}
}
