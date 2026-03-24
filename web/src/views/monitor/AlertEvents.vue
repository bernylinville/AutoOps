<template>
  <div class="alert-events-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>告警事件</span>
          <div class="header-actions">
            <el-select v-model="filterSeverity" placeholder="严重级别" clearable size="small" style="width:120px;margin-right:8px" @change="fetchEvents">
              <el-option label="Critical" value="critical" />
              <el-option label="Warning" value="warning" />
              <el-option label="Info" value="info" />
            </el-select>
            <el-select v-model="filterStatus" placeholder="状态" clearable size="small" style="width:120px;margin-right:8px" @change="fetchEvents">
              <el-option label="告警中" value="firing" />
              <el-option label="已恢复" value="resolved" />
            </el-select>
            <el-button size="small" @click="fetchEvents"><el-icon><Refresh /></el-icon> 刷新</el-button>
          </div>
        </div>
      </template>

      <!-- 统计卡片 -->
      <el-row :gutter="16" style="margin-bottom:16px">
        <el-col :span="6">
          <el-statistic title="总事件" :value="stats.total" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="告警中" :value="stats.firing" value-style="color:#F56C6C" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="已恢复" :value="stats.resolved" value-style="color:#67C23A" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="严重告警" :value="stats.critical" value-style="color:#F56C6C" />
        </el-col>
      </el-row>

      <!-- 事件表格 -->
      <el-table :data="events" v-loading="loading" stripe>
        <el-table-column prop="alertName" label="告警名称" min-width="180" show-overflow-tooltip />
        <el-table-column prop="severity" label="级别" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="severityType(row.severity)" size="small" effect="dark">{{ row.severity }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'firing' ? 'danger' : 'success'" size="small">
              {{ row.status === 'firing' ? '🔴 告警中' : '✅ 已恢复' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="ruleName" label="匹配规则" width="150" show-overflow-tooltip />
        <el-table-column prop="notifyStatus" label="通知状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="notifyStatusType(row.notifyStatus)" size="small">{{ row.notifyStatus }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="startsAt" label="开始时间" width="170" />
        <el-table-column prop="createTime" label="接收时间" width="170" />
      </el-table>

      <el-pagination
        v-model:current-page="page"
        :page-size="20"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="fetchEvents"
        style="margin-top: 16px; justify-content: flex-end;"
      />
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import request from '@/utils/request'

const events = ref([])
const loading = ref(false)
const page = ref(1)
const total = ref(0)
const filterSeverity = ref('')
const filterStatus = ref('')
const stats = reactive({ total: 0, firing: 0, resolved: 0, critical: 0, warning: 0 })

const fetchEvents = async () => {
  loading.value = true
  try {
    const params = { page: page.value, pageSize: 20 }
    if (filterSeverity.value) params.severity = filterSeverity.value
    if (filterStatus.value) params.status = filterStatus.value
    const res = await request({ url: 'n9e/alert/events', method: 'get', params })
    const d = res.data || res
    if (d.code === 200 && d.data) {
      events.value = d.data.list || []
      total.value = d.data.total || 0
    }
  } catch (e) { console.error(e) }
  loading.value = false
}

const fetchStats = async () => {
  try {
    const res = await request({ url: 'n9e/alert/events/stats', method: 'get' })
    const d = res.data || res
    if (d.code === 200 && d.data) Object.assign(stats, d.data)
  } catch (e) { console.error(e) }
}

const severityType = (s) => ({ critical: 'danger', warning: 'warning', info: 'info' }[s] || '')
const notifyStatusType = (s) => ({ sent: 'success', failed: 'danger', pending: 'warning' }[s] || 'info')

onMounted(() => { fetchEvents(); fetchStats() })
</script>

<style scoped>
.alert-events-container { padding: 20px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.header-actions { display: flex; align-items: center; }
</style>
