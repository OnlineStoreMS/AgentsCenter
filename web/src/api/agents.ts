import http, { unwrap, type PageData } from './client'

export interface DashboardStats {
  agentTotal: number
  agentOnline: number
  shopTotal: number
  jobPending: number
  jobRunning: number
}

export interface AgentItem {
  id: number
  machineId: string
  name: string
  hostname: string
  os: string
  agentVersion: string
  status: string
  skillsJson: string
  shopCount: number
  shops?: Array<{
    platform: string
    platformShopId: string
    platformShopName: string
    browserChannel: string
    status: string
  }>
  lastHeartbeat?: string
  createdAt: string
}

export interface ShopItem {
  id: number
  agentId: number
  agentName: string
  platform: string
  platformShopId: string
  platformShopName: string
  browserChannel: string
  capabilitiesJson: string
  status: string
  lastSeenAt?: string
  agentOnline: boolean
}

export interface JobItem {
  id: number
  jobType: string
  platform: string
  platformShopId: string
  platformShopName: string
  paramsJson: string
  source: string
  priority: number
  status: string
  agentId?: number
  agentName: string
  errorMessage: string
  createdAt: string
  finishedAt?: string
}

export interface SkillItem {
  id: string
  name: string
  platform: string
  description: string
  runPolicies: string[]
  defaultIntervalMinutes?: number
}

export interface AssignmentItem {
  id: number
  jobType: string
  jobTypeName: string
  platform: string
  platformShopId: string
  platformShopName: string
  enabled: boolean
  runPolicy: string
  intervalMinutes?: number | null
  lastEnqueuedAt?: string | null
  nextRunAt?: string | null
  createdAt: string
  updatedAt: string
}

export async function fetchDashboardStats() {
  return unwrap<DashboardStats>(await http.get('/dashboard/stats'))
}

export async function listAgents(params: { page?: number; pageSize?: number }) {
  return unwrap<PageData<AgentItem>>(await http.get('/agents', { params }))
}

export async function listShops(params: { page?: number; pageSize?: number; platform?: string }) {
  return unwrap<PageData<ShopItem>>(await http.get('/shops', { params }))
}

export async function listJobs(params: { page?: number; pageSize?: number; status?: string; jobType?: string }) {
  return unwrap<PageData<JobItem>>(await http.get('/jobs', { params }))
}

export async function createJob(body: {
  jobType: string
  platform: string
  platformShopId: string
  platformShopName?: string
  paramsJson?: string
  source?: string
  priority?: number
}) {
  return unwrap(await http.post('/jobs', body))
}

export async function listSkills() {
  return unwrap<SkillItem[]>(await http.get('/skills'))
}

export async function listAssignments(params: { page?: number; pageSize?: number; jobType?: string }) {
  return unwrap<PageData<AssignmentItem>>(await http.get('/assignments', { params }))
}

export async function upsertAssignment(body: {
  jobType: string
  platform: string
  platformShopId: string
  platformShopName?: string
  enabled?: boolean
  runPolicy?: string
  intervalMinutes?: number
  triggerNow?: boolean
  paramsJson?: string
  source?: string
  priority?: number
}) {
  return unwrap(await http.post('/assignments', body))
}

export async function triggerAssignment(id: number, body?: { paramsJson?: string; source?: string }) {
  return unwrap(await http.post(`/assignments/${id}/trigger`, body || {}))
}

export async function setAssignmentEnabled(id: number, enabled: boolean) {
  return unwrap(await http.put(`/assignments/${id}/enabled`, { enabled }))
}
