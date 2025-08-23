package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gpt-load/internal/models"
	"gpt-load/internal/types"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BatchCheckProgress 批量檢查進度
type BatchCheckProgress struct {
	TaskID       string    `json:"task_id"`
	Status       string    `json:"status"` // running, paused, completed, cancelled, error
	TotalKeys    int64     `json:"total_keys"`
	ProcessedKeys int64    `json:"processed_keys"`
	ValidKeys    int64     `json:"valid_keys"`
	InvalidKeys  int64     `json:"invalid_keys"`
	CurrentBatch int       `json:"current_batch"`
	TotalBatches int       `json:"total_batches"`
	StartTime    time.Time `json:"start_time"`
	EstimatedEnd *time.Time `json:"estimated_end,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Speed        float64   `json:"speed"` // keys per second
}

// BatchCheckResult 單個密鑰檢查結果
type BatchCheckResult struct {
	KeyID        uint      `json:"key_id"`
	Key          string    `json:"key"`
	GroupID      uint      `json:"group_id"`
	Valid        bool      `json:"valid"`
	ResponseTime int64     `json:"response_time_ms"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CheckedAt    time.Time `json:"checked_at"`
}

// BatchCheckTask 批量檢查任務
type BatchCheckTask struct {
	ID           string
	GroupID      uint
	Status       string
	Progress     *BatchCheckProgress
	Results      []BatchCheckResult
	CancelFunc   context.CancelFunc
	PauseChannel chan bool
	ResumeChannel chan bool
	mu           sync.RWMutex
	db           *gorm.DB
	logger       *zap.Logger
	wsClients    map[*websocket.Conn]bool
	wsClientsMu  sync.RWMutex
}

// KeyBatchChecker 批量密鑰檢查器
type KeyBatchChecker struct {
	db        *gorm.DB
	logger    *zap.Logger
	tasks     map[string]*BatchCheckTask
	tasksMu   sync.RWMutex
	settings  *types.SystemSettings
}

// NewKeyBatchChecker 創建批量密鑰檢查器
func NewKeyBatchChecker(db *gorm.DB, logger *zap.Logger, settings *types.SystemSettings) *KeyBatchChecker {
	return &KeyBatchChecker{
		db:       db,
		logger:   logger,
		tasks:    make(map[string]*BatchCheckTask),
		settings: settings,
	}
}

// StartBatchCheck 開始批量檢查
func (c *KeyBatchChecker) StartBatchCheck(groupID uint, batchSize int, concurrency int) (*BatchCheckTask, error) {
	// 生成任務 ID
	taskID := fmt.Sprintf("batch_check_%d_%d", groupID, time.Now().Unix())

	// 獲取要檢查的密鑰
	var keys []models.APIKey
	query := c.db.Where("group_id = ?", groupID)
	if err := query.Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("獲取密鑰列表失敗: %w", err)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("沒有找到要檢查的密鑰")
	}

	// 創建任務
	ctx, cancel := context.WithCancel(context.Background())
	task := &BatchCheckTask{
		ID:            taskID,
		GroupID:       groupID,
		Status:        "running",
		CancelFunc:    cancel,
		PauseChannel:  make(chan bool, 1),
		ResumeChannel: make(chan bool, 1),
		db:            c.db,
		logger:        c.logger,
		wsClients:     make(map[*websocket.Conn]bool),
		Progress: &BatchCheckProgress{
			TaskID:       taskID,
			Status:       "running",
			TotalKeys:    int64(len(keys)),
			StartTime:    time.Now(),
			TotalBatches: (len(keys) + batchSize - 1) / batchSize,
		},
	}

	// 儲存任務
	c.tasksMu.Lock()
	c.tasks[taskID] = task
	c.tasksMu.Unlock()

	// 啟動檢查協程
	go c.runBatchCheck(ctx, task, keys, batchSize, concurrency)

	return task, nil
}

// runBatchCheck 執行批量檢查
func (c *KeyBatchChecker) runBatchCheck(ctx context.Context, task *BatchCheckTask, keys []models.APIKey, batchSize int, concurrency int) {
	defer func() {
		task.mu.Lock()
		if task.Progress.Status == "running" {
			task.Progress.Status = "completed"
		}
		task.mu.Unlock()
		c.broadcastProgress(task)
	}()

	// 分批處理
	for i := 0; i < len(keys); i += batchSize {
		// 檢查是否被取消
		select {
		case <-ctx.Done():
			task.mu.Lock()
			task.Progress.Status = "cancelled"
			task.mu.Unlock()
			return
		default:
		}

		// 檢查是否暫停
		select {
		case <-task.PauseChannel:
			task.mu.Lock()
			task.Progress.Status = "paused"
			task.mu.Unlock()
			c.broadcastProgress(task)

			// 等待恢復
			<-task.ResumeChannel
			task.mu.Lock()
			task.Progress.Status = "running"
			task.mu.Unlock()
		default:
		}

		// 獲取當前批次
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		// 更新進度
		task.mu.Lock()
		task.Progress.CurrentBatch = (i / batchSize) + 1
		task.mu.Unlock()

		// 並發檢查當前批次
		c.checkBatch(ctx, task, batch, concurrency)

		// 廣播進度更新
		c.broadcastProgress(task)

		// 批次間延遲，避免 API 限制
		time.Sleep(time.Second * 2)
	}
}

