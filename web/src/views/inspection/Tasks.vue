<template>
  <div class="inspection-tasks">
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span class="title">任务管理</span>
        </div>
      </template>

      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="keyword"
          placeholder="搜索任务名称或业务组..."
          clearable
          style="width: 280px"
          @keyup.enter="loadTasks"
          @clear="loadTasks"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="enabledFilter" placeholder="启用状态" clearable style="width: 140px; margin-left: 8px">
          <el-option :value="true" label="已启用" />
          <el-option :value="false" label="已禁用" />
        </el-select>
        <el-button type="primary" @click="loadTasks" style="margin-left: 8px">查询</el-button>
      </div>

      <!-- 任务列表 -->
      <el-table :data="tasks" v-loading="loading" stripe border style="width: 100%; margin-top: 16px">
        <el-table-column type="index" label="#" width="50" />
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="n9eGroupName" label="业务组名称" min-width="120" show-overflow-tooltip />
        <el-table-column prop="name" label="任务名称" min-width="140" show-overflow-tooltip />
        <el-table-column prop="enabled" label="启用状态" width="100" align="center">
          <template #default="{ row }">
            <el-switch
              :model-value="row.enabled"
              @change="(val) => toggleEnabled(row, val)"
            />
          </template>
        </el-table-column>
        <el-table-column label="通知配置" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.notifyWebhookUrl" style="font-size: 12px; color: #888">{{ row.notifyWebhookUrl }}</span>
            <el-tag v-else size="small" type="info">未配置</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="cron" label="Cron" width="140">
          <template #default="{ row }">
            <el-tag size="small">{{ row.cron || '-' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right" align="center">
          <template #default="{ row }">
            <el-button text type="primary" size="small" @click="showEditDialog(row)">
              <el-icon><Edit /></el-icon>
            </el-button>
            <el-popconfirm title="确定立即执行此任务?" @confirm="handleTrigger(row.id)">
              <template #reference>
                <el-button text type="success" size="small">
                  <el-icon><VideoPlay /></el-icon>
                </el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper" v-if="total > 0">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadTasks"
          @current-change="loadTasks"
        />
      </div>
    </el-card>

    <!-- 编辑 Dialog -->
    <el-dialog
      v-model="dialogVisible"
      title="编辑任务配置"
      width="560px"
      destroy-on-close
    >
      <el-form :model="form" label-width="120px">
        <el-form-item label="Cron 表达式">
          <el-input v-model="form.cron" placeholder="如 */5 * * * *" />
        </el-form-item>
        <el-form-item label="通知 Webhook URL">
          <el-input v-model="form.notifyWebhookUrl" placeholder="请输入 Webhook 地址" />
        </el-form-item>
        <el-form-item label="通知 Secret">
          <el-input v-model="form.notifySecret" placeholder="请输入签名密钥" />
        </el-form-item>
        <el-form-item label="告警级别通知">
          <div style="display: flex; gap: 16px">
            <el-switch v-model="form.notifyOnWarning" active-text="警告" />
            <el-switch v-model="form.notifyOnCritical" active-text="严重" />
            <el-switch v-model="form.notifyOnFailure" active-text="失败" />
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取 消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitEdit">确 定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Edit, VideoPlay } from '@element-plus/icons-vue'
import inspectionApi from '@/api/inspection'

// 列表状态
const tasks = ref([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const enabledFilter = ref(undefined)

// 表单状态
const dialogVisible = ref(false)
const submitting = ref(false)
const editingTaskId = ref(null)
const form = ref({
  cron: '',
  notifyWebhookUrl: '',
  notifySecret: '',
  notifyOnWarning: false,
  notifyOnCritical: false,
  notifyOnFailure: false
})

onMounted(() => {
  loadTasks()
})

async function loadTasks() {
  loading.value = true
  try {
    const params = { page: page.value, pageSize: pageSize.value, keyword: keyword.value }
    if (enabledFilter.value !== undefined && enabledFilter.value !== '') {
      params.enabled = enabledFilter.value
    }
    const res = await inspectionApi.getTasks(params)
    if (res.data && res.data.data) {
      tasks.value = res.data.data.list || []
      total.value = res.data.data.total || 0
    }
  } catch (e) {
    ElMessage.error('加载任务列表失败')
  } finally {
    loading.value = false
  }
}

async function toggleEnabled(row, val) {
  try {
    await inspectionApi.updateTask(row.id, { enabled: val })
    row.enabled = val
    ElMessage.success(val ? '已启用' : '已禁用')
  } catch (e) {
    ElMessage.error('更新失败')
  }
}

function showEditDialog(row) {
  editingTaskId.value = row.id
  form.value = {
    cron: row.cron || '',
    notifyWebhookUrl: row.notifyWebhookUrl || '',
    notifySecret: row.notifySecret || '',
    notifyOnWarning: row.notifyOnWarning || false,
    notifyOnCritical: row.notifyOnCritical || false,
    notifyOnFailure: row.notifyOnFailure || false
  }
  dialogVisible.value = true
}

async function submitEdit() {
  submitting.value = true
  try {
    await inspectionApi.updateTask(editingTaskId.value, form.value)
    ElMessage.success('更新成功')
    dialogVisible.value = false
    loadTasks()
  } catch (e) {
    ElMessage.error('更新失败')
  } finally {
    submitting.value = false
  }
}

async function handleTrigger(id) {
  try {
    await inspectionApi.triggerTask(id)
    ElMessage.success('任务已触发')
  } catch (e) {
    ElMessage.error('触发失败')
  }
}
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.title {
  font-size: 16px;
  font-weight: 600;
}
.search-bar {
  display: flex;
  align-items: center;
}
.pagination-wrapper {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
