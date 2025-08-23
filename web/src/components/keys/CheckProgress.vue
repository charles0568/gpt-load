<template>
  <div class="check-progress">
    <!-- 狀態標題 -->
    <div class="status-header">
      <n-space align="center">
        <n-icon :component="statusIcon" :color="statusColor" size="24" />
        <span class="status-text">{{ statusText }}</span>
        <n-tag :type="statusTagType" size="small">{{ progress?.status || 'unknown' }}</n-tag>
      </n-space>
    </div>

    <!-- 進度條 -->
    <div class="progress-bar">
      <n-progress
        type="line"
        :percentage="progressPercentage"
        :status="progressStatus"
        :show-indicator="true"
        :height="24"
      >
        <template #default="{ percentage }">
          <span class="progress-text">
            {{ Math.round(percentage) }}%
            ({{ progress?.processed_keys || 0 }}/{{ progress?.total_keys || 0 }})
          </span>
        </template>
      </n-progress>
    </div>

    <!-- 統計資訊 -->
    <div class="stats-grid">
      <n-card size="small" class="stat-card">
        <n-statistic label="總密鑰數" :value="progress?.total_keys || 0">
          <template #prefix>
            <n-icon :component="KeyOutline" />
          </template>
        </n-statistic>
      </n-card>

      <n-card size="small" class="stat-card valid">
        <n-statistic label="有效密鑰" :value="progress?.valid_keys || 0">
          <template #prefix>
            <n-icon :component="CheckmarkCircleOutline" />
          </template>
        </n-statistic>
      </n-card>

      <n-card size="small" class="stat-card invalid">
        <n-statistic label="無效密鑰" :value="progress?.invalid_keys || 0">
          <template #prefix>
            <n-icon :component="CloseCircleOutline" />
          </template>
        </n-statistic>
      </n-card>

      <n-card size="small" class="stat-card">
        <n-statistic label="檢查速度" :value="Math.round(progress?.speed || 0)" suffix="個/秒">
          <template #prefix>
            <n-icon :component="SpeedometerOutline" />
          </template>
        </n-statistic>
      </n-card>
    </div>

    <!-- 詳細資訊 -->
    <div class="detail-info">
      <n-descriptions :column="2" size="small">
        <n-descriptions-item label="當前批次">
          {{ progress?.current_batch || 0 }} / {{ progress?.total_batches || 0 }}
        </n-descriptions-item>
        <n-descriptions-item label="開始時間">
          {{ formatTime(progress?.start_time) }}
        </n-descriptions-item>
        <n-descriptions-item label="已用時間">
          {{ formatDuration(progress?.start_time) }}
        </n-descriptions-item>
        <n-descriptions-item label="預估完成">
          {{ formatTime(progress?.estimated_end) }}
        </n-descriptions-item>
      </n-descriptions>
    </div>

    <!-- 錯誤訊息 -->
    <div v-if="progress?.error_message" class="error-message">
      <n-alert type="error" title="檢查錯誤">
        {{ progress.error_message }}
      </n-alert>
    </div>

    <!-- 控制按鈕 -->
    <div class="control-buttons">
      <n-space justify="center">
        <n-button
          v-if="progress?.status === 'running'"
          type="warning"
          @click="$emit('pause')"
          :loading="actionLoading"
        >
          <template #icon>
            <n-icon :component="PauseOutline" />
          </template>
          暫停
        </n-button>

        <n-button
          v-if="progress?.status === 'paused'"
          type="primary"
          @click="$emit('resume')"
          :loading="actionLoading"
        >
          <template #icon>
            <n-icon :component="PlayOutline" />
          </template>
          恢復
        </n-button>

        <n-button
          v-if="['running', 'paused'].includes(progress?.status)"
          type="error"
          @click="confirmCancel"
          :loading="actionLoading"
        >
          <template #icon>
            <n-icon :component="StopOutline" />
          </template>
          取消
        </n-button>
      </n-space>
    </div>

    <!-- 取消確認對話框 -->
    <n-modal v-model:show="showCancelDialog" preset="dialog" title="確認取消">
      <p>確定要取消當前的批量檢查嗎？</p>
      <p class="text-warning">已檢查的結果將會保留，但未完成的檢查將停止。</p>
      <template #action>
        <n-space>
          <n-button @click="showCancelDialog = false">取消</n-button>
          <n-button type="error" @click="handleCancel">確認取消</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

