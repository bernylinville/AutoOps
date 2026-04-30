<template>
  <div class="release-center">
    <el-card shadow="hover" class="release-card">
      <template #header>
        <div class="card-header">
          <span class="title">部署中心</span>
          <div class="header-actions">
            <el-button type="primary" size="small" v-authority="['deploy:request:create']" @click="createDialogVisible = true">
              新建申请
            </el-button>
            <el-button type="primary" size="small" @click="refreshAll">
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="部署申请" name="requests">
          <el-table :data="requestList" v-loading="loadingRequests" stripe>
            <el-table-column prop="requestNo" label="申请单号" min-width="180" />
            <el-table-column prop="mode" label="模式" width="100" />
            <el-table-column prop="resourceType" label="资源类型" width="120" />
            <el-table-column prop="releaseName" label="发布名" min-width="150" />
            <el-table-column prop="namespace" label="命名空间" min-width="160" />
            <el-table-column prop="approvalStatus" label="审批状态" width="120" />
            <el-table-column prop="approvalDispatchStatus" label="投递状态" width="120" />
            <el-table-column prop="executionStatus" label="执行状态" width="120" />
            <el-table-column prop="requesterDisplayName" label="申请人" width="120" />
            <el-table-column label="操作" fixed="right" min-width="220">
              <template #default="{ row }">
                <div class="actions">
                  <el-button
                    size="small"
                    type="primary"
                    text
                    v-authority="['deploy:request:approve']"
                    @click="handleSyncApproval(row)"
                  >
                    同步审批
                  </el-button>
                  <el-button
                    size="small"
                    type="warning"
                    text
                    v-authority="['deploy:request:approve']"
                    @click="handleRetryApproval(row)"
                  >
                    重发审批
                  </el-button>
                  <el-button
                    size="small"
                    type="success"
                    text
                    v-authority="['deploy:request:execute']"
                    @click="handleExecute(row)"
                  >
                    执行
                  </el-button>
                  <el-button
                    v-if="row.mode === 'gitops' && row.executionStatus === 'succeeded'"
                    size="small"
                    type="danger"
                    text
                    v-authority="['deploy:request:execute']"
                    @click="handleRollback(row)"
                  >
                    下线
                  </el-button>
                  <el-button
                    size="small"
                    type="info"
                    text
                    @click="showExecutions(row)"
                  >
                    执行记录
                  </el-button>
                  <el-button
                    size="small"
                    type="info"
                    text
                    @click="showNotifications(row)"
                  >
                    通知记录
                  </el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="部署目标" name="targets">
          <el-table :data="targetList" v-loading="loadingTargets" stripe>
            <el-table-column prop="name" label="目标名称" min-width="160" />
            <el-table-column prop="kubeClusterId" label="集群ID" width="100" />
            <el-table-column prop="envType" label="环境类型" width="100" />
            <el-table-column prop="gitOpsEnabled" label="GitOps" width="100">
              <template #default="{ row }">{{ row.gitOpsEnabled ? '启用' : '禁用' }}</template>
            </el-table-column>
            <el-table-column prop="directEnabled" label="Direct" width="100">
              <template #default="{ row }">{{ row.directEnabled ? '启用' : '禁用' }}</template>
            </el-table-column>
            <el-table-column prop="directNamespacePrefix" label="直连前缀" min-width="140" />
            <el-table-column prop="defaultTtlHours" label="默认TTL" width="100" />
            <el-table-column label="操作" fixed="right" min-width="300">
              <template #default="{ row }">
                <div class="actions">
                <el-button
                  size="small"
                  type="primary"
                  text
                  v-authority="['deploy:target:edit']"
                  @click="openTargetDialog(row)"
                >
                  编辑
                </el-button>
                <el-button
                  size="small"
                  type="primary"
                  text
                  v-authority="['deploy:target:edit']"
                  @click="handleValidateDirectCredential(row)"
                >
                  校验直连凭据
                </el-button>
                <el-button
                  size="small"
                  type="success"
                  text
                  v-authority="['deploy:target:edit']"
                  @click="handleValidateGitOpsRepo(row)"
                >
                  校验 GitOps 仓库
                </el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
          <div class="table-toolbar">
            <el-button
              size="small"
              type="primary"
              plain
              v-authority="['deploy:target:create']"
              @click="openTargetDialog()"
            >
              新建目标
            </el-button>
            <el-button
              size="small"
              type="success"
              plain
              v-authority="['deploy:target:view']"
              @click="handleValidateGitOpsWorkingTree"
            >
              校验 GitOps 工作树
            </el-button>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-dialog v-model="executionDialogVisible" title="执行记录" width="900px">
      <el-table :data="executionList" v-loading="loadingExecutions" stripe>
        <el-table-column prop="executorType" label="执行器" width="120" />
        <el-table-column prop="phase" label="阶段" width="120" />
        <el-table-column prop="status" label="状态" width="120" />
        <el-table-column prop="k8sNamespace" label="命名空间" min-width="160" />
        <el-table-column prop="detailJson" label="详情" min-width="320" show-overflow-tooltip />
      </el-table>
    </el-dialog>

    <el-dialog v-model="notificationDialogVisible" title="通知记录" width="980px">
      <el-table :data="notificationList" v-loading="loadingNotifications" stripe>
        <el-table-column prop="channel" label="渠道" width="140" />
        <el-table-column prop="stage" label="阶段" width="140" />
        <el-table-column prop="status" label="状态" width="120" />
        <el-table-column prop="sentAt" label="发送时间" min-width="180" />
        <el-table-column prop="errorMessage" label="错误信息" min-width="220" show-overflow-tooltip />
        <el-table-column prop="payloadJson" label="负载快照" min-width="280" show-overflow-tooltip />
      </el-table>
    </el-dialog>

    <el-dialog v-model="createDialogVisible" title="新建部署申请" width="720px">
      <el-form :model="createForm" label-width="120px">
        <el-form-item label="部署模式">
          <el-select v-model="createForm.mode" style="width: 100%">
            <el-option label="Direct" value="direct" />
            <el-option label="GitOps（存量兼容）" value="gitops" :disabled="isCreateTargetDevTest()" />
          </el-select>
          <div class="form-helper">开发/测试新服务默认使用 Direct，不依赖 GitOps/ArgoCD。</div>
        </el-form-item>
        <el-form-item label="资源类型">
          <el-select v-model="createForm.resourceType" style="width: 100%">
            <el-option label="Deployment" value="deployment" />
            <el-option label="Pod" value="pod" />
            <el-option label="Service" value="service" />
          </el-select>
        </el-form-item>
        <el-form-item label="部署目标">
          <el-select v-model="createForm.clusterTargetId" style="width: 100%">
            <el-option v-for="item in targetList" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="发布名称">
          <el-input v-model="createForm.releaseName" />
        </el-form-item>
        <el-form-item label="命名空间">
          <el-input v-model="createForm.namespace" placeholder="可留空，后端自动生成" />
        </el-form-item>
        <el-form-item label="镜像">
          <el-input v-model="createForm.image" />
        </el-form-item>
        <el-form-item label="副本数">
          <el-input-number v-model="createForm.replicas" :min="1" />
        </el-form-item>
        <el-form-item label="创建 Service">
          <el-switch v-model="createForm.serviceEnabled" />
        </el-form-item>
        <el-form-item v-if="createForm.serviceEnabled" label="Service 类型">
          <el-select v-model="createForm.serviceType" style="width: 100%">
            <el-option label="ClusterIP" value="ClusterIP" />
            <el-option label="NodePort" value="NodePort" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="createForm.serviceEnabled" label="Service 端口">
          <el-input-number v-model="createForm.servicePort" :min="1" />
        </el-form-item>
        <el-form-item v-if="createForm.serviceEnabled" label="目标端口">
          <el-input-number v-model="createForm.targetPort" :min="1" />
        </el-form-item>
        <el-form-item v-if="createForm.mode === 'direct'" label="TTL(小时)">
          <el-input-number v-model="createForm.ttlHours" :min="1" />
        </el-form-item>
        <el-form-item label="审批人ID">
          <el-input-number v-model="createForm.approverAdminId" :min="1" />
        </el-form-item>
        <el-form-item label="申请原因">
          <el-input v-model="createForm.reason" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreateRequest">提交</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="targetDialogVisible" :title="targetDialogTitle" width="720px">
      <el-form :model="targetForm" label-width="140px">
        <el-form-item label="目标名称" required>
          <el-input v-model="targetForm.name" placeholder="请输入目标名称" />
        </el-form-item>
        <el-form-item label="集群" required>
          <el-select v-model="targetForm.kubeClusterId" style="width: 100%" placeholder="请选择集群">
            <el-option v-for="item in clusterList" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="环境类型">
          <el-select v-model="targetForm.envType" style="width: 100%">
            <el-option label="开发" value="dev" />
            <el-option label="测试" value="test" />
            <el-option label="预发" value="staging" />
            <el-option label="生产" value="prod" />
          </el-select>
        </el-form-item>
        <el-form-item label="GitOps 启用">
          <el-switch v-model="targetForm.gitOpsEnabled" />
          <div class="form-helper">开发/测试新服务保持关闭；仅存量 GitOps 场景需要启用。</div>
        </el-form-item>
        <el-form-item label="Direct 启用">
          <el-switch v-model="targetForm.directEnabled" />
        </el-form-item>
        <el-form-item label="直连凭据引用">
          <el-input v-model="targetForm.directKubeconfigRef" placeholder="account:<kubeconfig-id>" />
          <div class="form-helper">格式: account:&lt;账户ID或别名&gt;</div>
        </el-form-item>
        <el-form-item label="直连命名空间前缀">
          <el-input v-model="targetForm.directNamespacePrefix" placeholder="ao-direct" />
        </el-form-item>
        <el-form-item label="默认TTL(小时)">
          <el-input-number v-model="targetForm.defaultTtlHours" :min="1" />
        </el-form-item>
        <el-form-item label="Harbor 服务器ID">
          <el-input-number v-model="targetForm.harborServerId" :min="1" controls-position="right" />
        </el-form-item>
        <el-form-item label="Jenkins 服务器ID">
          <el-input-number v-model="targetForm.jenkinsServerId" :min="1" controls-position="right" />
        </el-form-item>
        <el-form-item label="GitOps 仓库">
          <el-input v-model="targetForm.gitOpsRepo" placeholder="请输入GitOps仓库地址" />
        </el-form-item>
        <el-form-item label="GitOps 分支">
          <el-input v-model="targetForm.gitOpsBranch" placeholder="请输入GitOps分支" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="targetForm.description" type="textarea" :rows="3" placeholder="请输入说明" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="targetDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingTarget" @click="handleSubmitTarget">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import deployAPI from '@/api/deploy'
