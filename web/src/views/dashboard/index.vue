<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import * as echarts from 'echarts'
import { getDashboardStats, getBusinessDistribution } from '@/api/dashboard'
import { GetAllTools, CreateTool, UpdateTool, DeleteTool as DeleteToolAPI, UploadIcon } from '@/api/tool'
import n9eApi from '@/api/n9e'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()

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

// 图表实例
let trendChart = null
let pieChart = null
let heatChart = null

// 发布统计时间维度
const deployTimeRange = ref('week') // week, month, year

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
  Object.assign(toolForm, {
    title: '',
    icon: '',
    link: '',
    sort: 0
  })
  editDialogVisible.value = true
}

// 保存编辑
const saveToolEdit = async () => {
  if (!toolForm.title.trim()) {
    ElMessage.warning('请输入导航标题')
    return
  }

  if (!toolForm.icon) {
    ElMessage.warning('请上传导航图标')
    return
  }

  if (!toolForm.link.trim()) {
    ElMessage.warning('请输入链接地址')
    return
  }

  // 校验链接地址必须包含 http:// 或 https://
  const link = toolForm.link.trim()
  if (!link.startsWith('http://') && !link.startsWith('https://')) {
    ElMessage.warning('链接地址必须以 http:// 或 https:// 开头')
    return
  }

  try {
    if (editingIndex.value >= 0) {
      // 编辑现有工具
      await UpdateTool({
        id: editingTool.value.id,
        title: toolForm.title,
        icon: toolForm.icon,
        link: toolForm.link,
        sort: toolForm.sort
      })
      ElMessage.success('更新成功')
    } else {
      // 添加新工具
      await CreateTool({
        title: toolForm.title,
        icon: toolForm.icon,
        link: toolForm.link,
        sort: toolForm.sort
      })
      ElMessage.success('添加成功')
    }

    editDialogVisible.value = false
    // 重新加载导航工具列表
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
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await DeleteToolAPI(tool.id)
      ElMessage.success('删除成功')
      // 重新加载导航工具列表
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

  // 验证文件类型
  if (!file.type.startsWith('image/')) {
    ElMessage.error('请上传图片文件')
    return
  }

  // 验证文件大小（限制为2MB）
  if (file.size > 2 * 1024 * 1024) {
    ElMessage.error('图片大小不能超过2MB')
    return
  }

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

// 触发文件选择
const triggerIconUpload = () => {
  document.getElementById('iconUpload').click()
}

// 点击导航项
const handleToolClick = (tool) => {
  if (!tool.link) return

  // 判断是外部链接还是内部路由
  if (tool.link.startsWith('http://') || tool.link.startsWith('https://')) {
    // 外部链接，新窗口打开
    window.open(tool.link, '_blank')
  } else {
    // 内部路由
    router.push(tool.link)
  }
}

// 获取发布数据（根据时间维度）
const getDeploymentData = (timeRange) => {
  // 模拟数据，后续替换为真实API
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
    title: {
      text: '上线发布次数统计',
      left: 20,
      top: 10,
      textStyle: {
        fontSize: 16,
        fontWeight: 'normal',
        color: '#333'
      }
    },
    grid: {
      left: 60,
      right: 30,
      top: 60,
      bottom: 40
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        label: {
          backgroundColor: '#6a7985'
        }
      },
      formatter: (params) => {
        let result = params[0].name + '<br/>'
        params.forEach(item => {
          result += `${item.marker} ${item.seriesName}: ${item.value}次<br/>`
        })
        return result
      }
    },
    xAxis: {
      type: 'category',
      data: data.labels,
      axisLine: { lineStyle: { color: '#e0e0e0' } },
      axisTick: { show: false },
      axisLabel: { color: '#999' }
    },
    yAxis: {
      type: 'value',
      name: '发布次数',
      nameTextStyle: { color: '#999', fontSize: 12 },
      splitLine: { lineStyle: { color: '#f5f5f5' } },
      axisLabel: { color: '#999' }
    },
    legend: {
      data: ['生产环境', '测试环境'],
      top: 15,
      right: 120
    },
    series: [
      {
        name: '生产环境',
        type: 'line',
        smooth: true,
        data: data.production,
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(255, 107, 107, 0.4)' },
              { offset: 1, color: 'rgba(255, 107, 107, 0.05)' }
            ]
          }
        },
        lineStyle: { color: '#ff6b6b', width: 2 },
        itemStyle: { color: '#ff6b6b' }
      },
      {
        name: '测试环境',
        type: 'line',
        smooth: true,
        data: data.test,
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(78, 201, 176, 0.4)' },
              { offset: 1, color: 'rgba(78, 201, 176, 0.05)' }
            ]
          }
        },
        lineStyle: { color: '#4ecdc4', width: 2 },
        itemStyle: { color: '#4ecdc4' }
      }
    ]
  }
  trendChart.setOption(option)
}

