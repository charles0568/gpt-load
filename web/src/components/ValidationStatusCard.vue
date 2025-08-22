<script setup lang="ts">
import { keysApi } from "@/api/keys";
import {
  NCard,
  NIcon,
  NProgress,
  NSpace,
  NStatistic,
  NTag,
  NText,
  NButton,
  NEmpty,
  NSpin,
  useMessage,
} from "naive-ui";
import {
  FlashOutline,
  CheckmarkCircleOutline,
  CloseCircleOutline,
  TimeOutline,
  PlayCircleOutline,
} from "@vicons/ionicons5";
import { ref, onMounted, onUnmounted, computed } from "vue";

interface ValidationJob {
  id: string;
  status: "running" | "completed" | "failed" | "cancelled";
  stats: {
    total_keys: number;
    valid_keys: number;
    invalid_keys: number;
    processed_keys: number;
    start_time: string;
    duration: number;
    error_rate: number;
  };
}

const message = useMessage();
const activeJobs = ref<ValidationJob[]>([]);
const loading = ref(false);
const refreshInterval = ref<number | null>(null);

// 計算進度百分比
function getProgressPercentage(job: ValidationJob): number {
  if (job.stats.total_keys === 0) return 0;
  return Math.round((job.stats.processed_keys / job.stats.total_keys) * 100);
}

// 獲取狀態顏色
function getStatusColor(status: string): string {
  switch (status) {
    case "running": return "#2080f0";
    case "completed": return "#18a058";
    case "failed": return "#d03050";
    case "cancelled": return "#f0a020";
    default: return "#909399";
  }
}

// 獲取狀態圖標
function getStatusIcon(status: string) {
  switch (status) {
    case "running": return PlayCircleOutline;
    case "completed": return CheckmarkCircleOutline;
    case "failed": return CloseCircleOutline;
    case "cancelled": return TimeOutline;
    default: return TimeOutline;
  }
}

// 獲取狀態文字
function getStatusText(status: string): string {
  switch (status) {
    case "running": return "進行中";
    case "completed": return "已完成";
    case "failed": return "失敗";
    case "cancelled": return "已取消";
    default: return "未知";
  }
}

// 計算估計剩餘時間
function getEstimatedTimeRemaining(job: ValidationJob): string {
  if (job.status !== "running" || job.stats.processed_keys === 0) {
    return "-";
  }
  
  const { processed_keys, total_keys, duration } = job.stats;
  const remaining = total_keys - processed_keys;
  const avgTimePerKey = duration / processed_keys;
  const remainingTime = remaining * avgTimePerKey;
  
  if (remainingTime < 60 * 1000) {
    return "不到 1 分鐘";
  } else if (remainingTime < 60 * 60 * 1000) {
    return `約 ${Math.ceil(remainingTime / (60 * 1000))} 分鐘`;
  } else {
    return `約 ${Math.ceil(remainingTime / (60 * 60 * 1000))} 小時`;
  }
}

// 獲取活躍驗證任務
async function fetchActiveJobs() {
  try {
    loading.value = true;
    // 注意：這裡需要一個獲取所有活躍驗證任務的API
    // 目前我們暫時使用空數組，實際應該調用後端API
    activeJobs.value = [];
  } catch (error) {
    console.error("Failed to fetch active jobs:", error);
  } finally {
    loading.value = false;
  }
}

// 取消驗證任務
async function cancelJob(jobId: string) {
  try {
    await keysApi.cancelValidation(jobId);
    message.success("驗證任務已取消");
    await fetchActiveJobs();
  } catch (error) {
    console.error("Failed to cancel job:", error);
    message.error("取消任務失敗");
  }
}

// 開始自動刷新
function startAutoRefresh() {
  refreshInterval.value = setInterval(fetchActiveJobs, 3000); // 每3秒刷新
}

// 停止自動刷新
function stopAutoRefresh() {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value);
    refreshInterval.value = null;
  }
}