import {
  KeyOutline,
  CheckmarkCircleOutline,
  CloseCircleOutline,
  SpeedometerOutline,
  PauseOutline,
  PlayOutline,
  StopOutline,
  TimeOutline,
  CheckmarkOutline,
  CloseOutline
} from '@vicons/ionicons5'

interface Props {
  taskId: string
  progress: any
}

interface Emits {
  (e: 'pause'): void
  (e: 'resume'): void
  (e: 'cancel'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()


const actionLoading = ref(false)
const showCancelDialog = ref(false)

// 計算屬性
const progressPercentage = computed(() => {
  if (!props.progress?.total_keys || props.progress.total_keys === 0) return 0
  return (props.progress.processed_keys / props.progress.total_keys) * 100
})

const progressStatus = computed(() => {
  switch (props.progress?.status) {
    case 'running': return 'info'
    case 'paused': return 'warning'
    case 'completed': return 'success'
    case 'cancelled': return 'error'
    default: return 'info'
  }
})

const statusIcon = computed(() => {
  switch (props.progress?.status) {
    case 'running': return TimeOutline
    case 'paused': return PauseOutline
    case 'completed': return CheckmarkOutline
    case 'cancelled': return CloseOutline
    default: return TimeOutline
  }
})

const statusColor = computed(() => {
  switch (props.progress?.status) {
    case 'running': return '#2080f0'
    case 'paused': return '#f0a020'
    case 'completed': return '#18a058'
    case 'cancelled': return '#d03050'
    default: return '#666'
  }
})

const statusText = computed(() => {
  switch (props.progress?.status) {
    case 'running': return '檢查進行中'
    case 'paused': return '檢查已暫停'
    case 'completed': return '檢查已完成'
    case 'cancelled': return '檢查已取消'
    default: return '未知狀態'
  }
})

const statusTagType = computed(() => {
  switch (props.progress?.status) {
    case 'running': return 'info'
    case 'paused': return 'warning'
    case 'completed': return 'success'
    case 'cancelled': return 'error'
    default: return 'default'
  }
})

// 格式化時間
const formatTime = (timeStr: string) => {
  if (!timeStr) return '-'
  return new Date(timeStr).toLocaleString('zh-TW')
}

// 格式化持續時間
const formatDuration = (startTime: string) => {
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

// 確認取消
const confirmCancel = () => {
  showCancelDialog.value = true
}

// 處理取消
const handleCancel = () => {
  showCancelDialog.value = false
  emit('cancel')
}
</script>

<style scoped>
.check-progress {
  padding: 24px;
  max-width: 100%;
  margin: 0 auto;
  box-sizing: border-box;
}

.status-header {
  text-align: center;
  margin-bottom: 24px;
}

.status-text {
  font-size: 18px;
  font-weight: 600;
}

.progress-bar {
  margin-bottom: 32px;
}

.progress-text {
  font-weight: 600;
  color: #333;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  text-align: center;
}

.stat-card.valid :deep(.n-statistic-value) {
  color: #18a058;
}

.stat-card.invalid :deep(.n-statistic-value) {
  color: #d03050;
}

.detail-info {
  margin-bottom: 24px;
}

.error-message {
  margin-bottom: 24px;
}

.control-buttons {
  margin-top: 32px;
}

.text-warning {
  color: #f0a020;
  font-size: 14px;
}
</style>