import k8sAPI from '@/api/k8s'

const activeTab = ref('requests')
const loadingRequests = ref(false)
const loadingTargets = ref(false)
const loadingExecutions = ref(false)
const loadingNotifications = ref(false)
const requestList = ref([])
const targetList = ref([])
const executionList = ref([])
const notificationList = ref([])
const executionDialogVisible = ref(false)
const notificationDialogVisible = ref(false)
const createDialogVisible = ref(false)
const creating = ref(false)
const createForm = ref({
  mode: 'direct',
  resourceType: 'deployment',
  clusterTargetId: undefined,
  releaseName: '',
  namespace: '',
  image: 'nginx:1.27.4-alpine',
  replicas: 1,
  serviceEnabled: true,
  serviceType: 'ClusterIP',
  servicePort: 80,
  targetPort: 80,
  ttlHours: 72,
  approverAdminId: undefined,
  reason: ''
})

const targetDialogVisible = ref(false)
const targetDialogTitle = ref('新建部署目标')
const savingTarget = ref(false)
const editingTargetId = ref(null)
const clusterList = ref([])
const targetForm = ref({
  name: '',
  kubeClusterId: undefined,
  envType: 'dev',
  gitOpsEnabled: false,
  directEnabled: true,
  directKubeconfigRef: '',
  directNamespacePrefix: 'ao-direct',
  defaultTtlHours: 72,
  harborServerId: undefined,
  jenkinsServerId: undefined,
  gitOpsRepo: '',
  gitOpsBranch: '',
  description: ''
})