// 切换时间维度
const changeTimeRange = (range) => {
  deployTimeRange.value = range
  updateTrendChart()
}

// 初始化环形图
const initPieChart = async () => {
  const chartDom = document.getElementById('pieChart')
  if (!chartDom) return

  pieChart = echarts.init(chartDom)

  // 加载业务分布数据
  let businessData = []
  try {
    const response = await getBusinessDistribution()
    if (response.data && response.data.code === 200) {
      const data = response.data.data
      const colors = ['#5dade2', '#f8b739', '#48c9b0', '#9b59b6', '#ec7063', '#ff6b6b', '#4ecdc4', '#45b7d1']
      businessData = data.businessLines.map((line, index) => ({
        value: line.serviceCount,
        name: line.name,
        itemStyle: { color: colors[index % colors.length] }
      }))
    }
  } catch (error) {
    console.error('加载业务分布数据失败:', error)
    // 使用默认数据
    businessData = [
      { value: 10, name: '暂无数据', itemStyle: { color: '#e0e0e0' } }
    ]
  }

  const option = {
    title: {
      text: '业务组应用分布',
      left: 20,
      top: 10,
      textStyle: {
        fontSize: 16,
        fontWeight: 'normal',
        color: '#333'
      }
    },
    tooltip: {
      trigger: 'item',
      formatter: '{b}<br/>应用数: {c}<br/>占比: {d}%'
    },
    legend: {
      orient: 'vertical',
      right: 30,
      top: 'center',
      itemWidth: 12,
      itemHeight: 12,
      textStyle: { fontSize: 12, color: '#666' }
    },
    series: [
      {
        type: 'pie',
        radius: ['50%', '70%'],
        center: ['35%', '50%'],
        avoidLabelOverlap: false,
        label: { show: false },
        labelLine: { show: false },
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.3)'
          }
        },
        data: businessData
      }
    ]
  }
  pieChart.setOption(option)
}

// 资源使用率类型
const resourceType = ref('cpu') // cpu, memory, disk

// 获取资源使用率数据
const getResourceData = (type) => {
  // 模拟数据，后续替换为真实API
  const mockData = {
    cpu: [
      { name: '服务器-01', value: 89.5 },
      { name: '服务器-02', value: 76.3 },
      { name: '服务器-03', value: 68.7 },
      { name: '服务器-04', value: 62.1 },
      { name: '服务器-05', value: 58.9 }
    ],
    memory: [
      { name: '服务器-03', value: 92.3 },
      { name: '服务器-07', value: 85.6 },
      { name: '服务器-01', value: 78.9 },
      { name: '服务器-12', value: 71.2 },
      { name: '服务器-05', value: 68.4 }
    ],
    disk: [
      { name: '服务器-05', value: 94.7 },
      { name: '服务器-08', value: 88.2 },
      { name: '服务器-11', value: 82.5 },
      { name: '服务器-03', value: 75.8 },
      { name: '服务器-09', value: 69.3 }
    ]
  }
  return mockData[type]
}

// 初始化资源使用率图表
const initHeatChart = () => {
  const chartDom = document.getElementById('heatChart')
  if (!chartDom) return

  heatChart = echarts.init(chartDom)
  updateResourceChart()
}

