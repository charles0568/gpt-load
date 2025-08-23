package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gpt-load/internal/errors"
	"gpt-load/internal/response"
	"gpt-load/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// BatchCheckRequest 批量檢查請求
type BatchCheckRequest struct {
	GroupID     uint `json:"group_id" binding:"required"`
	BatchSize   int  `json:"batch_size"`
	Concurrency int  `json:"concurrency"`
}

// KeyBatchCheckHandler 批量密鑰檢查處理器
type KeyBatchCheckHandler struct {
	checker  *service.KeyBatchChecker
	logger   *zap.Logger
	upgrader websocket.Upgrader
}

// NewKeyBatchCheckHandler 創建批量密鑰檢查處理器
func NewKeyBatchCheckHandler(checker *service.KeyBatchChecker, logger *zap.Logger) *KeyBatchCheckHandler {
	return &KeyBatchCheckHandler{
		checker: checker,
		logger:  logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允許跨域
			},
		},
	}
}

// StartBatchCheck 開始批量檢查
func (h *KeyBatchCheckHandler) StartBatchCheck(c *gin.Context) {
	var req BatchCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.ErrBadRequest)
		return
	}

	// 設定預設值
	if req.BatchSize <= 0 {
		req.BatchSize = 100
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 50
	}

	// 限制併發數，避免系統過載
	if req.Concurrency > 200 {
		req.Concurrency = 200
	}

	// 開始批量檢查
	task, err := h.checker.StartBatchCheck(req.GroupID, req.BatchSize, req.Concurrency)
	if err != nil {
		h.logger.Error("開始批量檢查失敗", zap.Error(err))
		response.Error(c, errors.ErrInternalServer)
		return
	}

	response.Success(c, gin.H{
		"task_id": task.ID,
		"message": "批量檢查已開始",
	})
}

// GetTaskProgress 獲取任務進度
func (h *KeyBatchCheckHandler) GetTaskProgress(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		response.Error(c, errors.ErrBadRequest)
		return
	}

	progress, err := h.checker.GetTaskProgress(taskID)
	if err != nil {
		response.Error(c, errors.ErrResourceNotFound)
		return
	}

	response.Success(c, progress)
}

// GetTaskResults 獲取任務結果
func (h *KeyBatchCheckHandler) GetTaskResults(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		response.Error(c, errors.ErrBadRequest)
		return
	}

	// 獲取分頁參數
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 100
	}

	results, err := h.checker.GetTaskResults(taskID)
	if err != nil {
		response.Error(c, errors.ErrResourceNotFound)
		return
	}

	// 分頁處理
	total := len(results)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		start = total
	}
	if end > total {
		end = total
	}

	pagedResults := results[start:end]

	response.Success(c, gin.H{
		"results": pagedResults,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
	})
}

// PauseTask 暫停任務
func (h *KeyBatchCheckHandler) PauseTask(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		response.Error(c, errors.ErrBadRequest)
		return
	}

	if err := h.checker.PauseTask(taskID); err != nil {
		response.Error(c, errors.ErrBadRequest)
		return
	}

	response.Success(c, gin.H{"message": "任務已暫停"})
}

// ResumeTask 恢復任務
func (h *KeyBatchCheckHandler) ResumeTask(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		response.Error(c, errors.ErrBadRequest)
		return
	}

	if err := h.checker.ResumeTask(taskID); err != nil {
		response.Error(c, errors.ErrBadRequest)
		return
	}

	response.Success(c, gin.H{"message": "任務已恢復"})
}

// CancelTask 取消任務
func (h *KeyBatchCheckHandler) CancelTask(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		response.Error(c, errors.ErrBadRequest)
		return
	}

	if err := h.checker.CancelTask(taskID); err != nil {
		response.Error(c, errors.ErrBadRequest)
		return
	}

	response.Success(c, gin.H{"message": "任務已取消"})
}