// 計算總體統計
const totalStats = computed(() => {
  const runningJobs = activeJobs.value.filter(job => job.status === "running");
  
  if (runningJobs.length === 0) {
    return null;
  }
  
  return {
    totalJobs: runningJobs.length,
    totalKeys: runningJobs.reduce((sum, job) => sum + job.stats.total_keys, 0),
    processedKeys: runningJobs.reduce((sum, job) => sum + job.stats.processed_keys, 0),
    validKeys: runningJobs.reduce((sum, job) => sum + job.stats.valid_keys, 0),
    invalidKeys: runningJobs.reduce((sum, job) => sum + job.stats.invalid_keys, 0),
  };
});

onMounted(() => {
  fetchActiveJobs();
  startAutoRefresh();
});

onUnmounted(() => {
  stopAutoRefresh();
});
</script>

<template>
  <n-card title="批量驗證狀態" :bordered="false">
    <template #header-extra>
      <n-icon size="20" color="var(--primary-color)">
        <FlashOutline />
      </n-icon>
    </template>

    <n-spin :show="loading">
      <!-- 總體統計 -->
      <div v-if="totalStats" class="total-stats">
        <n-space>
          <n-statistic label="運行任務" :value="totalStats.totalJobs" />
          <n-statistic label="總密鑰" :value="totalStats.totalKeys" />
          <n-statistic label="已處理" :value="totalStats.processedKeys" />
          <n-statistic label="有效" :value="totalStats.validKeys" />
          <n-statistic label="無效" :value="totalStats.invalidKeys" />
        </n-space>
      </div>

      <!-- 活躍任務列表 -->
      <div v-if="activeJobs.length > 0" class="jobs-list">
        <div 
          v-for="job in activeJobs" 
          :key="job.id" 
          class="job-item"
        >
          <div class="job-header">
            <n-space align="center">
              <n-icon 
                size="16" 
                :color="getStatusColor(job.status)"
              >
                <component :is="getStatusIcon(job.status)" />
              </n-icon>
              <n-tag 
                :type="job.status === 'running' ? 'info' : 
                       job.status === 'completed' ? 'success' : 
                       job.status === 'failed' ? 'error' : 'warning'"
                size="small"
              >
                {{ getStatusText(job.status) }}
              </n-tag>
              <n-text depth="2" style="font-size: 12px;">
                任務 {{ job.id.substring(0, 8) }}...
              </n-text>
            </n-space>
            
            <n-button 
              v-if="job.status === 'running'"
              size="tiny" 
              type="error" 
              @click="cancelJob(job.id)"
            >
              取消
            </n-button>
          </div>

          <div class="job-progress">
            <n-progress
              type="line"
              :percentage="getProgressPercentage(job)"
              :status="job.status === 'completed' ? 'success' : 
                       job.status === 'failed' ? 'error' : 'info'"
              :show-indicator="true"
              :height="6"
            />
          </div>

          <div class="job-stats">
            <n-space size="small">
              <n-text depth="2" style="font-size: 12px;">
                {{ job.stats.processed_keys }}/{{ job.stats.total_keys }}
              </n-text>
              <n-text depth="2" style="font-size: 12px;">
                成功率: {{ job.stats.total_keys > 0 ? 
                  Math.round((job.stats.valid_keys / job.stats.processed_keys) * 100) : 0 }}%
              </n-text>
              <n-text 
                v-if="job.status === 'running'" 
                depth="2" 
                style="font-size: 12px;"
              >
                剩餘: {{ getEstimatedTimeRemaining(job) }}
              </n-text>
            </n-space>
          </div>
        </div>
      </div>

      <!-- 空狀態 -->
      <n-empty 
        v-else-if="!loading"
        description="目前沒有正在運行的批量驗證任務"
        size="small"
      >
        <template #icon>
          <n-icon size="48" color="#909399">
            <FlashOutline />
          </n-icon>
        </template>
      </n-empty>
    </n-spin>
  </n-card>
</template>

<style scoped>
.total-stats {
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
  margin-bottom: 16px;
}

.jobs-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.job-item {
  padding: 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
  transition: all 0.2s;
}

.job-item:hover {
  border-color: #d1d5db;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.job-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.job-progress {
  margin: 8px 0;
}

.job-stats {
  margin-top: 8px;
}

.total-stats :deep(.n-statistic) {
  text-align: center;
}

.job-item :deep(.n-progress-line) {
  margin: 4px 0;
}
</style>