// checkBatch 檢查單個批次
func (c *KeyBatchChecker) checkBatch(ctx context.Context, task *BatchCheckTask, keys []models.APIKey, concurrency int) {
	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var processed int64

	for _, key := range keys {
		wg.Add(1)
		go func(k models.APIKey) {
			defer wg.Done()

			// 獲取信號量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 檢查密鑰
			result := c.checkSingleKey(ctx, k)

			// 儲存結果
			task.mu.Lock()
			task.Results = append(task.Results, result)
			if result.Valid {
				atomic.AddInt64(&task.Progress.ValidKeys, 1)
			} else {
				atomic.AddInt64(&task.Progress.InvalidKeys, 1)
			}
			task.mu.Unlock()

			// 更新進度
			processedCount := atomic.AddInt64(&processed, 1)
			atomic.AddInt64(&task.Progress.ProcessedKeys, 1)

			// 計算速度和預估時間
			if processedCount%10 == 0 { // 每 10 個更新一次
				c.updateProgressStats(task)
			}
		}(key)
	}

	wg.Wait()
}

// checkSingleKey 檢查單個密鑰
func (c *KeyBatchChecker) checkSingleKey(ctx context.Context, key models.APIKey) BatchCheckResult {
	startTime := time.Now()
	result := BatchCheckResult{
		KeyID:     key.ID,
		Key:       key.KeyValue,
		GroupID:   key.GroupID,
		CheckedAt: startTime,
	}

	// 根據分組類型選擇檢查方法
	var group models.Group
	if err := c.db.First(&group, key.GroupID).Error; err != nil {
		result.ErrorMessage = "獲取分組資訊失敗"
		return result
	}

	// 執行實際的 API 檢查
	valid, responseTime, errMsg := c.performAPICheck(ctx, key.KeyValue, group)

	result.Valid = valid
	result.ResponseTime = responseTime
	result.ErrorMessage = errMsg

	// 更新資料庫中的密鑰狀態
	if !valid {
		c.db.Model(&key).Updates(map[string]interface{}{
			"status":      "invalid",
			"last_error":  errMsg,
			"updated_at":  time.Now(),
		})
	} else {
		c.db.Model(&key).Updates(map[string]interface{}{
			"status":     "active",
			"last_error": "",
			"updated_at": time.Now(),
		})
	}

	return result
}

// performAPICheck 執行實際的 API 檢查
func (c *KeyBatchChecker) performAPICheck(ctx context.Context, apiKey string, group models.Group) (bool, int64, string) {
	startTime := time.Now()

	// 根據分組類型構建測試請求
	var testURL string
	var testPayload map[string]interface{}
	var headers map[string]string

	switch group.ChannelType {
	case "gemini":
		testURL = group.ValidationEndpoint
		if testURL == "" {
			testURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent"
		}
		testPayload = map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]interface{}{
						{"text": "Hello"},
					},
				},
			},
		}
		headers = map[string]string{
			"Content-Type": "application/json",
			"x-goog-api-key": apiKey,
		}
	case "openai":
		testURL = group.ValidationEndpoint
		if testURL == "" {
			testURL = "https://api.openai.com/v1/chat/completions"
		}
		testPayload = map[string]interface{}{
			"model": "gpt-3.5-turbo",
			"messages": []map[string]interface{}{
				{"role": "user", "content": "Hello"},
			},
			"max_tokens": 1,
		}
		headers = map[string]string{
			"Content-Type": "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		}
	default:
		return false, 0, "不支援的分組類型"
	}

	// 創建 HTTP 請求
	client := &http.Client{
		Timeout: time.Duration(c.settings.KeyValidationTimeoutSeconds) * time.Second,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	req, err := http.NewRequestWithContext(ctx, "POST", testURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return false, 0, fmt.Sprintf("創建請求失敗: %v", err)
	}

	// 設定請求標頭
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 發送請求
	resp, err := client.Do(req)
	if err != nil {
		responseTime := time.Since(startTime).Milliseconds()
		return false, responseTime, fmt.Sprintf("請求失敗: %v", err)
	}
	defer resp.Body.Close()

	responseTime := time.Since(startTime).Milliseconds()

	// 檢查回應狀態
	if resp.StatusCode == 200 {
		return true, responseTime, ""
	} else if resp.StatusCode == 429 {
		return false, responseTime, "API 配額超限"
	} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return false, responseTime, "密鑰無效或無權限"
	} else {
		return false, responseTime, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
}

