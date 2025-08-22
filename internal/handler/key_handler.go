package handler

import (
	"fmt"
	app_errors "gpt-load/internal/errors"
	"gpt-load/internal/models"
	"gpt-load/internal/response"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// validateGroupIDFromQuery validates and parses group ID from a query parameter.
func validateGroupIDFromQuery(c *gin.Context) (uint, error) {
	groupIDStr := c.Query("group_id")
	if groupIDStr == "" {
		return 0, fmt.Errorf("group_id query parameter is required")
	}

	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil || groupID <= 0 {
		return 0, fmt.Errorf("invalid group_id format")
	}

	return uint(groupID), nil
}

// validateKeysText validates the keys text input
func validateKeysText(keysText string) error {
	if strings.TrimSpace(keysText) == "" {
		return fmt.Errorf("keys text cannot be empty")
	}

	return nil
}

// findGroupByID is a helper function to find a group by its ID.
func (s *Server) findGroupByID(c *gin.Context, groupID uint) (*models.Group, bool) {
	var group models.Group
	if err := s.DB.First(&group, groupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, app_errors.ErrResourceNotFound)
		} else {
			response.Error(c, app_errors.ParseDBError(err))
		}
		return nil, false
	}
	return &group, true
}

// KeyTextRequest defines a generic payload for operations requiring a group ID and a text block of keys.
type KeyTextRequest struct {
	GroupID  uint   `json:"group_id" binding:"required"`
	KeysText string `json:"keys_text" binding:"required"`
}

// GroupIDRequest defines a generic payload for operations requiring only a group ID.
type GroupIDRequest struct {
	GroupID uint `json:"group_id" binding:"required"`
}

// ValidateGroupKeysRequest defines the payload for validating keys in a group.
type ValidateGroupKeysRequest struct {
	GroupID uint   `json:"group_id" binding:"required"`
	Status  string `json:"status,omitempty"`
}

// BatchValidationRequest defines the payload for batch key validation
type BatchValidationRequest struct {
	GroupID uint     `json:"group_id" binding:"required"`
	KeyIDs  []uint   `json:"key_ids,omitempty"` // If empty, validate all keys in group
	Config  *BatchValidationConfig `json:"config,omitempty"`
}

// BatchValidationConfig represents configuration for batch validation
type BatchValidationConfig struct {
	Concurrency        int    `json:"concurrency,omitempty"`
	TimeoutSeconds     int    `json:"timeout_seconds,omitempty"`
	MaxRetries         int    `json:"max_retries,omitempty"`
	RateLimitPerSec    int    `json:"rate_limit_per_sec,omitempty"`
	EnableMultiplexing bool   `json:"enable_multiplexing,omitempty"`
	ProxyURL           string `json:"proxy_url,omitempty"`
}

// AddMultipleKeys handles creating new keys from a text block within a specific group.
func (s *Server) AddMultipleKeys(c *gin.Context) {
	var req KeyTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	if _, ok := s.findGroupByID(c, req.GroupID); !ok {
		return
	}

	if err := validateKeysText(req.KeysText); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, err.Error()))
		return
	}

	result, err := s.KeyService.AddMultipleKeys(req.GroupID, req.KeysText)
	if err != nil {
		if strings.Contains(err.Error(), "batch size exceeds the limit") {
			response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, err.Error()))
		} else if err.Error() == "no valid keys found in the input text" {
			response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, err.Error()))
		} else {
			response.Error(c, app_errors.ParseDBError(err))
		}
		return
	}

	response.Success(c, result)
}

// AddMultipleKeysAsync handles creating new keys from a text block within a specific group.
func (s *Server) AddMultipleKeysAsync(c *gin.Context) {
	var req KeyTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	group, ok := s.findGroupByID(c, req.GroupID)
	if !ok {
		return
	}

	if err := validateKeysText(req.KeysText); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, err.Error()))
		return
	}

	taskStatus, err := s.KeyImportService.StartImportTask(group, req.KeysText)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrTaskInProgress, err.Error()))
		return
	}

	response.Success(c, taskStatus)
}

// ListKeysInGroup handles listing all keys within a specific group with pagination.
func (s *Server) ListKeysInGroup(c *gin.Context) {
	groupID, err := validateGroupIDFromQuery(c)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrBadRequest, err.Error()))
		return
	}

	if _, ok := s.findGroupByID(c, groupID); !ok {
		return
	}

	statusFilter := c.Query("status")
	if statusFilter != "" && statusFilter != models.KeyStatusActive && statusFilter != models.KeyStatusInvalid {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, "Invalid status filter"))
		return
	}

	searchKeyword := c.Query("key")

	query := s.KeyService.ListKeysInGroupQuery(groupID, statusFilter, searchKeyword)

	var keys []models.APIKey
	paginatedResult, err := response.Paginate(c, query, &keys)
	if err != nil {
		response.Error(c, app_errors.ParseDBError(err))
		return
	}

	response.Success(c, paginatedResult)
}

