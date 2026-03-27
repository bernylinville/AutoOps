<template>
  <div class="cmdb-group-management">
    <el-card shadow="hover" class="group-card">
      <template #header>
        <div class="card-header">
          <span class="title">分组管理</span>
        </div>
      </template>

      <!-- 搜索表单 -->
      <div class="search-section">
        <el-form :inline="true" :model="queryParams" ref="queryForm" class="search-form">
          <el-form-item label="分组名称" prop="groupName">
            <el-input
              v-model="queryParams.groupName"
              placeholder="请输入分组名称"
              clearable
              size="small"
              style="width: 200px"
              @keyup.enter="handleQuery"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="small" @click="handleQuery">
              <el-icon><Search /></el-icon>
              搜索
            </el-button>
            <el-button type="warning" size="small" @click="resetQuery">
              <el-icon><Refresh /></el-icon>
              重置
            </el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 操作按钮 -->
      <div class="action-section">
        <el-button
          type="success"
          size="small"
          @click="addGroupDialogVisible = true"
          v-authority="['cmdb:group:add']">
          <el-icon><Plus /></el-icon>
          创建分组
        </el-button>
        <el-button
          type="info"
          size="small"
          @click="toggleExpandAll">
          <el-icon><Sort /></el-icon>
          {{ isExpandAll ? '折叠' : '展开' }}
        </el-button>
      </div>

      <!-- 表格 -->
      <div class="table-section">
        <el-table
          v-loading="loading"
          :data="groupList"
          row-key="id"
          :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
          :default-expand-all="isExpandAll"
          :key="refreshTable"
          class="modern-table"
          stripe>
          <el-table-column label="分组名称" prop="name" min-width="200">
            <template v-slot="scope">
              <div class="group-name-cell" :class="getGroupLevelClass(scope.row)">
                <el-icon class="group-icon" :class="getGroupIconClass(scope.row)">
                  <component :is="getGroupIcon(scope.row)" />
                </el-icon>
                <span class="group-name" :class="getGroupNameClass(scope.row)">{{ scope.row.name }}</span>
                <el-tag v-if="scope.row.parentId === 0" size="small" type="primary" class="level-tag">一级</el-tag>
                <el-tag v-else size="small" type="info" class="level-tag">二级</el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" prop="createTime" min-width="180">
            <template v-slot="scope">
              <div class="time-cell">
                <el-icon class="time-icon"><Clock /></el-icon>
                <span>{{ scope.row.createTime }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" fixed="right">
            <template v-slot="scope">
              <div class="operation-buttons" v-if="scope.row.id !== DEFAULT_GROUP_ID">
                <el-tooltip content="编辑" placement="top">
                  <el-button
                    type="warning"
                    size="small"
                    circle
                    @click="showEditGroupDialog(scope.row.id)"
                    v-authority="['cmdb:group:update']">
                    <el-icon><Edit /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top">
                  <el-button
                    type="danger"
                    size="small"
                    circle
                    @click="handleGroupDelete(scope.row)"
                    v-authority="['cmdb:group:delete']">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </el-tooltip>
              </div>
              <div v-else class="default-group-tag">
                <el-tag type="info" size="small">默认分组</el-tag>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
    
    <!--新增分组-->
    <el-dialog title="新增业务分组" v-model="addGroupDialogVisible" width="30%" @close="addGroupDialogClosed">
      <el-form :model="addGroupForm" :rules="addGroupFormRules" ref="addGroupFormRef" label-width="80px">
        <el-form-item label="分组名称" prop="name">
          <el-input v-model="addGroupForm.name"></el-input>
        </el-form-item>
        <el-form-item label="父级分组" prop="parentId" v-if="optionsGroupList.length > 0">
          <el-tree-select
            v-model="addGroupForm.parentId"
            :data="optionsGroupList"
            :props="{ 
              label: 'name', 
              value: 'id',
              disabled: (data) => data.id === DEFAULT_GROUP_ID
            }"
            check-strictly
            placeholder="请选择父级分组"
          />
        </el-form-item>
      </el-form>
      <span slot="footer" class="dialog-footer">
        <el-button type="primary" @click="addGroup">确 定</el-button>
        <el-button type="primary" @click="addGroupDialogVisible = false">取 消</el-button>
      </span>
    </el-dialog>
    
    <!--修改分组-->
    <el-dialog title="编辑业务分组" v-model="editGroupDialogVisible" width="30%">
      <el-form :model="groupInfo" :rules="editGroupFormRules" ref="editGroupFormRef" label-width="80px">
        <el-form-item label="分组名称" prop="name">
          <el-input v-model="groupInfo.name" :disabled="groupInfo.id === DEFAULT_GROUP_ID"></el-input>
        </el-form-item>
        <el-form-item label="父级分组" prop="parentId" v-if="optionsGroupList.length > 0">
          <el-tree-select
            v-model="groupInfo.parentId"
            :data="optionsGroupList"
            :props="{ 
              label: 'name', 
              value: 'id',
              disabled: (data) => data.id === DEFAULT_GROUP_ID || groupInfo.id === DEFAULT_GROUP_ID
            }"
            check-strictly
            placeholder="请选择父级分组"
            :disabled="groupInfo.id === DEFAULT_GROUP_ID"
          />
        </el-form-item>
      </el-form>
      <span slot="footer" class="dialog-footer">
        <el-button type="primary" @click="editGroup">确 定</el-button>
        <el-button type="primary" @click="editGroupDialogVisible = false">取 消</el-button>
      </span>
    </el-dialog>
    </el-card>
  </div>
