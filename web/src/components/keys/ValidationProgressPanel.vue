<script setup lang="ts">
import { keysApi } from "@/api/keys";
import {
  NButton,
  NCard,
  NIcon,
  NProgress,
  NSpace,
  NStatistic,
  NTag,
  NText,
  NTime,
  NDataTable,
  NSpin,
  useMessage,
  type DataTableColumns,
} from "naive-ui";
import {
  CheckmarkCircleOutline,
  CloseCircleOutline,
  PauseCircleOutline,
  PlayCircleOutline,
  StopCircleOutline,
  TimeOutline,
} from "@vicons/ionicons5";
import { ref, computed, onMounted, onUnmounted, h } from "vue";

interface ValidationResult {
  key: {
    id: number;
    key_value: string;
    status: string;
  };
  is_valid: boolean;
  error?: string;
  duration: number;
  timestamp: string;
}

interface ValidationStats {
  total_keys: number;
  valid_keys: number;
  invalid_keys: number;
  processed_keys: number;
  start_time: string;
  duration: number;
  error_rate: number;
}

interface ValidationJob {
  id: string;
  status: "running" | "completed" | "failed" | "cancelled";
  stats: ValidationStats;
  results: ValidationResult[];
}

interface Props {
  jobId: string;
  visible: boolean;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  "update:visible": [value: boolean];
  "job-completed": [job: ValidationJob];
}>();

const message = useMessage();
const job = ref<ValidationJob | null>(null);
const loading = ref(true);
const autoRefresh = ref(true);
const refreshInterval = ref<number | null>(null);

// Progress calculations
const progressPercentage = computed(() => {
  if (!job.value?.stats) return 0;
  const { processed_keys, total_keys } = job.value.stats;
  return Math.round((processed_keys / total_keys) * 100);
});

const throughputPerMinute = computed(() => {
  if (!job.value?.stats || job.value.stats.duration === 0) return 0;
  const { processed_keys, duration } = job.value.stats;
  return Math.round((processed_keys / (duration / 1000)) * 60);
});

const estimatedTimeRemaining = computed(() => {
  if (!job.value?.stats || throughputPerMinute.value === 0) return "計算中...";

  const { processed_keys, total_keys } = job.value.stats;
  const remaining = total_keys - processed_keys;
  const remainingMinutes = remaining / (throughputPerMinute.value / 60);

  if (remainingMinutes < 1) {
    return "不到 1 分鐘";
  } else if (remainingMinutes < 60) {
    return `約 ${Math.ceil(remainingMinutes)} 分鐘`;
  } else {
    return `約 ${Math.ceil(remainingMinutes / 60)} 小時`;
  }
});

const statusColor = computed(() => {
  switch (job.value?.status) {
    case "running": return "info";
    case "completed": return "success";
    case "failed": return "error";
    case "cancelled": return "warning";
    default: return "default";
  }
});

const statusIcon = computed(() => {
  switch (job.value?.status) {
    case "running": return PlayCircleOutline;
    case "completed": return CheckmarkCircleOutline;
    case "failed": return CloseCircleOutline;
    case "cancelled": return PauseCircleOutline;
    default: return TimeOutline;
  }
});

// Results table configuration
const resultColumns: DataTableColumns<ValidationResult> = [
  {
    title: "密鑰",
    key: "key.key_value",
    width: 200,
    ellipsis: {
      tooltip: true
    },
    render: (row) => {
      const keyValue = row.key?.key_value || "";
      return keyValue.length > 20 ?
        `${keyValue.substring(0, 20)}...` :
        keyValue;
    }
  },
  {
    title: "狀態",
    key: "is_valid",
    width: 100,
    render: (row) => {
      return h(NTag, {
        type: row.is_valid ? "success" : "error",
        size: "small"
      }, {
        default: () => row.is_valid ? "有效" : "無效"
      });
    }
  },
  {
    title: "耗時",
    key: "duration",
    width: 100,
    render: (row) => `${row.duration}ms`
  },
  {
    title: "錯誤",
    key: "error",
    ellipsis: {
      tooltip: true
    },
    render: (row) => row.error || "-"
  }
];

// Fetch job status
async function fetchJobStatus() {
  try {
    const response = await keysApi.getValidationStatus(props.jobId);
    job.value = response;

    if (response.status === "completed" || response.status === "failed" || response.status === "cancelled") {
      stopAutoRefresh();
      emit("job-completed", response);

      if (response.status === "completed") {
        message.success("批量驗證已完成！");
      } else if (response.status === "failed") {
        message.error("批量驗證失敗");
      } else {
        message.warning("批量驗證已取消");
      }
    }
  } catch (error) {
    console.error("Failed to fetch job status:", error);
    message.error("獲取驗證狀態失敗");
    stopAutoRefresh();
  } finally {
    loading.value = false;
  }
}

