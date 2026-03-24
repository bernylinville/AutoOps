<template>
  <div class="n9e-monitor-container">
    <!-- 顶部筛选栏 -->
    <div class="filter-bar">
      <el-row :gutter="16" align="middle">
        <el-col :span="6">
          <el-select v-model="filters.groupId" placeholder="选择业务组" clearable @change="handleFilterChange" style="width: 100%">
            <el-option v-for="group in busiGroups" :key="group.id" :label="`${group.name} (${group.hostCount || 0})`" :value="group.id" />
          </el-select>
        </el-col>
        <el-col :span="5">
          <el-input v-model="filters.keyword" placeholder="搜索主机名/IP" clearable @clear="handleFilterChange" @keyup.enter="handleFilterChange">
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
        </el-col>
        <el-col :span="4">
          <el-select v-model="filters.status" placeholder="在线状态" clearable @change="handleFilterChange" style="width: 100%">
            <el-option label="在线" :value="1" />
            <el-option label="离线" :value="3" />
            <el-option label="失联" :value="4" />
          </el-select>
        </el-col>
        <el-col :span="9" style="text-align: right;">
          <el-button type="primary" :icon="Refresh" @click="handleRefresh" :loading="loading">刷新</el-button>
          <el-button @click="handleSync" :loading="syncing" :icon="Download">从 N9E 同步</el-button>
        </el-col>
      </el-row>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stats-row">
      <el-col :span="5">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon total"><el-icon :size="28"><Monitor /></el-icon></div>
            <div class="stat-info">
              <div class="stat-number">{{ pagination.total }}</div>
              <div class="stat-label">N9E 主机</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="5">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon online"><el-icon :size="28"><CircleCheckFilled /></el-icon></div>
            <div class="stat-info">
              <div class="stat-number">{{ stats.online }}</div>
              <div class="stat-label">在线</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="5">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon offline"><el-icon :size="28"><CircleCloseFilled /></el-icon></div>
            <div class="stat-info">
              <div class="stat-number">{{ stats.offline }}</div>
              <div class="stat-label">离线</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="5">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon stale"><el-icon :size="28"><WarningFilled /></el-icon></div>
            <div class="stat-info">
              <div class="stat-number">{{ stats.stale }}</div>
              <div class="stat-label">失联</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon groups"><el-icon :size="28"><FolderOpened /></el-icon></div>
            <div class="stat-info">
              <div class="stat-number">{{ busiGroups.length }}</div>
              <div class="stat-label">业务组</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 主机列表表格 -->
    <el-card shadow="hover" class="table-card">
      <el-table :data="targets" stripe v-loading="loading" border style="width: 100%">
        <el-table-column prop="n9eIdent" label="标识 (Ident)" min-width="150" show-overflow-tooltip />
        <el-table-column prop="hostName" label="主机名" min-width="140" show-overflow-tooltip />
        <el-table-column label="IP 地址" min-width="130">
          <template #default="{ row }">
            {{ row.publicIp || row.privateIp || row.sshIp || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="os" label="操作系统" min-width="120" show-overflow-tooltip />
        <el-table-column prop="cpu" label="CPU" min-width="100" show-overflow-tooltip />
        <el-table-column prop="memory" label="内存" min-width="80" />
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.status === 4" type="info" size="small" effect="plain">失联</el-tag>
            <el-tag v-else-if="row.status === 1" type="success" size="small">在线</el-tag>
            <el-tag v-else type="danger" size="small">离线</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="groupName" label="分组" min-width="100" show-overflow-tooltip />
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[20, 50, 100, 200]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadTargets"
          @current-change="loadTargets"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Refresh, Download, Monitor, CircleCheckFilled, CircleCloseFilled, WarningFilled, FolderOpened } from '@element-plus/icons-vue'
import n9eApi from '@/api/n9e'
import cmdbApi from '@/api/cmdb'
import { useRouter } from 'vue-router'

const router = useRouter()
const loading = ref(false)
const syncing = ref(false)
const busiGroups = ref([])
const targets = ref([])

const filters = reactive({
  groupId: null,
  keyword: '',
  status: null
})

const pagination = reactive({
  page: 1,
  pageSize: 50,
  total: 0
})

const stats = reactive({ online: 0, offline: 0, stale: 0 })

// 加载统计 + 业务组（从 overview API 一次性获取）
const loadStats = async () => {
  try {
    const res = await n9eApi.getOverview()
    if (res.data?.code === 200 && res.data.data) {
      const d = res.data.data
      // 主机统计
      if (d.hosts) {
        stats.online = d.hosts.online || 0
        stats.offline = (d.hosts.total || 0) - (d.hosts.online || 0) - (d.hosts.stale || 0)
        stats.stale = d.hosts.stale || 0
      }
      // 业务组含主机计数
      if (d.busiGroupStats) {
        busiGroups.value = d.busiGroupStats
      }
    }
  } catch (err) {
    console.error('Failed to load stats:', err)
  }
}

// 跳转到主机列表（按业务组筛选）
const handleGroupNavigate = (groupId, groupName) => {
  router.push({ path: '/cmdb/host', query: { sourceType: 'n9e', groupName: groupName } })
}

// 加载主机列表（后端分页 + sourceType=n9e 过滤）
const loadTargets = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      pageSize: pagination.pageSize,
      sourceType: 'n9e'
    }
    if (filters.keyword) params.keyword = filters.keyword
    if (filters.groupId) params.groupId = filters.groupId
    if (filters.status !== null && filters.status !== '') params.status = filters.status

    const res = await cmdbApi.getCmdbHostList(params)
    if (res.data?.code === 200) {
      const data = res.data.data
      targets.value = data?.list || []
      pagination.total = data?.total || 0
    }
  } catch (err) {
    console.error('Failed to load targets:', err)
  } finally {
    loading.value = false
  }
}

