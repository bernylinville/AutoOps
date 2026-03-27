<template>
  <div class="dept-container">
    <div >
      <div class="card-header">
        <h2 class="gradient-title">部门管理</h2>
      </div>
      
      <!--搜索-->
      <div class="search-section">
        <el-form :inline="true" :model="queryParams" class="search-form">
          <el-form-item label="部门名称">
            <el-input 
              placeholder="请输入部门名称" 
              clearable 
              v-model="queryParams.deptName"
              @keyup.enter="handleQuery"
              size="small" 
              class="modern-input"
            />
          </el-form-item>
          <el-form-item label="部门状态">
            <el-select 
              placeholder="部门状态"  
              v-model="queryParams.deptStatus" 
              style="width: 150px;" 
              size="small"
              class="modern-select"
            >
              <el-option 
                v-for="item in deptStatusList" 
                :key="item.value" 
                :label="item.label" 
                :value="item.value"
              />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button 
              type="primary" 
              size="small"
              @click="handleQuery"
              class="modern-btn primary-btn"
            >
              <el-icon><Search /></el-icon>
              搜索
            </el-button>
            <el-button 
              size="small"
              @click="resetQuery"
              class="modern-btn reset-btn"
            >
              <el-icon><Refresh /></el-icon>
              重置
            </el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 操作按钮区域 -->
      <div class="action-section">
        <div class="action-buttons">
          <el-button 
            type="success" 
            size="small"
            @click="addDeptDialogVisible = true" 
            v-authority="['base:dept:add']"
            class="modern-btn success-btn"
          >
            <el-icon><Plus /></el-icon>
            新增部门
          </el-button>
          <el-button 
            size="small"
            @click="toggleExpandAll"
            class="modern-btn secondary-btn"
          >
            <el-icon><Sort /></el-icon>
            展开/折叠
          </el-button>
        </div>
      </div>
      
      <!--列表-->
      <div class="table-section">
        <el-table 
          v-if="refreshTable"
          v-loading="loading" 
          :data="deptList" 
          row-key="id" 
          :default-expand-all="isExpandAll"
          :tree-props="{ children: 'children' }"
          class="modern-table"
          :header-cell-style="{ background: 'transparent', color: '#2c3e50', fontWeight: 'bold' }"
          :row-style="{ background: 'rgba(255, 255, 255, 0.05)' }"
        >
          <el-table-column label="部门名称" prop="deptName" />
          <el-table-column label="部门类型" prop="deptType">
            <template v-slot="scope">
              <el-tag 
                v-if="scope.row.deptType === 1" 
                class="modern-tag company-tag"
              >
                公司
              </el-tag>
              <el-tag 
                v-else-if="scope.row.deptType === 2" 
                class="modern-tag center-tag"
              >
                中心
              </el-tag>
              <el-tag 
                v-else-if="scope.row.deptType === 3" 
                class="modern-tag dept-tag"
              >
                部门
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="部门状态" prop="deptStatus">
            <template v-slot="scope">
              <el-tag 
                v-if="scope.row.deptStatus === 1" 
                class="modern-tag success-tag"
              >
                正常
              </el-tag>
              <el-tag 
                v-else-if="scope.row.deptStatus === 2" 
                class="modern-tag danger-tag"
              >
                停用
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" prop="createTime" />
          <el-table-column label="操作" width="150" fixed="right">
            <template v-slot="scope">
              <div class="operation-buttons">
                <el-tooltip content="修改" placement="top">
                  <el-button
                    type="warning"
                    size="small"
                    circle
                    @click="showEditDeptDialog(scope.row.id)"
                    v-authority="['base:dept:edit']"
                  >
                    <el-icon><Edit /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top">
                  <el-button
                    type="danger"
                    size="small"
                    circle
                    @click="handleDeptDelete(scope.row)"
                    :disabled="scope.row.deptType == '1' ? true : false"
                    v-authority="['base:dept:delete']"
                  >
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!--新增部门-->
    <el-dialog 
      title="新增部门" 
      v-model="addDeptDialogVisible" 
      width="35%" 
      @close="addDeptDialogClosed"
      class="modern-dialog"
    >
      <div class="dialog-content">
        <el-form 
          :model="addDeptForm" 
          :rules="addDeptFormRules" 
          ref="addDeptFormRefForm" 
          label-width="90px"
          class="modern-form"
        >
          <el-form-item label="部门类型" prop="deptType">
            <el-radio-group v-model="addDeptForm.deptType" class="modern-radio-group">
              <el-radio :label="1" class="modern-radio">公司</el-radio>
              <el-radio :label="2" class="modern-radio">中心</el-radio>
              <el-radio :label="3" class="modern-radio">部门</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="上级部门" prop="parentId" v-if="addDeptForm.deptType != 1">
            <treeselect 
              :options="optionsDeptList" 
              placeholder="请选择上级部门" 
              v-model="addDeptForm.parentId"
              class="modern-treeselect"
            />
          </el-form-item>
          <el-form-item label="部门名称" prop="deptName">
            <el-input v-model="addDeptForm.deptName" class="modern-input" />
          </el-form-item>
          <el-form-item label="部门状态" prop="deptStatus" >
            <el-radio-group v-model="addDeptForm.deptStatus" class="modern-radio-group">
              <el-radio :label="1" class="modern-radio">正常</el-radio>
              <el-radio :label="2" class="modern-radio">停用</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="addDept" class="modern-btn primary-btn">确 定</el-button>
          <el-button @click="addDeptDialogVisible = false" class="modern-btn secondary-btn">取 消</el-button>
        </div>
      </template>
    </el-dialog>

    <!--修改部门-->
    <el-dialog 
      title="编辑部门" 
      v-model="editDeptDialogVisible" 
      width="35%"
      class="modern-dialog"
    >
      <div class="dialog-content">
        <el-form 
          :model="deptInfo" 
          :rules="editDeptFormRules" 
          ref="editDeptFormRefForm" 
          label-width="90px"
          class="modern-form"
        >
          <el-form-item label="部门类型" prop="deptType">
            <el-radio-group v-model="deptInfo.deptType" class="modern-radio-group">
              <el-radio :label="1" class="modern-radio">公司</el-radio>
              <el-radio :label="2" class="modern-radio">中心</el-radio>
              <el-radio :label="3" class="modern-radio">部门</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="上级部门" prop="parentId" v-if="deptInfo.deptType != 1">
            <treeselect 
              :options="optionsDeptList" 
              placeholder="请选择上级部门" 
              v-model="deptInfo.parentId"
              class="modern-treeselect"
            />
          </el-form-item>
          <el-form-item label="部门名称" prop="deptName">
            <el-input v-model="deptInfo.deptName" class="modern-input" />
          </el-form-item>
          <el-form-item label="部门状态" prop="deptStatus">
            <el-radio-group v-model="deptInfo.deptStatus" class="modern-radio-group">
              <el-radio :label="1" class="modern-radio">正常</el-radio>
              <el-radio :label="2" class="modern-radio">停用</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="editDept" class="modern-btn primary-btn">确 定</el-button>
          <el-button @click="editDeptDialogVisible = false" class="modern-btn secondary-btn">取 消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import Treeselect from 'vue3-treeselect'
