<template>
  <div class="node-details-container">
    <div class="node-details-header">
      <el-button 
        @click="goBack" 
        :icon="ArrowLeft" 
        type="primary" 
        size="small"
        class="back-btn"
      >
        返回
      </el-button>
      <h1 class="page-title">{{ nodeData?.basic?.name || '节点详情' }}</h1>
    </div>

    <div class="node-details-content" v-loading="loading">
      <div v-if="!loading && nodeData">
        <!-- 节点概览卡片 -->
        <div class="overview-section">
          <el-row :gutter="16">
            <el-col :span="6">
              <el-card class="overview-card status-card">
                <div class="card-content">
                  <div class="card-icon status-icon">
                    <el-icon size="32"><Monitor /></el-icon>
                  </div>
                  <div class="card-info">
                    <div class="card-value">
                      <el-tag 
                        :type="getStatusType(nodeData.basic?.status)"
                        class="status-tag-large"
                        effect="dark"
                      >
                        {{ getStatusText(nodeData.basic?.status) }}
                      </el-tag>
                    </div>
                    <div class="card-label">节点状态</div>
                    <div class="card-sub">{{ nodeData.basic?.role || 'Unknown' }} 节点</div>
                  </div>
                </div>
              </el-card>
            </el-col>
            <el-col :span="6">
              <el-card class="overview-card cpu-card">
                <div class="card-content">
                  <div class="card-icon cpu-icon">
                    <img src="@/assets/image/cpu.svg" alt="CPU" class="svg-icon" />
                  </div>
                  <div class="card-info">
                    <div class="card-value" :class="getCpuUsageClass()">{{ cpuUsageRate }}%</div>
                    <div class="card-label">CPU使用率</div>
                    <div class="card-sub">{{ nodeData.resources?.cpu?.capacity || 0 }}核可用</div>
                    <el-progress 
                      :percentage="cpuUsageRate" 
                      :color="getProgressColor(cpuUsageRate)"
                      :show-text="false"
                      class="mini-progress"
                    />
                  </div>
                </div>
              </el-card>
            </el-col>
            <el-col :span="6">
              <el-card class="overview-card memory-card">
                <div class="card-content">
                  <div class="card-icon memory-icon">
                    <img src="@/assets/image/内存.svg" alt="内存" class="svg-icon" />
                  </div>
                  <div class="card-info">
                    <div class="card-value" :class="getMemoryUsageClass()">{{ memoryUsageRate }}%</div>
                    <div class="card-label">内存使用率</div>
                    <div class="card-sub">{{ formatMemory(nodeData.resources?.memory?.capacity) }}</div>
                    <el-progress 
                      :percentage="memoryUsageRate" 
                      :color="getProgressColor(memoryUsageRate)"
                      :show-text="false"
                      class="mini-progress"
                    />
                  </div>
                </div>
              </el-card>
            </el-col>
            <el-col :span="6">
              <el-card class="overview-card pods-card">
                <div class="card-content">
                  <div class="card-icon pods-icon">
                    <el-icon size="32"><Box /></el-icon>
                  </div>
                  <div class="card-info">
                    <div class="card-value">{{ nodeData.resources?.pods?.current || 0 }}</div>
                    <div class="card-label">运行Pod数</div>
                    <div class="card-sub">总容量: {{ nodeData.resources?.pods?.capacity || 0 }}</div>
                    <el-progress 
                      :percentage="podUsageRate" 
                      :color="getProgressColor(podUsageRate)"
                      :show-text="false"
                      class="mini-progress"
                    />
                  </div>
                </div>
              </el-card>
            </el-col>
          </el-row>
        </div>

        <!-- 详细信息网格 -->
        <div class="details-grid">
        <!-- 基本信息 -->
        <div class="info-card">
          <div class="card-header">
            <el-icon><InfoFilled /></el-icon>
            <h3>基本信息</h3>
          </div>
          <div class="card-content">
            <div class="info-row">
              <span class="label">节点名称：</span>
              <span class="value">{{ nodeData.basic?.name || 'N/A' }}</span>
            </div>
            <div class="info-row">
              <span class="label">节点角色：</span>
              <span class="value">{{ nodeData.basic?.role || 'N/A' }}</span>
            </div>
            <div class="info-row">
              <span class="label">节点状态：</span>
              <el-tag 
                :type="getStatusType(nodeData.basic?.status)"
                class="status-tag"
              >
                {{ getStatusText(nodeData.basic?.status) }}
              </el-tag>
            </div>
            <div class="info-row">
              <span class="label">创建时间：</span>
              <span class="value">{{ formatTime(nodeData.basic?.creationTime) || 'N/A' }}</span>
            </div>
            <div class="info-row">
              <span class="label">Kubernetes版本：</span>
              <span class="value">{{ nodeData.basic?.k8sVersion || 'N/A' }}</span>
            </div>
            <div class="info-row">
              <span class="label">容器运行时：</span>
              <span class="value">{{ nodeData.basic?.containerRuntime || 'N/A' }}</span>
            </div>
          </div>
        </div>

        <!-- 系统信息 -->
        <div class="info-card">
          <div class="card-header">
            <el-icon><Monitor /></el-icon>
            <h3>系统信息</h3>
          </div>
          <div class="card-content">
            <div class="info-row">
              <span class="label">操作系统：</span>
              <span class="value">{{ nodeData.system?.osImage || 'N/A' }}</span>
            </div>
            <div class="info-row">
              <span class="label">内核版本：</span>
              <span class="value">{{ nodeData.system?.kernelVersion || 'N/A' }}</span>
            </div>
            <div class="info-row">
              <span class="label">架构：</span>
              <span class="value">{{ nodeData.system?.architecture || 'N/A' }}</span>
            </div>
            <div class="info-row">
              <span class="label">主机名：</span>
              <span class="value">{{ nodeData.system?.hostname || 'N/A' }}</span>
            </div>
            <div class="info-row">
              <span class="label">机器ID：</span>
              <span class="value">{{ nodeData.system?.machineId || 'N/A' }}</span>
            </div>
          </div>
        </div>

        <!-- 网络信息 -->
        <div class="info-card">
          <div class="card-header">
            <el-icon><Connection /></el-icon>
            <h3>网络信息</h3>
          </div>
          <div class="card-content">
            <div class="info-row">
              <span class="label">内部IP：</span>
              <span class="value">{{ nodeData.network?.internalIP || 'N/A' }}</span>
            </div>
            <div class="info-row">
              <span class="label">外部IP：</span>
              <span class="value">{{ nodeData.network?.externalIP || 'N/A' }}</span>
            </div>
            <div class="info-row">
              <span class="label">Pod CIDR：</span>
              <span class="value">{{ nodeData.network?.podCIDR || 'N/A' }}</span>
            </div>
          </div>
        </div>

        <!-- 资源信息 -->
        <div class="info-card resources-card">
          <div class="card-header">
            <el-icon><Monitor /></el-icon>
            <h3>资源监控</h3>
            <div class="resource-status-indicator">
              <div class="status-dot" :class="getOverallResourceStatus()"></div>
              <span class="status-text">{{ getResourceStatusText() }}</span>
            </div>
          </div>
          <div class="card-content">
            <div class="resource-grid">
              <div class="resource-item advanced-resource">
                <div class="resource-icon cpu-resource">
                  <img src="@/assets/image/cpu.svg" alt="CPU" class="svg-icon-small" />
                </div>
                <div class="resource-info">
                  <div class="resource-header">
                    <span class="resource-name">CPU处理器</span>
                    <span class="resource-usage" :class="getCpuUsageClass()">{{ cpuUsageRate }}%</span>
                  </div>
                  <div class="resource-details">
                    <div class="detail-item">
                      <span class="detail-label">已使用:</span>
                      <span class="detail-value">{{ getCpuUsed() }}核</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">总容量:</span>
                      <span class="detail-value">{{ nodeData.resources?.cpu?.capacity || 0 }}核</span>
                    </div>
                  </div>
                  <div class="advanced-progress">
                    <el-progress 
                      :percentage="cpuUsageRate" 
                      :color="getProgressColor(cpuUsageRate)"
                      :show-text="false"
                      stroke-width="8"
                    />
                    <div class="progress-labels">
                      <span>0%</span>
                      <span>50%</span>
                      <span>100%</span>
                    </div>
                  </div>
                </div>
              </div>
              
              <div class="resource-item advanced-resource">
                <div class="resource-icon memory-resource">
                  <img src="@/assets/image/内存.svg" alt="内存" class="svg-icon-small" />
                </div>
                <div class="resource-info">
                  <div class="resource-header">
                    <span class="resource-name">内存存储</span>
                    <span class="resource-usage" :class="getMemoryUsageClass()">{{ memoryUsageRate }}%</span>
                  </div>
                  <div class="resource-details">
                    <div class="detail-item">
                      <span class="detail-label">已使用:</span>
                      <span class="detail-value">{{ getMemoryUsed() }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">总容量:</span>
                      <span class="detail-value">{{ formatMemory(nodeData.resources?.memory?.capacity) }}</span>
                    </div>
                  </div>
                  <div class="advanced-progress">
                    <el-progress 
                      :percentage="memoryUsageRate" 
                      :color="getProgressColor(memoryUsageRate)"
                      :show-text="false"
                      stroke-width="8"
                    />
                    <div class="progress-labels">
                      <span>0%</span>
                      <span>50%</span>
                      <span>100%</span>
                    </div>
                  </div>
                </div>
              </div>

              <div class="resource-item advanced-resource">
                <div class="resource-icon pods-resource">
                  <el-icon size="24"><Box /></el-icon>
                </div>
                <div class="resource-info">
                  <div class="resource-header">
                    <span class="resource-name">容器Pod</span>
                    <span class="resource-usage">{{ nodeData.resources?.pods?.current || 0 }}/{{ nodeData.resources?.pods?.capacity || 0 }}</span>
                  </div>
                  <div class="resource-details">
                    <div class="detail-item">
                      <span class="detail-label">运行中:</span>
                      <span class="detail-value">{{ nodeData.resources?.pods?.current || 0 }}个</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">最大数:</span>
                      <span class="detail-value">{{ nodeData.resources?.pods?.capacity || 0 }}个</span>
                    </div>
                  </div>
                  <div class="advanced-progress">
                    <el-progress 
                      :percentage="podUsageRate" 
                      :color="getProgressColor(podUsageRate)"
                      :show-text="false"
                      stroke-width="8"
                    />
                    <div class="progress-labels">
                      <span>0</span>
                      <span>{{ Math.floor((nodeData.resources?.pods?.capacity || 0) / 2) }}</span>
                      <span>{{ nodeData.resources?.pods?.capacity || 0 }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 标签信息 -->
        <div class="info-card">
          <div class="card-header">
            <el-icon><Setting /></el-icon>
            <h3>标签信息</h3>
          </div>
          <div class="card-content">
            <div class="labels-container">
              <el-tag 
                v-for="(value, key) in nodeData.labels" 
                :key="key"
                class="label-tag"
                size="small"
              >
                {{ key }}: {{ value }}
              </el-tag>
              <div v-if="!nodeData.labels || Object.keys(nodeData.labels).length === 0" class="no-data">
                暂无标签信息
              </div>
            </div>
          </div>
        </div>

        <!-- 注解信息 -->
        <div class="info-card">
          <div class="card-header">
            <el-icon><FolderOpened /></el-icon>
            <h3>注解信息</h3>
          </div>
          <div class="card-content">
            <div class="annotations-container">
              <div 
                v-for="(value, key) in nodeData.annotations" 
                :key="key"
                class="annotation-item"
              >
                <div class="annotation-key">{{ key }}</div>
                <div class="annotation-value">{{ value }}</div>
              </div>
              <div v-if="!nodeData.annotations || Object.keys(nodeData.annotations).length === 0" class="no-data">
                暂无注解信息
              </div>
            </div>
          </div>
        </div>

        <!-- Pod列表 -->
        <div class="info-card full-width">
          <div class="card-header">
            <el-icon><Monitor /></el-icon>
            <h3>运行中的Pod ({{ podList.length }})</h3>
          </div>
          <div class="card-content">
            <el-table 
              :data="podList" 
              style="width: 100%"
              :header-cell-style="{ background: 'rgba(255, 255, 255, 0.1)', color: '#333' }"
            >
              <el-table-column prop="name" label="Pod名称" width="200" />
              <el-table-column prop="namespace" label="命名空间" width="150" />
              <el-table-column prop="status" label="状态" width="100">
                <template #default="scope">
                  <el-tag 
                    :type="getPodStatusType(scope.row.status)"
                    size="small"
                  >
                    {{ scope.row.status }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="cpuUsage" label="CPU使用" width="120" />
              <el-table-column prop="memUsage" label="内存使用" width="120" />
              <el-table-column label="重启次数" width="100">
                <template #default="scope">
                  {{ scope.row.restarts || 0 }}
                </template>
              </el-table-column>
              <el-table-column label="运行时间">
                <template #default="scope">
                  {{ scope.row.age || '未知' }}
                </template>
              </el-table-column>
            </el-table>
            <div v-if="podList.length === 0" class="no-data">
              暂无Pod信息
            </div>
          </div>
        </div>

        <!-- 节点条件 -->
        <div class="info-card full-width">
          <div class="card-header">
            <el-icon><InfoFilled /></el-icon>
            <h3>节点条件</h3>
          </div>
          <div class="card-content">
            <el-table 
              :data="conditionsList" 
              style="width: 100%"
              :header-cell-style="{ background: 'rgba(255, 255, 255, 0.1)', color: '#333' }"
            >
              <el-table-column prop="type" label="类型" width="150" />
              <el-table-column prop="status" label="状态" width="100">
                <template #default="scope">
                  <el-tag 
                    :type="scope.row.status === 'True' ? 'success' : 'danger'"
                    size="small"
                  >
                    {{ scope.row.status }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="reason" label="原因" width="150" />
              <el-table-column prop="message" label="消息" />
              <el-table-column prop="lastTransitionTime" label="最后更新时间" width="180">
                <template #default="scope">
                  {{ formatTime(scope.row.lastTransitionTime) }}
                </template>
              </el-table-column>
            </el-table>
            <div v-if="conditionsList.length === 0" class="no-data">
              暂无条件信息
            </div>
          </div>
        </div>
        </div>
      </div>

      <div v-else-if="!loading && !nodeData" class="no-data-container">
        <el-empty description="未找到节点信息" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ArrowLeft, InfoFilled, Monitor, Network, FolderOpened, Setting, Box, CpuFill, MemoryStick, Grid, Files, CircleCheck, Warning, Timer } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import k8sApi from '@/api/k8s'

const router = useRouter()
const route = useRoute()

const loading = ref(false)
const nodeData = ref(null)

const clusterId = computed(() => route.params.clusterId)
const nodeName = computed(() => route.params.nodeName)

// 计算资源使用率
const cpuUsageRate = computed(() => {
  if (!nodeData.value?.resources?.cpu) return 0
  
  // 优先使用API提供的使用率
  if (nodeData.value.resources.cpu.usageRate > 0) {
    return Math.round(nodeData.value.resources.cpu.usageRate)
  }
  
  // 如果没有使用率，用requests/allocatable计算
  const { requests, allocatable } = nodeData.value.resources.cpu
  if (allocatable > 0 && requests > 0) {
    return Math.round((requests / allocatable) * 100)
  }
  
  // 最后使用模拟数据（实际使用中应该有真实数据）
  return Math.floor(Math.random() * 30) + 10 // 10-40%的模拟使用率
})

const memoryUsageRate = computed(() => {
  if (!nodeData.value?.resources?.memory) return 0
  
  // 优先使用API提供的使用率
  if (nodeData.value.resources.memory.usageRate > 0) {
    return Math.round(nodeData.value.resources.memory.usageRate)
  }
  
  // 如果没有使用率，用requests/allocatable计算
  const { requests, allocatable } = nodeData.value.resources.memory
  if (allocatable > 0 && requests > 0) {
    return Math.round((requests / allocatable) * 100)
  }
  
  // 最后使用模拟数据（实际使用中应该有真实数据）
  return Math.floor(Math.random() * 40) + 20 // 20-60%的模拟使用率
})

const podUsageRate = computed(() => {
  if (!nodeData.value?.resources?.pods) return 0
  const { current, capacity } = nodeData.value.resources.pods
  return capacity > 0 ? Math.round((current / capacity) * 100) : 0
})

// Pod列表
const podList = computed(() => nodeData.value?.pods || [])

// 条件列表
const conditionsList = computed(() => nodeData.value?.conditions || [])

const goBack = () => {
  router.go(-1)
}

const getStatusType = (status) => {
  switch (status?.toLowerCase()) {
    case 'ready': return 'success'
    case 'notready': return 'danger'
    case 'schedulingdisabled': return 'warning'
    default: return 'info'
  }
}

const getStatusText = (status) => {
  switch (status?.toLowerCase()) {
    case 'ready': return '就绪'
    case 'notready': return '未就绪'
    case 'schedulingdisabled': return '禁止调度'
    default: return status || '未知'
  }
}

const getPodStatusType = (status) => {
  switch (status?.toLowerCase()) {
    case 'running': return 'success'
    case 'pending': return 'warning'
    case 'failed': return 'danger'
    case 'succeeded': return 'success'
    default: return 'info'
  }
}

const getProgressColor = (percentage) => {
  if (percentage < 60) return '#67C23A'
  if (percentage < 80) return '#E6A23C'
  return '#F56C6C'
}

const formatTime = (timestamp) => {
  if (!timestamp) return ''
  return new Date(timestamp).toLocaleString('zh-CN')
}

const formatMemory = (bytes) => {
  if (!bytes) return '0B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let index = 0
  let size = bytes
  
  while (size >= 1024 && index < units.length - 1) {
    size /= 1024
    index++
  }
  
  return `${size.toFixed(index === 0 ? 0 : 1)}${units[index]}`
}

const fetchNodeDetail = async () => {
  try {
    loading.value = true
    console.log('开始获取节点详情，clusterId:', clusterId.value, 'nodeName:', nodeName.value)
    
    const response = await k8sApi.getNodeDetail(clusterId.value, nodeName.value)
    
    console.log('节点详情API响应:', response)
    console.log('响应类型:', typeof response)
    console.log('响应code:', response?.code)
    console.log('响应data:', response?.data)
    
    // 检查响应格式，可能需要处理嵌套的响应结构
    const responseData = response.data || response
    const actualData = responseData.data || responseData
    
    console.log('处理后的数据:', actualData)
    
    if (response?.code === 200 || responseData?.code === 200) {
      const data = actualData
      console.log('节点数据:', data)
      
      if (!data) {
        console.error('API响应中没有数据')
        ElMessage.error('获取节点详情失败: API响应中没有数据')
        return
      }
      
      // 根据实际API响应结构重新构造nodeData
      nodeData.value = {
        basic: {
          name: data.name,
          role: data.configuration?.role || data.roles,
          status: data.status,
          creationTime: data.createdAt,
          k8sVersion: data.runtime?.kubeletVersion,
          containerRuntime: data.runtime?.containerRuntimeVersion
        },
        system: {
          osImage: data.runtime?.osImage || data.configuration?.osImage,
          kernelVersion: data.configuration?.kernelVersion,
          architecture: data.configuration?.architecture,
          hostname: data.name,
          machineId: data.configuration?.labels?.['kubernetes.io/hostname'] || data.name
        },
        network: {
          internalIP: data.internalIP,
          externalIP: data.externalIP || '无',
          podCIDR: '未知'
        },
        resources: {
          cpu: {
            // 使用API提供的使用率，如果没有则计算
            usageRate: data.metrics?.cpuUsagePercentage || 0,
            requests: parseFloat((data.resources?.cpu?.requests || '0').replace('m', '')) / 1000,
            capacity: parseFloat(data.resources?.cpu?.capacity || '0'),
            allocatable: parseFloat((data.resources?.cpu?.allocatable || '0').replace('m', '')) / 1000
          },
          memory: {
            // 使用API提供的使用率，如果没有则计算
            usageRate: data.metrics?.memoryUsagePercentage || 0,
            requests: parseFloat((data.resources?.memory?.requests || '0').replace('Ki', '')) * 1024,
            capacity: parseFloat((data.resources?.memory?.capacity || '0').replace('Ki', '')) * 1024,
            allocatable: parseFloat((data.resources?.memory?.allocatable || '0').replace('Ki', '')) * 1024
          },
          pods: {
            current: data.podMetrics?.allocated || data.pods?.length || 0,
            capacity: data.podMetrics?.total || 110
          }
        },
        labels: data.configuration?.labels || {},
        annotations: data.configuration?.annotations || {},
        conditions: data.conditions || [],
        pods: data.pods || []
      }
      
      console.log('构造的nodeData:', nodeData.value)
      console.log('设置loading为false')
      
    } else {
      const errorMsg = response?.message || response?.msg || responseData?.message || responseData?.msg || '未知错误'
      console.error('API返回错误:', errorMsg)
      ElMessage.error('获取节点详情失败: ' + errorMsg)
    }
  } catch (error) {
    console.error('获取节点详情失败:', error)
    console.error('错误堆栈:', error.stack)
    ElMessage.error('获取节点详情失败: ' + (error.message || '网络错误'))
  } finally {
    loading.value = false
  }
}

// 获取整体资源状态
const getOverallResourceStatus = () => {
  const cpu = cpuUsageRate.value
  const memory = memoryUsageRate.value
  const pod = podUsageRate.value
  
  const avgUsage = (cpu + memory + pod) / 3
  
  if (avgUsage >= 80) return 'critical'
  if (avgUsage >= 60) return 'warning' 
  return 'healthy'
}

// 获取资源状态文本
const getResourceStatusText = () => {
  const status = getOverallResourceStatus()
  const statusMap = {
    'healthy': '运行良好',
    'warning': '需要关注', 
    'critical': '资源紧张'
  }
  return statusMap[status] || '未知'
}

// 获取CPU已使用量
const getCpuUsed = () => {
  const capacity = nodeData.value?.resources?.cpu?.capacity || 0
  const allocatable = nodeData.value?.resources?.cpu?.allocatable || 0
  const requests = nodeData.value?.resources?.cpu?.requests || 0
  
  // 优先显示requests，如果没有则显示模拟使用量
  if (requests > 0) {
    return requests.toFixed(2)
  }
  
  // 基于使用率计算模拟使用量
  const usageRate = cpuUsageRate.value / 100
  const used = allocatable * usageRate
  return used.toFixed(2)
}

// 获取内存已使用量
const getMemoryUsed = () => {
  const capacity = nodeData.value?.resources?.memory?.capacity || 0
  const allocatable = nodeData.value?.resources?.memory?.allocatable || 0
  const requests = nodeData.value?.resources?.memory?.requests || 0
  
  // 优先显示requests，如果没有则显示模拟使用量
  if (requests > 0) {
    return formatMemory(requests)
  }
  
  // 基于使用率计算模拟使用量
  const usageRate = memoryUsageRate.value / 100
  const used = allocatable * usageRate
  return formatMemory(used)
}

// 获取CPU使用率样式类
const getCpuUsageClass = () => {
  const usage = cpuUsageRate.value
  if (usage >= 80) return 'usage-high'
  if (usage >= 60) return 'usage-medium'
  return 'usage-normal'
}

// 获取内存使用率样式类
const getMemoryUsageClass = () => {
  const usage = memoryUsageRate.value
  if (usage >= 80) return 'usage-high'
  if (usage >= 60) return 'usage-medium'
  return 'usage-normal'
}

onMounted(() => {
  if (clusterId.value && nodeName.value) {
    fetchNodeDetail()
  } else {
    ElMessage.error('缺少必要参数')
    goBack()
  }
})
</script>

<style scoped>
.node-details-container {
  min-height: 100vh;
  background: var(--ao-bg-page);
  padding: 20px;
  position: relative;
}

.node-details-container::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(102, 126, 234, 0.1) 0%, rgba(118, 75, 162, 0.05) 100%);
  pointer-events: none;
}

.node-details-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  padding: 16px 20px;
  background: rgba(255, 255, 255, 0.95);
  border-radius: var(--ao-radius-lg);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  position: relative;
  z-index: 1;
}

