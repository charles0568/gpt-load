import http from '@/utils/http'

export interface BatchCheckRequest {
  group_id: number
  batch_size?: number
  concurrency?: number
}

export interface BatchCheckProgress {
  task_id: string
  status: 'running' | 'paused' | 'completed' | 'cancelled' | 'error'
  total_keys: number
  processed_keys: number
  valid_keys: number
  invalid_keys: number
  current_batch: number
  total_batches: number
  start_time: string
  estimated_end?: string
  error_message?: string
  speed: number
}

export interface BatchCheckResult {
  key_id: number
  key: string
  group_id: number
  valid: boolean
  response_time_ms: number
  error_message?: string
  checked_at: string
}

export interface GetResultsParams {
  page?: number
  page_size?: number
}

export interface GetResultsResponse {
  results: BatchCheckResult[]
  pagination: {
    page: number
    page_size: number
    total: number
    total_pages: number
  }
}

/**
 * 批量檢查 API
 */
export const batchCheckAPI = {
  /**
   * 開始批量檢查
   */
  start(data: BatchCheckRequest) {
    return http.post<{ task_id: string; message: string }>('/api/keys/batch-check/start', data)
  },

  /**
   * 獲取任務進度
   */
  getProgress(taskId: string) {
    return http.get<BatchCheckProgress>(`/api/keys/batch-check/${taskId}/progress`)
  },

  /**
   * 獲取任務結果
   */
  getResults(taskId: string, params?: GetResultsParams) {
    return http.get<GetResultsResponse>(`/api/keys/batch-check/${taskId}/results`, { params })
  },

  /**
   * 暫停任務
   */
  pause(taskId: string) {
    return http.post<{ message: string }>(`/api/keys/batch-check/${taskId}/pause`)
  },

  /**
   * 恢復任務
   */
  resume(taskId: string) {
    return http.post<{ message: string }>(`/api/keys/batch-check/${taskId}/resume`)
  },

  /**
   * 取消任務
   */
  cancel(taskId: string) {
    return http.post<{ message: string }>(`/api/keys/batch-check/${taskId}/cancel`)
  },

  /**
   * 匯出結果
   */
  export(taskId: string, format: 'csv' | 'json' = 'csv', filter?: 'valid' | 'invalid') {
    const params = new URLSearchParams({ format })
    if (filter === 'valid') {
      params.append('only_valid', 'true')
    } else if (filter === 'invalid') {
      params.append('only_invalid', 'true')
    }

    const url = `/api/keys/batch-check/${taskId}/export?${params.toString()}`
    window.open(url, '_blank')
  },

  /**
   * 批量刪除無效密鑰
   */
  deleteInvalid(taskId: string) {
    return http.post<{ message: string; deleted_count: number }>(`/api/keys/batch-check/${taskId}/delete-invalid`)
  },

  /**
   * 建立 WebSocket 連接
   */
  createWebSocket(taskId: string, onMessage: (progress: BatchCheckProgress) => void, onError?: (error: Event) => void) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/keys/batch-check/${taskId}/ws`

    const ws = new WebSocket(wsUrl)

    ws.onmessage = (event) => {
      try {
        const progress = JSON.parse(event.data) as BatchCheckProgress
        onMessage(progress)
      } catch (error) {
        console.error('解析 WebSocket 訊息失敗:', error)
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket 錯誤:', error)
      onError?.(error)
    }

    ws.onclose = () => {
      console.log('WebSocket 連接已關閉')
    }

    return ws
  }
}

/**
 * 批量檢查狀態文字映射
 */
export const statusTextMap = {
  running: '檢查中',
  paused: '已暫停',
  completed: '已完成',
  cancelled: '已取消',
  error: '錯誤'
}

/**
 * 格式化檢查速度
 */
export const formatSpeed = (speed: number): string => {
  if (speed < 1) {
    return `${Math.round(speed * 60)}/分鐘`
  }
  return `${Math.round(speed)}/秒`
}

/**
 * 格式化持續時間
 */
export const formatDuration = (startTime: string): string => {
  if (!startTime) return '-'

  const start = new Date(startTime)
  const now = new Date()
  const diff = now.getTime() - start.getTime()

  const hours = Math.floor(diff / (1000 * 60 * 60))
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
  const seconds = Math.floor((diff % (1000 * 60)) / 1000)

  if (hours > 0) {
    return `${hours}時${minutes}分${seconds}秒`
  } else if (minutes > 0) {
    return `${minutes}分${seconds}秒`
  } else {
    return `${seconds}秒`
  }
}

/**
 * 格式化預估時間
 */
export const formatEstimatedTime = (estimatedEnd?: string): string => {
  if (!estimatedEnd) return '-'

  const end = new Date(estimatedEnd)
  const now = new Date()
  const diff = end.getTime() - now.getTime()

  if (diff <= 0) return '即將完成'

  const hours = Math.floor(diff / (1000 * 60 * 60))
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))

  if (hours > 0) {
    return `約 ${hours}時${minutes}分後`
  } else if (minutes > 0) {
    return `約 ${minutes}分鐘後`
  } else {
    return '不到1分鐘'
  }
}

/**
 * 計算有效率
 */
export const calculateValidRate = (validKeys: number, totalKeys: number): number => {
  if (totalKeys === 0) return 0
  return Math.round((validKeys / totalKeys) * 100)
}

/**
 * 獲取狀態顏色
 */
export const getStatusColor = (status: string): string => {
  switch (status) {
    case 'running': return '#2080f0'
    case 'paused': return '#f0a020'
    case 'completed': return '#18a058'
    case 'cancelled': return '#d03050'
    case 'error': return '#d03050'
    default: return '#666'
  }
}

/**
 * 獲取狀態圖示
 */
export const getStatusIcon = (status: string): string => {
  switch (status) {
    case 'running': return 'time-outline'
    case 'paused': return 'pause-outline'
    case 'completed': return 'checkmark-outline'
    case 'cancelled': return 'close-outline'
    case 'error': return 'warning-outline'
    default: return 'help-outline'
  }
}
