<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { createJob, listJobs, listSkills, type JobItem, type SkillItem } from '../api/agents'

const route = useRoute()
const loading = ref(false)
const list = ref<JobItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const status = ref((route.query.status as string) || '')
const skills = ref<SkillItem[]>([])
const dialog = ref(false)
const form = reactive({
  jobType: 'doudian.order.decrypt-phone',
  platform: 'doudian',
  platformShopId: '',
  platformShopName: '',
  paramsJson: '{\n  "orderNo": "",\n  "browserChannel": "msedge"\n}',
  source: 'manual',
  priority: 100,
})

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
    skills.value = (await listSkills()) || []
  } catch {
    skills.value = []
  }
  dialog.value = true
}

async function submit() {
  try {
    JSON.parse(form.paramsJson || '{}')
  } catch {
    ElMessage.error('paramsJson 不是合法 JSON')
    return
  }
  try {
    await createJob({ ...form })
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
            <el-option label="淘宝" value="taobao" />
            <el-option label="拼多多" value="pdd" />
          </el-select>
        </el-form-item>
        <el-form-item label="店铺 ID">
          <el-input v-model="form.platformShopId" placeholder="平台侧店铺 ID" />
        </el-form-item>
        <el-form-item label="店铺名称">
          <el-input v-model="form.platformShopName" />
        </el-form-item>
        <el-form-item label="参数 JSON">
          <el-input v-model="form.paramsJson" type="textarea" :rows="6" />
        </el-form-item>
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