// DeleteMultipleKeys handles deleting keys from a text block within a specific group.
func (s *Server) DeleteMultipleKeys(c *gin.Context) {
	var req KeyTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	if _, ok := s.findGroupByID(c, req.GroupID); !ok {
		return
	}

	if err := validateKeysText(req.KeysText); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, err.Error()))
		return
	}

	result, err := s.KeyService.DeleteMultipleKeys(req.GroupID, req.KeysText)
	if err != nil {
		if strings.Contains(err.Error(), "batch size exceeds the limit") {
			response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, err.Error()))
		} else if err.Error() == "no valid keys found in the input text" {
			response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, err.Error()))
		} else {
			response.Error(c, app_errors.ParseDBError(err))
		}
		return
	}

	response.Success(c, result)
}

// RestoreMultipleKeys handles restoring keys from a text block within a specific group.
func (s *Server) RestoreMultipleKeys(c *gin.Context) {
	var req KeyTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	if _, ok := s.findGroupByID(c, req.GroupID); !ok {
		return
	}

	if err := validateKeysText(req.KeysText); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, err.Error()))
		return
	}

	result, err := s.KeyService.RestoreMultipleKeys(req.GroupID, req.KeysText)
	if err != nil {
		if strings.Contains(err.Error(), "batch size exceeds the limit") {
			response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, err.Error()))
		} else if err.Error() == "no valid keys found in the input text" {
			response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, err.Error()))
		} else {
			response.Error(c, app_errors.ParseDBError(err))
		}
		return
	}

	response.Success(c, result)
}

// TestMultipleKeys handles a one-off validation test for multiple keys.
func (s *Server) TestMultipleKeys(c *gin.Context) {
	var req KeyTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	groupDB, ok := s.findGroupByID(c, req.GroupID)
	if !ok {
		return
	}

	group, err := s.GroupManager.GetGroupByName(groupDB.Name)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrResourceNotFound, fmt.Sprintf("Group '%s' not found", groupDB.Name)))
		return
	}

	if err := validateKeysText(req.KeysText); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, err.Error()))
		return
	}

	start := time.Now()
	results, err := s.KeyService.TestMultipleKeys(group, req.KeysText)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		if strings.Contains(err.Error(), "batch size exceeds the limit") {
			response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, err.Error()))
		} else if err.Error() == "no valid keys found in the input text" {
			response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, err.Error()))
		} else {
			response.Error(c, app_errors.ParseDBError(err))
		}
		return
	}

	response.Success(c, gin.H{
		"results":        results,
		"total_duration": duration,
	})
}

// ValidateGroupKeys initiates a manual validation task for all keys in a group.
func (s *Server) ValidateGroupKeys(c *gin.Context) {
	var req ValidateGroupKeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	// Validate status if provided
	if req.Status != "" && req.Status != models.KeyStatusActive && req.Status != models.KeyStatusInvalid {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, "Invalid status value"))
		return
	}

	groupDB, ok := s.findGroupByID(c, req.GroupID)
	if !ok {
		return
	}

	group, err := s.GroupManager.GetGroupByName(groupDB.Name)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrResourceNotFound, fmt.Sprintf("Group '%s' not found", groupDB.Name)))
		return
	}

	taskStatus, err := s.KeyManualValidationService.StartValidationTask(group, req.Status)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrTaskInProgress, err.Error()))
		return
	}

	response.Success(c, taskStatus)
}

// RestoreAllInvalidKeys sets the status of all 'inactive' keys in a group to 'active'.
func (s *Server) RestoreAllInvalidKeys(c *gin.Context) {
	var req GroupIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	if _, ok := s.findGroupByID(c, req.GroupID); !ok {
		return
	}

	rowsAffected, err := s.KeyService.RestoreAllInvalidKeys(req.GroupID)
	if err != nil {
		response.Error(c, app_errors.ParseDBError(err))
		return
	}

	response.Success(c, gin.H{"message": fmt.Sprintf("%d keys restored.", rowsAffected)})
}

// ClearAllInvalidKeys deletes all 'inactive' keys from a group.
func (s *Server) ClearAllInvalidKeys(c *gin.Context) {
	var req GroupIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	if _, ok := s.findGroupByID(c, req.GroupID); !ok {
		return
	}

	rowsAffected, err := s.KeyService.ClearAllInvalidKeys(req.GroupID)
	if err != nil {
		response.Error(c, app_errors.ParseDBError(err))
		return
	}

	response.Success(c, gin.H{"message": fmt.Sprintf("%d invalid keys cleared.", rowsAffected)})
}

// ClearAllKeys deletes all keys from a group.
func (s *Server) ClearAllKeys(c *gin.Context) {
	var req GroupIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	if _, ok := s.findGroupByID(c, req.GroupID); !ok {
		return
	}

	rowsAffected, err := s.KeyService.ClearAllKeys(req.GroupID)
	if err != nil {
		response.Error(c, app_errors.ParseDBError(err))
		return
	}

	response.Success(c, gin.H{"message": fmt.Sprintf("%d keys cleared.", rowsAffected)})
}