// 更新资源使用率图表
const updateResourceChart = () => {
  if (!heatChart) return

  const data = getResourceData(resourceType.value)
  const titles = {
    cpu: 'CPU使用率 TOP5',
    memory: '内存使用率 TOP5',
    disk: '磁盘占用 TOP5'
  }
  const colors = {
    cpu: ['#ff6b6b', '#ee5a52', '#e74c3c', '#c0392b', '#a93226'],
    memory: ['#3498db', '#2980b9', '#2472a4', '#1f618d', '#1a5276'],
    disk: ['#f39c12', '#e67e22', '#d68910', '#ca6f1e', '#ba4a00']
  }

  const option = {
    title: {
      text: titles[resourceType.value],
      left: 20,
      top: 10,
      textStyle: {
        fontSize: 16,
        fontWeight: 'normal',
        color: '#333'
      }
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      },
      formatter: (params) => {
        const item = params[0]
        return `${item.name}<br/>${item.marker} ${item.value}%`
      }
    },
    grid: {
      left: 100,
      right: 40,
      top: 60,
      bottom: 40
    },
    xAxis: {
      type: 'value',
      max: 100,
      axisLabel: {
        formatter: '{value}%',
        color: '#999'
      },
      splitLine: {
        lineStyle: { color: '#f5f5f5' }
      }
    },
    yAxis: {
      type: 'category',
      data: data.map(item => item.name),
      axisLine: { lineStyle: { color: '#e0e0e0' } },
      axisTick: { show: false },
      axisLabel: { color: '#666' }
    },
    series: [
      {
        type: 'bar',
        data: data.map((item, index) => ({
          value: item.value,
          itemStyle: {
            color: colors[resourceType.value][index]
          }
        })),
        barWidth: 20,
        label: {
          show: true,
          position: 'right',
          formatter: '{c}%',
          color: '#666',
          fontSize: 12
        }
      }
    ]
  }
  heatChart.setOption(option)
}

// 切换资源类型
const changeResourceType = (type) => {
  resourceType.value = type
  updateResourceChart()
}

