<template>
  <div class="inspection-runs">
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span class="title">运行历史</span>
        </div>
      </template>

      <!-- 筛选栏 -->
      <div class="filter-bar">
        <el-input-number
          v-model="taskFilter"
          placeholder="任务 ID"
          :min="1"
          :controls="false"
          clearable
          style="width: 160px"
          @change="loadRuns"
        />
        <el-select v-model="statusFilter" placeholder="运行状态" clearable style="width: 140px; margin-left: 8px" @change="loadRuns">
          <el-option value="pending" label="等待中" />
          <el-option value="running" label="运行中" />
          <el-option value="success" label="正常" />
          <el-option value="partial" label="部分异常" />
          <el-option value="failed" label="失败" />
        </el-select>
        <el-select v-model="triggerTypeFilter" placeholder="触发类型" clearable style="width: 140px; margin-left: 8px" @change="loadRuns">
          <el-option value="cron" label="定时触发" />
          <el-option value="manual" label="手动触发" />
        </el-select>
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          format="YYYY-MM-DD"
          value-format="YYYY-MM-DD"
          style="margin-left: 8px"
          @change="loadRuns"
        />
        <el-button type="primary" @click="loadRuns" style="margin-left: 8px">查询</el-button>
      </div>

      <!-- 运行列表 -->
      <el-table
        :data="runs"
        v-loading="loading"
        stripe
        border
        highlight-current-row
        style="width: 100%; margin-top: 16px"
        @row-click="goDetail"
      >
        <el-table-column type="index" label="#" width="50" />
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="taskId" label="任务 ID" width="100" align="center" />
        <el-table-column prop="triggerType" label="触发类型" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.triggerType === 'cron' ? '' : 'primary'">
              {{ row.triggerType === 'cron' ? '定时' : '手动' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="主机统计" width="240" align="center">
          <template #default="{ row }">
            <div class="host-stats">
              <el-tag size="small" type="info">总 {{ row.totalHosts || 0 }}</el-tag>
              <el-tag size="small" type="success">正常 {{ row.normalHosts || 0 }}</el-tag>
              <el-tag size="small" type="warning">警告 {{ row.warningHosts || 0 }}</el-tag>
              <el-tag size="small" type="danger">严重 {{ row.criticalHosts || 0 }}</el-tag>
              <el-tag size="small" type="info">失败 {{ row.failedHosts || 0 }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="durationMs" label="耗时" width="100" align="center">
          <template #default="{ row }">
            <span>{{ formatDuration(row.durationMs) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="时间" width="170" show-overflow-tooltip />
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper" v-if="total > 0">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadRuns"
          @current-change="loadRuns"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import inspectionApi from '@/api/inspection'

const router = useRouter()

// 列表状态
const runs = ref([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

// 筛选状态
const taskFilter = ref(null)
const statusFilter = ref('')
const triggerTypeFilter = ref('')
const dateRange = ref([])

onMounted(() => {
  loadRuns()
})

async function loadRuns() {
  loading.value = true
  try {
    const params = { page: page.value, pageSize: pageSize.value }
    if (taskFilter.value) params.taskId = taskFilter.value
    if (statusFilter.value) params.status = statusFilter.value
    if (triggerTypeFilter.value) params.triggerType = triggerTypeFilter.value
    if (dateRange.value && dateRange.value.length === 2) {
      params.dateFrom = dateRange.value[0]
      params.dateTo = dateRange.value[1]
    }
    const res = await inspectionApi.getRuns(params)
    if (res.data && res.data.data) {
      runs.value = res.data.data.list || []
      total.value = res.data.data.total || 0
    }
  } catch (e) {
    ElMessage.error('加载运行历史失败')
  } finally {
    loading.value = false
  }
}

function goDetail(row) {
  router.push(`/inspection/runs/${row.id}`)
}

function statusTagType(status) {
  const map = { pending: 'info', running: 'primary', success: 'success', partial: 'warning', failed: 'danger' }
  return map[status] || 'info'
}

function statusText(status) {
  const map = { pending: '等待中', running: '运行中', success: '正常', partial: '部分异常', failed: '失败' }
  return map[status] || '未知'
}

function formatDuration(ms) {
  if (!ms && ms !== 0) return '-'
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const remainSeconds = seconds % 60
  return `${minutes}m${remainSeconds}s`
}
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.title {
  font-size: 16px;
  font-weight: 600;
}
.filter-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.host-stats {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
  justify-content: center;
}
.pagination-wrapper {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
