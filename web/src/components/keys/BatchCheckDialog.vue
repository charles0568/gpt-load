<template>
  <n-modal v-model:show="visible" preset="dialog" title="批量檢查密鑰" style="width: 900px; max-width: 95vw">
    <template #header>
      <div class="dialog-header">
        <n-icon :component="CheckmarkCircleOutline" size="24" />
        <span>批量檢查密鑰</span>
      </div>
    </template>

    <div class="batch-check-content">
      <!-- 配置區域 -->
      <div v-if="!isChecking && !isCompleted" class="config-section">
        <n-space vertical size="large">
          <n-alert type="info" title="檢查說明">
            <p>此功能將批量檢查所選分組中的所有密鑰有效性。</p>
            <p>• 支援暫停、恢復和取消操作</p>
            <p>• 檢查完成後可批量刪除無效密鑰</p>
            <p>• 大量密鑰建議在非高峰時段進行檢查</p>
          </n-alert>

          <n-form ref="formRef" :model="formData" label-placement="left" label-width="120px">
            <n-form-item label="目標分組" path="groupId">
              <n-select
                v-model:value="formData.groupId"
                :options="groupOptions"
                placeholder="選擇要檢查的分組"
                :disabled="isChecking"
              />
            </n-form-item>

            <n-form-item label="批次大小" path="batchSize">
              <n-input-number
                v-model:value="formData.batchSize"
                :min="10"
                :max="1000"
                :step="10"
                placeholder="每批處理的密鑰數量"
                :disabled="isChecking"
              />
              <template #suffix>
                <n-tooltip trigger="hover">
                  <template #trigger>
                    <n-icon :component="InformationCircleOutline" />
                  </template>
                  建議值：100-500，過大可能導致 API 限制
                </n-tooltip>
              </template>
            </n-form-item>

            <n-form-item label="併發數" path="concurrency">
              <n-input-number
                v-model:value="formData.concurrency"
                :min="1"
                :max="200"
                :step="5"
                placeholder="同時檢查的密鑰數量"
                :disabled="isChecking"
              />
              <template #suffix>
                <n-tooltip trigger="hover">
                  <template #trigger>
                    <n-icon :component="InformationCircleOutline" />
                  </template>
                  建議值：20-100，過高可能觸發 API 限制
                </n-tooltip>
              </template>
            </n-form-item>
          </n-form>

          <n-space justify="center">
            <n-button
              type="primary"
              size="large"
              :loading="startLoading"
              @click="startBatchCheck"
              :disabled="!formData.groupId"
            >
              <template #icon>
                <n-icon :component="PlayOutline" />
              </template>
              開始檢查
            </n-button>
          </n-space>
        </n-space>
      </div>

      <!-- 進度區域 -->
      <div v-if="isChecking" class="progress-section">
        <CheckProgress
          :task-id="currentTaskId"
          :progress="progress"
          @pause="pauseCheck"
          @resume="resumeCheck"
          @cancel="cancelCheck"
        />
      </div>

      <!-- 結果區域 -->
      <div v-if="isCompleted" class="results-section">
        <CheckResults
          :task-id="currentTaskId"
          :progress="progress"
          @restart="restartCheck"
          @export="exportResults"
          @delete-invalid="deleteInvalidKeys"
        />
      </div>
    </div>

    <template #action>
      <n-space>
        <n-button @click="closeDialog" :disabled="isChecking && progress?.status === 'running'">
          {{ isCompleted ? '關閉' : '取消' }}
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { useMessage } from 'naive-ui'
import {
  CheckmarkCircleOutline,
  InformationCircleOutline,
  PlayOutline
} from '@vicons/ionicons5'
import CheckProgress from './CheckProgress.vue'
import CheckResults from './CheckResults.vue'
import { batchCheckAPI } from '@/api/batchCheck'

