<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  listAssignments,
  listJobs,
  listShops,
  listSkills,
  setAssignmentEnabled,
  triggerAssignment,
  upsertAssignment,
  type AssignmentItem,
  type JobItem,
  type ShopItem,
  type SkillItem,
} from '../api/agents'

const route = useRoute()
const tab = ref<'assignments' | 'runs'>('assignments')

const loading = ref(false)
const assignLoading = ref(false)
const list = ref<JobItem[]>([])
const assignments = ref<AssignmentItem[]>([])
const total = ref(0)
const assignTotal = ref(0)
const page = ref(1)
const assignPage = ref(1)
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
  runPolicy: 'interval',
  intervalMinutes: 30,
  triggerNow: true,
  source: 'manual',
  priority: 100,
})

const selectedSkill = computed(() => skills.value.find((s) => s.id === form.jobType))
const supportsInterval = computed(() => (selectedSkill.value?.runPolicies || []).includes('interval'))
const selectedShop = computed(() =>
  shops.value.find(
    (s) => s.platform === form.platform && s.platformShopId === form.platformShopId,
  ),
)
const needsOrderNo = computed(() => form.jobType === 'doudian.order.decrypt-phone')

function policyLabel(p: string) {
  if (p === 'interval') return '定时间隔'
  if (p === 'on_demand') return '手动/服务端'
  if (p === 'daily') return '每日'
  return p
}

async function loadRuns() {
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

async function loadAssignments() {
  assignLoading.value = true
  try {
    const res = await listAssignments({ page: assignPage.value, pageSize: pageSize.value })
    assignments.value = res.list || []
    assignTotal.value = res.total || 0
  } catch (e: any) {
    ElMessage.error(e?.message || '加载订阅失败')
  } finally {
    assignLoading.value = false
  }
}

async function load() {
  if (tab.value === 'assignments') await loadAssignments()
  else await loadRuns()
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
  applySkillDefaults(form.jobType)
  form.platform = skills.value[0]?.platform || 'doudian'
  form.platformShopId = ''
  form.platformShopName = ''
  form.orderNo = ''
  form.browserChannel = ''
  form.triggerNow = true
  dialog.value = true
}

function applySkillDefaults(jobType: string) {
  const skill = skills.value.find((s) => s.id === jobType)
  if (!skill) return
  if ((skill.runPolicies || []).includes('interval')) {
    form.runPolicy = 'interval'
    form.intervalMinutes = skill.defaultIntervalMinutes || 30
  } else {
    form.runPolicy = 'on_demand'
  }
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
    applySkillDefaults(v)
  },
)

watch(tab, () => {
  page.value = 1
  assignPage.value = 1
  load()
})

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
  } else {
    paramsJson = JSON.stringify({
      browserChannel: form.browserChannel || selectedShop.value?.browserChannel || '',
    })
  }

  const runPolicy = supportsInterval.value ? form.runPolicy : 'on_demand'
  try {
    await upsertAssignment({
      jobType: form.jobType,
      platform: form.platform,
      platformShopId: form.platformShopId,
      platformShopName: form.platformShopName,
      enabled: true,
      runPolicy,
      intervalMinutes: runPolicy === 'interval' ? form.intervalMinutes : undefined,
      triggerNow: form.triggerNow,
      paramsJson,
      source: form.source,
      priority: form.priority,
    })
    ElMessage.success(form.triggerNow ? '已保存订阅并下发执行' : '已保存订阅')
    dialog.value = false
    tab.value = 'assignments'
    await loadAssignments()
    if (form.triggerNow) {
      tab.value = 'runs'
      await loadRuns()
    }
  } catch (e: any) {
    ElMessage.error(e?.message || '创建失败')
  }
}

async function onTrigger(row: AssignmentItem) {
  try {
    await triggerAssignment(row.id, { source: 'manual' })
    ElMessage.success('已下发执行单')
    tab.value = 'runs'
    await loadRuns()
  } catch (e: any) {
    ElMessage.error(e?.message || '下发失败')
  }
}

