<script setup lang="ts">
import type { Group } from "@/types/models";
import { keysApi } from "@/api/keys";
import {
  NButton,
  NCard,
  NModal,
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
import { FlashOutline, SettingsOutline, PlayOutline, CloseOutline } from "@vicons/ionicons5";
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
  <n-modal
    :show="props.visible"
    @update:show="(value: boolean) => emit('update:visible', value)"
    title="批量密鑰驗證"
    class="batch-validation-dialog"
    style="width: 1400px; max-width: 98vw; height: 90vh; max-height: 90vh"
    :teleport="true"
    :z-index="9999"
    :mask-closable="true"
    :close-on-esc="true"
    :show-mask="false"
    transform-origin="center"
    @mask-click="handleClose"
  >
    <n-card>
      <template #header>
        <n-space align="center" justify="space-between">
          <n-space align="center">
            <n-icon size="20" color="var(--primary-color)">
              <FlashOutline />
            </n-icon>
            <span>高效能批量驗證</span>
          </n-space>
          <n-button quaternary circle @click="handleClose">
            <template #icon>
              <n-icon><CloseOutline /></n-icon>
            </template>
          </n-button>
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
          <n-space size="small" wrap>
            <n-button
              v-for="preset in presetOptions"
              :key="preset.value"
              size="small"
              type="tertiary"
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
            <div style="width: 100%">
              <n-slider
                v-model:value="config.concurrency"
                :min="1"
                :max="100"
                :step="1"
                show-tooltip
                style="margin-bottom: 8px"
              />
              <div style="display: flex; justify-content: space-between; align-items: center">
                <span style="font-size: 12px; color: #999">1</span>
                <n-input-number
                  v-model:value="config.concurrency"
                  :min="1"
                  :max="100"
                  size="small"
                  style="width: 120px"
                />
                <span style="font-size: 12px; color: #999">100</span>
              </div>
            </div>
          </n-form-item>

          <n-form-item label="超時時間">
            <div style="width: 100%">
              <n-slider
                v-model:value="config.timeout_seconds"
                :min="5"
                :max="60"
                :step="1"
                show-tooltip
                :format-tooltip="(value) => `${value} 秒`"
                style="margin-bottom: 8px"
              />
              <div style="display: flex; justify-content: space-between; align-items: center">
                <span style="font-size: 12px; color: #999">5秒</span>
                <n-input-number
                  v-model:value="config.timeout_seconds"
                  :min="5"
                  :max="60"
                  size="small"
                  style="width: 120px"
                >
                  <template #suffix>秒</template>
                </n-input-number>
                <span style="font-size: 12px; color: #999">60秒</span>
              </div>
            </div>
          </n-form-item>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px">
            <n-form-item label="重試次數">
              <n-input-number
                v-model:value="config.max_retries"
                :min="0"
                :max="10"
                size="small"
                style="width: 100%"
              />
            </n-form-item>

            <n-form-item label="速率限制">
              <n-input-number
                v-model:value="config.rate_limit_per_sec"
                :min="10"
                :max="500"
                size="small"
                style="width: 100%"
              >
                <template #suffix>req/s</template>
              </n-input-number>
            </n-form-item>
          </div>

          <div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 16px">
            <n-form-item label="HTTP/2 多路復用">
              <n-switch v-model:value="config.enable_http2" />
            </n-form-item>

            <n-form-item label="智慧重試">
              <n-switch v-model:value="config.enable_jitter" />
            </n-form-item>

            <n-form-item label="結果備份">
              <n-switch v-model:value="config.backup_results" />
            </n-form-item>
          </div>
        </n-card>

        <!-- 效能預估 -->
        <n-card title="效能預估" size="small">
          <div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 16px; text-align: center">
            <div>
              <n-text depth="3" style="font-size: 12px">預估完成時間</n-text>
              <div style="font-size: 16px; font-weight: 600; color: var(--primary-color)">
                {{ estimatedTime }}
              </div>
            </div>
            <div>
              <n-text depth="3" style="font-size: 12px">理論吞吐量</n-text>
              <div style="font-size: 16px; font-weight: 600; color: var(--primary-color)">
                {{ throughput }} keys/min
              </div>
            </div>
            <div v-if="selectedGroup">
              <n-text depth="3" style="font-size: 12px">待驗證密鑰</n-text>
              <div style="font-size: 16px; font-weight: 600; color: var(--primary-color)">
                {{ selectedGroup.total_keys || 0 }} 個
              </div>
            </div>
          </div>
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
  </n-modal>
</template>

<style scoped>
/* 確保對話框覆蓋整個視窗 */
.batch-validation-dialog :deep(.n-modal) {
  position: fixed !important;
  top: 0 !important;
  left: 0 !important;
  width: 100vw !important;
  height: 100vh !important;
  z-index: 9999 !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
}

/* 完全隱藏遮罩層 */
.batch-validation-dialog :deep(.n-modal-mask) {
  display: none !important;
  visibility: hidden !important;
  opacity: 0 !important;
}

/* 全局隱藏所有模態框遮罩層 */
:global(.n-modal-mask) {
  display: none !important;
  visibility: hidden !important;
  opacity: 0 !important;
}

.batch-validation-dialog :deep(.n-modal-container) {
  position: relative !important;
  z-index: 10000 !important;
  width: 1400px !important;
  max-width: 98vw !important;
  height: 90vh !important;
  max-height: 90vh !important;
}

.batch-validation-dialog :deep(.n-modal__content) {
  height: 100% !important;
  max-height: 100% !important;
  overflow: hidden !important;
  display: flex !important;
  flex-direction: column !important;
  border-radius: 12px !important;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3), 0 0 0 1px rgba(255, 255, 255, 0.1) !important;
}

.batch-validation-dialog :deep(.n-card) {
  height: 100% !important;
  display: flex !important;
  flex-direction: column !important;
  border-radius: 12px !important;
  overflow: hidden !important;
}

.batch-validation-dialog :deep(.n-card__header) {
  flex-shrink: 0 !important;
  padding: 20px 20px 16px 20px !important;
  border-bottom: 1px solid var(--n-border-color) !important;
}

.batch-validation-dialog :deep(.n-card__content) {
  flex: 1 !important;
  overflow-y: auto !important;
  padding: 20px !important;
  min-height: 0 !important;
}

.batch-validation-dialog :deep(.n-card__footer) {
  flex-shrink: 0 !important;
  border-top: 1px solid var(--n-border-color) !important;
  padding: 16px 20px !important;
  background: var(--n-color) !important;
}

.batch-validation-dialog :deep(.n-form-item-label) {
  font-weight: 500;
  white-space: nowrap;
}

.batch-validation-dialog :deep(.n-form-item) {
  margin-bottom: 16px;
}

.batch-validation-dialog :deep(.n-slider) {
  margin: 8px 0;
}

.batch-validation-dialog :deep(.n-input-number) {
  min-width: 80px;
}

/* 響應式設計 */
@media (max-width: 768px) {
  .batch-validation-dialog :deep(.n-modal-container) {
    width: 98vw !important;
    max-width: none !important;
    height: 95vh !important;
    max-height: 95vh !important;
  }

  /* 在小屏幕上使用單列佈局 */
  div[style*="grid-template-columns: 1fr 1fr"] {
    display: block !important;
  }

  div[style*="grid-template-columns: 1fr 1fr 1fr"] {
    display: block !important;
  }
}

/* 確保對話框在所有情況下都能正確顯示 */
:global(.n-modal-container) {
  position: relative !important;
  z-index: 10000 !important;
}

:global(.n-modal-mask) {
  z-index: 9998 !important;
}
</style>
