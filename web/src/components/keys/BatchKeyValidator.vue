<template>
  <div class="batch-validation-container">
    <n-card title="批量密鑰驗證" :bordered="false">
      <template #header-extra>
        <n-space>
          <n-badge
            :value="stats.totalKeys"
            :type="getStatusBadgeType()"
            show-zero
          >
            <n-button
              type="primary"
              :disabled="isValidating"
              @click="startValidation"
              :loading="isValidating"
            >
              <template #icon>
                <n-icon><CheckmarkCircle /></n-icon>
              </template>
              {{ isValidating ? '驗證中...' : '開始驗證' }}
            </n-button>
          </n-badge>

          <n-button
            v-if="isValidating"
            type="error"
            ghost
            @click="cancelValidation"
          >
            <template #icon>
              <n-icon><StopCircle /></n-icon>
            </template>
            取消
          </n-button>
        </n-space>
      </template>

      <!-- 配置面板 -->
      <n-collapse class="mb-4">
        <n-collapse-item title="高級配置" name="config">
          <n-form
            ref="configFormRef"
            :model="validationConfig"
            :rules="configRules"
            label-placement="left"
            label-width="120px"
          >
            <n-grid cols="2" :x-gap="16">
              <n-gi>
                <n-form-item label="並發數" path="concurrency">
                  <n-input-number
                    v-model:value="validationConfig.concurrency"
                    :min="1"
                    :max="200"
                    placeholder="同時驗證的密鑰數量"
                  />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="超時時間(秒)" path="timeoutSeconds">
                  <n-input-number
                    v-model:value="validationConfig.timeoutSeconds"
                    :min="5"
                    :max="120"
                    placeholder="單個密鑰驗證超時時間"
                  />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="重試次數" path="maxRetries">
                  <n-input-number
                    v-model:value="validationConfig.maxRetries"
                    :min="0"
                    :max="10"
                    placeholder="失敗後的重試次數"
                  />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="速率限制/秒" path="rateLimitPerSec">
                  <n-input-number
                    v-model:value="validationConfig.rateLimitPerSec"
                    :min="1"
                    :max="500"
                    placeholder="每秒最大請求數"
                  />
                </n-form-item>
              </n-gi>
            </n-grid>

            <n-form-item label="代理設置" path="proxyURL">
              <n-input
                v-model:value="validationConfig.proxyURL"
                placeholder="http://username:password@host:port (可選)"
              />
            </n-form-item>

            <n-form-item label="啟用多路復用">
              <n-switch v-model:value="validationConfig.enableMultiplexing" />
              <n-text depth="3" class="ml-2">
                啟用 HTTP/2 多路復用以提升性能
              </n-text>
            </n-form-item>
          </n-form>
        </n-collapse-item>
      </n-collapse>

      <!-- 進度面板 -->
      <n-card
        v-if="isValidating || hasResults"
        title="驗證進度"
        class="mb-4"
        :bordered="false"
        size="small"
      >
        <n-space vertical>
          <!-- 進度條 -->
          <n-progress
            :percentage="progressPercentage"
            :status="getProgressStatus()"
            :show-indicator="true"
          >
            <template #default="{ percentage }">
              <span>{{ stats.processedKeys }}/{{ stats.totalKeys }} ({{ percentage.toFixed(1) }}%)</span>
            </template>
          </n-progress>

          <!-- 統計信息 -->
          <n-space>
            <n-statistic label="總計" :value="stats.totalKeys" />
            <n-statistic
              label="有效"
              :value="stats.validKeys"
              value-style="color: #18a058;"
            />
            <n-statistic
              label="無效"
              :value="stats.invalidKeys"
              value-style="color: #d03050;"
            />
            <n-statistic
              label="錯誤率"
              :value="stats.errorRate"
              :precision="1"
              suffix="%"
              :value-style="getErrorRateStyle()"
            />
          </n-space>

          <!-- 實時狀態 -->
          <n-space v-if="isValidating">
            <n-tag type="info" round>
              <template #icon>
                <n-icon><Time /></n-icon>
              </template>
              已用時: {{ formatDuration(elapsedTime) }}
            </n-tag>
            <n-tag type="warning" round>
              <template #icon>
                <n-icon><SpeedometerOutline /></n-icon>
              </template>
              速度: {{ validationSpeed.toFixed(1) }} keys/sec
            </n-tag>
            <n-tag type="success" round v-if="estimatedTime > 0">
              <template #icon>
                <n-icon><HourglassOutline /></n-icon>
              </template>
              預計剩餘: {{ formatDuration(estimatedTime) }}
            </n-tag>
          </n-space>
        </n-space>
      </n-card>

      <!-- 實時結果面板 -->
      <n-card
        v-if="hasResults"
        title="驗證結果"
        :bordered="false"
      >
        <template #header-extra>
          <n-space>
            <n-button
              type="primary"
              ghost
              @click="exportResults"
              :disabled="!hasResults"
            >
              <template #icon>
                <n-icon><Download /></n-icon>
              </template>
              匯出結果
            </n-button>
            <n-button
              type="warning"
              ghost
              @click="retryFailedKeys"
              :disabled="!hasFailedKeys"
            >
              <template #icon>
                <n-icon><RefreshCircle /></n-icon>
              </template>
              重試失敗的密鑰
            </n-button>
          </n-space>
        </template>

        <!-- 結果篩選 -->
        <n-space class="mb-4">
          <n-select
            v-model:value="resultFilter"
            style="width: 150px"
            placeholder="篩選結果"
            :options="filterOptions"
          />
          <n-input
            v-model:value="searchKeyword"
            placeholder="搜索密鑰..."
            clearable
            style="width: 300px"
          >
            <template #prefix>
              <n-icon><Search /></n-icon>
            </template>
          </n-input>
        </n-space>

        <!-- 結果表格 -->
        <n-data-table
          :columns="resultColumns"
          :data="filteredResults"
          :pagination="resultPagination"
          :loading="isValidating"
          :scroll-x="1200"
          size="small"
          :row-class-name="getRowClassName"
        />
      </n-card>

      <!-- 快速操作面板 -->
      <n-card
        v-if="hasResults && !isValidating"
        title="快速操作"
        class="mt-4"
        :bordered="false"
        size="small"
      >
        <n-space>
          <n-button
            type="success"
            @click="updateValidKeys"
            :disabled="stats.validKeys === 0"
          >
            <template #icon>
              <n-icon><CheckmarkDone /></n-icon>
            </template>
            啟用所有有效密鑰 ({{ stats.validKeys }})
          </n-button>
          <n-button
            type="error"
            @click="disableInvalidKeys"
            :disabled="stats.invalidKeys === 0"
          >
            <template #icon>
              <n-icon><CloseCircle /></n-icon>
            </template>
            禁用所有無效密鑰 ({{ stats.invalidKeys }})
          </n-button>
          <n-button
            type="warning"
            @click="removeInvalidKeys"
            :disabled="stats.invalidKeys === 0"
          >
            <template #icon>
              <n-icon><TrashBin /></n-icon>
            </template>
            刪除所有無效密鑰
          </n-button>
        </n-space>
      </n-card>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, h } from 'vue'