async function onToggle(row: AssignmentItem, enabled: boolean) {
  try {
    await setAssignmentEnabled(row.id, enabled)
    ElMessage.success(enabled ? '已启用' : '已停用')
    await loadAssignments()
  } catch (e: any) {
    ElMessage.error(e?.message || '操作失败')
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="head">
      <h2>任务</h2>
      <div class="filters">
        <el-button @click="load">刷新</el-button>
        <el-button type="primary" @click="openCreate">新建任务</el-button>
      </div>
    </div>

    <el-tabs v-model="tab">
      <el-tab-pane label="任务（可反复执行）" name="assignments">
        <el-table :data="assignments" v-loading="assignLoading" stripe>
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column label="任务" min-width="160">
            <template #default="{ row }">
              <div>{{ row.jobTypeName || row.jobType }}</div>
              <div class="sub">{{ row.jobType }}</div>
            </template>
          </el-table-column>
          <el-table-column prop="platformShopName" label="店铺" min-width="140" />
          <el-table-column prop="platformShopId" label="店铺ID" min-width="120" show-overflow-tooltip />
          <el-table-column label="运行周期" width="140">
            <template #default="{ row }">
              <span>{{ policyLabel(row.runPolicy) }}</span>
              <span v-if="row.runPolicy === 'interval' && row.intervalMinutes"> · {{ row.intervalMinutes }}分</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '停用' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="nextRunAt" label="下次执行" min-width="170">
            <template #default="{ row }">{{ row.nextRunAt || '—' }}</template>
          </el-table-column>
          <el-table-column prop="lastEnqueuedAt" label="最近下发" min-width="170">
            <template #default="{ row }">{{ row.lastEnqueuedAt || '—' }}</template>
          </el-table-column>
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link @click="onTrigger(row)">立即执行</el-button>
              <el-button type="primary" link @click="onToggle(row, !row.enabled)">
                {{ row.enabled ? '停用' : '启用' }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="pager">
          <el-pagination
            v-model:current-page="assignPage"
            v-model:page-size="pageSize"
            :total="assignTotal"
            layout="total, prev, pager, next"
            @current-change="loadAssignments"
          />
        </div>
      </el-tab-pane>

      <el-tab-pane label="执行记录" name="runs">
        <p class="hint" style="margin: 0 0 12px">每次触发执行会产生一条记录，不是新建任务。</p>
        <div class="filters" style="margin-bottom: 12px">
          <el-select v-model="status" clearable placeholder="状态" style="width: 140px" @change="loadRuns">
            <el-option label="pending" value="pending" />
            <el-option label="claimed" value="claimed" />
            <el-option label="running" value="running" />
            <el-option label="succeeded" value="succeeded" />
            <el-option label="failed" value="failed" />
          </el-select>
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
            @current-change="loadRuns"
          />
        </div>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="dialog" title="新建任务（订阅）" width="560px">
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
          <div v-if="selectedSkill" class="hint">
            支持：{{ (selectedSkill.runPolicies || []).map(policyLabel).join('、') }}
            <template v-if="supportsInterval">（定时间隔由对应业务中心触发下发，Agents 只转发）</template>
          </div>
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
        <el-form-item v-if="supportsInterval" label="运行周期">
          <el-radio-group v-model="form.runPolicy">
            <el-radio-button label="interval">定时间隔</el-radio-button>
            <el-radio-button label="on_demand">仅手动</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="supportsInterval && form.runPolicy === 'interval'" label="间隔分钟">
          <el-input-number v-model="form.intervalMinutes" :min="5" :max="1440" />
        </el-form-item>
        <el-form-item v-if="needsOrderNo" label="订单号">
          <el-input v-model="form.orderNo" placeholder="抖店订单号" />
        </el-form-item>
        <el-form-item label="立即执行">
          <el-switch v-model="form.triggerNow" />
        </el-form-item>
        <el-alert
          v-if="form.jobType === 'doudian.aftersale'"
          type="info"
          :closable="false"
          show-icon
          title="售后抓取参数（上报地址、凭证等）由售后中心创建/触发任务时写入。定时也由售后中心触发；Agents 只做转发，WindowsAgent 按参数执行。"
          style="margin-bottom: 12px"
        />
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; gap: 12px; }
.filters { display: flex; gap: 8px; align-items: center; }
.pager { margin-top: 12px; display: flex; justify-content: flex-end; }
.sub { color: #909399; font-size: 12px; }
.hint { color: #909399; font-size: 12px; margin-top: 4px; }
</style>
