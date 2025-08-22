<script setup lang="ts">
import type { Group } from "@/types/models";
import { keysApi } from "@/api/keys";
import {
  NButton,
  NCard,
  NDialog,
  NForm,
  NFormItem,
  NIcon,
  NInputNumber,
  NSelect,
  NSlider,
  NSpace,
  NSwitch,
  NText,
  useMessage,
  type FormInst,
} from "naive-ui";
import { FlashOutline, SettingsOutline, PlayOutline } from "@vicons/ionicons5";
import { ref, reactive, computed } from "vue";

interface Props {
  visible: boolean;
  selectedGroup: Group | null;
}

interface BatchValidationConfig {
  concurrency: number;
  timeout_seconds: number;
  max_retries: number;
  rate_limit_per_sec: number;
  enable_multiplexing: boolean;
  enable_http2: boolean;
  streaming_threshold: number;
  backup_results: boolean;
  enable_jitter: boolean;
  proxy_url: string;
}

interface BatchValidationRequest {
  group_id: number;
  keys: number[];
  config: BatchValidationConfig;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  "update:visible": [value: boolean];
  "validation-started": [jobId: string];
}>();

const message = useMessage();
const formRef = ref<FormInst>();
const loading = ref(false);

// Validation configuration with optimized defaults
const config = reactive<BatchValidationConfig>({
  concurrency: 50,
  timeout_seconds: 15,
  max_retries: 3,
  rate_limit_per_sec: 100,
  enable_multiplexing: true,
  enable_http2: true,
  streaming_threshold: 1000,
  backup_results: true,
  enable_jitter: true,
  proxy_url: "",
});

// Validation scope options
const scopeOptions = [
  { label: "所有密鑰", value: "all" },
  { label: "僅有效密鑰", value: "active" },
  { label: "僅無效密鑰", value: "invalid" },
];
const selectedScope = ref("all");

// Performance presets
const presetOptions = [
  { 
    label: "平衡模式", 
    value: "balanced",
    concurrency: 30,
    timeout: 15,
    rate_limit: 60
  },
  { 
    label: "高速模式", 
    value: "fast",
    concurrency: 50,
    timeout: 10,
    rate_limit: 100
  },
  { 
    label: "保守模式", 
    value: "conservative",
    concurrency: 20,
    timeout: 30,
    rate_limit: 30
  },
  { 
    label: "極速模式", 
    value: "extreme",
    concurrency: 80,
    timeout: 8,
    rate_limit: 150
  },
];

const estimatedTime = computed(() => {
  if (!props.selectedGroup) return "N/A";
  const keyCount = props.selectedGroup.total_keys ?? 0;
  const timePerKey = config.timeout_seconds + 2; // Add some overhead
  const parallelTime = Math.ceil(keyCount / config.concurrency) * timePerKey;
  
  if (parallelTime < 60) {
    return `約 ${parallelTime} 秒`;
  } else if (parallelTime < 3600) {
    return `約 ${Math.ceil(parallelTime / 60)} 分鐘`;
  } else {
    return `約 ${Math.ceil(parallelTime / 3600)} 小時`;
  }
});

const throughput = computed(() => {
  return Math.round(config.concurrency * (60 / config.timeout_seconds));
});

function applyPreset(preset: string) {
  const option = presetOptions.find(p => p.value === preset);
  if (option) {
    config.concurrency = option.concurrency;
    config.timeout_seconds = option.timeout;
    config.rate_limit_per_sec = option.rate_limit;
  }
}

function handleClose() {
  emit("update:visible", false);
}