</template>

<script>
import API from '@/api/cmdb'
import {
  Search,
  Refresh,
  Plus,
  Edit,
  Delete,
  FolderOpened,
  Folder,
  Clock,
  Sort
} from '@element-plus/icons-vue'

export default {
  data() {
    return {
      DEFAULT_GROUP_ID: 1, // 默认业务分组ID
      queryParams: {},
      loading: true,
      groupList: [],
      addGroupDialogVisible: false,
      addGroupForm: {
        parentId: 0
      },
      addGroupFormRules: {
        name: [{ required: true, message: '请输入分组名称', trigger: 'blur' }]
      },
      optionsGroupList: [],
      editGroupDialogVisible: false,
      groupInfo: {},
      editGroupFormRules: {
        name: [{ required: true, message: '请输入分组名称', trigger: 'blur' }]
      },
      isExpandAll: true,
      refreshTable: true
    }
  },
  methods: {
    // 获取分组列表
    async getList() {
      this.loading = true
      let retryCount = 0
      const maxRetries = 2
      
      const fetchData = async () => {
        try {
          const { data: res } = await API.getAllCmdbGroups({
            params: this.queryParams
          })
          
          if (res.code !== 200) {
            throw new Error(res.message || '获取数据失败')
          }
          
          // 转换数据格式
          const processData = (data) => {
            return data.map(item => ({
              ...item,
              children: item.children && item.children.length > 0 
                ? processData(item.children) 
                : [],
              hasChildren: !!(item.children && item.children.length)
            }))
          }
          
          const treeData = processData(res.data)
          this.groupList = treeData
          this.optionsGroupList = treeData
          return true
          
        } catch (error) {
          console.error('获取数据失败:', error)
          if (retryCount < maxRetries) {
            retryCount++
            console.log(`重试第 ${retryCount} 次...`)
            await new Promise(resolve => setTimeout(resolve, 1000))
            return fetchData()
          }
          throw error
        }
      }
      
      try {
        await fetchData()
      } catch (error) {
        this.$message.error(`获取分组列表失败: ${error?.message || '未知错误'}`)
      } finally {
        this.loading = false
      }
    },
    // 展开/折叠
    toggleExpandAll() {
      this.refreshTable = false
      this.isExpandAll = !this.isExpandAll
      this.$nextTick(() => {
        this.refreshTable = true
      })
    },
    
    // 搜索
    async handleQuery() {
        if (!this.queryParams.groupName || !this.queryParams.groupName.trim()) {
            this.$message.warning('请输入分组名称')
            return
        }
        
        try {
            const { data: res } = await API.getCmdbGroupByName(this.queryParams.groupName.trim())
            if (res.code === 200) {
                this.groupList = res.data ? (Array.isArray(res.data) ? res.data : [res.data]) : []
                this.$message.success('查询成功')
            } else {
                this.$message.error(res.message || '查询失败')
            }
        } catch (error) {
            this.$message.error('查询失败: ' + (error.message || '未知错误'))
        }
    },
    // 重置搜索
    resetQuery() {
      this.queryParams = {}
      this.getList()
      this.$message.success("重置成功")
    },
    // 监听新增分组对话框关闭
    addGroupDialogClosed() {
      this.$refs.addGroupFormRef?.resetFields()
      this.addGroupForm = {}
    },
    // 新增分组
    async addGroup() {
      try {
        await this.$refs.addGroupFormRef.validate()
        const { data: res } = await API.createCmdbGroup(this.addGroupForm)
        if (res.code !== 200) {
          this.$message.error(res.message)
        } else {
          this.$message.success('新增分组成功')
          this.addGroupDialogVisible = false
          await this.getList()
        }
      } catch (error) {
        console.error('新增分组失败:', error)
        this.$message.error('新增分组失败，请检查网络连接')
      }
    },
    // 展示编辑对话框
    async showEditGroupDialog(id) {
      const group = this.findGroupById(this.groupList, id)
      if (group) {
        this.groupInfo = {
          id: group.id,
          name: group.name,
          parentId: group.parentId
        }
        this.editGroupDialogVisible = true
      } else {
        this.$message.error('未找到分组信息')
      }
    },

    // 递归查找分组
    findGroupById(groups, id) {
      for (const group of groups) {
        if (group.id === id) return group
        if (group.children && group.children.length > 0) {
          const found = this.findGroupById(group.children, id)
          if (found) return found
        }
      }
      return null
    },
    // 修改分组信息
    async editGroup() {
      let retryCount = 0
      const maxRetries = 2
      
      const performUpdate = async () => {
        try {
          await this.$refs.editGroupFormRef.validate()
          const { data: res } = await API.updateCmdbGroup({
            ...this.groupInfo,
            id: this.groupInfo.id
          })
          
          if (res.code !== 200) {
            throw new Error(res.message || '更新分组失败')
          }
          
          this.editGroupDialogVisible = false
          await this.getList()
          this.$message.success("修改分组成功")
          return true
          
        } catch (error) {
          console.error('修改分组失败:', error)
          if (retryCount < maxRetries) {
            retryCount++
            console.log(`重试第 ${retryCount} 次...`)
            await new Promise(resolve => setTimeout(resolve, 1000))
            return performUpdate()
          }
          throw error
        }
      }
      
      try {
        await performUpdate()
      } catch (error) {
        this.$message.error(`修改分组失败: ${error.message}`)
      }
    },
    // 获取分组层级样式类
    getGroupLevelClass(row) {
      return row.parentId === 0 ? 'level-1-group' : 'level-2-group'
    },

    // 获取分组图标
    getGroupIcon(row) {
      return row.parentId === 0 ? 'FolderOpened' : 'Folder'
    },

    // 获取分组图标样式类
    getGroupIconClass(row) {
      return row.parentId === 0 ? 'level-1-icon' : 'level-2-icon'
    },

    // 获取分组名称样式类
    getGroupNameClass(row) {
      return row.parentId === 0 ? 'level-1-name' : 'level-2-name'
    },

    // 删除分组
    async handleGroupDelete(row) {
      if (row.id === this.DEFAULT_GROUP_ID) {
        this.$message.warning('默认分组不能删除')
        return
      }
      
      try {
        const confirmResult = await this.$confirm(
          `是否确认删除分组"${row.name}"?`, 
          '提示', 
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        ).catch(err => err)
        
        if (confirmResult !== 'confirm') {
          return this.$message.info('已取消删除')
        }
        
        const { data: res } = await API.deleteCmdbGroup(row.id)
        if (res.code !== 200) {
          this.$message.error(res.message)
        } else {
          // 删除成功后直接刷新列表
          await this.getList()
          this.$message.success('删除分组成功')
        }
      } catch (error) {
        console.error('删除分组失败:', error)
        this.$message.error('删除分组失败，请检查网络连接')
      }
    }
  },
  created() {
    this.getList()
  }
}
</script>

