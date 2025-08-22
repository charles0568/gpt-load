// Package services provides enhanced key validation with high-performance batch processing
package services

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"gpt-load/internal/channel"
	"gpt-load/internal/models"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// ValidationResult represents the result of a single key validation
type ValidationResult struct {
	Key       *models.APIKey `json:"key"`
	IsValid   bool           `json:"is_valid"`
	Error     string         `json:"error,omitempty"`
	Duration  time.Duration  `json:"duration"`
	Timestamp time.Time      `json:"timestamp"`
}

// BatchValidationStats represents statistics for batch validation
type BatchValidationStats struct {
	TotalKeys     int32         `json:"total_keys"`
	ValidKeys     int32         `json:"valid_keys"`
	InvalidKeys   int32         `json:"invalid_keys"`
	ProcessedKeys int32         `json:"processed_keys"`
	StartTime     time.Time     `json:"start_time"`
	Duration      time.Duration `json:"duration"`
	ErrorRate     float64       `json:"error_rate"`
}

// BatchValidationConfig represents configuration for batch validation
type BatchValidationConfig struct {
	Concurrency          int           `json:"concurrency"`
	TimeoutSeconds       int           `json:"timeout_seconds"`
	MaxRetries           int           `json:"max_retries"`
	RetryDelay           time.Duration `json:"retry_delay"`
	RateLimitPerSec      int           `json:"rate_limit_per_sec"`
	EnableMultiplexing   bool          `json:"enable_multiplexing"`
	ProxyURL             string        `json:"proxy_url,omitempty"`
	EnableHTTP2          bool          `json:"enable_http2"`
	StreamingThreshold   int           `json:"streaming_threshold"`
	BackupResults        bool          `json:"backup_results"`
	MaxRetryBackoffSec   int           `json:"max_retry_backoff_sec"`
	EnableJitter         bool          `json:"enable_jitter"`
}

// EnhancedKeyValidationService provides high-performance batch key validation
type EnhancedKeyValidationService struct {
	channelFactory *channel.Factory
	config         *BatchValidationConfig
	limiter        *rate.Limiter
	mu             sync.RWMutex
	activeJobs     map[string]*ValidationJob
}

// ValidationJob represents an active validation job
type ValidationJob struct {
	ID           string                `json:"id"`
	GroupID      uint                  `json:"group_id"`
	Status       string                `json:"status"` // "running", "completed", "failed"
	Stats        *BatchValidationStats `json:"stats"`
	Results      []ValidationResult    `json:"results"`
	Context      context.Context       `json:"-"`
	Cancel       context.CancelFunc    `json:"-"`
	ProgressChan chan ValidationResult `json:"-"`
}

// NewEnhancedKeyValidationService creates a new enhanced key validation service
func NewEnhancedKeyValidationService(channelFactory *channel.Factory) *EnhancedKeyValidationService {
	config := &BatchValidationConfig{
		Concurrency:          50,                // High concurrency like Gemini-Keychecker
		TimeoutSeconds:       15,                // Fast timeout for responsiveness
		MaxRetries:           3,                 // Smart retry mechanism
		RetryDelay:           time.Second * 2,   // Base retry delay
		RateLimitPerSec:      100,               // Rate limiting to avoid API limits
		EnableMultiplexing:   true,              // Enable HTTP/2 multiplexing
		EnableHTTP2:          true,              // Force HTTP/2 for better performance
		StreamingThreshold:   1000,              // Use streaming for large batches
		BackupResults:        true,              // Backup validation results
		MaxRetryBackoffSec:   30,                // Maximum retry backoff time
		EnableJitter:         true,              // Add jitter to retry delays
	}

	return &EnhancedKeyValidationService{
		channelFactory: channelFactory,
		config:         config,
		limiter:        rate.NewLimiter(rate.Limit(config.RateLimitPerSec), config.RateLimitPerSec),
		activeJobs:     make(map[string]*ValidationJob),
	}
}

// ValidateBatchAsync starts asynchronous batch validation
func (s *EnhancedKeyValidationService) ValidateBatchAsync(
	ctx context.Context,
	group *models.Group,
	keys []*models.APIKey,
) (*ValidationJob, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys provided for validation")
	}

	jobID := fmt.Sprintf("batch_%d_%d", group.ID, time.Now().Unix())
	jobCtx, cancel := context.WithCancel(ctx)

	job := &ValidationJob{
		ID:      jobID,
		GroupID: group.ID,
		Status:  "running",
		Stats: &BatchValidationStats{
			TotalKeys: int32(len(keys)),
			StartTime: time.Now(),
		},
		Results:      make([]ValidationResult, 0, len(keys)),
		Context:      jobCtx,
		Cancel:       cancel,
		ProgressChan: make(chan ValidationResult, len(keys)),
	}

	s.mu.Lock()
	s.activeJobs[jobID] = job
	s.mu.Unlock()

	// Start validation in background
	go s.executeBatchValidation(job, group, keys)

	return job, nil
}