// updateProgressStats 更新進度統計
func (c *KeyBatchChecker) updateProgressStats(task *BatchCheckTask) {
	task.mu.Lock()
	defer task.mu.Unlock()

	elapsed := time.Since(task.Progress.StartTime)
	if task.Progress.ProcessedKeys > 0 {
		// 計算速度 (keys/second)
		task.Progress.Speed = float64(task.Progress.ProcessedKeys) / elapsed.Seconds()

		// 預估完成時間
		if task.Progress.Speed > 0 {
			remaining := task.Progress.TotalKeys - task.Progress.ProcessedKeys
			estimatedSeconds := float64(remaining) / task.Progress.Speed
			estimatedEnd := time.Now().Add(time.Duration(estimatedSeconds) * time.Second)
			task.Progress.EstimatedEnd = &estimatedEnd
		}
	}
}

// broadcastProgress 廣播進度更新
func (c *KeyBatchChecker) broadcastProgress(task *BatchCheckTask) {
	task.wsClientsMu.RLock()
	defer task.wsClientsMu.RUnlock()

	task.mu.RLock()
	progressData, _ := json.Marshal(task.Progress)
	task.mu.RUnlock()

	for client := range task.wsClients {
		if err := client.WriteMessage(websocket.TextMessage, progressData); err != nil {
			// 移除斷開的客戶端
			delete(task.wsClients, client)
			client.Close()
		}
	}
}

// PauseTask 暫停任務
func (c *KeyBatchChecker) PauseTask(taskID string) error {
	c.tasksMu.RLock()
	task, exists := c.tasks[taskID]
	c.tasksMu.RUnlock()

	if !exists {
		return fmt.Errorf("任務不存在")
	}

	select {
	case task.PauseChannel <- true:
		return nil
	default:
		return fmt.Errorf("任務無法暫停")
	}
}

// ResumeTask 恢復任務
func (c *KeyBatchChecker) ResumeTask(taskID string) error {
	c.tasksMu.RLock()
	task, exists := c.tasks[taskID]
	c.tasksMu.RUnlock()

	if !exists {
		return fmt.Errorf("任務不存在")
	}

	select {
	case task.ResumeChannel <- true:
		return nil
	default:
		return fmt.Errorf("任務無法恢復")
	}
}

// CancelTask 取消任務
func (c *KeyBatchChecker) CancelTask(taskID string) error {
	c.tasksMu.RLock()
	task, exists := c.tasks[taskID]
	c.tasksMu.RUnlock()

	if !exists {
		return fmt.Errorf("任務不存在")
	}

	task.CancelFunc()
	return nil
}

// GetTaskProgress 獲取任務進度
func (c *KeyBatchChecker) GetTaskProgress(taskID string) (*BatchCheckProgress, error) {
	c.tasksMu.RLock()
	task, exists := c.tasks[taskID]
	c.tasksMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("任務不存在")
	}

	task.mu.RLock()
	defer task.mu.RUnlock()

	return task.Progress, nil
}

// GetTaskResults 獲取任務結果
func (c *KeyBatchChecker) GetTaskResults(taskID string) ([]BatchCheckResult, error) {
	c.tasksMu.RLock()
	task, exists := c.tasks[taskID]
	c.tasksMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("任務不存在")
	}

	task.mu.RLock()
	defer task.mu.RUnlock()

	return task.Results, nil
}

// AddWebSocketClient 添加 WebSocket 客戶端
func (c *KeyBatchChecker) AddWebSocketClient(taskID string, conn *websocket.Conn) error {
	c.tasksMu.RLock()
	task, exists := c.tasks[taskID]
	c.tasksMu.RUnlock()

	if !exists {
		return fmt.Errorf("任務不存在")
	}

	task.wsClientsMu.Lock()
	task.wsClients[conn] = true
	task.wsClientsMu.Unlock()

	return nil
}

// RemoveWebSocketClient 移除 WebSocket 客戶端
func (c *KeyBatchChecker) RemoveWebSocketClient(taskID string, conn *websocket.Conn) {
	c.tasksMu.RLock()
	task, exists := c.tasks[taskID]
	c.tasksMu.RUnlock()

	if !exists {
		return
	}

	task.wsClientsMu.Lock()
	delete(task.wsClients, conn)
	task.wsClientsMu.Unlock()
}

// CleanupCompletedTasks 清理已完成的任務
func (c *KeyBatchChecker) CleanupCompletedTasks() {
	c.tasksMu.Lock()
	defer c.tasksMu.Unlock()

	for taskID, task := range c.tasks {
		task.mu.RLock()
		status := task.Progress.Status
		startTime := task.Progress.StartTime
		task.mu.RUnlock()

		// 清理 24 小時前完成的任務
		if (status == "completed" || status == "cancelled") &&
		   time.Since(startTime) > 24*time.Hour {
			delete(c.tasks, taskID)
		}
	}
}
