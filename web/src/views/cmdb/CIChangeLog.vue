<template>
  <div>
    <!-- 查询栏 -->
    <el-card shadow="never" style="margin-bottom: 12px">
      <el-form :model="query" inline>
        <el-form-item label="实体类型">
          <el-select v-model="query.entityType" placeholder="全部" clearable style="width: 150px">
            <el-option label="CI 实例" value="ci_instance" />
            <el-option label="CMDB 主机" value="cmdb_host" />
            <el-option label="SQL 数据库" value="cmdb_sql" />
          </el-select>
        </el-form-item>
        <el-form-item label="实体 ID">
          <el-input v-model.number="query.entityId" placeholder="留空查全部" clearable style="width: 120px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadLogs(1)">查询</el-button>
          <el-button @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 日志表格 -->
    <el-card shadow="never" v-loading="loading">
      <el-table :data="list" stripe size="small" style="width: 100%">
        <el-table-column type="index" width="55" label="#" />
        <el-table-column label="实体类型" width="110">
          <template #default="{ row }">
            <el-tag :type="entityTagType(row.entityType)" size="small">
              {{ entityLabel(row.entityType) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="实体 ID" prop="entityId" width="80" />
        <el-table-column label="实体名称" prop="entityName" min-width="120" show-overflow-tooltip />
        <el-table-column label="变更字段" prop="field" width="120">
          <template #default="{ row }">
            <el-tag type="info" size="small">{{ row.field }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="变更前" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="old-value">{{ row.oldValue || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="变更后" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="new-value">{{ row.newValue || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作人" prop="operator" width="100" show-overflow-tooltip />
        <el-table-column label="变更时间" prop="createTime" width="160" />
      </el-table>

      <div style="margin-top: 12px; display: flex; justify-content: flex-end">
        <el-pagination
          v-model:current-page="query.page"
          v-model:page-size="query.pageSize"
          :page-sizes="[20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next"
          @size-change="loadLogs(1)"
          @current-change="loadLogs"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '@/api/changelog'

const loading = ref(false)
const list    = ref([])
const total   = ref(0)

const query = reactive({
  entityType: '',
  entityId:   undefined,
  page:       1,
  pageSize:   20,
})

const entityLabel = (t) => ({ ci_instance: 'CI 实例', cmdb_host: 'CMDB 主机', cmdb_sql: 'SQL 库' }[t] || t)
const entityTagType = (t) => ({ ci_instance: '', cmdb_host: 'success', cmdb_sql: 'warning' }[t] || 'info')

const loadLogs = async (page) => {
  if (page) query.page = page
  loading.value = true
  try {
    const params = {
      page:       query.page,
      pageSize:   query.pageSize,
    }
    if (query.entityType) params.entityType = query.entityType
    if (query.entityId)   params.entityId   = query.entityId

    const res = await api.getChangeLogs(params)
    if (res.code === 200) {
      list.value  = res.data?.list  || []
      total.value = res.data?.total || 0
    }
  } finally {
    loading.value = false
  }
}

const resetQuery = () => {
  query.entityType = ''
  query.entityId   = undefined
  query.page       = 1
  loadLogs()
}

onMounted(() => loadLogs())
</script>

<style scoped>
.old-value { color: #F56C6C; font-size: 12px; }
.new-value { color: #67C23A; font-size: 12px; }
</style>