const loadRequests = async () => {
  try {
    loadingRequests.value = true
    const response = await deployAPI.getDeployRequests({ page: 1, pageSize: 50 })
    const responseData = response.data || response
    const pageData = responseData.data || {}
    requestList.value = Array.isArray(pageData.list) ? pageData.list : []
  } catch (error) {
    console.error(error)
    ElMessage.error('获取部署申请失败')
  } finally {
    loadingRequests.value = false
  }
}

const loadTargets = async () => {
  try {
    loadingTargets.value = true
    const response = await deployAPI.getClusterTargets()
    const responseData = response.data || response
    targetList.value = responseData.data || []
  } catch (error) {
    console.error(error)
    ElMessage.error('获取部署目标失败')
  } finally {
    loadingTargets.value = false
  }
}

const loadClusters = async () => {
  try {
    const response = await k8sAPI.getClusterList()
    const responseData = response.data || response
    clusterList.value = responseData.data || []
  } catch (error) {
    console.error(error)
    ElMessage.error('获取集群列表失败')
  }
}

const resetTargetForm = () => {
  targetForm.value = {
    name: '',
    kubeClusterId: undefined,
    envType: 'dev',
    gitOpsEnabled: false,
    directEnabled: true,
    directKubeconfigRef: '',
    directNamespacePrefix: 'ao-direct',
    defaultTtlHours: 72,
    harborServerId: undefined,
    jenkinsServerId: undefined,
    gitOpsRepo: '',
    gitOpsBranch: '',
    description: ''
  }
}