// 加载导航工具列表
const loadTools = async () => {
  try {
    const response = await GetAllTools()
    if (response.data && response.data.code === 200) {
      // 清空现有数据
      quickTools.splice(0, quickTools.length)
      // 添加新数据
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

      // 更新资产详情
      stats.assets.items[0].value = data.hostStats?.total || 0
      stats.assets.items[1].value = data.databaseStats?.total || 0
      stats.assets.items[2].value = data.k8sClusterStats?.total || 0

      // 更新服务详情
      stats.services.items[0].value = data.serviceStats?.total || 0
      stats.services.items[1].value = data.serviceStats?.businessLines || 0

      // 更新发布详情
      stats.deployment.items[0].value = data.deploymentStats?.total || 0
      stats.deployment.items[1].value = data.taskStats?.total || 0
      stats.deployment.items[2].value = data.deploymentStats?.successRate || 0
    }

    // 加载 CMDB/N9E 概览数据更新监控卡片
    try {
      const overviewRes = await n9eApi.getOverview()
      if (overviewRes.data?.code === 200 && overviewRes.data.data) {
        const ov = overviewRes.data.data
        const hosts = ov.hosts || {}
        // 用 CMDB 真实主机数覆盖资产卡
        if (hosts.total > 0) {
          stats.assets.items[0].value = hosts.total
        }
        // 更新监控告警卡为 CMDB 健康度
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

// 窗口大小改变时重绘图表
const handleResize = () => {
  trendChart?.resize()
  pieChart?.resize()
  heatChart?.resize()
}

// 生命周期
onMounted(async () => {
  await loadData()
  await loadTools()
  setTimeout(async () => {
    initTrendChart()
    await initPieChart()
    initHeatChart()
  }, 100)
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  trendChart?.dispose()
  pieChart?.dispose()
  heatChart?.dispose()
})
</script>

<template>
  <div class="dashboard">
    <!-- 顶部统计卡片 -->
    <div class="stats-cards">
      <!-- 资产详情 -->
      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon" style="background: #e3f2fd;">
            <svg viewBox="0 0 48 48" fill="#2196f3">
              <path d="M40 8H8c-2.21 0-3.98 1.79-3.98 4L4 36c0 2.21 1.79 4 4 4h32c2.21 0 4-1.79 4-4V12c0-2.21-1.79-4-4-4zm0 28H8V16h32v20z"/>
            </svg>
          </div>
          <div class="stat-title">{{ stats.assets.title }}</div>
        </div>
        <div class="stat-items">
          <div class="stat-item" v-for="item in stats.assets.items" :key="item.label">
            <span class="item-label">{{ item.label }}</span>
            <span class="item-value">{{ item.value }}{{ item.unit || '' }}</span>
          </div>
        </div>
      </div>

      <!-- 服务详情 -->
      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon" style="background: #e8f5e9;">
            <svg viewBox="0 0 48 48" fill="#4caf50">
              <path d="M38 6H10c-2.21 0-4 1.79-4 4v28c0 2.21 1.79 4 4 4h28c2.21 0 4-1.79 4-4V10c0-2.21-1.79-4-4-4zM24 34c-5.52 0-10-4.48-10-10s4.48-10 10-10 10 4.48 10 10-4.48 10-10 10zm0-16c-3.31 0-6 2.69-6 6s2.69 6 6 6 6-2.69 6-6-2.69-6-6-6z"/>
            </svg>
          </div>
          <div class="stat-title">{{ stats.services.title }}</div>
        </div>
        <div class="stat-items">
          <div class="stat-item" v-for="item in stats.services.items" :key="item.label">
            <span class="item-label">{{ item.label }}</span>
            <span class="item-value">{{ item.value }}{{ item.unit || '' }}</span>
          </div>
        </div>
      </div>

      <!-- 发布详情 -->
      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon" style="background: #fff3e0;">
            <svg viewBox="0 0 48 48" fill="#ff9800">
              <path d="M20 8l-8 8h6v12h4V16h6l-8-8zm8 24h-6v-12h-4v12h-6l8 8 8-8z"/>
            </svg>
          </div>
          <div class="stat-title">{{ stats.deployment.title }}</div>
        </div>
        <div class="stat-items">
          <div class="stat-item" v-for="item in stats.deployment.items" :key="item.label">
            <span class="item-label">{{ item.label }}</span>
            <span class="item-value">{{ item.value }}{{ item.unit || '' }}</span>
          </div>
        </div>
      </div>

      <!-- 监控告警 -->
      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon" style="background: #fce4ec;">
            <svg viewBox="0 0 48 48" fill="#e91e63">
              <path d="M2 42h44L24 4 2 42zm24-6h-4v-4h4v4zm0-8h-4v-8h4v8z"/>
            </svg>
          </div>
          <div class="stat-title">{{ stats.monitor.title }}</div>
        </div>
        <div class="stat-items">
          <div class="stat-item" v-for="item in stats.monitor.items" :key="item.label">
            <span class="item-label">{{ item.label }}</span>
            <span class="item-value">{{ item.value }}{{ item.unit || '' }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 图表区域 -->
    <div class="charts-row">
      <!-- 发布统计图 -->
      <div class="chart-card large">
        <div class="chart-header">
          <div class="time-range-tabs">
            <button
              :class="['tab-btn', { active: deployTimeRange === 'week' }]"
              @click="changeTimeRange('week')"
            >
              周
            </button>
            <button
              :class="['tab-btn', { active: deployTimeRange === 'month' }]"
              @click="changeTimeRange('month')"
            >
              月
            </button>
            <button
              :class="['tab-btn', { active: deployTimeRange === 'year' }]"
              @click="changeTimeRange('year')"
            >
              年
            </button>
          </div>
        </div>
        <div id="trendChart" style="width: 100%; height: calc(100% - 40px);"></div>
      </div>

      <!-- 环形图 -->
      <div class="chart-card">
        <div id="pieChart" style="width: 100%; height: 100%;"></div>
      </div>
    </div>

    <!-- 底部区域 -->
    <div class="bottom-row">
      <!-- 快捷导航工具 -->
      <div class="tools-card">
        <div class="tools-header">
          <div class="tools-title">快捷导航工具</div>
          <button class="add-tool-btn" @click="addNewTool">
            <svg viewBox="0 0 24 24" fill="currentColor" style="width: 16px; height: 16px;">
              <path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
            </svg>
            添加
          </button>
        </div>
        <div class="tools-grid">
          <div
            class="tool-item"
            v-for="(tool, index) in quickTools"
            :key="tool.id"
          >
            <div class="tool-actions">
              <button class="action-btn edit-btn" @click.stop="openEditDialog(tool, index)">
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/>
                </svg>
              </button>
              <button class="action-btn delete-btn" @click.stop="deleteTool(index)">
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
                </svg>
              </button>
            </div>
            <div class="tool-content" @click="handleToolClick(tool)">
              <div class="tool-icon">
                <img v-if="tool.icon" :src="tool.icon" :alt="tool.title" />
                <div v-else class="icon-placeholder">?</div>
              </div>
              <div class="tool-info">
                <div class="tool-name">{{ tool.title }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 编辑弹窗 -->
      <el-dialog
        v-model="editDialogVisible"
        :title="editingIndex >= 0 ? '编辑导航' : '添加导航'"
        width="550px"
        :close-on-click-modal="false"
      >
        <div class="edit-form">
          <div class="form-item">
            <label><span class="required">*</span> 导航标题</label>
            <input
              v-model="toolForm.title"
              type="text"
              placeholder="例如：百度"
              class="form-input"
            />
          </div>

          <div class="form-item">
            <label><span class="required">*</span> 导航图标</label>
            <div class="icon-upload-wrapper">
              <div class="icon-preview-box">
                <img v-if="toolForm.icon" :src="toolForm.icon" alt="图标预览" />
                <div v-else class="empty-icon">
                  <svg viewBox="0 0 24 24" fill="currentColor">
                    <path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/>
                  </svg>
                  <span>暂无图标</span>
                </div>
              </div>
              <button class="upload-btn" @click="triggerIconUpload">
                <svg viewBox="0 0 24 24" fill="currentColor" style="width: 16px; height: 16px;">
                  <path d="M9 16h6v-6h4l-7-7-7 7h4zm-4 2h14v2H5z"/>
                </svg>
                选择图标
              </button>
              <input
                id="iconUpload"
                type="file"
                accept="image/*"
                @change="handleIconUpload"
                style="display: none;"
              />
            </div>
            <div class="form-tip">支持 PNG、JPG、SVG 格式，大小不超过 2MB</div>
          </div>

          <div class="form-item">
            <label><span class="required">*</span> 链接地址</label>
            <input
              v-model="toolForm.link"
              type="text"
              placeholder="例如：https://www.baidu.com/"
              class="form-input"
            />
            <div class="form-tip">必须以 http:// 或 https:// 开头</div>
          </div>
        </div>

        <template #footer>
          <div class="dialog-footer">
            <button class="btn-cancel" @click="editDialogVisible = false">取消</button>
            <button class="btn-confirm" @click="saveToolEdit">保存</button>
          </div>
        </template>
      </el-dialog>

      <!-- 资源使用率 -->
      <div class="chart-card">
        <div class="chart-header">
          <div class="resource-tabs">
            <button
              :class="['tab-btn', { active: resourceType === 'cpu' }]"
              @click="changeResourceType('cpu')"
            >
              CPU
            </button>
            <button
              :class="['tab-btn', { active: resourceType === 'memory' }]"
              @click="changeResourceType('memory')"
            >
              内存
            </button>
            <button
              :class="['tab-btn', { active: resourceType === 'disk' }]"
              @click="changeResourceType('disk')"
            >
              磁盘
            </button>
          </div>
        </div>
        <div id="heatChart" style="width: 100%; height: calc(100% - 40px);"></div>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.dashboard {
  padding: 20px;
  background: #eef1f6;
  min-height: calc(100vh - 60px);
}

// 顶部统计卡片
.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 20px;
}

.stat-card {
  background: white;
  border-radius: 8px;
  padding: 16px 18px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  transition: all 0.3s ease;

  &:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
    transform: translateY(-2px);
  }
}

.stat-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid #f0f0f0;
}

.stat-title {
  font-size: 15px;
  font-weight: 600;
  color: #333;
}

.stat-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;

  svg {
    width: 22px;
    height: 22px;
  }
}

.stat-items {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
}

.item-label {
  font-size: 13px;
  color: #666;
}

.item-value {
  font-size: 15px;
  font-weight: 600;
  color: #333;
}

// 图表行
.charts-row {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 20px;
  margin-bottom: 20px;
}

.chart-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  height: 400px;

  &.large {
    height: 400px;
  }
}

.chart-header {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 10px;
  padding-bottom: 10px;
}

.time-range-tabs,
.resource-tabs {
  display: flex;
  gap: 8px;
  background: #f5f7fa;
  padding: 4px;
  border-radius: 6px;
}

.tab-btn {
  padding: 6px 16px;
  border: none;
  background: transparent;
  color: #666;
  font-size: 13px;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.3s ease;

  &:hover {
    color: #333;
    background: rgba(255, 255, 255, 0.5);
  }

  &.active {
    background: white;
    color: #2196f3;
    font-weight: 500;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }
}

// 底部行
.bottom-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

// 快捷工具
.tools-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.tools-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.tools-title {
  font-size: 16px;
  font-weight: 500;
  color: #333;
}

.add-tool-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: #2196f3;
  color: white;
  border: none;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.3s ease;

  &:hover {
    background: #1976d2;
    transform: translateY(-1px);
  }
}

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
  padding: 16px 8px;
  border-radius: 8px;
  transition: all 0.3s ease;

  &:hover {
    background: #f8f9fa;

    .tool-actions {
      opacity: 1;
    }
  }
}

