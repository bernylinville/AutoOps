<template>
  <!-- 命名空间列表表格 -->
  <div class="table-section">
    <el-table
      :data="tableData"
      v-loading="loading"
      stripe
      style="width: 100%"
      class="namespace-table"
    >
      <el-table-column prop="name" label="命名空间名称" min-width="180">
        <template #default="{ row }">
          <div class="namespace-name-container">
            <img src="@/assets/image/k8s.svg" alt="k8s" class="k8s-icon" />
            <el-button
              type="primary"
              link
              @click="viewNamespaceDetail(row)"
              class="namespace-name-link"
            >
              {{ row.name }}
            </el-button>
          </div>
        </template>
      </el-table-column>
      
      <el-table-column prop="status" label="状态" min-width="100">
        <template #default="{ row }">
          <el-tag
            :type="getStatusTag(row.status)"
            size="small"
            effect="dark"
          >
            {{ getStatusText(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      
      <el-table-column prop="age" label="创建时间" min-width="100">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ row.age }}</el-tag>
        </template>
      </el-table-column>

      <el-table-column label="标签" min-width="100" align="center">
        <template #default="{ row }">
          <div class="label-container">
            <el-badge :value="getLabelCount(row.labels)" :max="99" class="label-badge">
              <el-button
                type="text"
                size="small"
                circle
                @click="viewNamespaceLabels(row)"
                class="label-icon-button"
              >
                <img src="@/assets/image/标签.svg" alt="标签" width="14" height="14" />
              </el-button>
            </el-badge>
          </div>
        </template>
      </el-table-column>
      
      <el-table-column label="资源统计" min-width="160">
        <template #default="{ row }">
          <div class="resource-stats">
            <div class="resource-item">
              <span class="resource-label">Pod:</span>
              <span class="resource-value">{{ row.resourceCount.pods }}</span>
            </div>
            <div class="resource-item">
              <span class="resource-label">Service:</span>
              <span class="resource-value">{{ row.resourceCount.services }}</span>
            </div>
            <div class="resource-item">
              <span class="resource-label">Secret:</span>
              <span class="resource-value">{{ row.resourceCount.secrets }}</span>
            </div>
            <div class="resource-item">
              <span class="resource-label">ConfigMap:</span>
              <span class="resource-value">{{ row.resourceCount.configMaps }}</span>
            </div>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="配额/限制" min-width="120" align="center">
        <template #default="{ row }">
          <div class="quota-limit-info">
            <el-badge :value="row.resourceQuotas.length" :max="99" class="quota-badge">
              <el-tag size="small" type="warning">配额</el-tag>
            </el-badge>
            <el-badge :value="row.limitRanges.length" :max="99" class="limit-badge">
              <el-tag size="small" type="info">限制</el-tag>
            </el-badge>
          </div>
        </template>
      </el-table-column>
      
      <el-table-column label="操作" width="300" fixed="right">
        <template #default="{ row }">
          <div class="operation-buttons">
            <el-tooltip content="查看详情" placement="top">
              <el-button
                type="primary"
                :icon="View"
                size="small"
                circle
                v-authority="['k8s:namespace:details']"
                @click="viewNamespaceDetail(row)"
              />
            </el-tooltip>
            
            <el-tooltip content="资源配额" placement="top">
              <el-button
                type="warning"
                :icon="Monitor"
                size="small"
                circle
                v-authority="['k8s:namespace:setup']"
                @click="manageResourceQuotas(row)"
              />
            </el-tooltip>
            
            <el-tooltip content="资源限制" placement="top">
              <el-button
                type="info"
                :icon="Setting"
                size="small"
                circle
                v-authority="['k8s:namespace:restriction']"
                @click="manageLimitRanges(row)"
              />
            </el-tooltip>
            
            <el-tooltip :content="isSystemNamespace(row.name) ? '系统命名空间不可删除' : '删除命名空间'" placement="top">
              <el-button
                type="danger"
                :icon="Delete"
                size="small"
                circle
                v-authority="['k8s:namespace:delete']"
                :disabled="isSystemNamespace(row.name)"
                @click="deleteNamespace(row)"
              />
            </el-tooltip>
          </div>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import {
  View,
  Delete,
  Monitor,
  Setting
} from '@element-plus/icons-vue'

// Props
const props = defineProps({
  tableData: {
    type: Array,
    default: () => []
  },
  loading: {
    type: Boolean,
    default: false
  }
})

// Emits
const emit = defineEmits([
  'viewNamespaceDetail',
  'viewNamespaceLabels', 
  'manageResourceQuotas',
  'manageLimitRanges',
  'deleteNamespace'
])

// 获取状态标签样式
const getStatusTag = (status) => {
  const tagMap = {
    'Active': 'success',
    'Terminating': 'warning',
    'Unknown': 'info'
  }
  return tagMap[status] || 'info'
}

// 获取状态文本
const getStatusText = (status) => {
  const textMap = {
    'Active': '活跃',
    'Terminating': '终止中',
    'Unknown': '未知'
  }
  return textMap[status] || '未知'
}

// 获取标签数量（过滤系统标签）
const getLabelCount = (labels) => {
  if (!labels) return 0
  
  const systemLabelPrefixes = [
    'kubernetes.io/',
    'beta.kubernetes.io/',
    'app.kubernetes.io/'
  ]
  
  return Object.keys(labels).filter(key => 
    !systemLabelPrefixes.some(prefix => key.startsWith(prefix))
  ).length
}

// 判断是否为系统命名空间
const isSystemNamespace = (namespaceName) => {
  const systemNamespaces = [
    'default',
    'kube-system', 
    'kube-public',
    'kube-node-lease'
  ]
  return systemNamespaces.includes(namespaceName)
}

// 事件处理函数
const viewNamespaceDetail = (row) => {
  emit('viewNamespaceDetail', row)
}

const viewNamespaceLabels = (row) => {
  emit('viewNamespaceLabels', row)
}

const manageResourceQuotas = (row) => {
  emit('manageResourceQuotas', row)
}

const manageLimitRanges = (row) => {
  emit('manageLimitRanges', row)
}

const deleteNamespace = (row) => {
  emit('deleteNamespace', row)
}
</script>

<style scoped>
.table-section {
  margin-top: 20px;
}

.namespace-table {
  border-radius: var(--ao-radius-lg);
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}

.namespace-table :deep(.el-table__header) {
  background: var(--ao-bg-page);
}

.namespace-table :deep(.el-table__header th) {
  background: transparent !important;
  color: #2c3e50 !important;
  font-weight: 700 !important;
  border-bottom: none;
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.8);
}