const handleFilterChange = () => {
  pagination.page = 1
  loadTargets()
}

const handleRefresh = () => {
  loadStats()
  loadTargets()
}

const handleSync = async () => {
  syncing.value = true
  try {
    const res = await n9eApi.triggerSync()
    if (res.data?.code === 200) {
      const data = res.data.data
      ElMessage.success(`同步完成：业务组 ${data.busiGroups?.created || 0} 新增，主机 ${data.hosts?.created || 0} 新增/${data.hosts?.updated || 0} 更新`)
      handleRefresh()
    } else {
      ElMessage.error(res.data?.message || '同步失败')
    }
  } catch (err) {
    ElMessage.error('同步失败: ' + (err.message || '未知错误'))
  } finally {
    syncing.value = false
  }
}

onMounted(() => {
  loadStats()
  loadTargets()
})
</script>

<style scoped>
.n9e-monitor-container { padding: 20px; }

.filter-bar {
  margin-bottom: 16px;
  padding: 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
}

.stats-row { margin-bottom: 16px; }
.stat-card { border-radius: 8px; }
.stat-content { display: flex; align-items: center; gap: 16px; }
.stat-icon {
  width: 56px; height: 56px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center; color: #fff;
}
.stat-icon.total { background: linear-gradient(135deg, #409eff, #337ecc); }
.stat-icon.online { background: linear-gradient(135deg, #67c23a, #529b2e); }
.stat-icon.offline { background: linear-gradient(135deg, #f56c6c, #c45656); }
.stat-icon.stale { background: linear-gradient(135deg, #909399, #6b6e75); }
.stat-icon.groups { background: linear-gradient(135deg, #e6a23c, #b88230); }
.stat-number { font-size: 28px; font-weight: 700; color: #303133; line-height: 1; }
.stat-label { font-size: 13px; color: #909399; margin-top: 4px; }

.table-card { border-radius: 8px; }
.pagination-wrapper {
  display: flex; justify-content: flex-end;
  align-items: center; padding: 16px 0 0;
}
</style>