const isDevTestEnvType = (envType) => ['dev', 'test'].includes(String(envType || '').trim().toLowerCase())

const selectedCreateTarget = () => targetList.value.find(item => item.id === createForm.value.clusterTargetId)

const isCreateTargetDevTest = () => isDevTestEnvType(selectedCreateTarget()?.envType)

const openTargetDialog = async (row = null) => {
  await loadClusters()
  if (row) {
    targetDialogTitle.value = '编辑部署目标'
    editingTargetId.value = row.id
    try {
      const response = await deployAPI.getClusterTargetById(row.id)
      const data = unwrapResult(response)
      targetForm.value = {
        name: data.name || '',
        kubeClusterId: data.kubeClusterId,
        envType: data.envType || 'dev',
        gitOpsEnabled: data.gitOpsEnabled ?? true,
        directEnabled: data.directEnabled ?? true,
        directKubeconfigRef: data.directKubeconfigRef || '',
        directNamespacePrefix: data.directNamespacePrefix || 'ao-direct',
        defaultTtlHours: data.defaultTtlHours ?? 72,
        harborServerId: data.harborServerId || undefined,
        jenkinsServerId: data.jenkinsServerId || undefined,
        gitOpsRepo: data.gitOpsRepo || '',
        gitOpsBranch: data.gitOpsBranch || '',
        description: data.description || ''
      }
    } catch (error) {
      console.error(error)
      return
    }
  } else {
    targetDialogTitle.value = '新建部署目标'
    editingTargetId.value = null
    resetTargetForm()
  }
  targetDialogVisible.value = true
}

const handleSubmitTarget = async () => {
  if (!targetForm.value.name) {
    ElMessage.warning('请输入目标名称')
    return
  }
  if (!targetForm.value.kubeClusterId) {
    ElMessage.warning('请选择集群')
    return
  }
  if (isDevTestEnvType(targetForm.value.envType) && !targetForm.value.directEnabled) {
    ElMessage.warning('开发/测试部署目标必须启用 Direct')
    return
  }

  try {
    savingTarget.value = true
    const payload = {
      name: targetForm.value.name,
      kubeClusterId: targetForm.value.kubeClusterId,
      envType: targetForm.value.envType,
      gitOpsEnabled: targetForm.value.gitOpsEnabled,
      directEnabled: targetForm.value.directEnabled,
      directKubeconfigRef: targetForm.value.directKubeconfigRef || undefined,
      directNamespacePrefix: targetForm.value.directNamespacePrefix || undefined,
      defaultTtlHours: targetForm.value.defaultTtlHours,
      harborServerId: targetForm.value.harborServerId || undefined,
      jenkinsServerId: targetForm.value.jenkinsServerId || undefined,
      gitOpsRepo: targetForm.value.gitOpsRepo || undefined,
      gitOpsBranch: targetForm.value.gitOpsBranch || undefined,
      description: targetForm.value.description || undefined
    }

    if (editingTargetId.value) {
      unwrapResult(await deployAPI.updateClusterTarget(editingTargetId.value, payload))
      ElMessage.success('部署目标已更新')
    } else {
      unwrapResult(await deployAPI.createClusterTarget(payload))
      ElMessage.success('部署目标已创建')
    }
    targetDialogVisible.value = false
    await loadTargets()
  } catch (error) {
    console.error(error)
    if (!error.__shown) {
      ElMessage.error(error.message || '保存部署目标失败')
    }
  } finally {
    savingTarget.value = false
  }
}

const refreshAll = async () => {
  await Promise.all([loadRequests(), loadTargets()])
}