<style scoped>
.cmdb-group-management {
  padding: 20px;
  min-height: 100vh;
  background: var(--ao-bg-page);
}

.group-card {
  background: rgba(255, 255, 255, 0.95);
  border-radius: var(--ao-radius-lg);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.title {
  font-size: 20px;
  font-weight: 600;
  color: #2c3e50;
  background: var(--ao-primary);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

/* 搜索区域样式 */
.search-section {
  margin-bottom: 20px;
  padding: 20px;
  background: rgba(64, 158, 255, 0.05);
  border-radius: var(--ao-radius-lg);
  border: 1px solid rgba(64, 158, 255, 0.1);
}

.search-form .el-form-item {
  margin-bottom: 0;
  margin-right: 16px;
}

.search-form .el-form-item__label {
  color: #606266;
  font-weight: 500;
}

/* 操作按钮区域 */
.action-section {
  margin-bottom: 20px;
  padding-left: 0;
}

/* 表格区域样式 */
.table-section {
  margin-top: 0;
}

.modern-table {
  border-radius: var(--ao-radius-lg);
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}

/* 让树形展开图标与内容同行 */
.modern-table :deep(td:first-child .cell) {
  display: flex;
  align-items: center;
  flex-wrap: nowrap;
}

.modern-table :deep(.el-table__header) {
  background: var(--ao-bg-page);
}

.modern-table :deep(.el-table__header th) {
  background: transparent !important;
  color: #2c3e50 !important;
  font-weight: 700 !important;
  border-bottom: none;
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.8);
}

.modern-table :deep(.el-table__header th .cell) {
  color: #2c3e50 !important;
  font-weight: 700 !important;
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.8);
}