// Cancel validation job
async function cancelJob() {
  try {
    await keysApi.cancelValidation(props.jobId);
    message.success("驗證任務已取消");
    await fetchJobStatus();
  } catch (error) {
    console.error("Failed to cancel job:", error);
    message.error("取消驗證任務失敗");
  }
}

// Auto refresh control
function startAutoRefresh() {
  autoRefresh.value = true;
  refreshInterval.value = setInterval(fetchJobStatus, 2000); // Refresh every 2 seconds
}

function stopAutoRefresh() {
  autoRefresh.value = false;
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value);
    refreshInterval.value = null;
  }
}

function toggleAutoRefresh() {
  if (autoRefresh.value) {
    stopAutoRefresh();
  } else {
    startAutoRefresh();
  }
}

function handleClose() {
  stopAutoRefresh();
  emit("update:visible", false);
}

onMounted(async () => {
  await fetchJobStatus();
  if (job.value?.status === "running") {
    startAutoRefresh();
  }
});

onUnmounted(() => {
  stopAutoRefresh();
});
</script>

<template>
  <n-card
    v-if="visible"
    title="批量驗證進度"
    class="validation-progress-panel"
    :bordered="false"
    size="small"
  >
    <template #header-extra>
      <n-space>
        <n-button
          size="small"
          :type="autoRefresh ? 'primary' : 'default'"
          @click="toggleAutoRefresh"
        >
          {{ autoRefresh ? "停止自動刷新" : "開始自動刷新" }}
        </n-button>
        <n-button
          size="small"
          @click="handleClose"
        >
          關閉
        </n-button>
      </n-space>
    </template>

    <n-spin :show="loading">
      <n-space vertical size="large" v-if="job">
        <!-- 狀態和進度 -->
        <n-space vertical>
          <n-space align="center">
            <n-icon size="24" :color="statusColor === 'success' ? '#18a058' : statusColor === 'error' ? '#d03050' : '#2080f0'">
              <component :is="statusIcon" />
            </n-icon>
            <n-tag :type="statusColor" size="large">
              {{ job.status === 'running' ? '進行中' :
                 job.status === 'completed' ? '已完成' :
                 job.status === 'failed' ? '失敗' : '已取消' }}
            </n-tag>
            <n-text depth="2">任務 ID: {{ job.id }}</n-text>
          </n-space>

          <n-progress
            type="line"
            :percentage="progressPercentage"
            :status="statusColor === 'error' ? 'error' : statusColor === 'success' ? 'success' : 'default'"
            :show-indicator="true"
          />
        </n-space>

        <!-- 統計資訊 -->
        <n-space>
          <n-statistic label="總計" :value="job.stats.total_keys" />
          <n-statistic label="已處理" :value="job.stats.processed_keys" />
          <n-statistic label="有效" :value="job.stats.valid_keys" />
          <n-statistic label="無效" :value="job.stats.invalid_keys" />
          <n-statistic label="錯誤率" :value="job.stats.error_rate.toFixed(1)" suffix="%" />
        </n-space>

        <!-- 效能指標 -->
        <n-space v-if="job.status === 'running'">
          <n-statistic label="處理速度" :value="throughputPerMinute" suffix=" keys/min" />
          <n-statistic label="預估剩餘時間">
            <n-text>{{ estimatedTimeRemaining }}</n-text>
          </n-statistic>
          <n-statistic label="已用時間">
            <n-time
              :time="new Date(job.stats.start_time)"
              :to="new Date()"
              type="relative"
            />
          </n-statistic>
        </n-space>

        <!-- 操作按鈕 -->
        <n-space v-if="job.status === 'running'">
          <n-button type="error" @click="cancelJob">
            <template #icon>
              <n-icon>
                <StopCircleOutline />
              </n-icon>
            </template>
            取消驗證
          </n-button>
        </n-space>

        <!-- 結果列表 -->
        <n-card title="驗證結果" size="small" v-if="job.results && job.results.length > 0">
          <n-data-table
            :columns="resultColumns"
            :data="job.results.slice(-20)"
            size="small"
            :max-height="300"
            :scroll-x="600"
          />
          <n-text depth="3" style="margin-top: 8px; display: block;">
            顯示最近 20 條結果（共 {{ job.results.length }} 條）
          </n-text>
        </n-card>
      </n-space>
    </n-spin>
  </n-card>
</template>

<style scoped>
.validation-progress-panel {
  margin: 16px 0;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
}

.validation-progress-panel :deep(.n-progress-line) {
  margin: 8px 0;
}

.validation-progress-panel :deep(.n-statistic) {
  text-align: center;
}
</style>