const unwrapResult = (response) => {
  const payload = response?.data || response || {}
  if (payload.code === 200 || payload.success) {
    return payload.data
  }
  const error = new Error(payload.message || '请求失败')
  error.__shown = true
  ElMessage.error(error.message)
  throw error
}

const handleSyncApproval = async (row) => {
  unwrapResult(await deployAPI.syncApprovalStatus(row.id))
  ElMessage.success('已触发审批状态同步')
  await loadRequests()
}

const handleRetryApproval = async (row) => {
  unwrapResult(await deployAPI.retryApprovalDispatch(row.id))
  ElMessage.success('已触发审批重发')
  await loadRequests()
}

const handleExecute = async (row) => {
  unwrapResult(await deployAPI.executeDeployRequest(row.id, {}))
  ElMessage.success('已触发执行')
  await loadRequests()
}

const handleRollback = async (row) => {
  await ElMessageBox.confirm(
    `将删除 GitOps release 文件并触发 ArgoCD 回收 ${row.releaseName}，是否继续？`,
    '确认下线',
    { type: 'warning' }
  )
  unwrapResult(await deployAPI.rollbackDeployRequest(row.id))
  ElMessage.success('已触发下线')
  await loadRequests()
}

const handleValidateDirectCredential = async (row) => {
  const data = unwrapResult(await deployAPI.validateDirectCredential(row.id))
  const message = data?.message || '校验完成'
  ElMessage.success(message)
}

const handleValidateGitOpsRepo = async (row) => {
  const data = unwrapResult(await deployAPI.validateGitOpsRepo(row.id))
  const message = data?.message || '校验完成'
  ElMessage.success(message)
}

const handleValidateGitOpsWorkingTree = async () => {
  const data = unwrapResult(await deployAPI.validateGitOpsWorkingTree())
  const message = data?.message || '校验完成'
  ElMessage.success(message)
}

const handleCreateRequest = async () => {
  if (createForm.value.mode === 'gitops' && isCreateTargetDevTest()) {
    ElMessage.warning('开发/测试新服务请使用 Direct 部署，不走 GitOps/ArgoCD')
    return
  }

  try {
    creating.value = true
    const payload = {
      mode: createForm.value.mode,
      resourceType: createForm.value.resourceType,
      clusterTargetId: createForm.value.clusterTargetId,
      releaseName: createForm.value.releaseName,
      namespace: createForm.value.namespace || undefined,
      image: createForm.value.image,
      replicas: createForm.value.replicas,
      serviceEnabled: createForm.value.serviceEnabled,
      serviceType: createForm.value.serviceEnabled ? createForm.value.serviceType : undefined,
      servicePort: createForm.value.serviceEnabled ? createForm.value.servicePort : undefined,
      targetPort: createForm.value.serviceEnabled ? createForm.value.targetPort : undefined,
      ttlHours: createForm.value.mode === 'direct' ? createForm.value.ttlHours : undefined,
      approverAdminId: createForm.value.approverAdminId || undefined,
      reason: createForm.value.reason
    }
    unwrapResult(await deployAPI.createDeployRequest(payload))
    ElMessage.success('部署申请已创建')
    createDialogVisible.value = false
    await loadRequests()
  } catch (error) {
    console.error(error)
    if (!error.__shown) {
      ElMessage.error(error.message || '创建部署申请失败')
    }
  } finally {
    creating.value = false
  }
}

const showExecutions = async (row) => {
  executionDialogVisible.value = true
  try {
    loadingExecutions.value = true
    const response = await deployAPI.getExecutionRecords(row.id)
    const responseData = response.data || response
    executionList.value = responseData.data || []
  } catch (error) {
    console.error(error)
    ElMessage.error('获取执行记录失败')
  } finally {
    loadingExecutions.value = false
  }
}

const showNotifications = async (row) => {
  notificationDialogVisible.value = true
  try {
    loadingNotifications.value = true
    const response = await deployAPI.getDeployNotifications(row.id)
    const responseData = response.data || response
    notificationList.value = responseData.data || []
  } catch (error) {
    console.error(error)
    ElMessage.error('获取通知记录失败')
  } finally {
    loadingNotifications.value = false
  }
}

onMounted(() => {
  refreshAll()
})
</script>

<style scoped>
.release-center {
  padding: 16px;
}

.release-card {
  border-radius: 12px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.title {
  font-size: 18px;
  font-weight: 600;
}

.actions {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.table-toolbar {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}

.form-helper {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