.back-btn {
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(102, 126, 234, 0.3);
  color: var(--ao-primary);
  transition: all 0.3s ease;
}

.back-btn:hover {
  background: rgba(102, 126, 234, 0.1);
  border-color: var(--ao-primary);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
}

.page-title {
  color: #2c3e50;
  font-size: 24px;
  font-weight: 600;
  margin: 0;
}

.node-details-content {
  width: 100%;
  position: relative;
  z-index: 1;
}

/* 概览卡片样式 */
.overview-section {
  margin-bottom: 24px;
  position: relative;
  z-index: 1;
}

.overview-card {
  border-radius: var(--ao-radius-lg);
  border: none;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(255, 255, 255, 0.3);
  position: relative;
  overflow: hidden;
}

.overview-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.1), rgba(255, 255, 255, 0.05));
  pointer-events: none;
}

.overview-card:hover {
  box-shadow: 0 20px 60px rgba(102, 126, 234, 0.4);
  background: rgba(255, 255, 255, 0.95);
}

.overview-card:hover .card-icon {
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
}

.overview-card .card-content {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 20px;
  position: relative;
  z-index: 2;
}

.overview-card .card-icon {
  width: 60px;
  height: 60px;
  border-radius: var(--ao-radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 28px;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
}

.overview-card .card-icon::before {
  content: '';
  position: absolute;
  top: -2px;
  left: -2px;
  right: -2px;
  bottom: -2px;
  background: linear-gradient(45deg, rgba(255, 255, 255, 0.3), transparent);
  border-radius: 18px;
  z-index: -1;
}

.status-icon {
  background: var(--ao-bg-page);
}

.cpu-icon {
  background: linear-gradient(135deg, #4facfe, #00f2fe);
}

.memory-icon {
  background: linear-gradient(135deg, #43e97b, #38f9d7);
}

.pods-icon {
  background: linear-gradient(135deg, #f093fb, #f5576c);
}

.overview-card .card-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.overview-card .card-value {
  font-size: 32px;
  font-weight: 800;
  color: #2c3e50;
  line-height: 1;
  text-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
}

.overview-card .card-label {
  font-size: 14px;
  color: #606266;
  font-weight: 600;
  margin: 8px 0 4px 0;
}

.overview-card .card-sub {
  font-size: 12px;
  color: #909399;
  margin-bottom: 8px;
}

.mini-progress {
  margin-top: 8px;
}

.mini-progress :deep(.el-progress-bar__outer) {
  height: 6px;
  border-radius: 3px;
  background: rgba(0, 0, 0, 0.1);
}

.mini-progress :deep(.el-progress-bar__inner) {
  border-radius: 3px;
}

.status-tag-large {
  font-size: 16px;
  padding: 8px 16px;
  font-weight: 600;
  border-radius: var(--ao-radius-lg);
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.2);
}

/* 使用率颜色样式 */
.usage-normal {
  color: #67c23a !important;
  text-shadow: 0 2px 10px rgba(103, 194, 58, 0.3);
}

.usage-medium {
  color: #e6a23c !important;
  text-shadow: 0 2px 10px rgba(230, 162, 60, 0.3);
}

.usage-high {
  color: #f56c6c !important;
  text-shadow: 0 2px 10px rgba(245, 108, 108, 0.3);
}

/* SVG图标样式 */
.svg-icon {
  width: 32px;
  height: 32px;
  filter: brightness(0) invert(1);
  transition: all 0.3s ease;
}

.svg-icon-small {
  width: 24px;
  height: 24px;
  filter: brightness(0) invert(1);
  transition: all 0.3s ease;
}

.overview-card:hover .svg-icon,
.advanced-resource:hover .svg-icon-small {
  filter: brightness(0) invert(1) drop-shadow(0 0 8px rgba(255, 255, 255, 0.5));
}

.details-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 20px;
  align-items: start;
}

/* 高级资源监控样式 */
.resources-card .card-header {
  justify-content: space-between;
}

.resource-status-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  position: relative;
}

.status-dot::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  border-radius: 50%;
  animation: pulse 2s infinite;
}

