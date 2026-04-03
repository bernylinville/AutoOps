<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import * as echarts from 'echarts'
import { getDashboardStats, getBusinessDistribution } from '@/api/dashboard'
import { GetAllTools, CreateTool, UpdateTool, DeleteTool as DeleteToolAPI, UploadIcon } from '@/api/tool'
import n9eApi from '@/api/n9e'
import flashdutyApi from '@/api/flashduty'
import FlashDutyIncidents from './FlashDutyIncidents.vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()

// ── 统一调色板（Phase 1 & 2 GeekBlue Theme）──
const AUTOOPS_PALETTE = [
  '#4F46E5',   // Indigo 600 (Primary)
  '#10B981',   // Emerald 500 (Success)
  '#F59E0B',   // Amber 500 (Warning)
  '#EF4444',   // Red 500 (Danger)
  '#6B7280',   // Gray 500 (Info)
  '#6366F1',   // Indigo 500 (Chart Info)
]

// 响应式数据
const loading = ref(true)
const editDialogVisible = ref(false)
const editingTool = ref(null)
const editingIndex = ref(-1)

// 统计数据
const stats = reactive({
  assets: {
    title: '资产详情',
    items: [
      { label: '主机总数', value: 0 },
      { label: '数据库总数', value: 0 },
      { label: 'K8s集群数量', value: 0 }
    ]
  },
  services: {
    title: '服务详情',
    items: [
      { label: '应用总数', value: 0 },
      { label: '业务线总数', value: 0 }
    ]
  },
  deployment: {
    title: '发布详情',
    items: [
      { label: '应用发布', value: 0 },
      { label: '任务执行', value: 0 },
      { label: '成功率', value: 0, unit: '%' }
    ]
  },
  monitor: {
    title: '监控告警',
    items: [
      { label: '在线主机', value: 0 },
      { label: '离线主机', value: 0 },
      { label: '健康度', value: 0, unit: '%' }
    ]
  }
})

// FlashDuty 数据
const fdConfigured = ref(false)
const fdLoading = ref(false)
const fdAlertSummary = reactive({
  activeAlerts: 0,
  criticalCount: 0,
  warningCount: 0,
  infoCount: 0,
  activeIncidents: 0,
  triggeredCount: 0,
  processingCount: 0
})
const fdOnCall = ref([])
const fdSREMetrics = reactive({
  mtta: 0,
  mttr: 0,
  noiseReductionPct: 0,
  ackPct: 0,
  totalIncidents: 0,
  totalAlerts: 0
})

const fdIncidentsVisible = ref(false)
const handleRefreshFlashDuty = () => {
  loadFlashDutyData()
  // Optional: could manually reload the chart here if needed
}

// 图表实例
let trendChart = null
let pieChart = null
let heatChart = null
let sreTrendChart = null

// 发布统计时间维度
const deployTimeRange = ref('week')

// 动态颜色判定
const getValueColorClass = (label, value) => {
  if (label.includes('离线') && value > 0) return 'val-danger'
  if (label.includes('健康度')) {
    let num = parseFloat(value)
    if (isNaN(num) || num === 0) return '' // 初始或零值淡化处理
    if (num < 60) return 'val-danger'
    if (num < 80) return 'val-warning'
    return 'val-success'
  }
  if (label.includes('成功率')) {
    let num = parseFloat(value)
    if (isNaN(num) || num === 0) return '' // 初始或零值淡化处理
    if (num < 90) return 'val-danger'
    if (num < 95) return 'val-warning'
    return 'val-success'
  }
  return ''
}

// 快捷导航工具数据
const quickTools = reactive([])

// 编辑工具表单
const toolForm = reactive({
  title: '',
  icon: '',
  link: '',
  sort: 0
})

// 打开编辑弹窗
const openEditDialog = (tool, index) => {
  editingIndex.value = index
  editingTool.value = tool
  Object.assign(toolForm, {
    title: tool.title,
    icon: tool.icon,
    link: tool.link,
    sort: tool.sort || 0
  })
  editDialogVisible.value = true
}

// 添加新工具
const addNewTool = () => {
  editingIndex.value = -1
  editingTool.value = null
  Object.assign(toolForm, { title: '', icon: '', link: '', sort: 0 })
  editDialogVisible.value = true
}

// 保存编辑
const saveToolEdit = async () => {
  if (!toolForm.title.trim()) { ElMessage.warning('请输入导航标题'); return }
  if (!toolForm.icon) { ElMessage.warning('请上传导航图标'); return }
  if (!toolForm.link.trim()) { ElMessage.warning('请输入链接地址'); return }

  const link = toolForm.link.trim()
  if (!link.startsWith('http://') && !link.startsWith('https://')) {
    ElMessage.warning('链接地址必须以 http:// 或 https:// 开头'); return
  }

  try {
    if (editingIndex.value >= 0) {
      await UpdateTool({ id: editingTool.value.id, title: toolForm.title, icon: toolForm.icon, link: toolForm.link, sort: toolForm.sort })
      ElMessage.success('更新成功')
    } else {
      await CreateTool({ title: toolForm.title, icon: toolForm.icon, link: toolForm.link, sort: toolForm.sort })
      ElMessage.success('添加成功')
    }
    editDialogVisible.value = false
    await loadTools()
  } catch (error) {
    console.error('保存失败:', error)
    ElMessage.error('保存失败，请稍后重试')
  }
}

