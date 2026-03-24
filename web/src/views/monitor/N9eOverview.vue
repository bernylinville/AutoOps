<template>
  <div class="n9e-overview-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <h3>CMDB 总览</h3>
      <p class="page-desc">资产统计与 N9E 同步状态概览</p>
    </div>

    <!-- 主机统计卡片 -->
    <el-row :gutter="16" class="stats-row">
      <el-col :span="4" v-for="item in hostCards" :key="item.label">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" :class="item.cls">
              <el-icon :size="26"><component :is="item.icon" /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-number">{{ item.value }}</div>
              <div class="stat-label">{{ item.label }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20">
      <!-- 左侧：数据来源分布 -->
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <span class="card-title">数据来源分布</span>
          </template>
          <div ref="sourceChartRef" class="chart-container"></div>
        </el-card>
      </el-col>

      <!-- 右侧：同步状态 -->
      <el-col :span="12">
        <el-card shadow="hover" class="sync-card">
          <template #header>
            <div class="card-header">
              <span class="card-title">N9E 同步状态</span>
              <el-tag :type="overview.n9eEnabled ? 'success' : 'info'" size="small">
                {{ overview.n9eEnabled ? '已启用' : '未启用' }}
              </el-tag>
            </div>
          </template>

          <div class="sync-info-list">
            <div class="sync-info-item">
              <span class="info-label">最后同步时间</span>
              <span class="info-value">{{ overview.lastSyncTime || '从未同步' }}</span>
            </div>
            <div class="sync-info-item">
              <span class="info-label">N9E 业务组</span>
              <span class="info-value">{{ overview.n9eBusiGroups || 0 }} 个</span>
            </div>
            <div class="sync-info-item">
              <span class="info-label">N9E 数据源</span>
              <span class="info-value">{{ overview.datasources || 0 }} 个</span>
            </div>
            <div class="sync-info-item">
              <span class="info-label">CMDB 分组</span>
              <span class="info-value">{{ overview.cmdbGroups || 0 }} 个</span>
            </div>
          </div>

          <el-divider />

          <!-- 同步结果摘要 -->
          <div v-if="syncResult" class="sync-result">
            <h4>最近同步结果</h4>
            <el-descriptions :column="3" size="small" border>
              <el-descriptions-item label="业务组新增">{{ syncResult.busiGroups?.created || 0 }}</el-descriptions-item>
              <el-descriptions-item label="主机新增">{{ syncResult.hosts?.created || 0 }}</el-descriptions-item>
              <el-descriptions-item label="数据源新增">{{ syncResult.datasources?.created || 0 }}</el-descriptions-item>
              <el-descriptions-item label="业务组更新">{{ syncResult.busiGroups?.updated || 0 }}</el-descriptions-item>
              <el-descriptions-item label="主机更新">{{ syncResult.hosts?.updated || 0 }}</el-descriptions-item>
              <el-descriptions-item label="数据源更新">{{ syncResult.datasources?.updated || 0 }}</el-descriptions-item>
              <el-descriptions-item label="主机跳过">{{ syncResult.hosts?.skipped || 0 }}</el-descriptions-item>
              <el-descriptions-item label="主机失联">{{ syncResult.hosts?.stale || 0 }}</el-descriptions-item>
              <el-descriptions-item label="">&nbsp;</el-descriptions-item>
            </el-descriptions>
          </div>
          <el-empty v-else description="暂无同步记录" :image-size="60" />
        </el-card>
      </el-col>
    </el-row>

    <!-- 最近同步记录 -->
    <el-card shadow="hover" class="log-card" style="margin-top: 20px;">
      <template #header>
        <div class="card-header">
          <span class="card-title">最近同步记录</span>
          <el-button text type="primary" size="small" @click="$router.push('/monitor/sync-logs')">查看全部</el-button>
        </div>
      </template>
      <el-table :data="recentLogs" stripe size="small" v-if="recentLogs.length > 0">
        <el-table-column label="时间" width="170">
          <template #default="{ row }">{{ formatLogTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="触发" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.triggerBy === 'cron' ? 'warning' : 'primary'" size="small" effect="plain">
              {{ row.triggerBy === 'cron' ? '定时' : '手动' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="80" align="center">
          <template #default="{ row }">
            {{ row.durationMs ? (row.durationMs / 1000).toFixed(1) + 's' : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="结果" min-width="200">
          <template #default="{ row }">
            <span v-if="row.parsedResult">
              主机 +{{ row.parsedResult.hosts?.created || 0 }}/↑{{ row.parsedResult.hosts?.updated || 0 }}
            </span>
            <span v-else-if="row.errorMsg" style="color: #f56c6c;">{{ row.errorMsg }}</span>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-else description="暂无同步记录" :image-size="40" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { Monitor, CircleCheckFilled, CircleCloseFilled, WarningFilled, FolderOpened, Connection } from '@element-plus/icons-vue'
import n9eApi from '@/api/n9e'
import * as echarts from 'echarts'

const overview = reactive({
  hosts: { total: 0, n9e: 0, manual: 0, cloud: 0, online: 0, offline: 0, stale: 0 },
  n9eBusiGroups: 0,
  datasources: 0,
  cmdbGroups: 0,
  lastSyncTime: '',
  lastSyncResult: '',
  n9eEnabled: false
})

const syncResult = ref(null)
const sourceChartRef = ref(null)
let chartInstance = null

const hostCards = computed(() => [
  { label: '主机总数', value: overview.hosts.total, icon: Monitor, cls: 'total' },
  { label: '在线', value: overview.hosts.online, icon: CircleCheckFilled, cls: 'online' },
  { label: '离线', value: overview.hosts.offline, icon: CircleCloseFilled, cls: 'offline' },
  { label: '失联', value: overview.hosts.stale, icon: WarningFilled, cls: 'stale' },
  { label: 'N9E 来源', value: overview.hosts.n9e, icon: Connection, cls: 'n9e' },
  { label: 'CMDB 分组', value: overview.cmdbGroups, icon: FolderOpened, cls: 'groups' }
])

const loadOverview = async () => {
  try {
    const res = await n9eApi.getOverview()
    if (res.data?.code === 200 && res.data.data) {
      const d = res.data.data
      Object.assign(overview.hosts, d.hosts || {})
      overview.n9eBusiGroups = d.n9eBusiGroups || 0
      overview.datasources = d.datasources || 0
      overview.cmdbGroups = d.cmdbGroups || 0
      overview.lastSyncTime = d.lastSyncTime || ''
      overview.n9eEnabled = d.n9eEnabled || false

      if (d.lastSyncResult) {
        try {
          syncResult.value = typeof d.lastSyncResult === 'string'
            ? JSON.parse(d.lastSyncResult)
            : d.lastSyncResult
        } catch (e) {
          syncResult.value = null
        }
      }

      await nextTick()
      renderSourceChart()
    }
  } catch (err) {
    console.error('Failed to load overview:', err)
  }
}

const renderSourceChart = () => {
  if (!sourceChartRef.value) return
  if (chartInstance) chartInstance.dispose()
  chartInstance = echarts.init(sourceChartRef.value)

  const option = {
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: { bottom: 0, textStyle: { fontSize: 12 } },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      avoidLabelOverlap: false,
      itemStyle: { borderRadius: 8, borderColor: '#fff', borderWidth: 2 },
      label: { show: true, formatter: '{b}\n{c}', fontSize: 12 },
      data: [
        { value: overview.hosts.n9e, name: 'N9E 同步', itemStyle: { color: '#409eff' } },
        { value: overview.hosts.manual, name: '手动录入', itemStyle: { color: '#67c23a' } },
        { value: overview.hosts.cloud, name: '云厂商', itemStyle: { color: '#e6a23c' } }
      ].filter(d => d.value > 0)
    }]
  }
  chartInstance.setOption(option)
}

const recentLogs = ref([])

const loadRecentLogs = async () => {
  try {
    const res = await n9eApi.getSyncLogs(5)
    if (res.data?.code === 200) {
      recentLogs.value = (res.data.data || []).map(log => ({
        ...log,
        parsedResult: (() => {
          if (!log.result) return null
          try { return typeof log.result === 'string' ? JSON.parse(log.result) : log.result }
          catch { return null }
        })()
      }))
    }
  } catch (err) {
    console.error('Failed to load recent logs:', err)
  }
}

const formatLogTime = (timeStr) => {
  if (!timeStr) return '-'
  const d = new Date(timeStr)
  return isNaN(d.getTime()) ? timeStr : d.toLocaleString('zh-CN', { hour12: false })
}

onMounted(() => {
  loadOverview()
  loadRecentLogs()
})

onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }
})
</script>

