<template>
  <div class="check-results">
    <!-- 結果摘要 -->
    <div class="results-summary">
      <n-alert :type="summaryAlertType" title="檢查完成">
        <div class="summary-content">
          <p>
            <strong>檢查結果：</strong>
            共檢查 {{ progress?.total_keys || 0 }} 個密鑰，
            有效 {{ progress?.valid_keys || 0 }} 個，
            無效 {{ progress?.invalid_keys || 0 }} 個
          </p>
          <p>
            <strong>有效率：</strong>
            {{ validPercentage }}%
          </p>
          <p v-if="progress?.error_message">
            <strong>錯誤訊息：</strong>
            <span class="error-text">{{ progress.error_message }}</span>
          </p>
        </div>
      </n-alert>
    </div>

    <!-- 統計圖表 -->
    <div class="stats-charts">
      <n-card title="檢查統計" size="small">
        <div class="chart-container">
          <div class="pie-chart">
            <n-progress
              type="circle"
              :percentage="validPercentage"
              :stroke-width="12"
              :show-indicator="false"
              status="success"
            />
            <div class="chart-center">
              <div class="chart-value">{{ validPercentage }}%</div>
              <div class="chart-label">有效率</div>
            </div>
          </div>

          <div class="stats-list">
            <div class="stat-item valid">
              <n-icon :component="CheckmarkCircleOutline" />
              <span class="stat-label">有效密鑰</span>
              <span class="stat-value">{{ progress?.valid_keys || 0 }}</span>
            </div>
            <div class="stat-item invalid">
              <n-icon :component="CloseCircleOutline" />
              <span class="stat-label">無效密鑰</span>
              <span class="stat-value">{{ progress?.invalid_keys || 0 }}</span>
            </div>
            <div class="stat-item total">
              <n-icon :component="KeyOutline" />
              <span class="stat-label">總計</span>
              <span class="stat-value">{{ progress?.total_keys || 0 }}</span>
            </div>
          </div>
        </div>
      </n-card>
    </div>

    <!-- 操作按鈕 -->
    <div class="action-buttons">
      <n-space justify="center" size="large">
        <!-- 匯出功能 -->
        <n-dropdown :options="exportOptions" @select="handleExport">
          <n-button type="primary">
            <template #icon>
              <n-icon :component="DownloadOutline" />
            </template>
            匯出結果
          </n-button>
        </n-dropdown>

        <!-- 批量操作 -->
        <n-dropdown :options="batchOptions" @select="handleBatchAction">
          <n-button type="warning">
            <template #icon>
              <n-icon :component="SettingsOutline" />
            </template>
            批量操作
          </n-button>
        </n-dropdown>

        <!-- 重新檢查 -->
        <n-button @click="$emit('restart')">
          <template #icon>
            <n-icon :component="RefreshOutline" />
          </template>
          重新檢查
        </n-button>
      </n-space>
    </div>

    <!-- 詳細結果表格 -->
    <div class="results-table">
      <n-card title="詳細結果" size="small">
        <template #header-extra>
          <n-space>
            <n-select
              v-model:value="filterStatus"
              :options="filterOptions"
              size="small"
              style="width: 120px"
              @update:value="loadResults"
            />
            <n-button size="small" @click="loadResults">
              <template #icon>
                <n-icon :component="RefreshOutline" />
              </template>
              重新整理
            </n-button>
          </n-space>
        </template>

        <n-data-table
          :columns="tableColumns"
          :data="tableData"
          :loading="tableLoading"
          :pagination="pagination"
          :row-key="(row: any) => row.key_id"
          size="small"
          striped
        />
      </n-card>
    </div>

    <!-- 刪除確認對話框 -->
    <n-modal v-model:show="showDeleteDialog" preset="dialog" title="確認刪除">
      <p>確定要刪除所有無效密鑰嗎？</p>
      <p class="text-warning">
        此操作將永久刪除 {{ progress?.invalid_keys || 0 }} 個無效密鑰，無法復原。
      </p>
      <template #action>
        <n-space>
          <n-button @click="showDeleteDialog = false">取消</n-button>
          <n-button type="error" @click="confirmDeleteInvalid" :loading="deleteLoading">
            確認刪除
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useMessage } from 'naive-ui'
import {
  CheckmarkCircleOutline,
  CloseCircleOutline,
  KeyOutline,
  DownloadOutline,
  SettingsOutline,
  RefreshOutline
} from '@vicons/ionicons5'
import { batchCheckAPI } from '@/api/batchCheck'

interface Props {
  taskId: string
  progress: any
}

