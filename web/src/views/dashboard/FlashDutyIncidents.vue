<template>
  <el-drawer
    v-model="drawerVisible"
    title="故障处理面板"
    size="50%"
    @close="handleClose"
    :destroy-on-close="true"
  >
    <div class="drawer-header">
      <el-button type="primary" size="small" @click="loadIncidents" :loading="loading">
        <el-icon><Refresh /></el-icon> 刷新
      </el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="incidents"
      stripe
      style="width: 100%"
      empty-text="暂无活动故障"
    >
      <el-table-column label="级别" width="80" align="center">
        <template v-slot="scope">
          <el-tag :type="getSeverityType(scope.row.incident_severity)" size="small">
            {{ getSeverityText(scope.row.incident_severity) }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="标题" prop="title" show-overflow-tooltip min-width="180" />

      <el-table-column label="状态" width="100" align="center">
        <template v-slot="scope">
          <el-tag :type="getProgressType(scope.row.progress)" size="small">
            {{ getProgressText(scope.row.progress) }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="触发时间" width="160">
        <template v-slot="scope">
          {{ formatTime(scope.row.created_at) }}
        </template>
      </el-table-column>

      <el-table-column label="当前处理人" width="120" show-overflow-tooltip>
        <template v-slot="scope">
          {{ scope.row.assignees && scope.row.assignees.length > 0 ? scope.row.assignees.map(a => a.person_name).join(', ') : '暂无' }}
        </template>
      </el-table-column>

      <el-table-column label="操作" width="120" fixed="right" align="center">
        <template v-slot="scope">
          <el-button
            v-if="scope.row.progress === 'Triggered'"
            type="primary"
            size="small"
            plain
            @click="handleClaim(scope.row)"
          >
            认领
          </el-button>
          <el-button
            v-else-if="scope.row.progress === 'Processing'"
            type="success"
            size="small"
            plain
            @click="handleCloseIncident(scope.row)"
          >
            关闭
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 关闭备注对话框 -->
    <el-dialog
      v-model="closeDialogVisible"
      title="填写关闭备注"
      width="400px"
      append-to-body
    >
      <el-input
        v-model="closeDesc"
        type="textarea"
        :rows="3"
        placeholder="选填: 请输入故障处理相关的结案备注"
      />
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="closeDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitClose" :loading="submitting">
            确认关闭
          </el-button>
        </span>
      </template>
    </el-dialog>
  </el-drawer>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import flashdutyApi from '@/api/flashduty'

const props = defineProps({
  visible: { type: Boolean, default: false }
})

const emit = defineEmits(['update:visible', 'close', 'refresh-dashboard'])

const drawerVisible = ref(false)
const loading = ref(false)
const incidents = ref([])

const closeDialogVisible = ref(false)
const closeDesc = ref('')
const currentIncidentId = ref('')
const submitting = ref(false)

watch(() => props.visible, (val) => {
  drawerVisible.value = val
  if (val) {
    loadIncidents()
  }
}, { immediate: true })

watch(drawerVisible, (val) => {
  if (!val) emit('update:visible', false)
})

const handleClose = () => {
  emit('close')
  emit('update:visible', false)
}

const loadIncidents = async () => {
  loading.value = true
  try {
    const res = await flashdutyApi.getActiveIncidents({ limit: 100 })
    if (res.data) {
      incidents.value = res.data.items || []
    }
  } catch (e) {
    console.error('加载活动故障失败:', e)
  } finally {
    loading.value = false
  }
}

// 标签样式处理
const getSeverityType = (severity) => {
  const map = { Critical: 'danger', Warning: 'warning', Info: 'info' }
  return map[severity] || 'info'
}

const getSeverityText = (severity) => {
  const map = { Critical: '严重', Warning: '警告', Info: '信息' }
  return map[severity] || severity
}

const getProgressType = (progress) => {
  const map = { Triggered: 'danger', Processing: 'warning', Closed: 'success' }
  return map[progress] || 'info'
}

const getProgressText = (progress) => {
  const map = { Triggered: '待处理', Processing: '处理中', Closed: '已关闭' }
  return map[progress] || progress
}

const formatTime = (ts) => {
  if (!ts) return '-'
  const d = new Date(ts * 1000)
  return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

// 认领操作
const handleClaim = async (row) => {
  try {
    await ElMessageBox.confirm(`确认认领故障: ${row.title}?`, '认领确认', {
      confirmButtonText: '认领',
      cancelButtonText: '取消',
      type: 'warning'
    })
    
    await flashdutyApi.claimIncident(row.incident_id)
    ElMessage.success('认领成功')
    
    // 刷新数据
    await loadIncidents()
    emit('refresh-dashboard') // 通知大盘刷新数据
  } catch (e) {
    if (e !== 'cancel') {
      console.error(e)
      ElMessage.error('认领失败')
    }
  }
}

// 关闭操作
const handleCloseIncident = (row) => {
  currentIncidentId.value = row.incident_id
  closeDesc.value = ''
  closeDialogVisible.value = true
}

// 提交关闭
const submitClose = async () => {
  if (!currentIncidentId.value) return
  submitting.value = true
  try {
    await flashdutyApi.closeIncident(currentIncidentId.value, closeDesc.value)
    ElMessage.success('关闭成功')
    closeDialogVisible.value = false
    
    // 刷新数据
    await loadIncidents()
    emit('refresh-dashboard') // 通知大盘刷新数据
  } catch (e) {
    console.error(e)
    ElMessage.error('关闭失败')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.drawer-header {
  margin-bottom: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