// 删除工具
const deleteTool = (index) => {
  const tool = quickTools[index]
  ElMessageBox.confirm('确定要删除这个导航吗？', '提示', {
    confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning'
  }).then(async () => {
    try {
      await DeleteToolAPI(tool.id)
      ElMessage.success('删除成功')
      await loadTools()
    } catch (error) {
      console.error('删除失败:', error)
      ElMessage.error('删除失败，请稍后重试')
    }
  }).catch(() => {})
}

// 上传图标
const handleIconUpload = async (event) => {
  const file = event.target.files[0]
  if (!file) return
  if (!file.type.startsWith('image/')) { ElMessage.error('请上传图片文件'); return }
  if (file.size > 2 * 1024 * 1024) { ElMessage.error('图片大小不能超过2MB'); return }

  try {
    const formData = new FormData()
    formData.append('file', file)
    const response = await UploadIcon(formData)
    if (response.data && response.data.code === 200) {
      toolForm.icon = response.data.data
      ElMessage.success('图标上传成功')
    } else {
      ElMessage.error(response.data?.message || '图标上传失败')
    }
  } catch (error) {
    console.error('上传图标失败:', error)
    ElMessage.error('图标上传失败，请稍后重试')
  }
}

const triggerIconUpload = () => { document.getElementById('iconUpload').click() }

// 点击导航项
const handleToolClick = (tool) => {
  if (!tool.link) return
  if (tool.link.startsWith('http://') || tool.link.startsWith('https://')) {
    window.open(tool.link, '_blank')
  } else {
    router.push(tool.link)
  }
}

// 获取发布数据（根据时间维度）
const getDeploymentData = (timeRange) => {
  const mockData = {
    week: {
      labels: ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
      production: [12, 15, 10, 18, 22, 8, 5],
      test: [25, 30, 28, 35, 40, 20, 15]
    },
    month: {
      labels: ['1日', '5日', '10日', '15日', '20日', '25日', '30日'],
      production: [45, 52, 48, 60, 55, 62, 58],
      test: [88, 95, 90, 102, 98, 105, 100]
    },
    year: {
      labels: ['1月', '2月', '3月', '4月', '5月', '6月', '7月', '8月', '9月', '10月', '11月', '12月'],
      production: [180, 165, 195, 210, 205, 220, 215, 230, 225, 240, 235, 250],
      test: [320, 310, 340, 360, 355, 380, 375, 390, 385, 400, 395, 410]
    }
  }
  return mockData[timeRange]
}

// 初始化发布统计图
const initTrendChart = () => {
  const chartDom = document.getElementById('trendChart')
  if (!chartDom) return
  trendChart = echarts.init(chartDom)
  updateTrendChart()
}

// 更新发布统计图
const updateTrendChart = () => {
  if (!trendChart) return
  const data = getDeploymentData(deployTimeRange.value)

  const option = {
    grid: { left: 48, right: 16, top: 40, bottom: 24, containLabel: true },
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(31, 41, 55, 0.95)',
      borderColor: 'rgba(255,255,255,0.1)',
      borderWidth: 1,
      padding: [8, 12],
      boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
      textStyle: { color: '#F9FAFB', fontSize: 13 },
      formatter: (params) => {
        let result = `<div style="margin-bottom:4px;color:#9CA3AF">${params[0].name}</div>`
        params.forEach(item => { result += `<div>${item.marker} <span style="color:#D1D5DB">${item.seriesName}:</span> <span style="font-family:'JetBrains Mono';font-weight:600;margin-left:8px">${item.value}次</span></div>` })
        return result
      }
    },
    legend: { data: ['生产环境', '测试环境'], top: 8, right: 16, itemWidth: 12, itemHeight: 8, textStyle: { color: '#6B7280' } },
    xAxis: {
      type: 'category', data: data.labels,
      axisLine: { lineStyle: { color: '#E5E7EB' } },
      axisTick: { show: false },
      axisLabel: { color: '#6B7280', fontFamily: 'Inter, sans-serif' }
    },
    yAxis: {
      type: 'value', name: '发布次数',
      nameTextStyle: { color: '#6B7280', fontSize: 12 },
      splitLine: { lineStyle: { type: 'dashed', color: '#F3F4F6' } },
      axisLabel: { color: '#9CA3AF', fontFamily: 'JetBrains Mono, monospace' }
    },
    series: [
      {
        name: '生产环境', type: 'line', smooth: true, data: data.production,
        areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(239, 68, 68, 0.15)' }, { offset: 1, color: 'rgba(239, 68, 68, 0.02)' }] } },
        lineStyle: { color: AUTOOPS_PALETTE[3], width: 3, shadowColor: 'rgba(239, 68, 68, 0.2)', shadowBlur: 8, shadowOffsetY: 3 },
        itemStyle: { color: AUTOOPS_PALETTE[3] },
        symbol: 'circle', symbolSize: 6
      },
      {
        name: '测试环境', type: 'line', smooth: true, data: data.test,
        areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(79, 70, 229, 0.15)' }, { offset: 1, color: 'rgba(79, 70, 229, 0.02)' }] } },
        lineStyle: { color: AUTOOPS_PALETTE[0], width: 3, shadowColor: 'rgba(79, 70, 229, 0.2)', shadowBlur: 8, shadowOffsetY: 3 },
        itemStyle: { color: AUTOOPS_PALETTE[0] },
        symbol: 'circle', symbolSize: 6
      }
    ]
  }
  trendChart.setOption(option)
}