<style scoped>
.n9e-overview-container { padding: 20px; }

.page-header { margin-bottom: 20px; }
.page-header h3 { font-size: 18px; font-weight: 600; margin: 0 0 8px 0; color: #303133; }
.page-desc { color: #909399; font-size: 13px; margin: 0; }

.stats-row { margin-bottom: 20px; }

.stat-card { border-radius: 8px; }
.stat-content { display: flex; align-items: center; gap: 14px; }
.stat-icon { width: 50px; height: 50px; border-radius: 10px; display: flex; align-items: center; justify-content: center; color: #fff; }
.stat-icon.total { background: linear-gradient(135deg, #409eff, #337ecc); }
.stat-icon.online { background: linear-gradient(135deg, #67c23a, #529b2e); }
.stat-icon.offline { background: linear-gradient(135deg, #f56c6c, #c45656); }
.stat-icon.stale { background: linear-gradient(135deg, #909399, #6b6e75); }
.stat-icon.n9e { background: linear-gradient(135deg, #7c3aed, #5b21b6); }
.stat-icon.groups { background: linear-gradient(135deg, #e6a23c, #b88230); }
.stat-number { font-size: 24px; font-weight: 700; color: #303133; line-height: 1; }
.stat-label { font-size: 12px; color: #909399; margin-top: 4px; }

.chart-card, .sync-card { border-radius: 8px; }
.card-title { font-weight: 600; font-size: 15px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.chart-container { width: 100%; height: 300px; }

.sync-info-list { padding: 4px 0; }
.sync-info-item { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #f0f0f0; }
.sync-info-item:last-child { border-bottom: none; }
.info-label { color: #606266; font-size: 14px; }
.info-value { color: #303133; font-weight: 500; font-size: 14px; }

.sync-result h4 { margin: 0 0 10px 0; color: #303133; font-size: 14px; }
</style>
