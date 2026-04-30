<template>
  <el-card class="content-card" shadow="hover">
    <template #header>
      <div class="card-header">
        <div class="card-title">
          <el-icon><Grid /></el-icon>
          <span>集群内容信息</span>
        </div>
        <div class="card-actions">
          <el-select
            v-model="selectedNamespace"
            size="small"
            filterable
            placeholder="选择命名空间"
            style="width: 220px"
            :loading="namespaceLoading"
            @change="fetchNamespaceContent"
          >
            <el-option
              v-for="namespace in namespaces"
              :key="namespace.name"
              :label="namespace.name"
              :value="namespace.name"
            />
          </el-select>
          <el-button size="small" :icon="Refresh" :loading="contentLoading" @click="fetchAll">
            刷新
          </el-button>
        </div>
      </div>
    </template>

    <div class="content-summary">
      <div class="summary-item">
        <div class="summary-label">命名空间</div>
        <div class="summary-value">{{ namespaces.length }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">当前工作负载</div>
        <div class="summary-value">{{ workloads.length }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">当前 Pod</div>
        <div class="summary-value">{{ pods.length }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">当前 Service</div>
        <div class="summary-value">{{ services.length }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">最近事件</div>
        <div class="summary-value">{{ events.length }}</div>
      </div>
    </div>

    <el-row :gutter="16" v-loading="contentLoading">
      <el-col :span="12">
        <el-card class="inner-card" shadow="never">
          <template #header>
            <div class="inner-header">
              <span>工作负载</span>
              <el-tag size="small" type="info">{{ workloads.length }}</el-tag>
            </div>
          </template>
          <el-table :data="workloads.slice(0, 8)" size="small" max-height="320">
            <el-table-column prop="name" label="名称" min-width="180" show-overflow-tooltip />
            <el-table-column prop="type" label="类型" width="120" />
            <el-table-column prop="status" label="状态" width="110">
              <template #default="{ row }">
                <el-tag size="small" :type="getWorkloadStatusTag(row.status)">
                  {{ row.status || 'Unknown' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="副本" width="90">
              <template #default="{ row }">
                {{ formatReplicas(row) }}
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!workloads.length" description="当前命名空间暂无工作负载" :image-size="72" />
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="inner-card" shadow="never">
          <template #header>
            <div class="inner-header">
              <span>Pods</span>
              <el-tag size="small" type="info">{{ pods.length }}</el-tag>
            </div>
          </template>
          <el-table :data="pods.slice(0, 8)" size="small" max-height="320">
            <el-table-column prop="name" label="名称" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">
                <el-button type="primary" link @click="goToPod(row.name)">
                  {{ row.name }}
                </el-button>
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="110">
              <template #default="{ row }">
                <el-tag size="small" :type="getPodStatusTag(row.status)">
                  {{ row.status || row.phase || 'Unknown' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="nodeName" label="节点" min-width="130" show-overflow-tooltip />
            <el-table-column prop="runningTime" label="运行时长" width="110" />
          </el-table>
          <el-empty v-if="!pods.length" description="当前命名空间暂无 Pod" :image-size="72" />
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="inner-card" shadow="never">
          <template #header>
            <div class="inner-header">
              <span>Services</span>
              <el-tag size="small" type="info">{{ services.length }}</el-tag>
            </div>
          </template>
          <el-table :data="services.slice(0, 8)" size="small" max-height="320">
            <el-table-column prop="name" label="名称" min-width="160" show-overflow-tooltip />
            <el-table-column prop="type" label="类型" width="110" />
            <el-table-column prop="clusterIP" label="ClusterIP" min-width="130" show-overflow-tooltip />
            <el-table-column label="端口" min-width="150" show-overflow-tooltip>
              <template #default="{ row }">
                {{ formatServicePorts(row.ports) }}
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!services.length" description="当前命名空间暂无 Service" :image-size="72" />
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="inner-card" shadow="never">
          <template #header>
            <div class="inner-header">
              <span>最近事件</span>
              <el-tag size="small" type="info">{{ events.length }}</el-tag>
            </div>
          </template>
          <el-table :data="events.slice(0, 8)" size="small" max-height="320">
            <el-table-column prop="type" label="类型" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.type === 'Warning' ? 'warning' : 'success'">
                  {{ row.type || 'Normal' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" width="120" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="220" show-overflow-tooltip />
            <el-table-column prop="lastTime" label="时间" width="160">
              <template #default="{ row }">
                {{ formatTime(row.lastTime || row.lastTimestamp) }}
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!events.length" description="当前命名空间暂无事件" :image-size="72" />
        </el-card>
      </el-col>
    </el-row>
  </el-card>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Grid, Refresh } from '@element-plus/icons-vue'
import k8sApi from '@/api/k8s'

const props = defineProps({
  clusterId: {
    type: [String, Number],
    required: true
  }
})

const router = useRouter()
const namespaceLoading = ref(false)
const contentLoading = ref(false)
const namespaces = ref([])
const selectedNamespace = ref('')
const workloads = ref([])
const pods = ref([])
const services = ref([])
const events = ref([])

const normalizedClusterId = computed(() => {
  if (props.clusterId === undefined || props.clusterId === null || props.clusterId === '') {
    return ''
  }
  return String(props.clusterId)
})

const pickDefaultNamespace = (items) => {
  if (!items.length) return ''
  const exactDefault = items.find((item) => item.name === 'default')
  if (exactDefault) return exactDefault.name

  const preferred = items.find((item) => !['kube-system', 'kube-public', 'kube-node-lease'].includes(item.name))
  return (preferred || items[0]).name
}

const parseSuccessData = (response) => {
  const payload = response?.data || response || {}
  if (payload.code === 200 || payload.success) {
    return payload.data ?? {}
  }
  throw new Error(payload.message || '请求失败')
}

const fetchNamespaces = async () => {
  if (!normalizedClusterId.value) return

  try {
    namespaceLoading.value = true
    const data = parseSuccessData(await k8sApi.getNamespaces(normalizedClusterId.value))
    const list = Array.isArray(data.namespaces) ? data.namespaces : Array.isArray(data) ? data : []
    namespaces.value = list.map((item) => ({
      name: item.name,
      status: item.status || 'Unknown',
      labels: item.labels || {}
    }))

    if (!selectedNamespace.value || !namespaces.value.some((item) => item.name === selectedNamespace.value)) {
      selectedNamespace.value = pickDefaultNamespace(namespaces.value)
    }
  } catch (error) {
    console.error('获取命名空间失败:', error)
    namespaces.value = []
    selectedNamespace.value = ''
    ElMessage.error('获取集群内容的命名空间列表失败')
  } finally {
    namespaceLoading.value = false
  }
}

const fetchNamespaceContent = async () => {
  if (!normalizedClusterId.value || !selectedNamespace.value) {
    workloads.value = []
    pods.value = []
    services.value = []
    events.value = []
    return
  }

  contentLoading.value = true
  try {
    const [workloadResp, podResp, serviceResp, eventResp] = await Promise.all([
      k8sApi.getWorkloadList(normalizedClusterId.value, selectedNamespace.value, { type: 'all' }).catch(() => null),
      k8sApi.getPodList(normalizedClusterId.value, selectedNamespace.value).catch(() => null),
      k8sApi.getServiceList(normalizedClusterId.value, selectedNamespace.value).catch(() => null),
      k8sApi.getNamespaceEvents(normalizedClusterId.value, selectedNamespace.value).catch(() => null)
    ])

    const workloadData = workloadResp ? parseSuccessData(workloadResp) : {}
    const podData = podResp ? parseSuccessData(podResp) : {}
    const serviceData = serviceResp ? parseSuccessData(serviceResp) : {}
    const eventData = eventResp ? parseSuccessData(eventResp) : {}

    workloads.value = Array.isArray(workloadData.workloads) ? workloadData.workloads : []
    pods.value = Array.isArray(podData.pods) ? podData.pods : []
    services.value = Array.isArray(serviceData.services) ? serviceData.services : []
    events.value = Array.isArray(eventData.events) ? eventData.events : []
  } catch (error) {
    console.error('获取命名空间内容失败:', error)
    workloads.value = []
    pods.value = []
    services.value = []
    events.value = []
    ElMessage.error('获取命名空间内容信息失败')
  } finally {
    contentLoading.value = false
  }
}

const fetchAll = async () => {
  await fetchNamespaces()
  await fetchNamespaceContent()
}

const formatReplicas = (row) => {
  const ready = row.readyReplicas ?? 0
  const total = row.replicas ?? 0
  if (!ready && !total) return '-'
  return `${ready}/${total}`
}

const formatServicePorts = (ports = []) => {
  if (!Array.isArray(ports) || !ports.length) return '-'
  return ports
    .map((port) => {
      const protocol = port.protocol || 'TCP'
      return `${port.port}${port.targetPort ? `→${port.targetPort}` : ''}/${protocol}`
    })
    .join(', ')
}

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN')
}

const getWorkloadStatusTag = (status) => {
  if (['Running', 'Available', 'Active'].includes(status)) return 'success'
  if (['Pending', 'Progressing'].includes(status)) return 'warning'
  if (['Failed', 'CrashLoopBackOff', 'Error'].includes(status)) return 'danger'
  return 'info'
}

const getPodStatusTag = (status) => {
  if (['Running', 'Succeeded'].includes(status)) return 'success'
  if (['Pending', 'ContainerCreating'].includes(status)) return 'warning'
  if (['Failed', 'ImagePullBackOff', 'CrashLoopBackOff', 'ErrImagePull'].includes(status)) return 'danger'
  return 'info'
}

const goToPod = (podName) => {
  router.push(`/k8s/pod/${normalizedClusterId.value}/${selectedNamespace.value}/${podName}`)
}

watch(
  () => normalizedClusterId.value,
  async (value) => {
    if (!value) return
    await fetchAll()
  },
  { immediate: true }
)
</script>

<style scoped>
.content-card {
  margin-bottom: 24px;
  border-radius: 8px;
}

.card-header,
.inner-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #303133;
}

.card-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.content-summary {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.summary-item {
  padding: 14px 16px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background: #fafbfc;
}

.summary-label {
  font-size: 12px;
  color: #909399;
  margin-bottom: 6px;
}

.summary-value {
  font-size: 24px;
  font-weight: 600;
  color: #303133;
  line-height: 1;
}

.inner-card {
  margin-bottom: 16px;
  min-height: 395px;
}

@media (max-width: 1400px) {
  .content-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
