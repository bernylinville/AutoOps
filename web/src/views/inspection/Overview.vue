<template>
  <div class="inspection-overview">
    <!-- 统计卡片 -->
    <el-row :gutter="16" style="margin-bottom: 16px">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-value">{{ overview.totalRuns || 0 }}</div>
          <div class="stat-label">今日运行次数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card hosts">
          <div class="stat-value">{{ overview.totalHosts || 0 }}</div>
          <div class="stat-label">已巡检主机</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card alerts">
          <div class="stat-value">{{ overview.totalAlerts || 0 }}</div>
          <div class="stat-label">今日告警</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card normal">
          <div class="stat-value">{{ overview.normalHosts || 0 }}</div>
          <div class="stat-label">正常主机</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 快捷操作 -->
    <el-row :gutter="16" style="margin-bottom: 16px">
      <el-col :span="24">
        <el-card shadow="hover">
          <template #header><span>快捷操作</span></template>
          <div class="action-buttons">
            <el-button type="primary" @click="$router.push('/inspection/tasks')">
              <el-icon><List /></el-icon>
              <span style="margin-left: 4px">任务管理</span>
            </el-button>
            <el-button type="success" @click="$router.push('/inspection/runs')">
              <el-icon><Clock /></el-icon>
              <span style="margin-left: 4px">运行历史</span>
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 最近严重告警 -->
    <el-card shadow="hover">
      <template #header><span>近期严重告警</span></template>
      <el-table :data="recentAlerts" v-loading="loading" stripe border style="width: 100%">
        <el-table-column type="index" label="#" width="50" />
        <el-table-column prop="hostname" label="主机名" min-width="140" show-overflow-tooltip />
        <el-table-column label="指标" min-width="120" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.metricDisplayName || row.metric }}
          </template>
        </el-table-column>
        <el-table-column prop="level" label="级别" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="levelTagType(row.level)" size="small">{{ levelText(row.level) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="告警信息" min-width="200" show-overflow-tooltip />
      </el-table>
      <el-empty v-if="!loading && recentAlerts.length === 0" description="暂无告警" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { List, Clock } from '@element-plus/icons-vue'
import inspectionApi from '@/api/inspection'

const loading = ref(false)
const overview = ref({ totalRuns: 0, totalHosts: 0, totalAlerts: 0, normalHosts: 0 })
const recentAlerts = ref([])

onMounted(() => {
  loadOverview()
})

async function loadOverview() {
  loading.value = true
  try {
    const res = await inspectionApi.getOverview()
    if (res.data && res.data.data) {
      const data = res.data.data
      overview.value = {
        totalRuns: data.stats?.totalRuns || 0,
        totalHosts: data.stats?.totalHosts || 0,
        totalAlerts: data.stats?.totalAlerts || 0,
        normalHosts: data.stats?.normalHosts || 0
      }
      recentAlerts.value = data.recentAlerts || []
    }
  } catch (e) {
    ElMessage.error('加载概览数据失败')
  } finally {
    loading.value = false
  }
}

function levelTagType(level) {
  const map = { normal: 'success', warning: 'warning', critical: 'danger' }
  return map[level] || 'info'
}

function levelText(level) {
  const map = { normal: '正常', warning: '警告', critical: '严重' }
  return map[level] || '未知'
}
</script>

<style scoped>
.stat-card {
  text-align: center;
  padding: 8px 0;
}
.stat-card .stat-value {
  font-size: 36px;
  font-weight: 700;
  color: #409eff;
}
.stat-card.hosts .stat-value { color: #67c23a; }
.stat-card.alerts .stat-value { color: #f56c6c; }
.stat-card.active .stat-value { color: #e6a23c; }
.stat-label {
  font-size: 13px;
  color: #888;
  margin-top: 4px;
}
.action-buttons {
  display: flex;
  align-items: center;
  gap: 12px;
}
</style>
