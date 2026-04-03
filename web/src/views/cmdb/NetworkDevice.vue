<template>
  <div>
    <!-- 顶部统计 -->
    <el-row :gutter="12" style="margin-bottom: 12px">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-val">{{ total }}</div>
          <div class="stat-lbl">设备总数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card ok">
          <div class="stat-val">{{ reachableCount }}</div>
          <div class="stat-lbl">上次可达</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card fail">
          <div class="stat-val">{{ unreachableCount }}</div>
          <div class="stat-lbl">上次不可达</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card pending">
          <div class="stat-val">{{ neverCheckedCount }}</div>
          <div class="stat-lbl">未巡检</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 设备列表 -->
    <el-card shadow="never">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center">
          <span style="font-weight:600">网络设备列表</span>
          <div style="display:flex; gap:8px">
            <el-input v-model="keyword" placeholder="搜索设备名/型号/IP" clearable
              style="width:220px" @keyup.enter="load(1)" @clear="load(1)">
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
            <el-button type="primary" :loading="batchLoading" @click="batchInspect">批量巡检</el-button>
            <el-button @click="load(1)">刷新</el-button>
          </div>
        </div>
      </template>

      <el-table :data="devices" v-loading="loading" stripe border size="small">
        <el-table-column type="index" width="50" label="#" />
        <el-table-column label="设备名称" prop="name" min-width="150" show-overflow-tooltip />
        <el-table-column label="设备类型" width="110">
          <template #default="{ row }">
            <el-tag size="small" type="warning">{{ attrVal(row, 'device_type') || '—' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="品牌/型号" width="130" show-overflow-tooltip>
          <template #default="{ row }">
            {{ attrVal(row, 'brand') }}{{ attrVal(row, 'model') ? ' / ' + attrVal(row, 'model') : '' }}
          </template>
        </el-table-column>
        <el-table-column label="管理 IP" width="140">
          <template #default="{ row }">
            <el-text type="primary" style="font-family: monospace; font-size:13px">
              {{ attrVal(row, 'mgmt_ip') || '未配置' }}
            </el-text>
          </template>
        </el-table-column>
        <el-table-column label="固件版本" width="120" show-overflow-tooltip>
          <template #default="{ row }">{{ attrVal(row, 'firmware') || '—' }}</template>
        </el-table-column>
        <el-table-column label="CI 状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="巡检状态" width="160" align="center">
          <template #default="{ row }">
            <template v-if="row.lastInspection">
              <el-tag :type="row.lastInspection.reachable ? 'success' : 'danger'" size="small">
                {{ row.lastInspection.reachable ? '可达' : '不可达' }}
              </el-tag>
              <span v-if="row.lastInspection.reachable" style="margin-left:4px; font-size:11px; color:#909399">
                {{ row.lastInspection.latencyMs }}ms / :{{ row.lastInspection.port }}
              </span>
            </template>
            <el-tag v-else type="info" size="small">未巡检</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="上次巡检" width="155" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.lastInspection ? row.lastInspection.createTime : '—' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right" align="center">
          <template #default="{ row }">
            <el-button text type="primary" size="small"
              :loading="inspecting[row.id]" @click="inspect(row)">
              <el-icon><Aim /></el-icon> 巡检
            </el-button>
            <el-button text size="small" @click="openHistory(row)">
              <el-icon><Clock /></el-icon> 历史
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div style="margin-top:12px; display:flex; justify-content:flex-end">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next"
          @size-change="load(1)"
          @current-change="load"
        />
      </div>
    </el-card>

    <!-- 巡检历史抽屉 -->
    <el-drawer v-model="historyVisible" :title="`${historyDevice?.name || ''} — 巡检历史`"
      size="520px" direction="rtl">
      <el-table :data="historyList" v-loading="historyLoading" stripe border size="small">
        <el-table-column label="结果" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.reachable ? 'success' : 'danger'" size="small">
              {{ row.reachable ? '可达' : '不可达' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="管理IP" prop="mgmtIp" width="130" />
        <el-table-column label="延迟" width="70" align="right">
          <template #default="{ row }">
            <span v-if="row.reachable">{{ row.latencyMs }}ms</span>
            <span v-else style="color:#F56C6C">—</span>
          </template>
        </el-table-column>
        <el-table-column label="端口" prop="port" width="55" align="center" />
        <el-table-column label="错误" prop="errorMsg" min-width="100" show-overflow-tooltip />
        <el-table-column label="操作人" prop="operator" width="80" />
        <el-table-column label="时间" prop="createTime" min-width="155" />
      </el-table>
      <div style="margin-top:12px; display:flex; justify-content:flex-end">
        <el-pagination
          v-model:current-page="historyPage"
          :page-size="historyPageSize"
          :total="historyTotal"
          layout="total, prev, pager, next"
          @current-change="loadHistory"
        />
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Aim, Clock } from '@element-plus/icons-vue'
import api from '@/api/networkDevice'

// ——— 列表状态 ———
const devices     = ref([])
const total       = ref(0)
const page        = ref(1)
const pageSize    = ref(20)
const keyword     = ref('')
const loading     = ref(false)
const inspecting  = reactive({})   // { [deviceId]: bool }
const batchLoading = ref(false)

// ——— 统计 ———
const reachableCount   = computed(() => devices.value.filter(d => d.lastInspection?.reachable).length)
const unreachableCount = computed(() => devices.value.filter(d => d.lastInspection && !d.lastInspection.reachable).length)
const neverCheckedCount = computed(() => devices.value.filter(d => !d.lastInspection).length)

// ——— 历史抽屉 ———
const historyVisible  = ref(false)
const historyDevice   = ref(null)
const historyList     = ref([])
const historyTotal    = ref(0)
const historyPage     = ref(1)
const historyPageSize = 20
const historyLoading  = ref(false)

// ——— 辅助 ———
const attrVal = (row, code) => {
  if (!row.attributes) return ''
  const v = row.attributes[code]
  return (v !== undefined && v !== null && v !== '') ? String(v) : ''
}

const statusText    = s => ({ 1: '运行中', 2: '已停机', 3: '维护中', 4: '已下线' }[s] || '未知')
const statusTagType = s => ({ 1: 'success', 2: 'danger', 3: 'warning', 4: 'info' }[s] || 'info')

// ——— 数据加载 ———
const load = async (p) => {
  if (p) page.value = p
  loading.value = true
  try {
    const res = await api.getNetworkDevices({ page: page.value, pageSize: pageSize.value, keyword: keyword.value })
    if (res.code === 200) {
      devices.value = res.data?.list  || []
      total.value   = res.data?.total || 0
    }
  } finally {
    loading.value = false
  }
}

// ——— 单设备巡检 ———
const inspect = async (row) => {
  inspecting[row.id] = true
  try {
    const res = await api.inspectDevice(row.id)
    if (res.code === 200) {
      const r = res.data
      row.lastInspection = r
      if (r.reachable) {
        ElMessage.success(`${row.name} 可达，延迟 ${r.latencyMs}ms（:${r.port}）`)
      } else {
        ElMessage.warning(`${row.name} 不可达`)
      }
    }
  } catch {
    ElMessage.error('巡检请求失败')
  } finally {
    inspecting[row.id] = false
  }
}

// ——— 批量巡检 ———
const batchInspect = async () => {
  if (!devices.value.length) return
  batchLoading.value = true
  const targets = devices.value.filter(d => attrVal(d, 'mgmt_ip'))
  let ok = 0, fail = 0
  for (const row of targets) {
    try {
      const res = await api.inspectDevice(row.id)
      if (res.code === 200) {
        row.lastInspection = res.data
        res.data.reachable ? ok++ : fail++
      }
    } catch { fail++ }
  }
  batchLoading.value = false
  ElMessage.info(`批量巡检完成：${ok} 台可达，${fail} 台不可达`)
}

// ——— 历史记录 ———
const openHistory = async (row) => {
  historyDevice.value = row
  historyPage.value   = 1
  historyVisible.value = true
  await loadHistory(1)
}

const loadHistory = async (p) => {
  if (p) historyPage.value = p
  historyLoading.value = true
  try {
    const res = await api.getInspectionHistory(historyDevice.value.id, historyPage.value, historyPageSize)
    if (res.code === 200) {
      historyList.value  = res.data?.list  || []
      historyTotal.value = res.data?.total || 0
    }
  } finally {
    historyLoading.value = false
  }
}

onMounted(() => load())
</script>

<style scoped>
.stat-card { text-align: center; }
.stat-card :deep(.el-card__body) { padding: 14px 8px; }
.stat-val { font-size: 28px; font-weight: 700; color: #303133; }
.stat-lbl { font-size: 12px; color: #909399; margin-top: 4px; }
.stat-card.ok  .stat-val { color: #67C23A; }
.stat-card.fail .stat-val { color: #F56C6C; }
.stat-card.pending .stat-val { color: #909399; }
</style>