// executeBatchValidation performs the actual batch validation with high concurrency
func (s *EnhancedKeyValidationService) executeBatchValidation(
	job *ValidationJob,
	group *models.Group,
	keys []*models.APIKey,
) {
	defer func() {
		job.Status = "completed"
		job.Stats.Duration = time.Since(job.Stats.StartTime)
		close(job.ProgressChan)

		// Calculate error rate
		if job.Stats.TotalKeys > 0 {
			job.Stats.ErrorRate = float64(job.Stats.InvalidKeys) / float64(job.Stats.TotalKeys) * 100
		}

		logrus.Infof("Batch validation completed for group %d: %d/%d valid keys (%.2f%% error rate)",
			group.ID, job.Stats.ValidKeys, job.Stats.TotalKeys, job.Stats.ErrorRate)
	}()

	// Get channel for validation
	channelHandler, err := s.channelFactory.GetChannel(group)
	if err != nil {
		logrus.Errorf("Failed to get channel for group %d: %v", group.ID, err)
		job.Status = "failed"
		return
	}

	// Use streaming processing for large batches (memory optimization)
	useStreaming := len(keys) >= s.config.StreamingThreshold
	
	if useStreaming {
		logrus.Infof("Using streaming processing for %d keys (threshold: %d)", len(keys), s.config.StreamingThreshold)
		s.executeStreamingValidation(job, channelHandler, keys, group)
	} else {
		s.executeBatchedValidation(job, channelHandler, keys, group)
	}
}

// executeBatchedValidation handles normal batch processing
func (s *EnhancedKeyValidationService) executeBatchedValidation(
	job *ValidationJob,
	channelHandler channel.ChannelProxy,
	keys []*models.APIKey,
	group *models.Group,
) {
	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, s.config.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, key := range keys {
		select {
		case <-job.Context.Done():
			logrus.Info("Batch validation cancelled")
			return
		case semaphore <- struct{}{}:
			wg.Add(1)
			go func(k *models.APIKey) {
				defer func() {
					<-semaphore
					wg.Done()
				}()

				result := s.validateSingleKeyWithRetry(job.Context, channelHandler, k, group)

				// Update stats atomically
				atomic.AddInt32(&job.Stats.ProcessedKeys, 1)
				if result.IsValid {
					atomic.AddInt32(&job.Stats.ValidKeys, 1)
				} else {
					atomic.AddInt32(&job.Stats.InvalidKeys, 1)
				}

				// Store result safely
				mu.Lock()
				job.Results = append(job.Results, result)
				mu.Unlock()

				// Send progress update
				select {
				case job.ProgressChan <- result:
				case <-job.Context.Done():
					return
				}
			}(key)
		}
	}

	wg.Wait()
}

