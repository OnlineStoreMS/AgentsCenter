<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { createJob, listJobs, listShops, listSkills, type JobItem, type ShopItem, type SkillItem } from '../api/agents'

const route = useRoute()
const loading = ref(false)
const list = ref<JobItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const status = ref((route.query.status as string) || '')
const skills = ref<SkillItem[]>([])
const shops = ref<ShopItem[]>([])
const dialog = ref(false)
const form = reactive({
  jobType: 'doudian.aftersale',
  platform: 'doudian',
  platformShopId: '',
  platformShopName: '',
  orderNo: '',
  browserChannel: '',
  source: 'manual',
  priority: 100,
})

const selectedShop = computed(() =>
  shops.value.find(
    (s) => s.platform === form.platform && s.platformShopId === form.platformShopId,
  ),
)

const needsOrderNo = computed(() => form.jobType === 'doudian.order.decrypt-phone')
const needsParamsJson = computed(
  () => form.jobType !== 'doudian.aftersale' && form.jobType !== 'doudian.order.decrypt-phone',
)

async function load() {
  loading.value = true
  try {
    const res = await listJobs({
      page: page.value,
      pageSize: pageSize.value,
      status: status.value || undefined,
    })
    list.value = res.list || []
    total.value = res.total || 0
  } catch (e: any) {
    ElMessage.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function openCreate() {
  try {
    const [sk, sh] = await Promise.all([
      listSkills(),
      listShops({ page: 1, pageSize: 200 }),
    ])
    skills.value = sk || []
    shops.value = sh?.list || []
  } catch {
    skills.value = []
    shops.value = []
  }
  form.jobType = skills.value[0]?.id || 'doudian.aftersale'
  form.platform = 'doudian'
  form.platformShopId = ''
  form.platformShopName = ''
  form.orderNo = ''
  form.browserChannel = ''
  dialog.value = true
}

function onShopPick(id: string) {
  form.platformShopId = id
  const shop = shops.value.find((s) => s.platformShopId === id && s.platform === form.platform)
  if (shop) {
    form.platformShopName = shop.platformShopName
    form.browserChannel = shop.browserChannel || ''
  }
}

watch(
  () => form.jobType,
  (v) => {
    const skill = skills.value.find((s) => s.id === v)
    if (skill?.platform) form.platform = skill.platform
  },
)

async function submit() {
  if (!form.platformShopId) {
    ElMessage.error('请选择店铺')
    return
  }
  let paramsJson = '{}'
  if (form.jobType === 'doudian.order.decrypt-phone') {
    if (!form.orderNo.trim()) {
      ElMessage.error('请填写订单号')
      return
    }
    paramsJson = JSON.stringify({
      orderNo: form.orderNo.trim(),
      browserChannel: form.browserChannel || selectedShop.value?.browserChannel || 'msedge',
    })
  } else if (form.jobType === 'doudian.aftersale') {
    paramsJson = JSON.stringify({
      browserChannel: form.browserChannel || selectedShop.value?.browserChannel || '',
    })
  } else if (needsParamsJson.value) {
    paramsJson = JSON.stringify({
      browserChannel: form.browserChannel || selectedShop.value?.browserChannel || '',
    })
  }

  try {
    await createJob({
      jobType: form.jobType,
      platform: form.platform,
      platformShopId: form.platformShopId,
      platformShopName: form.platformShopName,
      paramsJson,
      source: form.source,
      priority: form.priority,
    })
    ElMessage.success('已创建任务')
    dialog.value = false
    await load()
  } catch (e: any) {
    ElMessage.error(e?.message || '创建失败')
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="head">
      <h2>任务队列</h2>
      <div class="filters">
        <el-select v-model="status" clearable placeholder="状态" style="width: 140px" @change="load">
          <el-option label="pending" value="pending" />
          <el-option label="claimed" value="claimed" />
          <el-option label="running" value="running" />
          <el-option label="succeeded" value="succeeded" />
          <el-option label="failed" value="failed" />
        </el-select>
        <el-button @click="load">刷新</el-button>
        <el-button type="primary" @click="openCreate">提交任务</el-button>
      </div>
    </div>
    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="jobType" label="类型" min-width="180" show-overflow-tooltip />
      <el-table-column prop="platformShopName" label="店铺" min-width="140" />
      <el-table-column prop="platformShopId" label="店铺ID" min-width="120" show-overflow-tooltip />
      <el-table-column prop="status" label="状态" width="100" />
      <el-table-column prop="agentName" label="执行节点" width="120" />
      <el-table-column prop="source" label="来源" width="100" />
      <el-table-column prop="createdAt" label="创建时间" min-width="170" />
      <el-table-column prop="errorMessage" label="错误" min-width="160" show-overflow-tooltip />
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

    <el-dialog v-model="dialog" title="提交 Agent 任务" width="560px">
      <el-form label-width="110px">
        <el-form-item label="任务类型">
          <el-select v-model="form.jobType" style="width: 100%">
            <el-option
              v-for="s in skills"
              :key="s.id"
              :label="`${s.name} (${s.id})`"
              :value="s.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="平台">
          <el-select v-model="form.platform" style="width: 100%">
            <el-option label="抖店" value="doudian" />
          </el-select>
        </el-form-item>
        <el-form-item label="店铺">
          <el-select
            :model-value="form.platformShopId"
            filterable
            placeholder="选择 Agent 已上报店铺"
            style="width: 100%"
            @change="onShopPick"
          >
            <el-option
              v-for="s in shops.filter((x) => x.platform === form.platform)"
              :key="`${s.agentId}-${s.platformShopId}`"
              :label="`${s.platformShopName || s.platformShopId} (${s.platformShopId})${s.agentOnline ? '' : ' · 离线'}`"
              :value="s.platformShopId"
            />
          </el-select>
        </el-form-item>
        <el-form-item v-if="needsOrderNo" label="订单号">
          <el-input v-model="form.orderNo" placeholder="抖店订单号" />
        </el-form-item>
        <el-alert
          v-if="form.jobType === 'doudian.aftersale'"
          type="info"
          :closable="false"
          show-icon
          title="售后抓取凭证由售后中心自动注入，无需填写。请确保售后店铺已启用 Agent 采集且店铺 ID 一致。"
          style="margin-bottom: 12px"
        />
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="submit">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; gap: 12px; }
.filters { display: flex; gap: 8px; }
.pager { margin-top: 12px; display: flex; justify-content: flex-end; }
</style>
