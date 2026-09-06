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
