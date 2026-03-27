<template>
  <div class="cmdb-db-management">
    <el-card shadow="hover" class="db-card">
      <template #header>
        <div class="card-header">
          <span class="title">数据库管理</span>
        </div>
      </template>
      
      <!-- 搜索表单 -->
      <div class="search-section">
        <el-form :inline="true" :model="queryParams" ref="queryForm" class="search-form">
          <el-form-item label="数据库名称">
            <el-input
              v-model="queryParams.name"
              placeholder="请输入数据库名称"
              clearable
              size="small"
              style="width: 200px"
              @keyup.enter="handleQuery"
            />
          </el-form-item>
          <el-form-item label="数据库类型">
            <el-select
              v-model="queryParams.type"
              placeholder="请选择数据库类型"
              clearable
              size="small"
              style="width: 180px"
            >
              <el-option
                v-for="item in dbTypeOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              >
                <div style="display: flex; align-items: center; gap: 8px;">
                  <img 
                    :src="getDbIcon(item.value)" 
                    :alt="item.label"
                    style="width: 16px; height: 16px; object-fit: contain; flex-shrink: 0;"
                  />
                  <span>{{ item.label }}</span>
                </div>
              </el-option>
            </el-select>
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
        <el-button type="success" size="small" v-authority="['cmdb:db:add']" @click="showAddDialog">
          <el-icon><Plus /></el-icon>
          创建数据库
        </el-button>
      </div>
      
      <!-- 数据库列表表格 -->
      <div class="table-section">
        <el-table
          :data="dbList"
          v-loading="loading"
          stripe
          style="width: 100%"
          class="db-table"
        >
          <el-table-column prop="name" label="数据库名称" min-width="200">
            <template #default="{ row }">
              <div class="db-name-container">
                <img :src="getDbIcon(row.type)" :alt="getDbName(row.type)" class="db-icon" />
                <el-button
                  type="primary"
                  link
                  @click="goToDetails(row.id)"
                  class="db-name-link"
                >
                  {{ row.name }}
                </el-button>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="accountAlias" label="所属账号" min-width="140" />
          <el-table-column prop="charset" label="标签" min-width="120" show-overflow-tooltip />
          <el-table-column prop="type" label="数据库类型" min-width="140">
            <template #default="{ row }">
              <el-tag
                :type="getDbTagType(row.type)"
                size="small"
                effect="dark"
              >
                {{ getDbName(row.type) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" label="创建时间" min-width="180" />
          <el-table-column prop="remark" label="备注" min-width="150" show-overflow-tooltip />
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <div class="operation-buttons">
                <el-tooltip content="修改" placement="top">
                  <el-button
                    type="warning"
                    size="small"
                    circle
                    v-authority="['cmdb:db:edit']"
                    @click="showEditDialog(row)"
                  >
                    <el-icon><Edit /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top">
                  <el-button
                    type="danger"
                    size="small"
                    circle
                    v-authority="['cmdb:db:delete']"
                    @click="handleDelete(row)"
                  >
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
    
    <!--新增/编辑对话框-->
    <el-dialog :title="dialogTitle" v-model="dialogVisible" width="40%">
      <el-form :model="formData" :rules="formRules" ref="formRef" label-width="100px">
        <el-row>
          <el-col :span="12">
            <el-form-item label="数据库名称" prop="name">
              <el-input v-model="formData.name"></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="数据库类型" prop="type">
              <el-select v-model="formData.type" placeholder="请选择数据库类型">
                <el-option label="MySQL" :value="1" />
                <el-option label="PostgreSQL" :value="2" />
                <el-option label="Redis" :value="3" />
                <el-option label="MongoDB" :value="4" />
                <el-option label="Elasticsearch" :value="5" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="所属账号" prop="accountId">
              <el-select
                v-model="formData.accountId"
                placeholder="请选择所属账号"
                filterable
              >
                <el-option
                  v-for="account in accountList"
                  :key="account.id"
                  :label="account.alias"
                  :value="account.id">
                  <span style="float: left">{{ account.alias }}</span>
                  <span style="float: right; color: #8492a6; font-size: 13px">
                    {{ getDbName(account.type) }}
                  </span>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="业务分组" prop="groupId">
              <el-cascader
                v-model="formData.groupId"
                :options="groupList"
                :props="{
                  checkStrictly: true,
                  value: 'id',
                  label: 'name',
                  children: 'children',
                  expandTrigger: 'hover'
                }"
                placeholder="请选择业务分组"
                clearable
                filterable
                style="width: 100%"
              ></el-cascader>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="标签" prop="tags">
              <el-input v-model="formData.tags" placeholder="多个标签用逗号分隔"></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="描述" prop="description">
              <el-input v-model="formData.description"></el-input>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <span slot="footer" class="dialog-footer">
        <el-button type="primary" @click="submitForm">确 定</el-button>
        <el-button @click="dialogVisible = false">取 消</el-button>
      </span>
    </el-dialog>
    </el-card>
  </div>
</template>

<script>
import cmdbAPI from '@/api/cmdb'
import configAPI from '@/api/config'
import { Edit, Delete, Search, Refresh, Plus } from '@element-plus/icons-vue'

export default {
  components: {
    Edit,
    Delete,
    Search,
    Refresh,
    Plus
  },
  data() {
    return {
      queryParams: {
        name: '',
        type: undefined
      },
      dbTypeOptions: [
        { label: 'MySQL', value: 1 },
        { label: 'PostgreSQL', value: 2 },
        { label: 'Redis', value: 3 },
        { label: 'MongoDB', value: 4 },
        { label: 'Elasticsearch', value: 5 }
      ],
      loading: false,
      dbList: [],
      accountList: [], // 账号列表
      groupList: [], // 分组列表
      dialogVisible: false,
      dialogTitle: '',
      formData: {
        id: '',
        name: '',
        type: undefined,
        accountId: undefined,
        groupId: undefined,
        tags: '',
        description: ''
      },
      formRules: {
        name: [{ required: true, message: '请输入数据库名称', trigger: 'blur' }],
        type: [{ required: true, message: '请选择数据库类型', trigger: 'change' }],
        accountId: [{ required: true, message: '请选择所属账号', trigger: 'change' }],
        groupId: [{ required: true, message: '请选择业务分组', trigger: 'change' }]
      }
    }
  },
    methods: {
      // 获取数据库图标
      getDbIcon(type) {
        const iconMap = {
          1: require('@/assets/image/mysql.svg'),          // MySQL
          2: require('@/assets/image/PostgreSQL.svg'),        // PostgreSQL 
          3: require('@/assets/image/redis.svg'),          // Redis
          4: require('@/assets/image/mongodb.svg'),        // MongoDB
          5: require('@/assets/image/Elasticsearch.svg')   // Elasticsearch
        }
        return iconMap[type] || require('@/assets/image/mysql.svg')
      },
      
      // 获取数据库名称
      getDbName(type) {
        const nameMap = {
          1: 'MySQL',
          2: 'PostgreSQL', 
          3: 'Redis',
          4: 'MongoDB',
          5: 'Elasticsearch'
        }
        return nameMap[type] || 'MySQL'
      },

      // 获取数据库标签类型
      getDbTagType(type) {
        const tagTypeMap = {
          1: 'success',    // MySQL - 绿色
          2: 'warning',    // PostgreSQL - 黄色
          3: 'danger',     // Redis - 红色
          4: 'info',       // MongoDB - 蓝色
          5: 'primary'     // Elasticsearch - 主色
        }
        return tagTypeMap[type] || 'success'
      },
      
      goToDetails(id) {
        this.$router.push({
          path: '/cmdb/dbdetails',
          query: { id }
        })
      },
    // 获取数据库列表
    async getList() {
      this.loading = true
      try {
        const res = await cmdbAPI.listDatabases()
        if (res.data?.code === 200) {
          // 适配后端返回的数据结构
          this.dbList = res.data?.data?.list?.map(item => {
            // 查找对应的账号名称
            const account = this.accountList.find(a => a.id === item.accountId)
            return {
              ...item,
              accountAlias: account ? account.alias : item.accountId,
              charset: item.tags || '',
              collation: item.description || '',
              remark: item.description || '',
              createdAt: item.createdAt,
              updatedAt: item.updatedAt
            }
          }) || []
        } else {
          this.$message.error(res.data?.message || '获取数据库列表失败')
        }
      } catch (error) {
        console.error('获取数据库列表失败:', error)
        this.$message.error('获取数据库列表失败')
      } finally {
        this.loading = false
      }
    },
    
    // 获取账号列表
    async getAccountList() {
      try {
        console.log('开始获取账号列表...')
        const res = await configAPI.listAccountAuth({
          page: 1,
          pageSize: 100  // 获取账号，用于下拉选择
        })
        console.log('账号列表API完整响应:', JSON.stringify(res, null, 2))
        
        if (res.data?.code === 200) {
          console.log('API返回数据:', JSON.stringify(res.data.data, null, 2))
          this.accountList = (res.data.data?.list || []).map(account => ({
            id: account.id,
            alias: account.alias.trim(), // 去除可能的空白字符
            type: account.type
          }))
          console.log('格式化后的账号列表:', JSON.stringify(this.accountList, null, 2))
          
          // 检查选择器是否渲染
          this.$nextTick(() => {
            const selectEl = document.querySelector('.el-select')
            console.log('选择器DOM状态:', selectEl)
            console.log('选择器选项数量:', document.querySelectorAll('.el-select-dropdown__item').length)
          })
        } else {
          console.error('API返回错误:', res.data?.message)
          this.$message.error('获取账号列表失败: ' + (res.data?.message || '未知错误'))
        }
      } catch (error) {
        console.error('获取账号列表异常:', error)
        this.$message.error('获取账号列表失败: ' + error.message)
      }
    },

    // 获取业务分组列表
    async getGroupList() {
      try {
        const res = await cmdbAPI.getAllCmdbGroups()
        if (res.data?.code === 200) {
          this.groupList = res.data?.data || []
        }
      } catch (error) {
        console.error('获取业务分组列表失败:', error)
      }
    },
    
    // 搜索
    async handleQuery() {
      this.loading = true
      try {
        let res
        if (this.queryParams.name) {
          res = await cmdbAPI.getDatabasesByName(this.queryParams.name)
        } else if (this.queryParams.type) {
          res = await cmdbAPI.getDatabasesByType(this.queryParams.type)
        } else {
          res = await cmdbAPI.listDatabases()
        }

        if (res.data?.code === 200) {
          this.dbList = res.data?.data || []
        } else {
          this.$message.error(res.data?.message || '查询失败')
        }
      } catch (error) {
        console.error('查询失败:', error)
        this.$message.error('查询失败')
      } finally {
        this.loading = false
      }
    },
    
    // 重置搜索
    resetQuery() {
      this.queryParams = {
        name: '',
        type: undefined
      }
      this.getList()
    },
    
    // 显示新增对话框
    showAddDialog() {
      this.dialogTitle = '创建数据库'
      this.$nextTick(() => {
        this.formData = {
          name: '',
          type: undefined,
          accountId: undefined,
          groupId: 1, // 默认选择第一个分组
          tags: '',
          description: ''
        }
        this.dialogVisible = true
      })
    },
    
    // 显示编辑对话框
    showEditDialog(row) {
      this.dialogTitle = '修改数据库'
      this.$nextTick(() => {
        this.formData = {
          id: row.id,
          name: row.name,
          type: row.type,
          accountId: Number(row.accountId), // 确保转换为数字
          groupId: row.groupId,
          tags: row.tags,
          description: row.description
        }
        console.log('编辑表单数据:', this.formData)
        this.dialogVisible = true
      })
    },
    
    // 提交表单
    async submitForm() {
      try {
        await this.$refs.formRef.validate()
        
        // 处理级联选择器的值（可能是数组或单个值）
        let groupId = this.formData.groupId
        if (Array.isArray(groupId)) {
          groupId = groupId[groupId.length - 1] // 取最后一级的ID
        }
        
        // 准备提交数据，移除id字段，确保groupId有效
        const formData = {
          name: this.formData.name,
          type: Number(this.formData.type),
          accountId: Number(this.formData.accountId),
          groupId: groupId ? Number(groupId) : 1, // 默认分组ID为1
          tags: this.formData.tags,
          description: this.formData.description
        }
        
        console.log('处理后的groupId:', groupId)
        // 如果是更新操作，添加id字段
        if (this.formData.id) {
          formData.id = Number(this.formData.id)
        }
        console.log('提交给API的数据:', JSON.stringify(formData, null, 2))

        let res
        if (formData.id) {
          res = await cmdbAPI.updateDatabase(formData)
        } else {
          res = await cmdbAPI.createDatabase(formData)
        }

        if (res.data.code === 200) {
          this.$message.success(formData.id ? '修改成功' : '创建成功')
          this.dialogVisible = false
          await this.getList()
        } else {
          this.$message.error(res.data.message || (formData.id ? '修改失败' : '创建失败'))
        }
      } catch (error) {
        console.error('操作失败:', error)
        this.$message.error('操作失败: ' + error.message)
      }
    },
    
    // 删除数据库
    async handleDelete(row) {
      try {
        await this.$confirm(`确定删除数据库"${row.name}"?`, '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        
        const res = await cmdbAPI.deleteDatabase(row.id)
        if (res.data.code === 200) {
          this.$message.success('删除成功')
          this.getList()
        } else {
          this.$message.error(res.data.message || '删除失败')
        }
      } catch (error) {
        console.error('删除失败:', error)
      }
    }
  },
    created() {
      this.getList()
      this.getAccountList().then(() => {
        console.log('账号列表加载完成:', this.accountList)
      })
      this.getGroupList()
    },
    
    mounted() {
      console.log('组件挂载完成，检查账号选择器:')
      this.$nextTick(() => {
        const selectEl = document.querySelector('.el-select')
        console.log('选择器DOM:', selectEl)
      })
    }
}
</script>

<style scoped>
/* 🎨 现代化科技感设计风格 - 仿照k8s-cplony.vue */

.cmdb-db-management {
  padding: 20px;
  min-height: 100vh;
  background: var(--ao-bg-page);
}

.db-card {
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

/* 🔍 搜索区域样式 */
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

/* 🎯 操作按钮区域 */
.action-section {
  margin-bottom: 20px;
  padding-left: 0;
}

/* 📊 表格区域样式 */
.table-section {
  margin-top: 0;
}

.db-table {
  border-radius: var(--ao-radius-lg);
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}

.db-table :deep(.el-table__header) {
  background: var(--ao-bg-page);
}

.db-table :deep(.el-table__header th) {
  background: transparent !important;
  color: #2c3e50 !important;
  font-weight: 700 !important;
  border-bottom: none;
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.8);
}

.db-table :deep(.el-table__header th .cell) {
  color: #2c3e50 !important;
  font-weight: 700 !important;
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.8);
}

.db-table :deep(.el-table__row) {
  transition: all 0.3s ease;
}

.db-table :deep(.el-table__row:hover) {
  background-color: rgba(64, 158, 255, 0.05) !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

/* 数据库名称容器样式 */
.db-name-container {
  display: flex;
  align-items: center;
  gap: 10px;
}

.db-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.1));
}

.db-name-link {
  font-weight: 600;
  color: var(--ao-primary);
  text-decoration: none;
  transition: all 0.3s ease;
}

.db-name-link:hover {
  color: #764ba2;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
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

/* 标签样式优化 */
.el-tag {
  font-weight: 500;
  border-radius: 8px;
  border: none;
}

/* 按钮样式优化 */
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

/* 对话框样式保持默认（根据zf.md要求） */
.el-dialog {
  border-radius: 8px;
}
</style>