.tool-actions {
  position: absolute;
  top: 4px;
  right: 4px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.3s ease;
}

.action-btn {
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.3s ease;

  svg {
    width: 14px;
    height: 14px;
  }

  &.edit-btn {
    background: #e3f2fd;
    color: #2196f3;

    &:hover {
      background: #2196f3;
      color: white;
    }
  }

  &.delete-btn {
    background: #ffebee;
    color: #f44336;

    &:hover {
      background: #f44336;
      color: white;
    }
  }
}

.tool-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.tool-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: #f5f7fa;
  border: 1px solid #e4e7ed;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .icon-placeholder {
    font-size: 24px;
    color: #c0c4cc;
  }
}

.tool-info {
  text-align: center;
}

.tool-name {
  font-size: 13px;
  color: #606266;
  font-weight: 500;
}

// 编辑弹窗样式
.edit-form {
  padding: 10px 0;
}

.form-item {
  margin-bottom: 24px;

  label {
    display: block;
    font-size: 14px;
    color: #333;
    margin-bottom: 10px;
    font-weight: 500;

    .required {
      color: #f56c6c;
      margin-right: 4px;
    }
  }
}

.form-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  font-size: 14px;
  transition: border-color 0.3s ease;
  box-sizing: border-box;

  &:focus {
    outline: none;
    border-color: #2196f3;
  }

  &::placeholder {
    color: #c0c4cc;
  }
}