const changeTimeRange = (range) => { deployTimeRange.value = range; updateTrendChart() }

// 初始化环形图
const initPieChart = async () => {
  const chartDom = document.getElementById('pieChart')
  if (!chartDom) return
  pieChart = echarts.init(chartDom)
  pieChart.showLoading({ text: '加载中...', color: '#4F46E5', maskColor: 'rgba(255, 255, 255, 0.4)' })

  let businessData = []
  try {
    const response = await getBusinessDistribution()
    if (response.data && response.data.code === 200) {
      const data = response.data.data
      businessData = data.businessLines.map((line, index) => ({
        value: line.serviceCount,
        name: line.name,
        itemStyle: { color: AUTOOPS_PALETTE[index % AUTOOPS_PALETTE.length] }
      }))
    }
  } catch (error) {
    console.error('加载业务分布数据失败:', error)
    businessData = [{ value: 10, name: '暂无数据', itemStyle: { color: '#E4E7ED' } }]
  }

  const option = {
    tooltip: { trigger: 'item', backgroundColor: 'rgba(31, 41, 55, 0.95)', borderColor: 'rgba(255,255,255,0.1)', borderWidth: 1, padding: [8, 12], textStyle: { color: '#F9FAFB' }, formatter: (p) => `<div style="margin-bottom:4px;color:#9CA3AF">${p.name}</div>${p.marker} 应用数: <span style="font-family:'JetBrains Mono';font-weight:600;margin-left:8px">${p.value}</span><br/>${p.marker} 占比: <span style="font-family:'JetBrains Mono';font-weight:600;margin-left:8px">${p.percent}%</span>` },
    legend: { orient: 'vertical', right: 16, top: 'center', itemWidth: 12, itemHeight: 12, itemGap: 14, textStyle: { fontSize: 12, color: '#6B7280' } },
    series: [{
      type: 'pie', radius: ['40%', '60%'], center: ['35%', '50%'], avoidLabelOverlap: false,
      label: { show: false }, labelLine: { show: false },
      emphasis: { itemStyle: { shadowBlur: 8, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.15)' } },
      data: businessData
    }]
  }
  pieChart.hideLoading()
  pieChart.setOption(option)
}

// 资源使用率
const resourceType = ref('cpu')

const getResourceData = (type) => {
  const mockData = {
    cpu: [
      { name: '服务器-01', value: 89.5 }, { name: '服务器-02', value: 76.3 },
      { name: '服务器-03', value: 68.7 }, { name: '服务器-04', value: 62.1 },
      { name: '服务器-05', value: 58.9 }
    ],
    memory: [
      { name: '服务器-03', value: 92.3 }, { name: '服务器-07', value: 85.6 },
      { name: '服务器-01', value: 78.9 }, { name: '服务器-12', value: 71.2 },
      { name: '服务器-05', value: 68.4 }
    ],
    disk: [
      { name: '服务器-05', value: 94.7 }, { name: '服务器-08', value: 88.2 },
      { name: '服务器-11', value: 82.5 }, { name: '服务器-03', value: 75.8 },
      { name: '服务器-09', value: 69.3 }
    ]
  }
  return mockData[type]
}

const initHeatChart = () => {
  const chartDom = document.getElementById('heatChart')
  if (!chartDom) return
  heatChart = echarts.init(chartDom)
  updateResourceChart()
}

const updateResourceChart = () => {
  if (!heatChart) return
  const data = getResourceData(resourceType.value)
  const titles = { cpu: 'CPU 使用率 TOP5', memory: '内存使用率 TOP5', disk: '磁盘占用 TOP5' }

  // 根据数值确定颜色
  const getBarColor = (value) => {
    if (value >= 90) return AUTOOPS_PALETTE[3]      // danger
    if (value >= 70) return AUTOOPS_PALETTE[2]      // warning
    return AUTOOPS_PALETTE[0]                         // primary
  }

  const option = {
    grid: { left: 90, right: 40, top: 40, bottom: 24 },
    tooltip: {
      trigger: 'axis', axisPointer: { type: 'shadow' },
      backgroundColor: 'rgba(31, 41, 55, 0.95)', borderColor: 'rgba(255,255,255,0.1)', borderWidth: 1, padding: [8, 12], textStyle: { color: '#F9FAFB' },
      formatter: (params) => `<div style="margin-bottom:4px;color:#9CA3AF">${params[0].name}</div>${params[0].marker} 占用率: <span style="font-family: 'JetBrains Mono';font-weight:600">${params[0].value}%</span>`
    },
    xAxis: {
      type: 'value', max: 100,
      axisLabel: { formatter: '{value}%', color: '#9CA3AF', fontSize: 12, fontFamily: 'JetBrains Mono, monospace' },
      splitLine: { lineStyle: { type: 'dashed', color: '#F3F4F6' } }
    },
    yAxis: {
      type: 'category', data: data.map(item => item.name),
      axisLine: { lineStyle: { color: '#E5E7EB' } },
      axisTick: { show: false },
      axisLabel: { color: '#6B7280', fontSize: 12 }
    },
    series: [{
      type: 'bar',
      data: data.map(item => ({
        value: item.value,
        itemStyle: { color: getBarColor(item.value), borderRadius: [0, 2, 2, 0] }
      })),
      barWidth: 18,
      label: { show: true, position: 'right', formatter: '{c}%', color: '#606266', fontSize: 12 }
    }]
  }
  heatChart.setOption(option)
}

const changeResourceType = (type) => { resourceType.value = type; updateResourceChart() }

// 加载导航工具列表
const loadTools = async () => {
  try {
    const response = await GetAllTools()
    if (response.data && response.data.code === 200) {
      quickTools.splice(0, quickTools.length)
      quickTools.push(...response.data.data)
    }
  } catch (error) {
    console.error('加载导航工具失败:', error)
  }
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const response = await getDashboardStats()
    if (response.data && response.data.code === 200) {
      const data = response.data.data
      stats.assets.items[0].value = data.hostStats?.total || 0
      stats.assets.items[1].value = data.databaseStats?.total || 0
      stats.assets.items[2].value = data.k8sClusterStats?.total || 0
      stats.services.items[0].value = data.serviceStats?.total || 0
      stats.services.items[1].value = data.serviceStats?.businessLines || 0
      stats.deployment.items[0].value = data.deploymentStats?.total || 0
      stats.deployment.items[1].value = data.taskStats?.total || 0
      stats.deployment.items[2].value = data.deploymentStats?.successRate || 0
    }

    try {
      const overviewRes = await n9eApi.getOverview()
      if (overviewRes.data?.code === 200 && overviewRes.data.data) {
        const ov = overviewRes.data.data
        const hosts = ov.hosts || {}
        if (hosts.total > 0) { stats.assets.items[0].value = hosts.total }
        stats.monitor.items[0].value = hosts.online || 0
        stats.monitor.items[1].value = hosts.offline || 0
        stats.monitor.items[2].value = ov.healthScore || 0
      }
    } catch (e) {
      console.warn('N9E overview not available:', e)
    }
  } catch (error) {
    console.error('加载数据失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载 FlashDuty 数据
const loadFlashDutyData = async () => {
  fdLoading.value = true
  try {
    const statusRes = await flashdutyApi.getConfigStatus()
    if (statusRes.data?.configured) {
      fdConfigured.value = true

      // 并行加载所有 FlashDuty 数据
      const [summaryRes, onCallRes, metricsRes] = await Promise.allSettled([
        flashdutyApi.getAlertSummary(),
        flashdutyApi.getTodayOnCall(),
        flashdutyApi.getSREMetrics(7)
      ])

      if (summaryRes.status === 'fulfilled' && summaryRes.value.data) {
        Object.assign(fdAlertSummary, summaryRes.value.data)
      }
      if (onCallRes.status === 'fulfilled' && onCallRes.value.data?.items) {
        fdOnCall.value = onCallRes.value.data.items
      }
      if (metricsRes.status === 'fulfilled' && metricsRes.value.data) {
        Object.assign(fdSREMetrics, metricsRes.value.data)
      }
    }
  } catch (e) {
    console.warn('FlashDuty data not available:', e)
  } finally {
    fdLoading.value = false
  }
}

// 格式化秒数
const formatDuration = (seconds) => {
  if (!seconds || seconds <= 0) return '0s'
  if (seconds < 60) return Math.round(seconds) + 's'
  if (seconds < 3600) return Math.round(seconds / 60) + 'm'
  return (seconds / 3600).toFixed(1) + 'h'
}

// 初始化 SRE 指标趋势图
const initSreTrendChart = async () => {
  const chartDom = document.getElementById('sreTrendChart')
  if (!chartDom) return
  if (!sreTrendChart) sreTrendChart = echarts.init(chartDom)
  sreTrendChart.showLoading({ text: '加载中...', color: AUTOOPS_PALETTE[0] })

  try {
    const res = await flashdutyApi.getTrendData(7)
    if (res.data && res.data.trendData) {
      const tData = res.data.trendData
      const dates = tData.map(item => {
        const d = new Date(item.timestamp * 1000)
        return `${d.getMonth() + 1}-${d.getDate()}`
      })
      const mtta = tData.map(item => (item.mtta / 60).toFixed(1)) // 转换为分钟
      const mttr = tData.map(item => (item.mttr / 3600).toFixed(1)) // 转换为小时
      const incidents = tData.map(item => item.incidentCount)

      const option = {
        grid: { left: 50, right: 50, top: 40, bottom: 40 },
        tooltip: {
          trigger: 'axis',
          backgroundColor: 'rgba(31, 41, 55, 0.95)', // 深色毛玻璃观感
          borderColor: 'rgba(255,255,255,0.1)',
          borderWidth: 1,
          textStyle: { color: '#F9FAFB' },
          padding: [8, 12],
          boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)'
        },
        legend: {
          data: ['MTTA (分钟)', 'MTTR (小时)', '触发故障数'],
          bottom: 4, icon: 'circle',
          textStyle: { color: '#6B7280' }
        },
        xAxis: {
          type: 'category', data: dates,
          axisLine: { lineStyle: { color: '#E5E7EB' } },
          axisTick: { show: false },
          axisLabel: { color: '#6B7280', fontFamily: 'Inter, sans-serif' }
        },
        yAxis: [
          {
            type: 'value',
            name: '时长',
            position: 'left',
            splitLine: { lineStyle: { type: 'dashed', color: '#F3F4F6' } },
            axisLabel: { color: '#9CA3AF', fontFamily: 'JetBrains Mono, monospace' }
          },
          {
            type: 'value',
            name: '笔数',
            position: 'right',
            splitLine: { show: false },
            axisLabel: { color: '#9CA3AF', fontFamily: 'JetBrains Mono, monospace' },
            minInterval: 1 // 整数
          }
        ],
        series: [
          {
            name: '触发故障数',
            type: 'bar',
            yAxisIndex: 1,
            barWidth: '24%',
            itemStyle: { color: 'rgba(79, 70, 229, 0.15)', borderRadius: [4, 4, 0, 0] }, // Indigo 600 at 15%
            data: incidents
          },
          {
            name: 'MTTA (分钟)',
            type: 'line',
            symbol: 'circle', symbolSize: 6,
            itemStyle: { color: AUTOOPS_PALETTE[1] }, // Emerald (Success represents speed/health)
            lineStyle: { width: 3, shadowColor: 'rgba(16, 185, 129, 0.3)', shadowBlur: 10, shadowOffsetY: 3 },
            data: mtta
          },
          {
            name: 'MTTR (小时)',
            type: 'line',
            symbol: 'circle', symbolSize: 6,
            itemStyle: { color: AUTOOPS_PALETTE[2] }, // Amber 
            lineStyle: { width: 3, shadowColor: 'rgba(245, 158, 11, 0.3)', shadowBlur: 10, shadowOffsetY: 3 },
            data: mttr
          }
        ]
      }
      sreTrendChart.setOption(option)
    }
  } catch (error) {
    console.error('加载 SRE 趋势失败:', error)
  } finally {
    sreTrendChart.hideLoading()
  }
}

// 窗口大小改变时重绘图表
const handleResize = () => {
  trendChart?.resize()
  pieChart?.resize()
  heatChart?.resize()
  sreTrendChart?.resize()
}

onMounted(() => {
  // 1. 无阻塞发起数据获取
  loadData()
  loadTools()
  loadFlashDutyData().then(() => {
    if (fdConfigured.value) {
      setTimeout(() => initSreTrendChart(), 50)
    }
  })

  // 2. 立即初始化图表以避免白板等待（提高刷新感官首屏时间）
  setTimeout(() => {
    initTrendChart()
    initPieChart()
    initHeatChart()
  }, 50)

  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  trendChart?.dispose()
  pieChart?.dispose()
  heatChart?.dispose()
  sreTrendChart?.dispose()
})
</script>

<template>
  <div class="dashboard page-container">
    
    <!-- ════════════════ 管理大盘视窗 ════════════════ -->
    <div class="dashboard-section">
      <div class="section-header">
        <span class="section-title">业务大盘</span>
        <span class="section-subtitle">Executive Overview</span>
      </div>

      <!-- 顶部统计卡片 -->
      <div class="stats-cards">
        <div class="stat-card" v-for="section in [stats.assets, stats.services, stats.deployment, stats.monitor]" :key="section.title">
          <div class="stat-card-title">{{ section.title }}</div>
          <div class="stat-items">
            <div class="stat-item" v-for="item in section.items" :key="item.label">
              <span class="item-label">{{ item.label }}</span>
              <div class="item-value-box">
                <span :class="['item-value', 'mono-font', getValueColorClass(item.label, item.value)]">{{ item.value }}</span>
                <span class="item-unit" v-if="item.unit">{{ item.unit }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 图表区域 -->
      <div class="charts-row">
        <div class="chart-card chart-card--large">
          <div class="chart-card-header">
            <span class="chart-card-title">上线发布次数趋势</span>
            <div class="tab-group">
              <button :class="['tab-btn', { active: deployTimeRange === 'week' }]" @click="changeTimeRange('week')">周</button>
              <button :class="['tab-btn', { active: deployTimeRange === 'month' }]" @click="changeTimeRange('month')">月</button>
              <button :class="['tab-btn', { active: deployTimeRange === 'year' }]" @click="changeTimeRange('year')">年</button>
            </div>
          </div>
          <div id="trendChart" class="chart-body"></div>
        </div>

        <div class="chart-card">
          <div class="chart-card-header">
            <span class="chart-card-title">业务组应用分布</span>
          </div>
          <div id="pieChart" class="chart-body"></div>
        </div>
      </div>
    </div>

    <!-- ════════════════ FlashDuty 告警中心 ════════════════ -->
    <div class="dashboard-section mt-4" v-if="fdConfigured">
      <div class="section-header">
        <span class="section-title">告警中心</span>
        <span class="section-subtitle">FlashDuty Alert Center</span>
      </div>

      <div class="fd-grid" v-loading="fdLoading">
        <!-- 告警统计 -->
        <div class="fd-card fd-card--alert">
          <div class="fd-card-header">
            <span class="fd-card-title">告警概况</span>
            <span class="fd-badge" :class="fdAlertSummary.activeAlerts > 0 ? 'fd-badge--danger' : 'fd-badge--success'">
              {{ fdAlertSummary.activeAlerts > 0 ? '有告警' : '正常' }}
            </span>
          </div>
          <div class="fd-stats-row">
            <div class="fd-stat">
              <span class="fd-stat-value mono-font val-danger">{{ fdAlertSummary.criticalCount }}</span>
              <span class="fd-stat-label">严重</span>
            </div>
            <div class="fd-stat">
              <span class="fd-stat-value mono-font val-warning">{{ fdAlertSummary.warningCount }}</span>
              <span class="fd-stat-label">警告</span>
            </div>
            <div class="fd-stat">
              <span class="fd-stat-value mono-font">{{ fdAlertSummary.infoCount }}</span>
              <span class="fd-stat-label">信息</span>
            </div>
          </div>
        </div>

        <!-- 故障统计 -->
        <div class="fd-card fd-card--incident" style="cursor: pointer" @click="fdIncidentsVisible = true" title="点击展开处理故障">
          <div class="fd-card-header">
            <span class="fd-card-title">故障处理</span>
            <span class="fd-badge" :class="fdAlertSummary.triggeredCount > 0 ? 'fd-badge--danger' : 'fd-badge--success'">
              {{ fdAlertSummary.activeIncidents }} 活跃
            </span>
          </div>
          <div class="fd-stats-row">
            <div class="fd-stat">
              <span class="fd-stat-value mono-font val-danger">{{ fdAlertSummary.triggeredCount }}</span>
              <span class="fd-stat-label">待处理</span>
            </div>
            <div class="fd-stat">
              <span class="fd-stat-value mono-font val-warning">{{ fdAlertSummary.processingCount }}</span>
              <span class="fd-stat-label">处理中</span>
            </div>
          </div>
        </div>

        <!-- 今日值班 -->
        <div class="fd-card fd-card--oncall">
          <div class="fd-card-header">
            <span class="fd-card-title">今日值班</span>
          </div>
          <div class="fd-oncall-list" v-if="fdOnCall.length > 0">
            <div class="fd-oncall-item" v-for="person in fdOnCall" :key="person.personName">
              <div class="fd-oncall-avatar">{{ person.personName?.charAt(0) || '?' }}</div>
              <div class="fd-oncall-info">
                <span class="fd-oncall-name">{{ person.personName }}</span>
                <span class="fd-oncall-schedule">{{ person.scheduleName }}</span>
              </div>
            </div>
          </div>
          <div class="fd-oncall-empty" v-else>
            <span>暂无值班信息</span>
          </div>
        </div>

        <!-- SRE 指标 -->
        <div class="fd-card fd-card--sre">
          <div class="fd-card-header">
            <span class="fd-card-title">SRE 指标 <small>(近 7 天)</small></span>
          </div>
          <div class="fd-sre-grid">
            <div class="fd-sre-item">
              <span class="fd-sre-value mono-font">{{ formatDuration(fdSREMetrics.mtta) }}</span>
              <span class="fd-sre-label">MTTA</span>
            </div>
            <div class="fd-sre-item">
              <span class="fd-sre-value mono-font">{{ formatDuration(fdSREMetrics.mttr) }}</span>
              <span class="fd-sre-label">MTTR</span>
            </div>
            <div class="fd-sre-item">
              <span class="fd-sre-value mono-font">{{ fdSREMetrics.noiseReductionPct.toFixed(1) }}%</span>
              <span class="fd-sre-label">降噪率</span>
            </div>
            <div class="fd-sre-item">
              <span class="fd-sre-value mono-font">{{ fdSREMetrics.ackPct.toFixed(1) }}%</span>
              <span class="fd-sre-label">响应率</span>
            </div>
          </div>
        </div>
      </div>

      <!-- SRE 趋势图表区 -->
      <div class="charts-row mt-4">
        <div class="chart-card" style="grid-column: span 2">
          <div class="chart-card-header">
            <span class="chart-card-title">SRE 黄金指标趋势 <small class="text-muted" style="margin-left: 8px; font-weight: normal; color: #909399">(近 7 天)</small></span>
          </div>
          <div id="sreTrendChart" class="chart-body" style="height: 280px; width: 100%;"></div>
        </div>
      </div>
    </div>

    <!-- ════════════════ 运维操作视窗 ════════════════ -->
    <div class="dashboard-section mt-4">
      <div class="section-header">
        <span class="section-title">运维控制台</span>
        <span class="section-subtitle">Ops Workspace</span>
      </div>

      <div class="bottom-row">
        <!-- 快捷导航工具 -->
        <div class="panel-card">
          <div class="panel-card-header">
            <span class="panel-card-title">快捷入口</span>
            <el-button type="primary" size="small" text @click="addNewTool">
              <el-icon><Plus /></el-icon> 增加入口
            </el-button>
          </div>
          <div class="tools-grid" v-if="quickTools.length > 0">
            <div class="tool-item" v-for="(tool, index) in quickTools" :key="tool.id">
              <div class="tool-actions">
                <el-button text size="small" @click.stop="openEditDialog(tool, index)" title="编辑">
                  <el-icon size="14"><Edit /></el-icon>
                </el-button>
                <el-button text size="small" type="danger" @click.stop="deleteTool(index)" title="删除">
                  <el-icon size="14"><Delete /></el-icon>
                </el-button>
              </div>
              <div class="tool-content" @click="handleToolClick(tool)">
                <div class="tool-icon">
                  <img v-if="tool.icon" :src="tool.icon" :alt="tool.title" />
                  <el-icon v-else :size="24" class="text-info"><Link /></el-icon>
                </div>
                <span class="tool-name">{{ tool.title }}</span>
              </div>
            </div>
          </div>
          <el-empty v-else description="暂无快捷入口" :image-size="48" />
        </div>

        <!-- 资源使用率 -->
        <div class="chart-card">
          <div class="chart-card-header">
            <span class="chart-card-title">资源高水位 TOP5</span>
            <div class="tab-group">
              <button :class="['tab-btn', { active: resourceType === 'cpu' }]" @click="changeResourceType('cpu')">CPU</button>
              <button :class="['tab-btn', { active: resourceType === 'memory' }]" @click="changeResourceType('memory')">内存</button>
              <button :class="['tab-btn', { active: resourceType === 'disk' }]" @click="changeResourceType('disk')">磁盘</button>
            </div>
          </div>
          <div id="heatChart" class="chart-body"></div>
        </div>
      </div>
    </div>

    <!-- 编辑弹窗 -->
    <el-dialog
      v-model="editDialogVisible"
      :title="editingIndex >= 0 ? '编辑入口' : '新增入口'"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-form label-width="80px">
        <el-form-item label="入口名称" required>
          <el-input v-model="toolForm.title" placeholder="例如：GitLab" />
        </el-form-item>
        <el-form-item label="入口图标" required>
          <div class="icon-upload-row">
            <div class="icon-preview">
              <img v-if="toolForm.icon" :src="toolForm.icon" alt="图标预览" />
              <el-icon v-else :size="28" class="text-info"><Picture /></el-icon>
            </div>
            <el-button @click="triggerIconUpload">浏览文件</el-button>
            <input id="iconUpload" type="file" accept="image/*" @change="handleIconUpload" style="display: none;" />
          </div>
          <div class="form-tip">建议比例 1:1，支持 PNG/SVG，不超过 2MB</div>
        </el-form-item>
        <el-form-item label="链接地址" required>
          <el-input v-model="toolForm.link" placeholder="https://..." />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveToolEdit">保存入口</el-button>
      </template>
    </el-dialog>

    <!-- FlashDuty 故障操作抽屉 -->
    <FlashDutyIncidents v-model:visible="fdIncidentsVisible" @refresh-dashboard="handleRefreshFlashDuty" />
  </div>
</template>

<style scoped lang="less">
// ═══════════════════════════════════════════════════════════
//  视窗标题
// ═══════════════════════════════════════════════════════════
.dashboard-section {
  margin-bottom: var(--ao-spacing-lg);
}

.section-header {
  display: flex;
  align-items: baseline;
  gap: 12px;
  margin-bottom: var(--ao-spacing-md);
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--ao-text-primary);
  border-left: 3px solid var(--ao-primary);
  padding-left: 8px;
  line-height: 1.2;
}

.section-subtitle {
  font-size: 13px;
  color: var(--ao-text-secondary);
  font-family: var(--ao-font-family-mono);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

// ═══════════════════════════════════════════════════════════
//  统计卡片
// ═══════════════════════════════════════════════════════════
.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--ao-spacing-md);
  margin-bottom: var(--ao-spacing-lg);
}

.stat-card {
  background: var(--ao-bg-white);
  border-radius: var(--ao-radius);
  border: 1px solid var(--ao-border-lighter);
  padding: 16px 20px;
}

.stat-card-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--ao-text-primary);
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--ao-border-lighter);
}

