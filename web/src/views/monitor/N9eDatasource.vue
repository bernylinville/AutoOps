<template>
  <div class="n9e-datasource-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <h3>数据源管理</h3>
      <p class="page-desc">管理 N9E 同步的数据源（Prometheus / VictoriaMetrics），支持 PromQL 查询。</p>
    </div>

    <el-row :gutter="20">
      <!-- 左侧：数据源列表 -->
      <el-col :span="10">
        <el-card shadow="hover" class="ds-card">
          <template #header>
            <div class="card-header">
              <span><el-icon><Coin /></el-icon> 数据源列表</span>
              <div>
                <el-button size="small" @click="handleCheckAll" :loading="checkingAll" :icon="Connection">
                  全量检测
                </el-button>
                <el-button type="primary" size="small" @click="handleSyncDatasources" :loading="syncing" :icon="Refresh">
                  同步数据源
                </el-button>
              </div>
            </div>
          </template>

          <el-table :data="datasources" stripe v-loading="loading" border style="width: 100%"
                    highlight-current-row @current-change="handleDatasourceSelect">
            <el-table-column prop="name" label="名称" min-width="120" show-overflow-tooltip />
            <el-table-column prop="pluginType" label="类型" width="140">
              <template #default="{ row }">
                <el-tag v-if="row.pluginType === 'prometheus'" type="warning" size="small">Prometheus</el-tag>
                <el-tag v-else-if="row.pluginType === 'victoriametrics'" type="primary" size="small">VictoriaMetrics</el-tag>
                <el-tag v-else type="info" size="small">{{ row.pluginType || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="80" align="center">
              <template #default="{ row }">
                <el-tag :type="row.status === 'enabled' ? 'success' : 'danger'" size="small">
                  {{ row.status === 'enabled' ? '启用' : row.status || '未知' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="连通性" width="120" align="center">
              <template #default="{ row }">
                <el-tag v-if="row.checkResult === 'ok'" type="success" size="small">{{ row.checkLatency }}ms</el-tag>
                <el-tag v-else-if="row.checkResult === 'error'" type="danger" size="small">不可达</el-tag>
                <span v-else style="color: #909399; font-size: 12px;">未检测</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" align="center">
              <template #default="{ row }">
                <el-button size="small" type="primary" text :loading="row.checking" @click.stop="handleCheckDatasource(row)">
                  {{ row.checking ? '检测中' : '测试' }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <div v-if="datasources.length === 0 && !loading" class="empty-tip">
            <el-empty description="暂无数据源，请先同步" :image-size="60" />
          </div>
        </el-card>
      </el-col>

      <!-- 右侧：PromQL 查询 -->
      <el-col :span="14">
        <el-card shadow="hover" class="query-card">
          <template #header>
            <div class="card-header">
              <span><el-icon><DataAnalysis /></el-icon> PromQL 查询</span>
            </div>
          </template>

          <el-form label-width="90px">
            <el-form-item label="数据源">
              <el-select v-model="queryForm.datasourceId" placeholder="选择数据源" style="width: 100%">
                <el-option v-for="ds in datasources" :key="ds.id" :label="ds.name + ' (' + ds.pluginType + ')'" :value="ds.id" />
              </el-select>
            </el-form-item>

            <el-form-item label="PromQL">
              <el-input v-model="queryForm.query" type="textarea" :rows="3"
                        placeholder='例: up{job="node"} 或 node_cpu_seconds_total' />
            </el-form-item>

            <el-row :gutter="12">
              <el-col :span="8">
                <el-form-item label="时间范围">
                  <el-select v-model="queryForm.timeRange" @change="handleTimeRangeChange" style="width: 100%">
                    <el-option label="最近 15 分钟" value="15m" />
                    <el-option label="最近 1 小时" value="1h" />
                    <el-option label="最近 3 小时" value="3h" />
                    <el-option label="最近 6 小时" value="6h" />
                    <el-option label="最近 12 小时" value="12h" />
                    <el-option label="最近 24 小时" value="24h" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="步长">
                  <el-input v-model="queryForm.step" placeholder="15s" />
                </el-form-item>
              </el-col>
              <el-col :span="8" style="text-align: right; padding-top: 30px;">
                <el-button type="primary" @click="handleQuery" :loading="querying" :icon="Search">
                  执行查询
                </el-button>
              </el-col>
            </el-row>
          </el-form>

          <!-- 查询结果 -->
          <div v-if="queryResult" class="query-result">
            <el-divider content-position="left">查询结果</el-divider>

            <!-- 图表展示 -->
            <div ref="chartRef" class="chart-container" v-show="chartData.length > 0"></div>

            <!-- 原始数据 -->
            <div v-if="queryResult.data?.result?.length > 0" class="raw-data">
              <el-collapse>
                <el-collapse-item title="原始数据" name="raw">
                  <el-table :data="queryResult.data.result" stripe border size="small" max-height="300">
                    <el-table-column label="指标" min-width="200">
                      <template #default="{ row }">
                        <code>{{ JSON.stringify(row.metric) }}</code>
                      </template>
                    </el-table-column>
                    <el-table-column label="值" width="120">
                      <template #default="{ row }">
                        {{ row.value ? row.value[1] : (row.values ? row.values.length + ' points' : '-') }}
                      </template>
                    </el-table-column>
                  </el-table>
                </el-collapse-item>
              </el-collapse>
            </div>
          </div>

          <el-empty v-else-if="!querying" description="输入 PromQL 表达式并执行查询" :image-size="80" />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Coin, Refresh, DataAnalysis, Search, Connection } from '@element-plus/icons-vue'
import n9eApi from '@/api/n9e'
import * as echarts from 'echarts'

const loading = ref(false)
const syncing = ref(false)
const querying = ref(false)
const checkingAll = ref(false)
const datasources = ref([])
const queryResult = ref(null)
const chartRef = ref(null)
const chartData = ref([])
let chartInstance = null

const queryForm = reactive({
  datasourceId: null,
  query: '',
  timeRange: '1h',
  step: '15s',
  start: '',
  end: ''
})

// 加载数据源
const loadDatasources = async () => {
  loading.value = true
  try {
    const res = await n9eApi.getDatasources()
    if (res.data?.code === 200) {
      datasources.value = (res.data.data || []).map(d => ({
        ...d,
        checking: false,
        checkResult: null,
        checkLatency: null
      }))
      if (datasources.value.length > 0 && !queryForm.datasourceId) {
        queryForm.datasourceId = datasources.value[0].id
      }
    }
  } catch (err) {
    console.error('Failed to load datasources:', err)
  } finally {
    loading.value = false
  }
}

// 同步数据源
const handleSyncDatasources = async () => {
  syncing.value = true
  try {
    const res = await n9eApi.triggerSync()
    if (res.data?.code === 200) {
      ElMessage.success('同步完成')
      loadDatasources()
    } else {
      ElMessage.error(res.data?.message || '同步失败')
    }
  } catch (err) {
    ElMessage.error('同步失败')
  } finally {
    syncing.value = false
  }
}

// 选择数据源
const handleDatasourceSelect = (row) => {
  if (row) {
    queryForm.datasourceId = row.id
  }
}

// 测试数据源连通性
const handleCheckDatasource = async (row) => {
  row.checking = true
  row.checkResult = null
  try {
    const res = await n9eApi.checkDatasource(row.id)
    if (res.data?.code === 200) {
      const d = res.data.data
      row.checkResult = d.status
      row.checkLatency = d.latencyMs
      if (d.status === 'ok') {
        ElMessage.success(`${row.name} 连接正常 (${d.latencyMs}ms)`)
      } else {
        ElMessage.error(`${row.name} 连接失败: ${d.error || '未知错误'}`)
      }
    }
  } catch (err) {
    row.checkResult = 'error'
    ElMessage.error('检测失败: ' + (err.message || '未知错误'))
  } finally {
    row.checking = false
  }
}

// 批量检测所有数据源
const handleCheckAll = async () => {
  if (datasources.value.length === 0) {
    ElMessage.warning('暂无数据源可检测')
    return
  }
  checkingAll.value = true
  let okCount = 0
  let errCount = 0
  for (const ds of datasources.value) {
    await handleCheckDatasource(ds)
    if (ds.checkResult === 'ok') okCount++
    else errCount++
  }
  checkingAll.value = false
  ElMessage.info(`全量检测完成: ${okCount} 正常, ${errCount} 异常`)
}

// 时间范围变更
const handleTimeRangeChange = () => {
  const now = Math.floor(Date.now() / 1000)
  const ranges = {
    '15m': 15 * 60,
    '1h': 3600,
    '3h': 3 * 3600,
    '6h': 6 * 3600,
    '12h': 12 * 3600,
    '24h': 24 * 3600
  }
  const duration = ranges[queryForm.timeRange] || 3600
  queryForm.start = String(now - duration)
  queryForm.end = String(now)
}

// 执行查询
const handleQuery = async () => {
  if (!queryForm.datasourceId) {
    ElMessage.warning('请选择数据源')
    return
  }
  if (!queryForm.query) {
    ElMessage.warning('请输入 PromQL 表达式')
    return
  }

  handleTimeRangeChange()
  querying.value = true
  try {
    const res = await n9eApi.queryPromQL({
      datasourceId: queryForm.datasourceId,
      query: queryForm.query,
      start: queryForm.start,
      end: queryForm.end,
      step: queryForm.step
    })

    // 响应可能直接是 Prometheus 格式
    const data = res.data?.data ? res.data : (typeof res.data === 'string' ? JSON.parse(res.data) : res.data)
    queryResult.value = data

    if (data?.data?.resultType === 'matrix' && data.data.result?.length > 0) {
      renderChart(data.data.result)
    } else {
      chartData.value = []
    }
  } catch (err) {
    ElMessage.error('查询失败: ' + (err.message || '未知错误'))
  } finally {
    querying.value = false
  }
}

// 渲染 ECharts 图表
const renderChart = async (results) => {
  chartData.value = results
  await nextTick()

  if (!chartRef.value) return
  if (chartInstance) {
    chartInstance.dispose()
  }
  chartInstance = echarts.init(chartRef.value)

  const series = results.map((item, index) => {
    const label = Object.entries(item.metric || {}).map(([k, v]) => `${k}=${v}`).join(', ')
    return {
      name: label.length > 60 ? label.substring(0, 60) + '...' : label,
      type: 'line',
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 1.5 },
      data: (item.values || []).map(([ts, val]) => [ts * 1000, parseFloat(val)])
    }
  })

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross' }
    },
    legend: {
      type: 'scroll',
      bottom: 0,
      textStyle: { fontSize: 11 }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '15%',
      top: '5%',
      containLabel: true
    },
    xAxis: {
      type: 'time',
      axisLabel: {
        formatter: (value) => {
          const d = new Date(value)
          return d.getHours().toString().padStart(2, '0') + ':' + d.getMinutes().toString().padStart(2, '0')
        }
      }
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { type: 'dashed' } }
    },
    series
  }

  chartInstance.setOption(option)
}

onMounted(() => {
  loadDatasources()
  handleTimeRangeChange()
})

onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }
})
</script>

<style scoped>
.n9e-datasource-container {
  padding: 20px;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h3 {
  font-size: 18px;
  font-weight: 600;
  margin: 0 0 8px 0;
  color: #303133;
}

.page-desc {
  color: #909399;
  font-size: 13px;
  margin: 0;
}

.ds-card,
.query-card {
  border-radius: 8px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
}

.card-header span {
  display: flex;
  align-items: center;
  gap: 6px;
}

.empty-tip {
  padding: 20px 0;
}

.chart-container {
  width: 100%;
  height: 320px;
  margin-top: 10px;
}

.raw-data {
  margin-top: 10px;
}

.raw-data code {
  font-size: 11px;
  word-break: break-all;
}
</style>