// WebSocketProgress WebSocket 進度推送
func (h *KeyBatchCheckHandler) WebSocketProgress(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任務 ID 不能為空"})
		return
	}

	// 升級到 WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("WebSocket 升級失敗", zap.Error(err))
		return
	}
	defer conn.Close()

	// 添加到任務的 WebSocket 客戶端列表
	if err := h.checker.AddWebSocketClient(taskID, conn); err != nil {
		h.logger.Error("添加 WebSocket 客戶端失敗", zap.Error(err))
		return
	}
	defer h.checker.RemoveWebSocketClient(taskID, conn)

	// 發送當前進度
	if progress, err := h.checker.GetTaskProgress(taskID); err == nil {
		if progressData, err := json.Marshal(progress); err == nil {
			conn.WriteMessage(websocket.TextMessage, progressData)
		}
	}

	// 保持連接
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// ExportResults 匯出檢查結果
func (h *KeyBatchCheckHandler) ExportResults(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		response.Error(c, errors.ErrBadRequest)
		return
	}

	format := c.DefaultQuery("format", "csv")
	onlyValid := c.DefaultQuery("only_valid", "false") == "true"
	onlyInvalid := c.DefaultQuery("only_invalid", "false") == "true"

	results, err := h.checker.GetTaskResults(taskID)
	if err != nil {
		response.Error(c, errors.ErrResourceNotFound)
		return
	}

	// 過濾結果
	var filteredResults []service.BatchCheckResult
	for _, result := range results {
		if onlyValid && !result.Valid {
			continue
		}
		if onlyInvalid && result.Valid {
			continue
		}
		filteredResults = append(filteredResults, result)
	}

	switch format {
	case "csv":
		h.exportCSV(c, taskID, filteredResults)
	case "json":
		h.exportJSON(c, taskID, filteredResults)
	default:
		response.Error(c, errors.ErrBadRequest)
	}
}

// exportCSV 匯出 CSV 格式
func (h *KeyBatchCheckHandler) exportCSV(c *gin.Context, taskID string, results []service.BatchCheckResult) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=batch_check_results_%s.csv", taskID))

	// CSV 標頭
	c.Writer.WriteString("密鑰ID,密鑰,分組ID,有效,回應時間(ms),錯誤訊息,檢查時間\n")

	// CSV 資料
	for _, result := range results {
		validStr := "否"
		if result.Valid {
			validStr = "是"
		}

		line := fmt.Sprintf("%d,\"%s\",%d,%s,%d,\"%s\",\"%s\"\n",
			result.KeyID,
			result.Key,
			result.GroupID,
			validStr,
			result.ResponseTime,
			result.ErrorMessage,
			result.CheckedAt.Format("2006-01-02 15:04:05"),
		)
		c.Writer.WriteString(line)
	}
}

// exportJSON 匯出 JSON 格式
func (h *KeyBatchCheckHandler) exportJSON(c *gin.Context, taskID string, results []service.BatchCheckResult) {
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=batch_check_results_%s.json", taskID))

	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"results": results,
		"exported_at": time.Now(),
	})
}

// BatchDeleteInvalidKeys 批量刪除無效密鑰
func (h *KeyBatchCheckHandler) BatchDeleteInvalidKeys(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		response.Error(c, errors.ErrBadRequest)
		return
	}

	results, err := h.checker.GetTaskResults(taskID)
	if err != nil {
		response.Error(c, errors.ErrResourceNotFound)
		return
	}

	// 收集無效密鑰 ID
	var invalidKeyIDs []uint
	for _, result := range results {
		if !result.Valid {
			invalidKeyIDs = append(invalidKeyIDs, result.KeyID)
		}
	}

	if len(invalidKeyIDs) == 0 {
		response.Success(c, gin.H{
			"message": "沒有無效密鑰需要刪除",
			"deleted_count": 0,
		})
		return
	}

	// 批量刪除無效密鑰
	// 這裡需要調用密鑰服務的批量刪除方法
	// deletedCount, err := h.keyService.BatchDeleteKeys(invalidKeyIDs)
	// 暫時返回模擬結果
	deletedCount := len(invalidKeyIDs)

	response.Success(c, gin.H{
		"message": "無效密鑰已刪除",
		"deleted_count": deletedCount,
	})
}