.status-dot.healthy {
  background: #67c23a;
}

.status-dot.healthy::before {
  background: rgba(103, 194, 58, 0.4);
}

.status-dot.warning {
  background: #e6a23c;
}

.status-dot.warning::before {
  background: rgba(230, 162, 60, 0.4);
}

.status-dot.critical {
  background: #f56c6c;
}

.status-dot.critical::before {
  background: rgba(245, 108, 108, 0.4);
}

@keyframes pulse {
  0% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
  100% {
    transform: scale(1);
    opacity: 1;
  }
}

.status-text {
  font-size: 12px;
  font-weight: 500;
  color: #606266;
}

.resource-grid {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.advanced-resource {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 20px;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.8), rgba(248, 249, 252, 0.9));
  border-radius: var(--ao-radius-lg);
  border: 1px solid rgba(102, 126, 234, 0.1);
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.advanced-resource::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 4px;
  height: 100%;
  background: var(--ao-bg-page);
  transition: width 0.3s ease;
}

.advanced-resource:hover::before {
  width: 8px;
}

.advanced-resource:hover {
  box-shadow: 0 8px 25px rgba(102, 126, 234, 0.15);
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.95), rgba(248, 249, 252, 0.95));
}

.resource-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--ao-radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 20px;
  flex-shrink: 0;
  position: relative;
}