interface Emits {
  (e: 'restart'): void
  (e: 'export', format: string, filter: string): void
  (e: 'delete-invalid'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const message = useMessage()

// 狀態管理
const tableData = ref<any[]>([])
const tableLoading = ref(false)
const deleteLoading = ref(false)
const showDeleteDialog = ref(false)
const filterStatus = ref('all')

// 分頁配置
const pagination = ref({
  page: 1,
  pageSize: 50,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [20, 50, 100],
  onChange: (page: number) => {
    pagination.value.page = page
    loadResults()
  },
  onUpdatePageSize: (pageSize: number) => {
    pagination.value.pageSize = pageSize
    pagination.value.page = 1
    loadResults()
  }
})

// 計算屬性
const validPercentage = computed(() => {
  if (!props.progress?.total_keys || props.progress.total_keys === 0) return 0
  return Math.round((props.progress.valid_keys / props.progress.total_keys) * 100)
})

const summaryAlertType = computed(() => {
  if (props.progress?.status === 'completed') {
    return validPercentage.value >= 80 ? 'success' : 'warning'
  }
  return 'error'
})

// 匯出選項
const exportOptions = [
  {
    label: '匯出全部 (CSV)',
    key: 'all-csv'
  },
  {
    label: '匯出全部 (JSON)',
    key: 'all-json'
  },
  {
    label: '僅匯出有效密鑰 (CSV)',
    key: 'valid-csv'
  },
  {
    label: '僅匯出無效密鑰 (CSV)',
    key: 'invalid-csv'
  }
]

// 批量操作選項
const batchOptions = computed(() => [
  {
    label: `刪除無效密鑰 (${props.progress?.invalid_keys || 0}個)`,
    key: 'delete-invalid',
    disabled: !props.progress?.invalid_keys
  }
])

// 過濾選項
const filterOptions = [
  { label: '全部', value: 'all' },
  { label: '僅有效', value: 'valid' },
  { label: '僅無效', value: 'invalid' }
]

// 表格列配置
const tableColumns = [
  {
    title: '密鑰ID',
    key: 'key_id',
    width: 80
  },
  {
    title: '密鑰',
    key: 'key',
    width: 200,
    render: (row: any) => {
      const key = row.key || ''
      return key.length > 20 ? key.substring(0, 20) + '...' : key
    }
  },
  {
    title: '狀態',
    key: 'valid',
    width: 80,
    render: (row: any) => {
      return h(
        'n-tag',
        {
          type: row.valid ? 'success' : 'error',
          size: 'small'
        },
        row.valid ? '有效' : '無效'
      )
    }
  },
  {
    title: '回應時間',
    key: 'response_time_ms',
    width: 100,
    render: (row: any) => `${row.response_time_ms}ms`
  },
  {
    title: '錯誤訊息',
    key: 'error_message',
    render: (row: any) => row.error_message || '-'
  },
  {
    title: '檢查時間',
    key: 'checked_at',
    width: 160,
    render: (row: any) => {
      return new Date(row.checked_at).toLocaleString('zh-TW')
    }
  }
]

// 載入結果數據
const loadResults = async () => {
  try {
    tableLoading.value = true

    const response = await batchCheckAPI.getResults(props.taskId, {
      page: pagination.value.page,
      page_size: pagination.value.pageSize
    })

    let results = response.data.results || []

    // 根據過濾條件篩選
    if (filterStatus.value === 'valid') {
      results = results.filter((item: any) => item.valid)
    } else if (filterStatus.value === 'invalid') {
      results = results.filter((item: any) => !item.valid)
    }

    tableData.value = results
    pagination.value.itemCount = response.data.pagination?.total || results.length
  } catch (error) {
    message.error('載入結果失敗：' + (error instanceof Error ? error.message : String(error)))
  } finally {
    tableLoading.value = false
  }
}

// 處理匯出
const handleExport = (key: string) => {
  const [filter, format] = key.split('-')
  emit('export', format, filter)
}

// 處理批量操作
const handleBatchAction = (key: string) => {
  if (key === 'delete-invalid') {
    showDeleteDialog.value = true
  }
}

// 確認刪除無效密鑰
const confirmDeleteInvalid = async () => {
  try {
    deleteLoading.value = true
    emit('delete-invalid')
    showDeleteDialog.value = false
  } finally {
    deleteLoading.value = false
  }
}

// 組件掛載時載入數據
onMounted(() => {
  loadResults()
})
</script>

<style scoped>
.check-results {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.results-summary {
  margin-bottom: 32px;
}

.summary-content p {
  margin: 8px 0;
}

.error-text {
  color: #d03050;
}

.stats-charts {
  margin-bottom: 32px;
}

.chart-container {
  display: flex;
  align-items: center;
  gap: 32px;
  padding: 16px;
}

.pie-chart {
  position: relative;
  flex-shrink: 0;
}

.chart-center {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  text-align: center;
}

.chart-value {
  font-size: 24px;
  font-weight: bold;
  color: #18a058;
}

.chart-label {
  font-size: 12px;
  color: #666;
}

.stats-list {
  flex: 1;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}

.stat-item:last-child {
  border-bottom: none;
}

.stat-item.valid {
  color: #18a058;
}

.stat-item.invalid {
  color: #d03050;
}

.stat-item.total {
  color: #333;
  font-weight: 600;
}

.stat-label {
  flex: 1;
}

.stat-value {
  font-weight: 600;
  font-size: 16px;
}

.action-buttons {
  margin-bottom: 32px;
}

.results-table {
  margin-bottom: 24px;
}

.text-warning {
  color: #f0a020;
  font-size: 14px;
}
</style>