// ExportKeys handles exporting keys to a text file.
func (s *Server) ExportKeys(c *gin.Context) {
	groupID, err := validateGroupIDFromQuery(c)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrBadRequest, err.Error()))
		return
	}

	statusFilter := c.Query("status")
	if statusFilter == "" {
		statusFilter = "all"
	}

	switch statusFilter {
	case "all", models.KeyStatusActive, models.KeyStatusInvalid:
	default:
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, "Invalid status filter"))
		return
	}

	group, ok := s.findGroupByID(c, groupID)
	if !ok {
		return
	}

	filename := fmt.Sprintf("keys-%s-%s.txt", group.Name, statusFilter)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "text/plain; charset=utf-8")

	err = s.KeyService.StreamKeysToWriter(groupID, statusFilter, c.Writer)
	if err != nil {
		log.Printf("Failed to stream keys: %v", err)
	}
}

// ValidateBatchAsync starts an asynchronous batch validation job
func (s *Server) ValidateBatchAsync(c *gin.Context) {
	var req BatchValidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	groupDB, ok := s.findGroupByID(c, req.GroupID)
	if !ok {
		return
	}

	// Get group from group manager
	group, err := s.GroupManager.GetGroupByName(groupDB.Name)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrResourceNotFound, fmt.Sprintf("Group '%s' not found", groupDB.Name)))
		return
	}

	// Get keys to validate
	var keys []*models.APIKey
	if len(req.KeyIDs) > 0 {
		// Validate specific keys
		if err := s.DB.Where("group_id = ? AND id IN ?", req.GroupID, req.KeyIDs).Find(&keys).Error; err != nil {
			response.Error(c, app_errors.ParseDBError(err))
			return
		}
	} else {
		// Validate all keys in group
		if err := s.DB.Where("group_id = ?", req.GroupID).Find(&keys).Error; err != nil {
			response.Error(c, app_errors.ParseDBError(err))
			return
		}
	}

	if len(keys) == 0 {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, "No keys found for validation"))
		return
	}

	// Update validation configuration if provided
	if req.Config != nil {
		config := &services.BatchValidationConfig{
			Concurrency:        req.Config.Concurrency,
			TimeoutSeconds:     req.Config.TimeoutSeconds,
			MaxRetries:         req.Config.MaxRetries,
			RateLimitPerSec:    req.Config.RateLimitPerSec,
			EnableMultiplexing: req.Config.EnableMultiplexing,
			ProxyURL:           req.Config.ProxyURL,
		}

		// Apply default values if not provided
		if config.Concurrency <= 0 {
			config.Concurrency = 50
		}
		if config.TimeoutSeconds <= 0 {
			config.TimeoutSeconds = 15
		}
		if config.MaxRetries < 0 {
			config.MaxRetries = 3
		}
		if config.RateLimitPerSec <= 0 {
			config.RateLimitPerSec = 100
		}

		s.EnhancedKeyValidationService.UpdateConfig(config)
	}

	// Start batch validation
	job, err := s.EnhancedKeyValidationService.ValidateBatchAsync(c.Request.Context(), group, keys)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, fmt.Sprintf("Failed to start batch validation: %v", err)))
		return
	}

	response.Success(c, job)
}

// GetValidationStatus returns the status of a batch validation job
func (s *Server) GetValidationStatus(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrBadRequest, "Job ID is required"))
		return
	}

	job, err := s.EnhancedKeyValidationService.GetJobStatus(jobID)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrResourceNotFound, err.Error()))
		return
	}

	response.Success(c, job)
}

// CancelValidation cancels a running batch validation job
func (s *Server) CancelValidation(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrBadRequest, "Job ID is required"))
		return
	}

	err := s.EnhancedKeyValidationService.CancelJob(jobID)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrResourceNotFound, err.Error()))
		return
	}

	response.Success(c, gin.H{"message": "Validation job cancelled successfully"})
}

// GetBatchValidationConfig returns the current batch validation configuration
func (s *Server) GetBatchValidationConfig(c *gin.Context) {
	config := s.EnhancedKeyValidationService.GetConfig()
	response.Success(c, config)
}

// UpdateBatchValidationConfig updates the batch validation configuration
func (s *Server) UpdateBatchValidationConfig(c *gin.Context) {
	var config services.BatchValidationConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	// Validate configuration
	if config.Concurrency <= 0 || config.Concurrency > 200 {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, "Concurrency must be between 1 and 200"))
		return
	}
	if config.TimeoutSeconds <= 0 || config.TimeoutSeconds > 120 {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, "Timeout must be between 1 and 120 seconds"))
		return
	}
	if config.MaxRetries < 0 || config.MaxRetries > 10 {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, "Max retries must be between 0 and 10"))
		return
	}
	if config.RateLimitPerSec <= 0 || config.RateLimitPerSec > 500 {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrValidation, "Rate limit must be between 1 and 500 per second"))
		return
	}

	s.EnhancedKeyValidationService.UpdateConfig(&config)
	response.Success(c, gin.H{"message": "Configuration updated successfully"})
}
