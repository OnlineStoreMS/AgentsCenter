<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { fetchDashboardStats, type DashboardStats } from '../api/agents'

const router = useRouter()
const loading = ref(false)
const stats = ref<DashboardStats>({
  agentTotal: 0,
  agentOnline: 0,
  shopTotal: 0,
  jobPending: 0,
  jobRunning: 0,
})

async function load() {
  loading.value = true
  try {
    const res = await fetchDashboardStats()
    stats.value = res
  } catch (e: any) {
    ElMessage.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>Agents 工作台</h2>
    <p class="tip">业务中心只提交「任务类型 + 店铺 + 参数」；中心按在线 Agent 与已登录店铺做匹配分配。</p>
    <div class="cards">
      <el-card shadow="hover" class="card" @click="router.push('/agents')">
        <div class="label">在线节点</div>
        <div class="value">{{ stats.agentOnline }} <span>/ {{ stats.agentTotal }}</span></div>
      </el-card>
      <el-card shadow="hover" class="card" @click="router.push('/shops')">
        <div class="label">店铺会话</div>
        <div class="value">{{ stats.shopTotal }}</div>
      </el-card>
      <el-card shadow="hover" class="card" @click="router.push({ path: '/jobs', query: { status: 'pending' } })">
        <div class="label">待分配任务</div>
        <div class="value">{{ stats.jobPending }}</div>
      </el-card>
      <el-card shadow="hover" class="card" @click="router.push({ path: '/jobs', query: { status: 'running' } })">
        <div class="label">执行中</div>
        <div class="value">{{ stats.jobRunning }}</div>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.page { padding: 4px; }
.tip { color: #666; margin: 0 0 16px; }
.cards { display: grid; grid-template-columns: repeat(4, minmax(140px, 1fr)); gap: 12px; }
.card { cursor: pointer; }
.label { color: #888; font-size: 13px; }
.value { font-size: 28px; font-weight: 600; margin-top: 8px; }
.value span { font-size: 14px; color: #999; font-weight: 400; }
@media (max-width: 900px) {
  .cards { grid-template-columns: repeat(2, 1fr); }
}
</style>
