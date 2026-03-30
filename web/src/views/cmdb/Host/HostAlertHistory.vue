<template>
  <el-dialog
    title="主机告警历史"
    v-model="dialogVisible"
    width="70%"
    top="5vh"
    @close="handleClose"
    :append-to-body="true"
  >
    <div class="alert-header">
      <div class="alert-host-info">
        <el-tag type="primary" size="large">{{ hostIdent }}</el-tag>
        <span class="alert-host-name" v-if="hostName">{{ hostName }}</span>
      </div>
      <el-button size="small" @click="loadAlerts" :loading="loading">
        <el-icon><Refresh /></el-icon>刷新
      </el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="alerts"
      stripe
      style="width: 100%"
      class="alert-table"
      empty-text="暂无告警记录"
    >
      <el-table-column label="严重级别" width="100" align="center">
        <template v-slot="scope">
          <el-tag :type="getSeverityType(scope.row.alert_severity)" size="small">
            {{ getSeverityText(scope.row.alert_severity) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="告警标题" prop="title" show-overflow-tooltip min-width="200" />
      <el-table-column label="状态" width="90" align="center">
        <template v-slot="scope">
          <el-tag :type="scope.row.end_time ? 'success' : 'danger'" size="small">
            {{ scope.row.end_time ? '已恢复' : '活跃' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="协作空间" prop="channel_name" width="140" show-overflow-tooltip />
      <el-table-column label="触发时间" width="170">
        <template v-slot="scope">
          {{ formatTime(scope.row.start_time) }}
        </template>
      </el-table-column>
      <el-table-column label="持续时长" width="120">
        <template v-slot="scope">
          {{ calcDuration(scope.row.start_time, scope.row.end_time) }}
        </template>
      </el-table-column>
      <el-table-column label="事件数" prop="event_cnt" width="80" align="center" />
      <el-table-column label="关联故障" width="100" align="center">
        <template v-slot="scope">
          <el-tag v-if="scope.row.incident" type="warning" size="small">
            {{ scope.row.incident.progress === 'Closed' ? '已关闭' : '处理中' }}
          </el-tag>
          <span v-else class="text-muted">-</span>
        </template>
      </el-table-column>
    </el-table>

    <div class="alert-footer" v-if="total > 0">
      <span class="alert-total">共 {{ total }} 条告警</span>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import flashdutyApi from '@/api/flashduty'

const props = defineProps({
  visible: { type: Boolean, default: false },
  hostIdent: { type: String, required: true },
  hostName: { type: String, default: '' }
})

const emit = defineEmits(['update:visible', 'close'])

const dialogVisible = ref(false)
const loading = ref(false)
const alerts = ref([])
const total = ref(0)

watch(() => props.visible, (val) => {
  dialogVisible.value = val
  if (val && props.hostIdent) {
    loadAlerts()
  }
}, { immediate: true })

watch(dialogVisible, (val) => {
  if (!val) emit('update:visible', false)
})

const handleClose = () => {
  emit('close')
  emit('update:visible', false)
}

const loadAlerts = async () => {
  loading.value = true
  try {
    const res = await flashdutyApi.getAlertsByHost(props.hostIdent, { limit: 50 })
    if (res.data) {
      alerts.value = res.data.items || []
      total.value = res.data.total || 0
    }
  } catch (e) {
    console.error('加载主机告警失败:', e)
  } finally {
    loading.value = false
  }
}

const getSeverityType = (severity) => {
  const map = { Critical: 'danger', Warning: 'warning', Info: 'info' }
  return map[severity] || 'info'
}

const getSeverityText = (severity) => {
  const map = { Critical: '严重', Warning: '警告', Info: '信息' }
  return map[severity] || severity
}

const formatTime = (ts) => {
  if (!ts) return '-'
  const d = new Date(ts * 1000)
  return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

const calcDuration = (start, end) => {
  if (!start) return '-'
  const endTs = end || Math.floor(Date.now() / 1000)
  const diff = endTs - start
  if (diff < 60) return diff + '秒'
  if (diff < 3600) return Math.round(diff / 60) + '分钟'
  if (diff < 86400) return (diff / 3600).toFixed(1) + '小时'
  return (diff / 86400).toFixed(1) + '天'
}
</script>

<style scoped>
.alert-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.alert-host-info {
  display: flex;
  align-items: center;
  gap: 10px;
}
.alert-host-name {
  font-size: 14px;
  color: var(--ao-text-secondary);
}
.alert-table {
  border-radius: var(--ao-radius);
}
.alert-footer {
  margin-top: 12px;
  text-align: right;
}
.alert-total {
  font-size: 13px;
  color: var(--ao-text-secondary);
}
.text-muted {
  color: var(--ao-text-secondary);
}
</style>