.stat-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.stat-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.item-label {
  font-size: 13px;
  color: var(--ao-text-secondary);
}

.item-value-box {
  display: flex;
  align-items: baseline;
  position: relative;
}

.item-value {
  font-size: 18px;
  font-weight: 700;
  color: var(--ao-text-primary);
}

.item-unit {
  position: absolute;
  left: 100%;
  margin-left: 4px;
  font-size: 12px;
  line-height: 1;
  color: var(--ao-text-muted); // uses muted for less intrusion
}

// 动态颜色
.val-success { color: var(--ao-success); }
.val-warning { color: var(--ao-warning); }
.val-danger { color: var(--ao-danger); }

// ═══════════════════════════════════════════════════════════
//  图表卡片
// ═══════════════════════════════════════════════════════════
.chart-card, .panel-card {
  background: var(--ao-bg-white);
  border-radius: var(--ao-radius);
  border: 1px solid var(--ao-border-lighter);
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
}

.chart-card {
  height: 380px;
}

.chart-card--large {
  height: 380px;
}

.chart-card-header, .panel-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  flex-shrink: 0;
}

.chart-card-title, .panel-card-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--ao-text-primary);
}

.chart-body {
  width: 100%;
  flex: 1;
  min-height: 0;
}

// ═══════════════════════════════════════════════════════════
//  Tab 切换按钮组
// ═══════════════════════════════════════════════════════════
.tab-group {
  display: flex;
  gap: 2px;
  background: var(--ao-fill);
  padding: 3px;
  border-radius: var(--ao-radius);
}

