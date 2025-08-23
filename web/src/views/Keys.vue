<script setup lang="ts">
import { keysApi } from "@/api/keys";
import GroupInfoCard from "@/components/keys/GroupInfoCard.vue";
import GroupList from "@/components/keys/GroupList.vue";
import KeyTable from "@/components/keys/KeyTable.vue";

import type { Group } from "@/types/models";
import { onMounted, ref, computed } from "vue";
import { useRoute, useRouter } from "vue-router";


const groups = ref<Group[]>([]);
const loading = ref(false);
const selectedGroup = ref<Group | null>(null);
const router = useRouter();
const route = useRoute();

onMounted(async () => {
  await loadGroups();
});

async function loadGroups() {
  try {
    loading.value = true;
    groups.value = await keysApi.getGroups();
    // 選擇默認分組
    if (groups.value.length > 0 && !selectedGroup.value) {
      const groupId = route.query.groupId;
      const found = groups.value.find(g => String(g.id) === String(groupId));
      if (found) {
        selectedGroup.value = found;
        await loadGroupKeys(found);
      } else {
        await handleGroupSelect(groups.value[0]);
      }
    }
  } finally {
    loading.value = false;
  }
}

// 新增函數：加載分組的密鑰數據
async function loadGroupKeys(group: Group) {
  if (!group?.id) return;

  try {
    const result = await keysApi.getGroupKeys({
      group_id: group.id,
      page: 1,
      page_size: 1000, // 獲取所有密鑰用於批量驗證
    });

    // 更新選中分組的 api_keys 屬性
    if (selectedGroup.value?.id === group.id) {
      selectedGroup.value.api_keys = result.items;
    }
  } catch (error) {
    console.error('Failed to load group keys:', error);
  }
}

async function handleGroupSelect(group: Group | null) {
  selectedGroup.value = group || null;
  if (String(group?.id) !== String(route.query.groupId)) {
    router.push({ name: "keys", query: { groupId: group?.id || "" } });
  }

  // 加載選中分組的密鑰數據
  if (group) {
    await loadGroupKeys(group);
  }
}

async function handleGroupRefresh() {
  await loadGroups();
  if (selectedGroup.value) {
    // 重新加载当前选中的分组信息
    handleGroupSelect(groups.value.find(g => g.id === selectedGroup.value?.id) || null);
  }
}

async function handleGroupRefreshAndSelect(targetGroupId: number) {
  await loadGroups();
  // 刷新完成后，切换到指定的分组
  const targetGroup = groups.value.find(g => g.id === targetGroupId);
  if (targetGroup) {
    handleGroupSelect(targetGroup);
  }
}

function handleGroupDelete(deletedGroup: Group) {
  // 从分组列表中移除已删除的分组
  groups.value = groups.value.filter(g => g.id !== deletedGroup.id);

  // 如果删除的是当前选中的分组，则切换到第一个分组
  if (selectedGroup.value?.id === deletedGroup.id) {
    handleGroupSelect(groups.value.length > 0 ? groups.value[0] : null);
  }
}

async function handleGroupCopySuccess(newGroup: Group) {
  // 重新加载分组列表以包含新创建的分组
  await loadGroups();
  // 自动切换到新创建的分组
  const createdGroup = groups.value.find(g => g.id === newGroup.id);
  if (createdGroup) {
    handleGroupSelect(createdGroup);
  }
}
</script>

<template>
  <div class="keys-container">
    <div class="sidebar">
      <group-list
        :groups="groups"
        :selected-group="selectedGroup"
        :loading="loading"
        @group-select="handleGroupSelect"
        @refresh="handleGroupRefresh"
        @refresh-and-select="handleGroupRefreshAndSelect"
      />
    </div>

    <!-- 右側主內容區域 -->
    <div class="main-content">
      <!-- 分組信息卡片 -->
      <div class="group-info">
        <group-info-card
          :group="selectedGroup"
          @refresh="handleGroupRefresh"
          @delete="handleGroupDelete"
          @copy-success="handleGroupCopySuccess"
        />
      </div>

      <!-- 主要內容區域 -->
      <div class="content-main">
        <div class="key-table-section">
          <key-table :selected-group="selectedGroup" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.keys-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
  max-width: 100vw;
  overflow: hidden;
}

.sidebar {
  width: 100%;
  flex-shrink: 0;
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  overflow: hidden;
}

.group-info {
  flex-shrink: 0;
}

.content-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.key-table-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.batch-validation-section {
  flex: 1;
  padding: 16px;
  overflow-y: auto;
  max-width: 100%;
  box-sizing: border-box;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
}

@media (min-width: 768px) {
  .keys-container {
    flex-direction: row;
  }

  .sidebar {
    width: 240px;
    height: calc(100vh - 159px);
  }
}

/* 確保選項卡內容區域能正確填滿空間 */
:deep(.n-tabs-pane-wrapper) {
  flex: 1;
  display: flex;
  flex-direction: column;
}

:deep(.n-tab-pane) {
  flex: 1;
  display: flex;
  flex-direction: column;
}
</style>
