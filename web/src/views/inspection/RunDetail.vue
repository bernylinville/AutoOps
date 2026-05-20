<template>
  <div class="run-detail" v-loading="pageLoading">
    <!-- 顶部：返回 + 运行信息 -->
    <el-card shadow="hover" style="margin-bottom: 16px">
      <div class="run-header">
        <div class="run-info">
          <el-button text @click="$router.back()" style="margin-right: 12px">
            <el-icon><ArrowLeft /></el-icon>
          </el-button>
          <span class="run-title">{{ run.taskId ? `任务 #${run.taskId}` : '运行详情' }}</span>
          <el-tag size="small" :type="run.triggerType === 'cron' ? '' : 'primary'" style="margin-left: 8px">
            {{ run.triggerType === 'cron' ? '定时触发' : '手动触发' }}
          </el-tag>
          <el-tag :type="statusTagType(run.status)" size="small" style="margin-left: 6px">
            {{ statusText(run.status) }}
          </el-tag>
        </div>
        <div class="run-meta">
          <span>开始时间: {{ run.createdAt || '-' }}</span>
          <span style="margin-left: 16px">耗时: {{ formatDuration(run.durationMs) }}</span>
        </div>
      </div>
    </el-card>

    <!-- 统计卡片 -->
    <el-row :gutter="16" style="margin-bottom: 16px">
      <el-col :span="4.8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-value">{{ stats.total || 0 }}</div>
          <div class="stat-label">总主机数</div>
        </el-card>
      </el-col>
      <el-col :span="4.8">
        <el-card shadow="hover" class="stat-card normal">
          <div class="stat-value">{{ stats.normal || 0 }}</div>
          <div class="stat-label">正常</div>
        </el-card>
      </el-col>
      <el-col :span="4.8">
        <el-card shadow="hover" class="stat-card warning">
          <div class="stat-value">{{ stats.warning || 0 }}</div>
          <div class="stat-label">警告</div>
        </el-card>
      </el-col>
      <el-col :span="4.8">
        <el-card shadow="hover" class="stat-card critical">
          <div class="stat-value">{{ stats.critical || 0 }}</div>
          <div class="stat-label">严重</div>
        </el-card>
      </el-col>
      <el-col :span="4.8">
        <el-card shadow="hover" class="stat-card failed">
          <div class="stat-value">{{ stats.failed || 0 }}</div>
          <div class="stat-label">失败</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 下载按钮 -->
    <div style="margin-bottom: 16px; text-align: right">
      <el-button type="primary" @click="handleDownload">
        <el-icon><Download /></el-icon>
        <span style="margin-left: 4px">下载报告</span>
      </el-button>
    </div>

    <!-- 详情 Tabs -->
    <el-card shadow="hover">
      <el-tabs v-model="activeTab" @tab-change="onTabChange">
        <!-- 主机列表 -->
        <el-tab-pane label="主机列表" name="hosts">
          <el-table :data="results" v-loading="resultsLoading" stripe border>
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="hostname" label="主机名" min-width="140" show-overflow-tooltip />
            <el-table-column prop="ip" label="IP" width="130" />
            <el-table-column prop="os" label="操作系统" min-width="120" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="statusTagType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="指标概要" min-width="150" show-overflow-tooltip>
              <template #default="{ row }">
                <span style="font-size: 12px; color: #888">{{ row.metricSummary || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="alertCount" label="告警数" width="80" align="center">
              <template #default="{ row }">
                <el-tag :type="row.alertCount > 0 ? 'danger' : 'success'" size="small">{{ row.alertCount || 0 }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
          <div class="pagination-wrapper" v-if="resultsTotal > 0">
            <el-pagination
              v-model:current-page="resultsPage"
              v-model:page-size="resultsPageSize"
              :total="resultsTotal"
              :page-sizes="[10, 20, 50]"
              layout="total, sizes, prev, pager, next"
              @size-change="loadResults"
              @current-change="loadResults"
            />
          </div>
        </el-tab-pane>

        <!-- 异常明细 -->
        <el-tab-pane label="异常明细" name="alerts">
          <el-table :data="alerts" v-loading="alertsLoading" stripe border>
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="hostname" label="主机名" min-width="140" show-overflow-tooltip />
            <el-table-column prop="metric" label="指标" min-width="120" show-overflow-tooltip />
            <el-table-column prop="currentValue" label="当前值" width="100" align="center" />
            <el-table-column label="阈值" min-width="140" show-overflow-tooltip>
              <template #default="{ row }">
                <span style="font-size: 12px">{{ formatThreshold(row) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="level" label="级别" width="80" align="center">
              <template #default="{ row }">
                <el-tag :type="levelTagType(row.level)" size="small">{{ levelText(row.level) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="message" label="消息" min-width="160" show-overflow-tooltip />
          </el-table>
          <div class="pagination-wrapper" v-if="alertsTotal > 0">
            <el-pagination
              v-model:current-page="alertsPage"
              v-model:page-size="alertsPageSize"
              :total="alertsTotal"
              :page-sizes="[10, 20, 50]"
              layout="total, sizes, prev, pager, next"
              @size-change="loadAlerts"
              @current-change="loadAlerts"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Download } from '@element-plus/icons-vue'
import inspectionApi from '@/api/inspection'

const route = useRoute()
const runId = parseInt(route.params.id)
if (isNaN(runId)) {
  ElMessage.error('无效的运行 ID')
}

// 页面状态
const pageLoading = ref(false)
const run = ref({})
const stats = ref({ total: 0, normal: 0, warning: 0, critical: 0, failed: 0 })

// Tabs
const activeTab = ref('hosts')

// 主机结果
const results = ref([])
const resultsLoading = ref(false)
const resultsTotal = ref(0)
const resultsPage = ref(1)
const resultsPageSize = ref(20)

// 异常明细
const alerts = ref([])
const alertsLoading = ref(false)
const alertsTotal = ref(0)
const alertsPage = ref(1)
const alertsPageSize = ref(20)

onMounted(async () => {
  pageLoading.value = true
  await loadRun()
  pageLoading.value = false
  loadResults()
})

async function loadRun() {
  try {
    const res = await inspectionApi.getRun(runId)
    if (res.data && res.data.data) {
      const data = res.data.data
      run.value = data
      stats.value = {
        total: data.totalHosts || 0,
        normal: data.normalHosts || 0,
        warning: data.warningHosts || 0,
        critical: data.criticalHosts || 0,
        failed: data.failedHosts || 0
      }
    }
  } catch (e) {
    ElMessage.error('获取运行详情失败')
  }
}

async function loadResults() {
  resultsLoading.value = true
  try {
    const res = await inspectionApi.getRunResults(runId, { page: resultsPage.value, pageSize: resultsPageSize.value })
    if (res.data && res.data.data) {
      results.value = res.data.data.list || []
      resultsTotal.value = res.data.data.total || 0
    }
  } catch (e) {
    ElMessage.error('加载主机列表失败')
  } finally {
    resultsLoading.value = false
  }
}

async function loadAlerts() {
  alertsLoading.value = true
  try {
    const res = await inspectionApi.getRunAlerts(runId, { page: alertsPage.value, pageSize: alertsPageSize.value })
    if (res.data && res.data.data) {
      alerts.value = res.data.data.list || []
      alertsTotal.value = res.data.data.total || 0
    }
  } catch (e) {
    ElMessage.error('加载异常明细失败')
  } finally {
    alertsLoading.value = false
  }
}

function onTabChange(tab) {
  if (tab === 'alerts' && alerts.value.length === 0) loadAlerts()
}

async function handleDownload() {
  try {
    const res = await inspectionApi.downloadReport(runId)
    // Detect JSON error responses wrapped as blobs
    const contentType = res.headers['content-type'] || ''
    if (contentType.includes('application/json') || res.data.type === 'application/json') {
      const text = await res.data.text()
      let errMsg = '下载失败'
      try {
        const json = JSON.parse(text)
        errMsg = json.message || json.msg || errMsg
      } catch (_) { /* not JSON */ }
      ElMessage.error(errMsg)
      return
    }
    const url = window.URL.createObjectURL(new Blob([res.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', `inspection-report-${runId}.xlsx`)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
    ElMessage.success('下载成功')
  } catch (e) {
    ElMessage.error('下载失败')
  }
}

// 工具方法
function statusTagType(status) {
  const map = { pending: 'info', running: 'primary', success: 'success', partial: 'warning', failed: 'danger', normal: 'success' }
  return map[status] || 'info'
}

function statusText(status) {
  const map = { pending: '等待中', running: '运行中', success: '正常', partial: '部分异常', failed: '失败', normal: '正常', warning: '警告', critical: '严重' }
  return map[status] || '未知'
}

function levelTagType(level) {
  const map = { normal: 'success', warning: 'warning', critical: 'danger' }
  return map[level] || 'info'
}

function levelText(level) {
  const map = { normal: '正常', warning: '警告', critical: '严重' }
  return map[level] || '未知'
}

function formatDuration(ms) {
  if (!ms && ms !== 0) return '-'
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const remainSeconds = seconds % 60
  return `${minutes}m${remainSeconds}s`
}

function formatThreshold(row) {
  const parts = []
  if (row.warningThreshold !== undefined && row.warningThreshold !== null) parts.push(`警告: ${row.warningThreshold}`)
  if (row.criticalThreshold !== undefined && row.criticalThreshold !== null) parts.push(`严重: ${row.criticalThreshold}`)
  return parts.length > 0 ? parts.join(' / ') : '-'
}
</script>

<style scoped>
.run-header {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.run-info {
  display: flex;
  align-items: center;
}
.run-title {
  font-size: 18px;
  font-weight: 700;
}
.run-meta {
  color: #888;
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
.stat-card.normal .stat-value { color: #67c23a; }
.stat-card.warning .stat-value { color: #e6a23c; }
.stat-card.critical .stat-value { color: #f56c6c; }
.stat-card.failed .stat-value { color: #909399; }
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