.tab-btn {
  padding: 4px 14px;
  border: none;
  background: transparent;
  color: var(--ao-text-secondary);
  font-size: 12px;
  cursor: pointer;
  border-radius: var(--ao-radius-sm);
  transition: all var(--ao-transition-fast);
  line-height: 1.5;
  font-weight: 500;

  &:hover { color: var(--ao-text-primary); }
  &.active {
    background: var(--ao-bg-white);
    color: var(--ao-primary);
    box-shadow: var(--ao-shadow-sm);
  }
}

// ═══════════════════════════════════════════════════════════
//  布局与网格
// ═══════════════════════════════════════════════════════════
.charts-row {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: var(--ao-spacing-md);
}

.bottom-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--ao-spacing-md);
}

// 快捷工具
.tools-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 12px;
}

.tool-item {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 12px 8px;
  border-radius: var(--ao-radius);
  transition: background-color var(--ao-transition);
  border: 1px solid transparent;

  &:hover {
    background: var(--ao-fill);
    border-color: var(--ao-border-lighter);

    .tool-actions { opacity: 1; }
  }
}

.tool-actions {
  position: absolute;
  top: 4px;
  right: 4px;
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity var(--ao-transition);
  
  .el-button { margin-left: 0; padding: 4px; }
}

.tool-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.tool-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--ao-radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: var(--ao-bg-white);
  border: 1px solid var(--ao-border-lighter);
  box-shadow: var(--ao-shadow-sm);

  img { width: 100%; height: 100%; object-fit: contain; padding: 6px; }
}