interface Props {
  visible: boolean
  groupId?: number
  groups: Array<{ label: string; value: number }>
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'completed'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const message = useMessage()

// 表單數據
const formData = ref({
  groupId: props.groupId || 0,
  batchSize: 200,
  concurrency: 50
})

// 狀態管理
const startLoading = ref(false)
const currentTaskId = ref('')
const progress = ref<any>(null)
const wsConnection = ref<WebSocket | null>(null)

// 計算屬性
const groupOptions = computed(() => props.groups)
const isChecking = computed(() =>
  progress.value && ['running', 'paused'].includes(progress.value.status)
)
const isCompleted = computed(() =>
  progress.value && ['completed', 'cancelled'].includes(progress.value.status)
)

// 監聽 props 變化
watch(() => props.groupId, (newVal) => {
  if (newVal) {
    formData.value.groupId = newVal
  }
})

// 開始批量檢查
const startBatchCheck = async () => {
  try {
    startLoading.value = true

    const response = await batchCheckAPI.start({
      group_id: formData.value.groupId,
      batch_size: formData.value.batchSize,
      concurrency: formData.value.concurrency
    })

    currentTaskId.value = response.data.task_id

    // 建立 WebSocket 連接
    connectWebSocket()

    message.success('批量檢查已開始')
  } catch (error) {
    message.error('開始檢查失敗：' + (error instanceof Error ? error.message : String(error)))
  } finally {
    startLoading.value = false
  }
}

// 建立 WebSocket 連接
const connectWebSocket = () => {
  if (!currentTaskId.value) return

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/api/keys/batch-check/${currentTaskId.value}/ws`

  wsConnection.value = new WebSocket(wsUrl)

  wsConnection.value.onmessage = (event) => {
    try {
      progress.value = JSON.parse(event.data)
    } catch (error) {
      console.error('解析 WebSocket 訊息失敗:', error)
    }
  }

  wsConnection.value.onclose = () => {
    console.log('WebSocket 連接已關閉')
  }

  wsConnection.value.onerror = (error) => {
    console.error('WebSocket 錯誤:', error)
    message.error('即時進度連接失敗')
  }
}

// 暫停檢查
const pauseCheck = async () => {
  try {
    await batchCheckAPI.pause(currentTaskId.value)
    message.success('檢查已暫停')
  } catch (error) {
    message.error('暫停失敗：' + (error instanceof Error ? error.message : String(error)))
  }
}

// 恢復檢查
const resumeCheck = async () => {
  try {
    await batchCheckAPI.resume(currentTaskId.value)
    message.success('檢查已恢復')
  } catch (error) {
    message.error('恢復失敗：' + (error instanceof Error ? error.message : String(error)))
  }
}

// 取消檢查
const cancelCheck = async () => {
  try {
    await batchCheckAPI.cancel(currentTaskId.value)
    message.success('檢查已取消')
  } catch (error) {
    message.error('取消失敗：' + (error instanceof Error ? error.message : String(error)))
  }
}

// 重新開始檢查
const restartCheck = () => {
  progress.value = null
  currentTaskId.value = ''
  closeWebSocket()
}

// 匯出結果
const exportResults = (format: string, filter: string) => {
  const url = `/api/keys/batch-check/${currentTaskId.value}/export?format=${format}&${filter}=true`
  window.open(url, '_blank')
}

// 刪除無效密鑰
const deleteInvalidKeys = async () => {
  try {
    const response = await batchCheckAPI.deleteInvalid(currentTaskId.value)
    message.success(`已刪除 ${response.data.deleted_count} 個無效密鑰`)
    emit('completed')
  } catch (error) {
    message.error('刪除失敗：' + (error instanceof Error ? error.message : String(error)))
  }
}

// 關閉對話框
const closeDialog = () => {
  emit('update:visible', false)
  closeWebSocket()
}

// 關閉 WebSocket 連接
const closeWebSocket = () => {
  if (wsConnection.value) {
    wsConnection.value.close()
    wsConnection.value = null
  }
}

// 組件卸載時清理
onUnmounted(() => {
  closeWebSocket()
})
</script>

<style scoped>
.dialog-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}

.batch-check-content {
  min-height: 400px;
}

.config-section,
.progress-section,
.results-section {
  padding: 16px 0;
}

.n-form-item :deep(.n-form-item-feedback-wrapper) {
  min-height: 0;
}
</style>
