<template>
  <div class="sync-log-container">
    <div class="page-header">
      <h3>同步日志</h3>
      <p class="page-desc">N9E 数据同步记录查询</p>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <el-row :gutter="16" align="middle">
        <el-col :span="4">
          <el-select v-model="filters.status" placeholder="同步状态" clearable @change="loadLogs" style="width: 100%">
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-col>
        <el-col :span="4">
          <el-select v-model="filters.triggerBy" placeholder="触发方式" clearable @change="loadLogs" style="width: 100%">
            <el-option label="手动" value="manual" />
            <el-option label="定时" value="cron" />
          </el-select>
        </el-col>
        <el-col :span="3">
          <el-select v-model="filters.limit" @change="loadLogs" style="width: 100%">
            <el-option :label="'最近 20 条'" :value="20" />
            <el-option :label="'最近 50 条'" :value="50" />
            <el-option :label="'最近 100 条'" :value="100" />
          </el-select>
        </el-col>
        <el-col :span="13" style="text-align: right;">
          <el-button type="primary" :icon="Refresh" @click="loadLogs" :loading="loading">刷新</el-button>
        </el-col>
      </el-row>
    </div>

    <!-- 日志表格 -->
    <el-card shadow="hover" class="table-card">
      <el-table :data="filteredLogs" stripe v-loading="loading" border style="width: 100%"
        :row-class-name="tableRowClassName">
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="expand-content" v-if="row.parsedResult">
              <el-descriptions :column="3" size="small" border>
                <el-descriptions-item label="业务组新增">{{ row.parsedResult.busiGroups?.created || 0 }}</el-descriptions-item>
                <el-descriptions-item label="业务组更新">{{ row.parsedResult.busiGroups?.updated || 0 }}</el-descriptions-item>
                <el-descriptions-item label="业务组跳过">{{ row.parsedResult.busiGroups?.skipped || 0 }}</el-descriptions-item>
                <el-descriptions-item label="主机新增">{{ row.parsedResult.hosts?.created || 0 }}</el-descriptions-item>
                <el-descriptions-item label="主机更新">{{ row.parsedResult.hosts?.updated || 0 }}</el-descriptions-item>
                <el-descriptions-item label="主机跳过">{{ row.parsedResult.hosts?.skipped || 0 }}</el-descriptions-item>
                <el-descriptions-item label="主机失联">{{ row.parsedResult.hosts?.stale || 0 }}</el-descriptions-item>
                <el-descriptions-item label="数据源新增">{{ row.parsedResult.datasources?.created || 0 }}</el-descriptions-item>
                <el-descriptions-item label="数据源更新">{{ row.parsedResult.datasources?.updated || 0 }}</el-descriptions-item>
              </el-descriptions>
            </div>
            <div v-else-if="row.errorMsg" class="expand-error">
              <el-alert :title="row.errorMsg" type="error" :closable="false" show-icon />
            </div>
            <div v-else class="expand-empty">
              <el-empty description="无详细信息" :image-size="40" />
            </div>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column prop="syncType" label="类型" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small">{{ row.syncType === 'full' ? '全量' : row.syncType }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="100" align="center">
          <template #default="{ row }">
            <span v-if="row.durationMs">{{ (row.durationMs / 1000).toFixed(1) }}s</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="触发方式" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.triggerBy === 'cron' ? 'warning' : 'primary'" size="small" effect="plain">
              {{ row.triggerBy === 'cron' ? '定时' : '手动' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="同步摘要" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.parsedResult">
              业务组 +{{ row.parsedResult.busiGroups?.created || 0 }}
              主机 +{{ row.parsedResult.hosts?.created || 0 }}/↑{{ row.parsedResult.hosts?.updated || 0 }}
              <span v-if="row.parsedResult.hosts?.stale" style="color: #909399;">
                失联{{ row.parsedResult.hosts.stale }}
              </span>
            </span>
            <span v-else-if="row.errorMsg" style="color: #f56c6c;">{{ row.errorMsg }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import n9eApi from '@/api/n9e'

const loading = ref(false)
const logs = ref([])

const filters = reactive({
  status: null,
  triggerBy: null,
  limit: 50
})

const filteredLogs = computed(() => {
  let list = logs.value
  if (filters.status) {
    list = list.filter(l => l.status === filters.status)
  }
  if (filters.triggerBy) {
    list = list.filter(l => l.triggerBy === filters.triggerBy)
  }
  return list
})

const loadLogs = async () => {
  loading.value = true
  try {
    const res = await n9eApi.getSyncLogs(filters.limit)
    if (res.data?.code === 200) {
      logs.value = (res.data.data || []).map(log => ({
        ...log,
        parsedResult: parseResult(log.result)
      }))
    }
  } catch (err) {
    console.error('Failed to load sync logs:', err)
  } finally {
    loading.value = false
  }
}

const parseResult = (resultStr) => {
  if (!resultStr) return null
  try {
    return typeof resultStr === 'string' ? JSON.parse(resultStr) : resultStr
  } catch {
    return null
  }
}

const formatTime = (timeStr) => {
  if (!timeStr) return '-'
  const d = new Date(timeStr)
  if (isNaN(d.getTime())) return timeStr
  return d.toLocaleString('zh-CN', { hour12: false })
}

const tableRowClassName = ({ row }) => {
  return row.status === 'failed' ? 'row-failed' : ''
}

onMounted(() => {
  loadLogs()
})
</script>

<style scoped>
.sync-log-container { padding: 20px; }
.page-header { margin-bottom: 20px; }
.page-header h3 { font-size: 18px; font-weight: 600; margin: 0 0 8px 0; color: #303133; }
.page-desc { color: #909399; font-size: 13px; margin: 0; }

.filter-bar {
  margin-bottom: 16px; padding: 16px;
  background: #fff; border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
}

.table-card { border-radius: 8px; }

.expand-content { padding: 12px 20px; }
.expand-error { padding: 12px 20px; }
.expand-empty { padding: 8px 20px; }

:deep(.row-failed) { background-color: #fef0f0 !important; }
:deep(.row-failed:hover > td) { background-color: #fde2e2 !important; }
</style>