.modern-table :deep(.el-table__row) {
  transition: all 0.3s ease;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: rgba(64, 158, 255, 0.05) !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

/* 表格内容样式 */
.group-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 一级分组样式 */
.level-1-group {
  padding: 4px 0;
  border-left: 3px solid #667eea;
  padding-left: 8px;
  background: linear-gradient(90deg, rgba(102, 126, 234, 0.05) 0%, rgba(255, 255, 255, 0) 100%);
}

.level-1-icon {
  color: var(--ao-primary);
  font-size: 18px;
  font-weight: bold;
}

.level-1-name {
  font-weight: 600;
  color: #2c3e50;
  font-size: 15px;
}

/* 二级分组样式 */
.level-2-group {
  padding: 2px 0;
  margin-left: 20px;
  border-left: 2px solid #909399;
  padding-left: 8px;
  background: linear-gradient(90deg, rgba(144, 147, 153, 0.03) 0%, rgba(255, 255, 255, 0) 100%);
}

.level-2-icon {
  color: #909399;
  font-size: 16px;
}

.level-2-name {
  font-weight: 400;
  color: #606266;
  font-size: 14px;
}

/* 层级标签样式 */
.level-tag {
  margin-left: 8px;
  font-size: 10px;
  height: 18px;
  line-height: 16px;
}

.time-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #606266;
}

.time-icon {
  color: #909399;
  font-size: 14px;
}

/* 操作按钮样式 */
.operation-buttons {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.operation-buttons .el-button {
  transition: all 0.3s ease;
}

.operation-buttons .el-button:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.default-group-tag {
  display: flex;
  justify-content: center;
}

/* 标签样式优化 */
.el-tag {
  font-weight: 500;
  border-radius: 8px;
  border: none;
}

/* 通用按钮样式 */
.el-button {
  border-radius: 8px;
  font-weight: 500;
  transition: all 0.3s ease;
}

.el-button:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

/* 输入框和选择器样式 */
.el-input :deep(.el-input__wrapper),
.el-select :deep(.el-input__wrapper) {
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(64, 158, 255, 0.2);
  border-radius: 8px;
  box-shadow: none;
  transition: all 0.3s ease;
}

.el-input :deep(.el-input__wrapper):hover,
.el-select :deep(.el-input__wrapper):hover {
  border-color: #c0c4cc;
}

.el-input :deep(.el-input__wrapper.is-focus),
.el-select :deep(.el-input__wrapper.is-focus) {
  border-color: var(--ao-primary);
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.2);
  background: rgba(255, 255, 255, 1);
}

.el-input :deep(.el-input__inner),
.el-select :deep(.el-input__inner) {
  background: transparent;
  border: none;
  color: #2c3e50;
}

/* 加载遮罩样式 */
.el-loading-mask {
  background-color: rgba(64, 158, 255, 0.1);
}

/* 对话框样式 */
.el-dialog {
  border-radius: 8px;
}
</style>