.namespace-table :deep(.el-table__header th .cell) {
  color: #2c3e50 !important;
  font-weight: 700 !important;
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.8);
}

.namespace-table :deep(.el-table__row) {
  transition: all 0.3s ease;
}

.namespace-table :deep(.el-table__row:hover) {
  background-color: rgba(64, 158, 255, 0.05) !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.namespace-name-container {
  display: flex;
  align-items: center;
  gap: 10px;
}

.k8s-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.1));
}

.namespace-name-link {
  font-weight: 600;
  color: var(--ao-primary);
  text-decoration: none;
  transition: all 0.3s ease;
}

.namespace-name-link:hover {
  color: #764ba2;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.resource-stats {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
}

.resource-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.resource-label {
  font-size: 12px;
  color: #909399;
  min-width: 50px;
}

.resource-value {
  font-size: 12px;
  color: #606266;
  font-weight: 500;
}

.quota-limit-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: center;
}

.quota-badge, .limit-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

/* 标签容器样式 */
.label-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  padding: 4px 0;
}

.label-badge {
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.label-icon-button {
  background: transparent;
  border: none;
  color: #606266;
  transition: all 0.3s ease;
}

.label-icon-button:hover {
  background: transparent;
  color: #409eff;
}

.operation-buttons {
  display: flex;
  gap: 6px;
  justify-content: center;
  flex-wrap: wrap;
}

.operation-buttons .el-button {
  transition: all 0.3s ease;
  pointer-events: auto;
  position: relative;
  z-index: 1;
}

.operation-buttons .el-button:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.operation-buttons .el-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .operation-buttons {
    gap: 4px;
  }
  
  .operation-buttons .el-button {
    margin: 1px;
  }
}

@media (max-width: 768px) {
  .operation-buttons {
    flex-direction: column;
    gap: 4px;
  }
  
  .namespace-table :deep(.el-table__row:hover) {
    transform: none;
  }
}
</style>