.tool-name {
  font-size: 13px;
  color: var(--ao-text-regular);
  text-align: center;
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 500;
}

// 弹窗相关
.icon-upload-row { display: flex; align-items: center; gap: 12px; }
.icon-preview {
  width: 64px; height: 64px;
  border-radius: var(--ao-radius);
  display: flex; align-items: center; justify-content: center;
  border: 1px dashed var(--ao-border);
  background: var(--ao-bg-white);
  img { width: 100%; height: 100%; object-fit: contain; padding: 4px; }
}
.form-tip { margin-top: 4px; font-size: 12px; color: var(--ao-text-secondary); }

// ═══════════════════════════════════════════════════════════
//  FlashDuty 告警中心
// ═══════════════════════════════════════════════════════════
.fd-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--ao-spacing-md);
}

.fd-card {
  background: var(--ao-bg-white);
  border-radius: var(--ao-radius);
  border: 1px solid var(--ao-border-lighter);
  padding: 16px 20px;
}

.fd-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}

.fd-card-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--ao-text-primary);

  small {
    font-weight: 400;
    font-size: 12px;
    color: var(--ao-text-secondary);
  }
}

.fd-badge {
  font-size: 12px;
  padding: 2px 10px;
  border-radius: 12px;
  font-weight: 500;
}
.fd-badge--danger { background: #fef0f0; color: var(--ao-danger); }
.fd-badge--success { background: #f0f9eb; color: var(--ao-success); }

.fd-stats-row {
  display: flex;
  gap: 20px;
}

.fd-stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.fd-stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--ao-text-primary);
}