import 'vue3-treeselect/dist/vue3-treeselect.css'
import {
  Search,
  Refresh,
  Plus,
  Sort,
  Edit,
  Delete
} from '@element-plus/icons-vue'

export default {
  components: { 
    Treeselect,
    Search,
    Refresh,
    Plus,
    Sort,
    Edit,
    Delete
  },
  data() {
    return {
      deptStatusList: [{
        value: '2',
        label: '停用'
      }, {
        value: '1',
        label: '正常'
      }],
      queryParams: {},
      loading: true,
      deptList: [],
      refreshTable: true,
      isExpandAll: true,
      optionsDeptList: [],
      addDeptDialogVisible: false,
      addDeptFormRules: {
        deptType: [{ required: true, message: "请选择部门类型", trigger: "blur" }],
        deptName: [{ required: true, message: '请输入部门名称', trigger: 'blur' }],
      },
      addDeptForm: {
        deptStatus: 1
      },
      editDeptDialogVisible: false,
      deptInfo: {},
      editDeptFormRules: {
        deptType: [{ required: true, message: "请选择部门类型", trigger: "blur" }],
        deptName: [{ required: true, message: '请输入部门名称', trigger: 'blur' }],
      }
    }
  },
  methods: {
    // 列表
    async getList() {
      this.loading = true
      const { data: res } = await this.$api.queryDeptList(this.queryParams)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.deptList = this.$handleTree.handleTree(res.data, "id")
        this.loading = false
      }
    },
    // 搜索
    handleQuery() {
      this.getList()
    },
    // 重置搜索
    resetQuery() {
      this.queryParams = {}
      this.getList()
      this.$message.success("重置成功")
    },
    // 展开和折叠
    toggleExpandAll() {
      this.refreshTable = false
      this.isExpandAll = !this.isExpandAll
      this.$nextTick(() => {
        this.refreshTable = true
      })
    },
    // 部门下拉列表
    async getDeptVoList() {
      const { data: res } = await this.$api.querySysDeptVoList()
      // console.log(res)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.optionsDeptList = this.$handleTree.handleTree(res.data, "id")
      }
    },
    // 监听新增部门对话框
    addDeptDialogClosed() {
      this.$refs.addDeptFormRefForm.resetFields()
    },
    // 新增
    addDept() {
      this.$refs.addDeptFormRefForm.validate(async valid => {
        if (!valid) return
        const { data: res } = await this.$api.addDept(this.addDeptForm);
        if (res.code !== 200) {
          this.$message.error(res.message)
        } else {
          this.$message.success('新增部门成功')
          this.addDeptDialogVisible = false
          await this.getList()
          await this.getDeptVoList()
        }
      })
    },
    // 展示编辑对话框
    async showEditDeptDialog(id) {
      const { data: res } = await this.$api.deptInfo(id)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.deptInfo = res.data
        this.editDeptDialogVisible = true
      }
    },
    // 监听编辑部门
    editDeptDialogClosed() {
      this.$refs.editDeptFormRefForm.resetFields()
    },
    // 修改部门信息并提交
    editDept() {
      this.$refs.editDeptFormRefForm.validate(async valid => {
        if (!valid) return
        const { data: res } = await this.$api.deptUpdate(this.deptInfo)
        if (res.code !== 200) {
          this.$message.error(res.message)
        } else {
          this.editDeptDialogVisible = false
          await this.getList()
          this.$message.success("修改部门成功")
        }
      })
    },
    // 删除部门
    async handleDeptDelete(row) {
      const confirmResult = await this.$confirm('是否确认删除部门为"' + row.deptName + '"的数据项？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).catch(err => err)
      if (confirmResult !== 'confirm') {
        return this.$message.info('已取消删除')
      }
      const { data: res } = await this.$api.deleteDept(row.id)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.$message.success('删除成功')
        await this.getList()
      }
    }
  },
  created() {
    this.getList()
    this.getDeptVoList()
  }
}
</script>

<style lang="less" scoped>
.dept-container {
  padding: var(--ao-page-padding);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.gradient-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--ao-text-primary);
}

.search-section {
  margin-bottom: 16px;
  :deep(.el-form-item) {
    margin-bottom: 0;
  }
}

.action-section {
  margin-bottom: 8px;
}

.table-section {
  overflow: hidden;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