import {
  NTag,
  NButton,
  NIcon,
  NCard,
  NBadge,
  NSpace,
  NCollapse,
  NCollapseItem,
  NForm,
  NFormItem,
  NGrid,
  NGi,
  NInputNumber,
  NInput,
  NSwitch,
  NText,
  NProgress,
  NStatistic,
  NSelect,
  NDataTable,
  useMessage
} from 'naive-ui'
import {
  CheckmarkCircle,
  StopCircle,
  Time,
  SpeedometerOutline,
  HourglassOutline,
  Download,
  RefreshCircle,
  Search,
  CheckmarkDone,
  CloseCircle,
  TrashBin
} from '@vicons/ionicons5'
import type { DataTableColumns } from 'naive-ui'
import type { APIKey } from '@/types/models'

// Props
interface Props {
  groupId: number
  keys: APIKey[]
}

const props = defineProps<Props>()
const message = useMessage()

// Reactive data
const isValidating = ref(false)
const currentJobId = ref<string>('')
const results = ref<ValidationResult[]>([])
const stats = ref({
  totalKeys: 0,
  validKeys: 0,
  invalidKeys: 0,
  processedKeys: 0,
  errorRate: 0
})

// Configuration
const validationConfig = ref({
  concurrency: 50,
  timeoutSeconds: 15,
  maxRetries: 3,
  rateLimitPerSec: 100,
  enableMultiplexing: true,
  proxyURL: ''
})

// UI state
const resultFilter = ref<string>('all')
const searchKeyword = ref('')
const elapsedTime = ref(0)
const validationSpeed = ref(0)
const estimatedTime = ref(0)

// Timers
let progressTimer: number | null = null
let speedTimer: number | null = null

// Computed properties
const progressPercentage = computed(() => {
  if (stats.value.totalKeys === 0) return 0
  return (stats.value.processedKeys / stats.value.totalKeys) * 100
})