// executeStreamingValidation handles memory-efficient streaming processing
func (s *EnhancedKeyValidationService) executeStreamingValidation(
	job *ValidationJob,
	channelHandler channel.ChannelProxy,
	keys []*models.APIKey,
	group *models.Group,
) {
	// Process keys in chunks to optimize memory usage
	chunkSize := s.config.Concurrency * 2 // Process 2x concurrency at a time
	
	for i := 0; i < len(keys); i += chunkSize {
		end := i + chunkSize
		if end > len(keys) {
			end = len(keys)
		}
		
		chunk := keys[i:end]
		
		// Check for cancellation
		select {
		case <-job.Context.Done():
			logrus.Info("Streaming validation cancelled")
			return
		default:
		}
		
		s.processChunk(job, channelHandler, chunk, group)
		
		// Add a small delay between chunks to prevent overwhelming the system
		if i+chunkSize < len(keys) {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// processChunk processes a chunk of keys concurrently
func (s *EnhancedKeyValidationService) processChunk(
	job *ValidationJob,
	channelHandler channel.ChannelProxy,
	chunk []*models.APIKey,
	group *models.Group,
) {
	semaphore := make(chan struct{}, s.config.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	for _, key := range chunk {
		select {
		case <-job.Context.Done():
			return
		case semaphore <- struct{}{}:
			wg.Add(1)
			go func(k *models.APIKey) {
				defer func() {
					<-semaphore
					wg.Done()
				}()

				result := s.validateSingleKeyWithRetry(job.Context, channelHandler, k, group)

				// Update stats atomically
				atomic.AddInt32(&job.Stats.ProcessedKeys, 1)
				if result.IsValid {
					atomic.AddInt32(&job.Stats.ValidKeys, 1)
				} else {
					atomic.AddInt32(&job.Stats.InvalidKeys, 1)
				}

				// Store result safely (append to results)
				mu.Lock()
				job.Results = append(job.Results, result)
				mu.Unlock()

				// Send progress update
				select {
				case job.ProgressChan <- result:
				case <-job.Context.Done():
					return
				}
			}(key)
		}
	}

	wg.Wait()
}

// validateSingleKeyWithRetry validates a single key with exponential backoff retry
func (s *EnhancedKeyValidationService) validateSingleKeyWithRetry(
	ctx context.Context,
	channelHandler channel.ChannelProxy,
	key *models.APIKey,
	group *models.Group,
) ValidationResult {
	start := time.Now()

	for attempt := 0; attempt < s.config.MaxRetries; attempt++ {
		// Rate limiting
		if err := s.limiter.Wait(ctx); err != nil {
			return ValidationResult{
				Key:       key,
				IsValid:   false,
				Error:     fmt.Sprintf("Rate limit error: %v", err),
				Duration:  time.Since(start),
				Timestamp: time.Now(),
			}
		}

		// Create timeout context for this attempt
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.TimeoutSeconds)*time.Second)

		isValid, err := channelHandler.ValidateKey(timeoutCtx, key, group)
		cancel()

		if err == nil {
			return ValidationResult{
				Key:       key,
				IsValid:   isValid,
				Duration:  time.Since(start),
				Timestamp: time.Now(),
			}
		}

		// Check if we should retry
		if attempt < s.config.MaxRetries-1 {
			// Enhanced exponential backoff with jitter (like Gemini-Keychecker)
			delay := s.config.RetryDelay * time.Duration(1<<attempt)
			
			// Apply maximum backoff limit
			maxDelay := time.Duration(s.config.MaxRetryBackoffSec) * time.Second
			if delay > maxDelay {
				delay = maxDelay
			}
			
			// Add jitter to prevent thundering herd
			if s.config.EnableJitter {
				jitter := time.Duration(rand.Int63n(int64(delay.Seconds()*0.1))) * time.Second
				delay += jitter
			}

			logrus.Debugf("Retrying key validation after %v (attempt %d/%d)", delay, attempt+1, s.config.MaxRetries)

			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return ValidationResult{
					Key:       key,
					IsValid:   false,
					Error:     "Validation cancelled",
					Duration:  time.Since(start),
					Timestamp: time.Now(),
				}
			}
		}

		// Final attempt failed
		return ValidationResult{
			Key:       key,
			IsValid:   false,
			Error:     fmt.Sprintf("Validation failed after %d attempts: %v", s.config.MaxRetries, err),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}

	return ValidationResult{
		Key:       key,
		IsValid:   false,
		Error:     "Unexpected validation flow",
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}
}

// GetJobStatus returns the status of a validation job
func (s *EnhancedKeyValidationService) GetJobStatus(jobID string) (*ValidationJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.activeJobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

// CancelJob cancels a running validation job
func (s *EnhancedKeyValidationService) CancelJob(jobID string) error {
	s.mu.RLock()
	job, exists := s.activeJobs[jobID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.Cancel()
	job.Status = "cancelled"
	return nil
}

// CleanupCompletedJobs removes completed jobs older than specified duration
func (s *EnhancedKeyValidationService) CleanupCompletedJobs(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for jobID, job := range s.activeJobs {
		if (job.Status == "completed" || job.Status == "failed" || job.Status == "cancelled") &&
			job.Stats.StartTime.Before(cutoff) {
			delete(s.activeJobs, jobID)
		}
	}
}

// UpdateConfig updates the validation configuration
func (s *EnhancedKeyValidationService) UpdateConfig(config *BatchValidationConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config
	s.limiter = rate.NewLimiter(rate.Limit(config.RateLimitPerSec), config.RateLimitPerSec)
}

// GetConfig returns the current validation configuration
func (s *EnhancedKeyValidationService) GetConfig() *BatchValidationConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.config
}
