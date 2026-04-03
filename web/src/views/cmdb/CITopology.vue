<template>
  <div class="topology-page">
    <!-- 控制栏 -->
    <el-card shadow="never" class="control-card">
      <el-row :gutter="12" align="middle">
        <el-col :span="9">
          <el-select
            v-model="selectedCI"
            filterable
            remote
            reserve-keyword
            placeholder="搜索 CI 实例（名称 / 类型）"
            :remote-method="searchInstances"
            :loading="searching"
            value-key="id"
            clearable
            style="width: 100%"
            @clear="clearTopology"
          >
            <el-option
              v-for="item in instanceOptions"
              :key="item.id"
              :label="`${item.typeName}  /  ${item.name}`"
              :value="item"
            >
              <span style="color: #909399; font-size: 12px">{{ item.typeName }}</span>
              <span style="margin-left: 8px">{{ item.name }}</span>
            </el-option>
          </el-select>
        </el-col>

        <el-col :span="8">
          <el-radio-group v-model="direction" size="default">
            <el-radio-button label="all">全链路</el-radio-button>
            <el-radio-button label="down">下游依赖</el-radio-button>
            <el-radio-button label="up">上游来源</el-radio-button>
          </el-radio-group>
        </el-col>

        <el-col :span="4">
          <el-button type="primary" :loading="loading" :disabled="!selectedCI" @click="loadTopology">
            查询拓扑
          </el-button>
          <el-button v-if="topo.nodes.length" plain @click="resetChart">重置视图</el-button>
        </el-col>

        <el-col :span="3" style="text-align: right">
          <span v-if="topo.nodes.length" class="topo-stats">
            {{ topo.nodes.length }} 节点 &nbsp;·&nbsp; {{ topo.edges.length }} 关系
          </span>
        </el-col>
      </el-row>
    </el-card>

    <!-- 图例 -->
    <div v-if="topo.nodes.length" class="legend-bar">
      <span v-for="(color, code) in typeColors" :key="code" class="legend-item">
        <span class="legend-dot" :style="{ background: color }"></span>
        {{ typeLabels[code] || code }}
      </span>
      <span class="legend-item">
        <span class="legend-dot root-dot"></span>根节点
      </span>
    </div>

    <!-- ECharts 容器 -->
    <el-card shadow="never" class="chart-card" v-loading="loading" element-loading-text="正在计算拓扑布局…">
      <div v-if="!topo.nodes.length" class="empty-wrap">
        <el-empty description="请搜索 CI 实例，选择方向后点击「查询拓扑」">
          <template #image>
            <el-icon style="font-size: 64px; color: #C0C4CC"><Share /></el-icon>
          </template>
        </el-empty>
      </div>
      <div v-show="topo.nodes.length" ref="chartRef" class="chart-container"></div>
    </el-card>

    <!-- 节点详情侧边栏 -->
    <el-drawer v-model="drawerVisible" title="节点详情" size="300px" direction="rtl">
      <el-descriptions :column="1" border v-if="activeNode">
        <el-descriptions-item label="名称">{{ activeNode.name }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ activeNode.typeName }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="statusTagType(activeNode.status)" size="small">
            {{ statusLabels[activeNode.status] || activeNode.status }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="是否根节点">
          <el-tag v-if="activeNode.isRoot" type="warning" size="small">根节点</el-tag>
          <span v-else>—</span>
        </el-descriptions-item>
      </el-descriptions>
      <div style="margin-top: 16px; text-align: center">
        <el-button type="primary" plain size="small" @click="setAsRoot">以此节点重新查询</el-button>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import * as echarts from 'echarts'
import * as api from '@/api/ciType'
import { Share } from '@element-plus/icons-vue'

// ——————————————————————————————
// 常量映射
// ——————————————————————————————
const typeColors = {
  server:          '#409EFF',
  database:        '#67C23A',
  network_device:  '#E6A23C',
  middleware:      '#9B59B6',
  storage:         '#F56C6C',
  load_balancer:   '#00BCD4',
}

const typeLabels = {
  server:          '服务器',
  database:        '数据库',
  network_device:  '网络设备',
  middleware:      '中间件',
  storage:         '存储',
  load_balancer:   '负载均衡',
}

const relationLabels = {
  depends_on:   '依赖',
  runs_on:      '运行于',
  connects_to:  '连接',
  contains:     '包含',
}

const statusLabels = { 1: '运行中', 2: '已停机', 3: '维护中', 4: '已下线' }

const statusTagType = (s) => ({ 1: 'success', 2: 'danger', 3: 'warning', 4: 'info' }[s] || '')

// ——————————————————————————————
// 响应式状态
// ——————————————————————————————
const selectedCI      = ref(null)
const direction       = ref('all')
const instanceOptions = ref([])
const searching       = ref(false)
const loading         = ref(false)
const topo            = ref({ nodes: [], edges: [] })

const chartRef        = ref(null)
let   chartInstance   = null

const drawerVisible   = ref(false)
const activeNode      = ref(null)

// ——————————————————————————————
// 搜索 CI 实例（远程）
// ——————————————————————————————
const searchInstances = async (keyword) => {
  if (!keyword) { instanceOptions.value = []; return }
  searching.value = true
  try {
    const res = await api.default.getAllCIInstances(keyword)
    if (res.code === 200) instanceOptions.value = res.data || []
  } finally {
    searching.value = false
  }
}

// ——————————————————————————————
// 查询拓扑
// ——————————————————————————————
const loadTopology = async () => {
  if (!selectedCI.value) return
  loading.value = true
  try {
    const res = await api.default.getCITopology(selectedCI.value.id, direction.value)
    if (res.code === 200) {
      topo.value = res.data
      await nextTick()
      renderChart()
    }
  } finally {
    loading.value = false
  }
}

const clearTopology = () => {
  topo.value = { nodes: [], edges: [] }
  if (chartInstance) chartInstance.clear()
}

// ——————————————————————————————
// ECharts 渲染
// ——————————————————————————————
const buildOption = () => {
  const nodes = topo.value.nodes.map(n => ({
    id:         String(n.id),
    name:       n.name,
    value:      n.id,
    symbolSize: n.isRoot ? 52 : 36,
    itemStyle: {
      color:       typeColors[n.typeCode] || '#909399',
      borderColor: n.isRoot ? '#fff'        : 'transparent',
      borderWidth: n.isRoot ? 3             : 0,
      shadowBlur:  n.isRoot ? 16            : 0,
      shadowColor: n.isRoot ? (typeColors[n.typeCode] || '#909399') : 'transparent',
    },
    label: {
      show:     true,
      fontSize: 12,
      color:    '#303133',
      position: 'bottom',
    },
    tooltip: {
      formatter: `<b>${n.name}</b><br/>类型：${n.typeName}<br/>状态：${statusLabels[n.status] || n.status}`,
    },
    // raw data for click handler
    _raw: n,
  }))

  const edges = topo.value.edges.map(e => ({
    source: String(e.fromCiId),
    target: String(e.toCiId),
    label: {
      show:      true,
      formatter: relationLabels[e.relationType] || e.relationType,
      fontSize:  11,
      color:     '#909399',
    },
    lineStyle: {
      color:     '#C0C4CC',
      curveness: 0.15,
      width:     1.5,
    },
  }))

  return {
    tooltip: { trigger: 'item', enterable: false },
    series: [{
      type:      'graph',
      layout:    'force',
      data:      nodes,
      edges,
      roam:      true,
      draggable: true,
      force: {
        repulsion:       280,
        gravity:         0.08,
        edgeLength:      [120, 220],
        layoutAnimation: true,
      },
      edgeSymbol:     ['none', 'arrow'],
      edgeSymbolSize: [4, 10],
      emphasis: {
        focus:     'adjacency',
        lineStyle: { width: 2.5 },
      },
    }],
  }
}

const renderChart = () => {
  if (!chartRef.value) return

  if (!chartInstance) {
    chartInstance = echarts.init(chartRef.value)
    chartInstance.on('click', ({ dataType, data }) => {
      if (dataType === 'node' && data._raw) {
        activeNode.value = data._raw
        drawerVisible.value = true
      }
    })
    window.addEventListener('resize', handleResize)
  }

  chartInstance.setOption(buildOption(), true)
}

const resetChart = () => {
  if (chartInstance) chartInstance.dispatchAction({ type: 'restore' })
}

const handleResize = () => chartInstance?.resize()

// ——————————————————————————————
// 侧边栏操作
// ——————————————————————————————
const setAsRoot = () => {
  if (!activeNode.value) return
  selectedCI.value = {
    id:       activeNode.value.id,
    name:     activeNode.value.name,
    typeName: activeNode.value.typeName,
  }
  drawerVisible.value = false
  loadTopology()
}

// ——————————————————————————————
// 生命周期
// ——————————————————————————————
onMounted(() => {})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  chartInstance?.dispose()
})
</script>

<style scoped>
.topology-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
}

.control-card :deep(.el-card__body) {
  padding: 12px 16px;
}

.topo-stats {
  font-size: 13px;
  color: #909399;
}

/* 图例 */
.legend-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  padding: 6px 4px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #606266;
}

.legend-dot {
  display: inline-block;
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.root-dot {
  background: #409EFF;
  box-shadow: 0 0 0 3px #fff, 0 0 0 5px #409EFF;
}

/* 图表区域 */
.chart-card {
  flex: 1;
  min-height: 0;
}

.chart-card :deep(.el-card__body) {
  height: 100%;
  padding: 0;
}

.chart-container {
  width: 100%;
  height: 600px;
}

.empty-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 400px;
}
</style>