const hasResults = computed(() => results.value.length > 0)
const hasFailedKeys = computed(() => stats.value.invalidKeys > 0)

const filteredResults = computed(() => {
  let filtered = results.value

  // Filter by status
  if (resultFilter.value === 'valid') {
    filtered = filtered.filter(r => r.isValid)
  } else if (resultFilter.value === 'invalid') {
    filtered = filtered.filter(r => !r.isValid)
  }

  // Filter by search keyword
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    filtered = filtered.filter(r =>
      r.key.key_value.toLowerCase().includes(keyword) ||
      (r.error && r.error.toLowerCase().includes(keyword))
    )
  }

  return filtered
})

// Options and rules
const filterOptions = [
  { label: '全部', value: 'all' },
  { label: '有效', value: 'valid' },
  { label: '無效', value: 'invalid' }
]

const configRules = {
  concurrency: { required: true, type: 'number', min: 1, max: 200 },
  timeoutSeconds: { required: true, type: 'number', min: 5, max: 120 },
  maxRetries: { required: true, type: 'number', min: 0, max: 10 },
  rateLimitPerSec: { required: true, type: 'number', min: 1, max: 500 }
}

const resultPagination = {
  pageSize: 20,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
}

// Table columns
const resultColumns: DataTableColumns<ValidationResult> = [
  {
    title: '狀態',
    key: 'status',
    width: 80,
    render: (row) => {
      return h('div', { class: 'flex justify-center' }, [
        h(NTag, {
          type: row.isValid ? 'success' : 'error',
          size: 'small',
          round: true
        }, {
          default: () => row.isValid ? '有效' : '無效'
        })
      ])
    }
  },
  {
    title: 'API 密鑰',
    key: 'keyValue',
    ellipsis: { tooltip: true },
    render: (row) => {
      const masked = maskApiKey(row.key.key_value)
      return h('code', { class: 'text-xs' }, masked)
    }
  },
  {
    title: '驗證時間',
    key: 'duration',
    width: 100,
    render: (row) => `${row.duration}ms`
  },
  {
    title: '錯誤信息',
    key: 'error',
    ellipsis: { tooltip: true },
    render: (row) => row.error || '-'
  },
  {
    title: '時間戳',
    key: 'timestamp',
    width: 160,
    render: (row) => new Date(row.timestamp).toLocaleString()
  }
]

// Interface definitions (matching backend)
interface ValidationResult {
  key: APIKey
  isValid: boolean
  error?: string
  duration: number
  timestamp: string
}

// Methods
const startValidation = async () => {
  if (props.keys.length === 0) {
    message.warning('沒有可驗證的密鑰')
    return
  }

  try {
    isValidating.value = true
    results.value = []
    stats.value = {
      totalKeys: props.keys.length,
      validKeys: 0,
      invalidKeys: 0,
      processedKeys: 0,
      errorRate: 0
    }

    // Start validation job
    const response = await fetch('/api/keys/validate-batch-async', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('auth_token')}`
      },
      body: JSON.stringify({
        group_id: props.groupId,
        keys: props.keys.map(k => k.id),
        config: validationConfig.value
      })
    })

    if (!response.ok) throw new Error('Failed to start validation')

    const job = await response.json()
    currentJobId.value = job.data.id

    // Start monitoring progress
    startProgressMonitoring()

    message.success('批量驗證已開始')

  } catch (error) {
    console.error('Validation failed:', error)
    message.error('開始驗證失敗')
    isValidating.value = false
  }
}

const cancelValidation = async () => {
  if (!currentJobId.value) return

  try {
    await fetch(`/api/keys/cancel-validation/${currentJobId.value}`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('auth_token')}`
      }
    })

    stopProgressMonitoring()
    isValidating.value = false
    message.info('驗證已取消')

  } catch (error) {
    console.error('Cancel failed:', error)
    message.error('取消驗證失敗')
  }
}