.fd-stat-label {
  font-size: 12px;
  color: var(--ao-text-secondary);
}

.fd-oncall-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.fd-oncall-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.fd-oncall-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--ao-primary), #79bbff);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 600;
  flex-shrink: 0;
}

.fd-oncall-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow: hidden;
}

.fd-oncall-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--ao-text-primary);
}

.fd-oncall-schedule {
  font-size: 12px;
  color: var(--ao-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fd-oncall-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 60px;
  color: var(--ao-text-secondary);
  font-size: 13px;
}

.fd-sre-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.fd-sre-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px;
  background: var(--ao-fill);
  border-radius: var(--ao-radius-sm);
}

.fd-sre-value {
  font-size: 18px;
  font-weight: 700;
  color: var(--ao-primary);
}

.fd-sre-label {
  font-size: 12px;
  color: var(--ao-text-secondary);
  font-weight: 500;
}

@media (max-width: 1400px) {
  .stats-cards { grid-template-columns: repeat(2, 1fr); }
  .charts-row, .bottom-row { grid-template-columns: 1fr; }
  .tools-grid { grid-template-columns: repeat(4, 1fr); }
  .fd-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 768px) {
  .stats-cards { grid-template-columns: 1fr; }
  .tools-grid { grid-template-columns: repeat(3, 1fr); }
  .fd-grid { grid-template-columns: 1fr; }
}
</style>
