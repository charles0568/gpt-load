package errors

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// APIError defines a standard error structure for API responses.
type APIError struct {
	HTTPStatus int
	Code       string
	Message    string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return e.Message
}

// Predefined API errors
var (
	ErrBadRequest         = &APIError{HTTPStatus: http.StatusBadRequest, Code: "BAD_REQUEST", Message: "無效的請求參數"}
	ErrInvalidJSON        = &APIError{HTTPStatus: http.StatusBadRequest, Code: "INVALID_JSON", Message: "無效的 JSON 格式"}
	ErrValidation         = &APIError{HTTPStatus: http.StatusBadRequest, Code: "VALIDATION_FAILED", Message: "輸入驗證失敗"}
	ErrDuplicateResource  = &APIError{HTTPStatus: http.StatusConflict, Code: "DUPLICATE_RESOURCE", Message: "資源已存在"}
	ErrResourceNotFound   = &APIError{HTTPStatus: http.StatusNotFound, Code: "NOT_FOUND", Message: "找不到資源"}
	ErrInternalServer     = &APIError{HTTPStatus: http.StatusInternalServerError, Code: "INTERNAL_SERVER_ERROR", Message: "發生未預期的錯誤"}
	ErrDatabase           = &APIError{HTTPStatus: http.StatusInternalServerError, Code: "DATABASE_ERROR", Message: "資料庫操作失敗"}
	ErrUnauthorized       = &APIError{HTTPStatus: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: "認證失敗"}
	ErrForbidden          = &APIError{HTTPStatus: http.StatusForbidden, Code: "FORBIDDEN", Message: "您沒有權限存取此資源"}
	ErrTaskInProgress     = &APIError{HTTPStatus: http.StatusConflict, Code: "TASK_IN_PROGRESS", Message: "已有任務正在進行中"}
	ErrBadGateway         = &APIError{HTTPStatus: http.StatusBadGateway, Code: "BAD_GATEWAY", Message: "上游服務錯誤"}
	ErrNoActiveKeys       = &APIError{HTTPStatus: http.StatusServiceUnavailable, Code: "NO_ACTIVE_KEYS", Message: "此分組沒有可用的活躍 API 密鑰"}
	ErrMaxRetriesExceeded = &APIError{HTTPStatus: http.StatusBadGateway, Code: "MAX_RETRIES_EXCEEDED", Message: "請求在達到最大重試次數後失敗"}
	ErrNoKeysAvailable    = &APIError{HTTPStatus: http.StatusServiceUnavailable, Code: "NO_KEYS_AVAILABLE", Message: "沒有可用的 API 密鑰來處理請求"}
)

// NewAPIError creates a new APIError with a custom message.
func NewAPIError(base *APIError, message string) *APIError {
	return &APIError{
		HTTPStatus: base.HTTPStatus,
		Code:       base.Code,
		Message:    message,
	}
}

// NewAPIErrorWithUpstream creates a new APIError specifically for wrapping raw upstream errors.
func NewAPIErrorWithUpstream(statusCode int, code string, upstreamMessage string) *APIError {
	return &APIError{
		HTTPStatus: statusCode,
		Code:       code,
		Message:    upstreamMessage,
	}
}

// ParseDBError intelligently converts a GORM error into a standard APIError.
func ParseDBError(err error) *APIError {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrResourceNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" { // unique_violation
			return ErrDuplicateResource
		}
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		if mysqlErr.Number == 1062 { // Duplicate entry
			return ErrDuplicateResource
		}
	}

	// Generic check for SQLite
	if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed") {
		return ErrDuplicateResource
	}

	return ErrDatabase
}
