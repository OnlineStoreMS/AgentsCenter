<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listShops, type ShopItem } from '../api/agents'

const loading = ref(false)
const list = ref<ShopItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const platform = ref('')

async function load() {
  loading.value = true
  try {
    const res = await listShops({ page: page.value, pageSize: pageSize.value, platform: platform.value || undefined })
    list.value = res.list || []
    total.value = res.total || 0
  } catch (e: any) {
    ElMessage.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="head">
      <h2>店铺会话</h2>
      <div class="filters">
        <el-select v-model="platform" clearable placeholder="平台" style="width: 140px" @change="load">
          <el-option label="抖店" value="doudian" />
          <el-option label="淘宝" value="taobao" />
          <el-option label="拼多多" value="pdd" />
        </el-select>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>
    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="platform" label="平台" width="90" />
      <el-table-column prop="platformShopName" label="店铺名称" min-width="160" />
      <el-table-column prop="platformShopId" label="店铺 ID" min-width="140" show-overflow-tooltip />
      <el-table-column prop="agentName" label="所在节点" width="140" />
      <el-table-column label="节点状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.agentOnline ? 'success' : 'info'" size="small">{{ row.agentOnline ? '在线' : '离线' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="browserChannel" label="浏览器" width="90" />
      <el-table-column prop="status" label="会话" width="90" />
      <el-table-column prop="lastSeenAt" label="最近上报" min-width="170" />
    </el-table>
    <div class="pager">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="load"
      />
    </div>
  </div>
</template>

<style scoped>
.head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; gap: 12px; }
.filters { display: flex; gap: 8px; }
.pager { margin-top: 12px; display: flex; justify-content: flex-end; }
</style>