async function startValidation() {
  if (!props.selectedGroup) {
    message.error("請選擇一個密鑰分組");
    return;
  }

  try {
    loading.value = true;

    const request: BatchValidationRequest = {
      group_id: props.selectedGroup.id!,
      keys: [], // Empty means validate all keys in the group
      config: { ...config }
    };

    const response = await keysApi.startBatchValidation(request);
    
    message.success("批量驗證已開始！");
    emit("validation-started", response.id);
    handleClose();
    
  } catch (error: any) {
    console.error("Failed to start batch validation:", error);
    message.error(error.response?.data?.message || "啟動批量驗證失敗");
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <n-dialog
    :show="visible"
    title="批量密鑰驗證"
    class="batch-validation-dialog"
    style="width: 700px"
    @update:show="handleClose"
  >
    <n-card>
      <template #header>
        <n-space align="center">
          <n-icon size="20" color="var(--primary-color)">
            <FlashOutline />
          </n-icon>
          <span>高效能批量驗證</span>
        </n-space>
      </template>

      <n-form ref="formRef" :model="config" label-placement="left" label-width="120px">
        <!-- 基本設定 -->
        <n-form-item label="驗證範圍">
          <n-select
            v-model:value="selectedScope"
            :options="scopeOptions"
            placeholder="選擇驗證範圍"
          />
        </n-form-item>

        <!-- 預設模式 -->
        <n-form-item label="效能預設">
          <n-space>
            <n-button
              v-for="preset in presetOptions"
              :key="preset.value"
              size="small"
              @click="applyPreset(preset.value)"
            >
              {{ preset.label }}
            </n-button>
          </n-space>
        </n-form-item>

        <!-- 進階配置 -->
        <n-card title="進階配置" size="small" style="margin: 16px 0">
          <template #header-extra>
            <n-icon>
              <SettingsOutline />
            </n-icon>
          </template>

          <n-form-item label="並發數">
            <n-space vertical style="width: 100%">
              <n-slider
                v-model:value="config.concurrency"
                :min="1"
                :max="100"
                :step="1"
                show-tooltip
              />
              <n-input-number
                v-model:value="config.concurrency"
                :min="1"
                :max="100"
                size="small"
                style="width: 120px"
              />
            </n-space>
          </n-form-item>

          <n-form-item label="超時時間">
            <n-space vertical style="width: 100%">
              <n-slider
                v-model:value="config.timeout_seconds"
                :min="5"
                :max="60"
                :step="1"
                show-tooltip
                :format-tooltip="(value) => `${value} 秒`"
              />
              <n-input-number
                v-model:value="config.timeout_seconds"
                :min="5"
                :max="60"
                size="small"
                style="width: 120px"
              >
                <template #suffix>秒</template>
              </n-input-number>
            </n-space>
          </n-form-item>

          <n-form-item label="重試次數">
            <n-input-number
              v-model:value="config.max_retries"
              :min="0"
              :max="10"
              size="small"
              style="width: 120px"
            />
          </n-form-item>

          <n-form-item label="速率限制">
            <n-input-number
              v-model:value="config.rate_limit_per_sec"
              :min="10"
              :max="500"
              size="small"
              style="width: 120px"
            >
              <template #suffix>req/s</template>
            </n-input-number>
          </n-form-item>

          <n-form-item label="HTTP/2 多路復用">
            <n-switch v-model:value="config.enable_http2" />
          </n-form-item>

          <n-form-item label="智慧重試">
            <n-switch v-model:value="config.enable_jitter" />
          </n-form-item>

          <n-form-item label="結果備份">
            <n-switch v-model:value="config.backup_results" />
          </n-form-item>
        </n-card>

        <!-- 效能預估 -->
        <n-card title="效能預估" size="small">
          <n-space vertical>
            <n-text depth="2">
              <strong>預估完成時間：</strong> {{ estimatedTime }}
            </n-text>
            <n-text depth="2">
              <strong>理論吞吐量：</strong> {{ throughput }} keys/min
            </n-text>
            <n-text depth="2" v-if="selectedGroup">
              <strong>待驗證密鑰：</strong> {{ selectedGroup.total_keys || 0 }} 個
            </n-text>
          </n-space>
        </n-card>
      </n-form>

      <template #action>
        <n-space justify="end">
          <n-button @click="handleClose">取消</n-button>
          <n-button
            type="primary"
            :loading="loading"
            @click="startValidation"
          >
            <template #icon>
              <n-icon>
                <PlayOutline />
              </n-icon>
            </template>
            開始驗證
          </n-button>
        </n-space>
      </template>
    </n-card>
  </n-dialog>
</template>

<style scoped>
.batch-validation-dialog :deep(.n-dialog) {
  max-height: 90vh;
  overflow-y: auto;
}

.batch-validation-dialog :deep(.n-card) {
  border-radius: 12px;
}

.batch-validation-dialog :deep(.n-form-item-label) {
  font-weight: 500;
}
</style>