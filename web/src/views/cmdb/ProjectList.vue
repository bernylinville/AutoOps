<template>
  <div class="project-list">
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span class="title">项目管理</span>
          <el-button type="primary" size="small" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            <span style="margin-left: 4px">新建项目</span>
          </el-button>
        </div>
      </template>

      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="keyword"
          placeholder="搜索项目名称或代码..."
          clearable
          style="width: 280px"
          @keyup.enter="loadProjects"
          @clear="loadProjects"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-button type="primary" @click="loadProjects" style="margin-left: 8px">查询</el-button>
      </div>

      <!-- 项目列表 -->
      <el-table :data="projects" v-loading="loading" stripe border style="width: 100%; margin-top: 16px">
        <el-table-column type="index" label="#" width="50" />
        <el-table-column prop="name" label="项目名称" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            <el-button text type="primary" @click="goDetail(row)">{{ row.name }}</el-button>
          </template>
        </el-table-column>
        <el-table-column prop="code" label="项目代码" width="130" show-overflow-tooltip>
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.code }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="160" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="主机数" width="80" align="center">
          <template #default="{ row }">
            <el-tag type="success" size="small">{{ row.hostCount }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="数据库数" width="90" align="center">
          <template #default="{ row }">
            <el-tag type="warning" size="small">{{ row.dbCount }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="应用数" width="80" align="center">
          <template #default="{ row }">
            <el-tag type="primary" size="small">{{ row.appCount }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createTime" label="创建时间" width="160" />
        <el-table-column label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
            <el-button text type="primary" size="small" @click="goDetail(row)">
              <el-icon><View /></el-icon>
            </el-button>
            <el-button text type="warning" size="small" @click="showEditDialog(row)">
              <el-icon><Edit /></el-icon>
            </el-button>
            <el-popconfirm title="确定删除此项目?" @confirm="deleteProject(row.id)">
              <template #reference>
                <el-button text type="danger" size="small">
                  <el-icon><Delete /></el-icon>
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
          @size-change="loadProjects"
          @current-change="loadProjects"
        />
      </div>
    </el-card>

    <!-- 创建/编辑 Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEditing ? '编辑项目' : '新建项目'"
      width="500px"
      destroy-on-close
    >
      <el-form :model="form" label-width="90px">
        <el-form-item label="项目名称" required>
          <el-input v-model="form.name" placeholder="请输入项目名称" />
        </el-form-item>
        <el-form-item label="项目代码" required>
          <el-input
            v-model="form.code"
            placeholder="英文标识，如 shop-backend"
            :disabled="isEditing"
          />
        </el-form-item>
        <el-form-item label="项目描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="请输入项目描述" />
        </el-form-item>
        <el-form-item label="状态" v-if="isEditing">
          <el-select v-model="form.status" style="width: 100%">
            <el-option :value="1" label="活跃" />
            <el-option :value="2" label="归档" />
            <el-option :value="3" label="已废弃" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取 消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitForm">确 定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Plus, Search, Edit, Delete, View } from '@element-plus/icons-vue'
import projectApi from '@/api/project'

const router = useRouter()

// 列表状态
const projects = ref([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')

// 表单状态
const dialogVisible = ref(false)
const isEditing = ref(false)
const submitting = ref(false)
const form = ref({ id: null, name: '', code: '', description: '', status: 1 })

onMounted(() => {
  loadProjects()
})

async function loadProjects() {
  loading.value = true
  try {
    const res = await projectApi.getProjectList({ page: page.value, pageSize: pageSize.value, keyword: keyword.value })
    if (res.data && res.data.data) {
      projects.value = res.data.data.list || []
      total.value = res.data.data.total || 0
    }
  } catch (e) {
    ElMessage.error('加载项目列表失败')
  } finally {
    loading.value = false
  }
}

function showCreateDialog() {
  isEditing.value = false
  form.value = { id: null, name: '', code: '', description: '', status: 1 }
  dialogVisible.value = true
}

function showEditDialog(row) {
  isEditing.value = true
  form.value = { id: row.id, name: row.name, code: row.code, description: row.description, status: row.status }
  dialogVisible.value = true
}

async function submitForm() {
  if (!form.value.name || (!isEditing.value && !form.value.code)) {
    ElMessage.warning('项目名称和代码为必填项')
    return
  }
  submitting.value = true
  try {
    if (isEditing.value) {
      await projectApi.updateProject({ id: form.value.id, name: form.value.name, description: form.value.description, status: form.value.status })
      ElMessage.success('更新成功')
    } else {
      await projectApi.createProject({ name: form.value.name, code: form.value.code, description: form.value.description })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadProjects()
  } catch (e) {
    ElMessage.error(isEditing.value ? '更新失败' : '创建失败')
  } finally {
    submitting.value = false
  }
}

async function deleteProject(id) {
  try {
    await projectApi.deleteProject(id)
    ElMessage.success('删除成功')
    loadProjects()
  } catch (e) {
    ElMessage.error('删除失败，项目下可能存在关联资源')
  }
}

function goDetail(row) {
  router.push(`/cmdb/project/detail/${row.id}`)
}

function statusTagType(status) {
  const map = { 1: 'success', 2: 'warning', 3: 'danger' }
  return map[status] || 'info'
}

function statusText(status) {
  const map = { 1: '活跃', 2: '归档', 3: '已废弃' }
  return map[status] || '未知'
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
