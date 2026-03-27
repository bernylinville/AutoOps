<template>
  <div class="audit-log-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>审计日志</span>
        </div>
      </template>

      <!-- 搜索区 -->
      <div class="filter-bar">
        <el-form :model="query" :inline="true">
          <el-form-item label="操作人">
            <el-input v-model="query.username" placeholder="请输入用户名" clearable
                      style="width: 160px" @keyup.enter="handleQuery" />
          </el-form-item>
          <el-form-item label="操作模块">
            <el-select v-model="query.module" placeholder="全部" clearable style="width: 140px">
              <el-option label="系统管理" value="system" />
              <el-option label="CMDB" value="cmdb" />
              <el-option label="配置中心" value="config" />
              <el-option label="监控中心" value="monitor" />
              <el-option label="N9E" value="n9e" />
              <el-option label="K8s" value="k8s" />
              <el-option label="任务中心" value="task" />
              <el-option label="应用管理" value="app" />
              <el-option label="工具" value="tool" />
            </el-select>
          </el-form-item>
          <el-form-item label="操作类型">
            <el-select v-model="query.operType" placeholder="全部" clearable style="width: 120px">
              <el-option label="新增" value="新增" />
              <el-option label="修改" value="修改" />
              <el-option label="删除" value="删除" />
              <el-option label="其他" value="其他" />
            </el-select>
          </el-form-item>
          <el-form-item label="日期范围">
            <el-date-picker v-model="dateRange" type="daterange" range-separator="至"
                            start-placeholder="开始日期" end-placeholder="结束日期"
                            value-format="YYYY-MM-DD" style="width: 260px" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" icon="Search" @click="handleQuery">搜索</el-button>
            <el-button icon="Refresh" @click="resetQuery">重置</el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 操作按钮 -->
      <div class="action-bar">
        <el-button type="danger" plain icon="Delete" :disabled="!selectedIds.length"
                   @click="batchDelete" v-authority="['base:audit:delete']">批量删除</el-button>
        <el-button type="danger" plain icon="Delete" @click="handleClean"
                   v-authority="['base:audit:clean']">清空日志</el-button>
      </div>

      <!-- 表格 -->
      <el-table v-loading="loading" :data="tableData" stripe row-key="id"
                @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="50" />
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="expand-detail">
              <p><strong>请求URL:</strong> {{ row.url }}</p>
              <p v-if="row.requestBody"><strong>请求体:</strong></p>
              <pre v-if="row.requestBody" class="request-body">{{ formatJson(row.requestBody) }}</pre>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="时间" prop="createTime" width="170" />
        <el-table-column label="操作人" prop="username" width="120" />
        <el-table-column label="操作模块" width="110">
          <template #default="{ row }">
            <el-tag :type="moduleTagType(row.module)" size="small" effect="plain">
              {{ moduleLabel(row.module) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作类型" width="90">
          <template #default="{ row }">
            <el-tag :type="operTypeTag(row.operType)" size="small" effect="dark">
              {{ row.operType }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="请求方式" width="90">
          <template #default="{ row }">
            <el-tag :type="methodTagType(row.method)" size="small" effect="dark">
              {{ row.method }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作描述" prop="description" min-width="160" show-overflow-tooltip />
        <el-table-column label="IP" prop="ip" width="140" />
        <el-table-column label="状态码" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.statusCode >= 200 && row.statusCode < 400 ? 'success' : 'danger'"
                    size="small" effect="plain">
              {{ row.statusCode }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="80" align="right">
          <template #default="{ row }">
            <span class="mono-font">{{ row.duration }}ms</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" fixed="right" align="center">
          <template #default="{ row }">
            <el-tooltip content="删除" placement="top">
              <el-button type="danger" icon="Delete" size="small" circle
                         @click="handleDelete(row.id)" v-authority="['base:audit:delete']" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <el-pagination @size-change="handleSizeChange" @current-change="handleCurrentChange"
                     :current-page="query.pageNum" :page-sizes="[10, 20, 50, 100]"
                     :page-size="query.pageSize" layout="total, sizes, prev, pager, next, jumper"
                     :total="total" />
    </el-card>
  </div>
</template>

<script>
export default {
  data() {
    return {
      loading: true,
      tableData: [],
      total: 0,
      selectedIds: [],
      dateRange: null,
      query: {
        username: '',
        module: '',
        operType: '',
        pageSize: 20,
        pageNum: 1
      }
    }
  },
  methods: {
    async fetchData() {
      this.loading = true
      const params = { ...this.query }
      if (this.dateRange && this.dateRange.length === 2) {
        params.beginTime = this.dateRange[0]
        params.endTime = this.dateRange[1]
      }
      try {
        const { data: res } = await this.$api.queryAuditLogList(params)
        if (res.code === 200) {
          this.tableData = res.data.list || []
          this.total = res.data.total || 0
        } else {
          this.$message.error(res.message)
        }
      } catch (e) {
        this.$message.error('获取审计日志失败')
      }
      this.loading = false
    },
    handleQuery() {
      this.query.pageNum = 1
      this.fetchData()
    },
    resetQuery() {
      this.query = { username: '', module: '', operType: '', pageSize: 20, pageNum: 1 }
      this.dateRange = null
      this.fetchData()
    },
    handleSizeChange(size) {
      this.query.pageSize = size
      this.fetchData()
    },
    handleCurrentChange(page) {
      this.query.pageNum = page
      this.fetchData()
    },
    handleSelectionChange(selection) {
      this.selectedIds = selection.map(i => i.id)
    },
    async handleDelete(id) {
      await this.$confirm('确认删除该条审计日志？', '提示', { type: 'warning' })
      const { data: res } = await this.$api.deleteAuditLog(id)
      if (res.code === 200) {
        this.$message.success('删除成功')
        this.fetchData()
      } else {
        this.$message.error(res.message)
      }
    },
    async batchDelete() {
      await this.$confirm(`确认删除选中的 ${this.selectedIds.length} 条审计日志？`, '提示', { type: 'warning' })
      const { data: res } = await this.$api.batchDeleteAuditLog(this.selectedIds)
      if (res.code === 200) {
        this.$message.success('删除成功')
        this.fetchData()
      } else {
        this.$message.error(res.message)
      }
    },
    async handleClean() {
      await this.$confirm('确认清空所有审计日志？此操作不可恢复！', '警告', { type: 'error' })
      const { data: res } = await this.$api.cleanAuditLog()
      if (res.code === 200) {
        this.$message.success('清空成功')
        this.fetchData()
      } else {
        this.$message.error(res.message)
      }
    },
    moduleLabel(mod) {
      const map = {
        system: '系统管理', cmdb: 'CMDB', config: '配置中心',
        monitor: '监控中心', n9e: 'N9E', k8s: 'K8s',
        task: '任务中心', app: '应用管理', tool: '工具',
        dashboard: '仪表盘', other: '其他'
      }
      return map[mod] || mod
    },
    moduleTagType(mod) {
      const map = {
        system: 'primary', cmdb: 'success', config: 'warning',
        monitor: '', n9e: 'danger', k8s: 'info',
        task: 'warning', app: 'success', tool: 'info'
      }
      return map[mod] || 'info'
    },
    operTypeTag(type) {
      const map = { '新增': 'success', '修改': 'warning', '删除': 'danger' }
      return map[type] || 'info'
    },
    methodTagType(method) {
      const map = { POST: 'success', PUT: 'warning', DELETE: 'danger', PATCH: 'info' }
      return map[method] || 'info'
    },
    formatJson(str) {
      try {
        return JSON.stringify(JSON.parse(str), null, 2)
      } catch {
        return str
      }
    }
  },
  created() {
    this.fetchData()
  }
}
</script>

<style scoped>
.audit-log-page {
  padding: 16px;
}

.card-header {
  font-weight: 600;
  font-size: 16px;
  color: var(--ao-text-primary);
}

.filter-bar {
  margin-bottom: 16px;
}

.action-bar {
  margin-bottom: 12px;
}

.expand-detail {
  padding: 12px 24px;
  font-size: 13px;
  color: var(--ao-text-regular);
  line-height: 1.8;
}

.request-body {
  background: var(--ao-fill);
  border: 1px solid var(--ao-border-lighter);
  border-radius: var(--ao-radius);
  padding: 12px;
  font-size: 12px;
  max-height: 200px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}

.mono-font {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 12px;
  color: var(--ao-text-secondary);
}
</style>