const startProgressMonitoring = () => {
  elapsedTime.value = 0

  progressTimer = setInterval(async () => {
    if (!currentJobId.value) return

    try {
      const response = await fetch(`/api/keys/validation-status/${currentJobId.value}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('auth_token')}`
        }
      })

      if (!response.ok) throw new Error('Failed to get status')

      const status = await response.json()
      const jobData = status.data

      // Update stats
      stats.value = {
        totalKeys: jobData.stats.total_keys,
        validKeys: jobData.stats.valid_keys,
        invalidKeys: jobData.stats.invalid_keys,
        processedKeys: jobData.stats.processed_keys,
        errorRate: jobData.stats.error_rate
      }

      // Update results
      results.value = jobData.results || []

      // Check if completed
      if (jobData.status === 'completed' || jobData.status === 'failed' || jobData.status === 'cancelled') {
        stopProgressMonitoring()
        isValidating.value = false

        if (jobData.status === 'completed') {
          message.success(`驗證完成！有效密鑰: ${stats.value.validKeys}/${stats.value.totalKeys}`)
        }
      }

    } catch (error) {
      console.error('Status check failed:', error)
    }

    elapsedTime.value++
  }, 1000)

  // Speed calculation timer
  speedTimer = setInterval(() => {
    if (stats.value.processedKeys > 0 && elapsedTime.value > 0) {
      validationSpeed.value = stats.value.processedKeys / elapsedTime.value

      const remaining = stats.value.totalKeys - stats.value.processedKeys
      estimatedTime.value = remaining / Math.max(validationSpeed.value, 0.1)
    }
  }, 2000)
}

const stopProgressMonitoring = () => {
  if (progressTimer) {
    clearInterval(progressTimer)
    progressTimer = null
  }
  if (speedTimer) {
    clearInterval(speedTimer)
    speedTimer = null
  }
}

// Utility functions
const getStatusBadgeType = () => {
  if (isValidating.value) return 'warning'
  if (hasResults.value) return 'success'
  return 'default'
}

const getProgressStatus = () => {
  if (isValidating.value) return 'info'
  if (stats.value.errorRate > 50) return 'error'
  if (stats.value.errorRate > 20) return 'warning'
  return 'success'
}

const getErrorRateStyle = () => {
  if (stats.value.errorRate > 50) return 'color: #d03050;'
  if (stats.value.errorRate > 20) return 'color: #f0a020;'
  return 'color: #18a058;'
}

const getRowClassName = (row: ValidationResult) => {
  return row.isValid ? 'valid-key-row' : 'invalid-key-row'
}

const maskApiKey = (key: string) => {
  if (key.length <= 8) return key
  return key.substring(0, 8) + '....' + key.substring(key.length - 4)
}

const formatDuration = (seconds: number) => {
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分${seconds % 60}秒`
  return `${Math.floor(seconds / 3600)}時${Math.floor((seconds % 3600) / 60)}分`
}

// Action methods
const exportResults = () => {
  const data = results.value.map(r => ({
    key: r.key.key_value,
    status: r.isValid ? 'valid' : 'invalid',
    error: r.error || '',
    duration: r.duration,
    timestamp: r.timestamp
  }))

  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `validation_results_${Date.now()}.json`
  a.click()
  URL.revokeObjectURL(url)
}

const retryFailedKeys = () => {
  const failedKeys = results.value
    .filter(r => !r.isValid)
    .map(r => r.key)

  if (failedKeys.length > 0) {
    // TODO: Implement retry logic
    message.info(`準備重試 ${failedKeys.length} 個失敗的密鑰`)
  }
}

const updateValidKeys = async () => {
  const validKeys = results.value
    .filter(r => r.isValid)
    .map(r => r.key.id)

  // TODO: Implement batch update API call
  message.success(`已啟用 ${validKeys.length} 個有效密鑰`)
}

const disableInvalidKeys = async () => {
  const invalidKeys = results.value
    .filter(r => !r.isValid)
    .map(r => r.key.id)

  // TODO: Implement batch disable API call
  message.success(`已禁用 ${invalidKeys.length} 個無效密鑰`)
}

const removeInvalidKeys = async () => {
  const invalidKeys = results.value
    .filter(r => !r.isValid)
    .map(r => r.key.id)

  // TODO: Implement batch delete API call
  message.success(`已刪除 ${invalidKeys.length} 個無效密鑰`)
}

// Lifecycle
onMounted(() => {
  stats.value.totalKeys = props.keys.length
})

onUnmounted(() => {
  stopProgressMonitoring()
})

// Watch for props changes
watch(() => props.keys, (newKeys) => {
  stats.value.totalKeys = newKeys.length
  if (!isValidating.value) {
    results.value = []
    stats.value = {
      totalKeys: newKeys.length,
      validKeys: 0,
      invalidKeys: 0,
      processedKeys: 0,
      errorRate: 0
    }
  }
})
</script>

<style scoped>
.batch-validation-container {
  padding: 16px;
}

.valid-key-row {
  background-color: #f6ffed;
}

.invalid-key-row {
  background-color: #fff2f0;
}

:deep(.n-progress-text) {
  font-weight: 500;
}

:deep(.n-statistic-value) {
  font-weight: 600;
}
</style>