.form-tip {
  margin-top: 6px;
  font-size: 12px;
  color: #909399;
}

.icon-upload-wrapper {
  display: flex;
  align-items: center;
  gap: 16px;
}

.icon-preview-box {
  width: 80px;
  height: 80px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px dashed #dcdfe6;
  overflow: hidden;
  background: #fafafa;

  img {
    width: 100%;
    height: 100%;
    object-fit: contain;
  }

  .empty-icon {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    color: #c0c4cc;

    svg {
      width: 32px;
      height: 32px;
    }

    span {
      font-size: 11px;
    }
  }
}

.upload-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  background: #f5f7fa;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  font-size: 14px;
  color: #606266;
  cursor: pointer;
  transition: all 0.3s ease;

  &:hover {
    background: #2196f3;
    border-color: #2196f3;
    color: white;
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 10px;
}

.btn-cancel,
.btn-confirm {
  padding: 8px 20px;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.btn-cancel {
  background: #f5f7fa;
  color: #606266;

  &:hover {
    background: #e0e3e8;
  }
}

.btn-confirm {
  background: #2196f3;
  color: white;

  &:hover {
    background: #1976d2;
  }
}

// 响应式设计
@media (max-width: 1400px) {
  .stats-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .charts-row {
    grid-template-columns: 1fr;
  }

  .bottom-row {
    grid-template-columns: 1fr;
  }

  .tools-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 768px) {
  .dashboard {
    padding: 12px;
  }

  .stats-cards {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .charts-row,
  .bottom-row {
    gap: 12px;
  }

  .tools-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