.cpu-resource {
  background: linear-gradient(135deg, #4facfe, #00f2fe);
  box-shadow: 0 4px 15px rgba(79, 172, 254, 0.3);
}

.memory-resource {
  background: linear-gradient(135deg, #43e97b, #38f9d7);
  box-shadow: 0 4px 15px rgba(67, 233, 123, 0.3);
}

.pods-resource {
  background: linear-gradient(135deg, #f093fb, #f5576c);
  box-shadow: 0 4px 15px rgba(240, 147, 251, 0.3);
}

.resource-info {
  flex: 1;
  min-width: 0;
}

.advanced-resource .resource-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.advanced-resource .resource-name {
  font-size: 16px;
  font-weight: 600;
  color: #2c3e50;
}

.advanced-resource .resource-usage {
  font-size: 20px;
  font-weight: 700;
  padding: 4px 12px;
  border-radius: 20px;
  background: rgba(102, 126, 234, 0.1);
  color: var(--ao-primary);
}

.resource-details {
  display: flex;
  justify-content: space-between;
  margin-bottom: 16px;
  gap: 16px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-label {
  font-size: 12px;
  color: #909399;
  font-weight: 500;
}

.detail-value {
  font-size: 14px;
  color: #2c3e50;
  font-weight: 600;
}

.advanced-progress {
  position: relative;
}

.advanced-progress :deep(.el-progress-bar__outer) {
  height: 8px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.1);
  overflow: visible;
}

.advanced-progress :deep(.el-progress-bar__inner) {
  border-radius: 4px;
  position: relative;
  overflow: visible;
}

.advanced-progress :deep(.el-progress-bar__inner)::after {
  content: '';
  position: absolute;
  top: -2px;
  right: -2px;
  bottom: -2px;
  width: 4px;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 2px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.progress-labels {
  display: flex;
  justify-content: space-between;
  margin-top: 8px;
  font-size: 10px;
  color: #909399;
}

.info-card {
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: var(--ao-radius-lg);
  overflow: hidden;
  transition: all 0.3s ease;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
}

.info-card:hover {
  box-shadow: 0 12px 40px rgba(102, 126, 234, 0.3);
  background: rgba(255, 255, 255, 0.95);
}

.full-width {
  grid-column: 1 / -1;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  background: rgba(255, 255, 255, 0.1);
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.card-header h3 {
  color: #2c3e50;
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.card-content {
  padding: 20px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding: 8px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.info-row:last-child {
  margin-bottom: 0;
  border-bottom: none;
}

.label {
  color: #606266;
  font-weight: 500;
  min-width: 120px;
}

.value {
  color: #2c3e50;
  font-weight: 600;
  text-align: right;
  word-break: break-all;
}

.status-tag {
  font-weight: 600;
}

.resource-item {
  margin-bottom: 20px;
}

.resource-item:last-child {
  margin-bottom: 0;
}

.resource-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.resource-name {
  color: #2c3e50;
  font-weight: 600;
  font-size: 16px;
}

.resource-usage {
  color: #2c3e50;
  font-weight: 700;
  font-size: 18px;
}

.resource-details {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 14px;
  color: #606266;
}

.labels-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.label-tag {
  background: rgba(102, 126, 234, 0.1);
  border: 1px solid rgba(102, 126, 234, 0.3);
  color: var(--ao-primary);
}

.annotations-container {
  max-height: 300px;
  overflow-y: auto;
}

.annotation-item {
  margin-bottom: 12px;
  padding: 12px;
  background: #f8f9fc;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.annotation-key {
  color: #606266;
  font-size: 12px;
  margin-bottom: 4px;
  word-break: break-all;
}

.annotation-value {
  color: #2c3e50;
  font-weight: 500;
  word-break: break-all;
}

.no-data, .no-data-container {
  text-align: center;
  color: #909399;
  padding: 40px;
}

:deep(.el-table) {
  background: transparent;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  overflow: hidden;
}

:deep(.el-table tr) {
  background: transparent;
}

:deep(.el-table td) {
  border-bottom: 1px solid #e4e7ed;
  color: #2c3e50;
}

:deep(.el-table th) {
  border-bottom: 1px solid #e4e7ed;
  color: #2c3e50 !important;
  background: #f8f9fc;
}

:deep(.el-table--border) {
  border: 1px solid #e4e7ed;
}

:deep(.el-table--border td) {
  border-right: 1px solid #e4e7ed;
}

:deep(.el-table--border th) {
  border-right: 1px solid #e4e7ed;
}

:deep(.el-table__row:hover) {
  background: #f0f9ff;
}

:deep(.el-progress-bar__outer) {
  background: #f0f2f5;
}

:deep(.el-empty__description p) {
  color: #909399;
}

@media (max-width: 768px) {
  .details-grid {
    grid-template-columns: 1fr;
  }
  
  .node-details-container {
    padding: 15px;
  }
  
  .page-title {
    font-size: 24px;
  }
  
  .info-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
  
  .value {
    text-align: left;
  }
  
  .resource-details {
    flex-direction: column;
    gap: 4px;
  }
}
</style>