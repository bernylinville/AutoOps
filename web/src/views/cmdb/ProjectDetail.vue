<template>
  <div class="project-detail" v-loading="pageLoading">
    <!-- 顶部：项目信息 + 返回 -->
    <el-card shadow="hover" style="margin-bottom: 16px">
      <div class="project-header">
        <div class="project-info">
          <el-button text @click="$router.back()" style="margin-right: 12px">
            <el-icon><ArrowLeft /></el-icon>
          </el-button>
          <span class="project-name">{{ project.name }}</span>
          <el-tag size="small" type="info" style="margin-left: 8px">{{ project.code }}</el-tag>
          <el-tag :type="statusTagType(project.status)" size="small" style="margin-left: 6px">
            {{ statusText(project.status) }}
          </el-tag>
        </div>
        <span class="project-desc">{{ project.description }}</span>
      </div>
    </el-card>

    <!-- 统计卡片 -->
    <el-row :gutter="16" style="margin-bottom: 16px">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-value">{{ stats.totalHosts }}</div>
          <div class="stat-label">主机总数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card online">
          <div class="stat-value">{{ stats.onlineHosts }}</div>
          <div class="stat-label">在线主机</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card db">
          <div class="stat-value">{{ stats.totalDatabases }}</div>
          <div class="stat-label">数据库数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card app">
          <div class="stat-value">{{ stats.totalApps }}</div>
          <div class="stat-label">应用数</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 图表区 -->
    <el-row :gutter="16" style="margin-bottom: 16px">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header><span>主机分组分布</span></template>
          <div ref="hostChartRef" style="height: 240px"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header><span>数据库类型分布</span></template>
          <div ref="dbChartRef" style="height: 240px"></div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 关联资产 Tabs -->
    <el-card shadow="hover">
      <el-tabs v-model="activeTab" @tab-change="onTabChange">
        <!-- 关联主机 -->
        <el-tab-pane label="关联主机" name="hosts">
          <el-table :data="hosts" v-loading="hostsLoading" stripe border>
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="hostName" label="主机名称" min-width="140" show-overflow-tooltip />
            <el-table-column prop="sshIp" label="SSH IP" width="130" />
            <el-table-column prop="privateIp" label="内网IP" width="130" />
            <el-table-column prop="os" label="操作系统" min-width="120" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="hostStatusType(row.status)" size="small">{{ hostStatusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="group.name" label="所属分组" width="120" />
          </el-table>
          <div class="pagination-wrapper" v-if="hostsTotal > 0">
            <el-pagination
              v-model:current-page="hostsPage"
              v-model:page-size="hostsPageSize"
              :total="hostsTotal"
              layout="total, prev, pager, next"
              @current-change="loadHosts"
            />
          </div>
        </el-tab-pane>

        <!-- 关联数据库 -->
        <el-tab-pane label="关联数据库" name="databases">
          <el-table :data="databases" v-loading="dbsLoading" stripe border>
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="name" label="数据库名称" min-width="140" show-overflow-tooltip />
            <el-table-column prop="type" label="类型" width="120" align="center">
              <template #default="{ row }">
                <el-tag size="small">{{ dbTypeText(row.type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="description" label="描述" min-width="160" show-overflow-tooltip />
            <el-table-column prop="tags" label="标签" min-width="120" show-overflow-tooltip />
          </el-table>
          <div class="pagination-wrapper" v-if="dbsTotal > 0">
            <el-pagination
              v-model:current-page="dbsPage"
              v-model:page-size="dbsPageSize"
              :total="dbsTotal"
              layout="total, prev, pager, next"
              @current-change="loadDatabases"
            />
          </div>
        </el-tab-pane>

        <!-- 关联应用 -->
        <el-tab-pane label="关联应用" name="apps">
          <el-table :data="apps" v-loading="appsLoading" stripe border>
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="name" label="应用名称" min-width="150" show-overflow-tooltip />
            <el-table-column prop="code" label="应用代码" width="140">
              <template #default="{ row }">
                <el-tag size="small" type="info">{{ row.code }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="programmingLang" label="技术栈" width="110" />
            <el-table-column prop="status" label="状态" width="80" align="center">
              <template #default="{ row }">
                <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
                  {{ row.status === 1 ? '正常' : '停用' }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!appsLoading && apps.length === 0" description="暂无关联应用" />
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import projectApi from '@/api/project'

const route = useRoute()
const projectId = parseInt(route.params.id)

// 页面状态
const pageLoading = ref(false)
const project = ref({ name: '', code: '', description: '', status: 1 })
const stats = ref({ totalHosts: 0, onlineHosts: 0, offlineHosts: 0, totalDatabases: 0, totalApps: 0, hostsByGroup: [], dbsByType: [] })

// 图表 ref
const hostChartRef = ref(null)
const dbChartRef = ref(null)
let hostChart = null
let dbChart = null

// Tabs
const activeTab = ref('hosts')

// 主机分页
const hosts = ref([])
const hostsLoading = ref(false)
const hostsTotal = ref(0)
const hostsPage = ref(1)
const hostsPageSize = ref(20)

// 数据库分页
const databases = ref([])
const dbsLoading = ref(false)
const dbsTotal = ref(0)
const dbsPage = ref(1)
const dbsPageSize = ref(20)

// 应用
const apps = ref([])
const appsLoading = ref(false)

onMounted(async () => {
  pageLoading.value = true
  await Promise.all([loadProject(), loadStats()])
  pageLoading.value = false
  loadHosts()
  await nextTick()
  initCharts()
})

async function loadProject() {
  try {
    const res = await projectApi.getProjectDetail(projectId)
    if (res.data && res.data.data) {
      project.value = res.data.data
    }
  } catch (e) {
    ElMessage.error('获取项目信息失败')
  }
}

async function loadStats() {
  try {
    const res = await projectApi.getProjectStats(projectId)
    if (res.data && res.data.data) {
      stats.value = res.data.data
    }
  } catch (e) {
    // ignore
  }
}

async function loadHosts() {
  hostsLoading.value = true
  try {
    const res = await projectApi.getProjectHosts({ projectId, page: hostsPage.value, pageSize: hostsPageSize.value })
    if (res.data && res.data.data) {
      hosts.value = res.data.data.list || []
      hostsTotal.value = res.data.data.total || 0
    }
  } finally {
    hostsLoading.value = false
  }
}

async function loadDatabases() {
  dbsLoading.value = true
  try {
    const res = await projectApi.getProjectDatabases({ projectId, page: dbsPage.value, pageSize: dbsPageSize.value })
    if (res.data && res.data.data) {
      databases.value = res.data.data.list || []
      dbsTotal.value = res.data.data.total || 0
    }
  } finally {
    dbsLoading.value = false
  }
}

async function loadApps() {
  appsLoading.value = true
  try {
    const res = await projectApi.getProjectApps(projectId)
    if (res.data && res.data.data) {
      apps.value = res.data.data || []
    }
  } finally {
    appsLoading.value = false
  }
}

function onTabChange(tab) {
  if (tab === 'databases' && databases.value.length === 0) loadDatabases()
  if (tab === 'apps' && apps.value.length === 0) loadApps()
}

function initCharts() {
  // 主机按分组饼图
  if (hostChartRef.value) {
    hostChart = echarts.init(hostChartRef.value)
    const groupData = (stats.value.hostsByGroup || []).map(g => ({ name: g.groupName || '未分组', value: g.count }))
    hostChart.setOption({
      tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
      legend: { orient: 'vertical', right: 10, top: 'center' },
      series: [{
        type: 'pie', radius: ['40%', '65%'],
        center: ['40%', '50%'],
        data: groupData.length ? groupData : [{ name: '暂无数据', value: 0 }],
        label: { show: false }
      }]
    })
  }

  // 数据库按类型柱图
  if (dbChartRef.value) {
    dbChart = echarts.init(dbChartRef.value)
    const dbTypeMap = { 1: 'MySQL', 2: 'PostgreSQL', 3: 'Redis', 4: 'MongoDB', 5: 'Elasticsearch' }
    const typeData = (stats.value.dbsByType || []).map(d => ({ name: dbTypeMap[d.type] || `类型${d.type}`, value: d.count }))
    dbChart.setOption({
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: typeData.map(d => d.name) },
      yAxis: { type: 'value', minInterval: 1 },
      series: [{ type: 'bar', data: typeData.map(d => d.value), barMaxWidth: 40 }]
    })
  }
}

// 工具方法
function statusTagType(status) {
  return { 1: 'success', 2: 'warning', 3: 'danger' }[status] || 'info'
}
function statusText(status) {
  return { 1: '活跃', 2: '归档', 3: '已废弃' }[status] || '未知'
}
function hostStatusType(status) {
  return { 1: 'success', 2: 'warning', 3: 'danger', 4: 'danger', 5: 'warning' }[status] || 'info'
}
function hostStatusText(status) {
  return { 1: '在线', 2: '未认证', 3: '离线', 4: '失联', 5: '降级' }[status] || '未知'
}
function dbTypeText(type) {
  return { 1: 'MySQL', 2: 'PostgreSQL', 3: 'Redis', 4: 'MongoDB', 5: 'Elasticsearch' }[type] || `类型${type}`
}
</script>

<style scoped>
.project-header {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.project-info {
  display: flex;
  align-items: center;
}
.project-name {
  font-size: 18px;
  font-weight: 700;
}
.project-desc {
  color: #666;
  font-size: 13px;
  padding-left: 44px;
}
.stat-card {
  text-align: center;
  padding: 8px 0;
}
.stat-card .stat-value {
  font-size: 36px;
  font-weight: 700;
  color: #409eff;
}
.stat-card.online .stat-value { color: #67c23a; }
.stat-card.db .stat-value { color: #e6a23c; }
.stat-card.app .stat-value { color: #9b59b6; }
.stat-label {
  font-size: 13px;
  color: #888;
  margin-top: 4px;
}
.pagination-wrapper {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}
</style>
