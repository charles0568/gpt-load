<script setup lang="ts">
import BaseInfoCard from "@/components/BaseInfoCard.vue";
import LineChart from "@/components/LineChart.vue";
import ValidationStatusCard from "@/components/ValidationStatusCard.vue";
import { NSpace } from "naive-ui";
</script>

<template>
  <div class="dashboard-container">
    <n-space vertical size="large">
      <base-info-card />
      <validation-status-card class="validation-status-card" />
      <line-chart class="dashboard-chart" />
    </n-space>
  </div>
</template>

<style scoped>
.dashboard-header-card {
  background: rgba(255, 255, 255, 0.98);
  border-radius: var(--border-radius-lg);
  border: 1px solid rgba(255, 255, 255, 0.3);
  animation: fadeInUp 0.2s ease-out;
}

.dashboard-title {
  font-size: 2.25rem;
  font-weight: 700;
  margin: 0;
  letter-spacing: -0.5px;
}

.dashboard-subtitle {
  font-size: 1.1rem;
  font-weight: 500;
}

.validation-status-card {
  animation: fadeInUp 0.2s ease-out 0.15s both;
}

.dashboard-chart {
  animation: fadeInUp 0.2s ease-out 0.3s both;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
