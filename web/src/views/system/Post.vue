<template>
  <div class="post-management">
    <div class="main-card">
      <!-- 卡片标题 -->
      <div class="card-header">
        <h1 class="gradient-title">岗位管理</h1>
      </div>
      
      <!-- 搜索区域 -->
      <div class="search-section">
        <el-form :model="queryParams" :inline="true" class="search-form">
        <el-form-item label="岗位名称" prop="postName">
          <el-input 
            placeholder="请输入岗位名称" 
            clearable 
            size="small"
            class="modern-input"
            v-model="queryParams.postName">
          </el-input>
        </el-form-item>
        <el-form-item label="岗位状态" prop="postStatus">
          <el-select 
            v-model="queryParams.postStatus" 
            placeholder="岗位状态" 
            size="small"
            style="width: 150px;" 
            class="modern-select">
            <el-option
                v-for="item in postStatusList" 
                :key="item.value" 
                :label="item.label" 
                :value="item.value"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button 
            type="primary" 
            size="small"
            @click="handleQuery"
            class="modern-btn primary-btn">
            <el-icon><Search /></el-icon>
            搜索
          </el-button>
          <el-button 
            size="small"
            @click="resetQuery"
            class="modern-btn reset-btn">
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
            @click="handleAddButtonClick" 
            v-authority="['base:post:add']"
            class="modern-btn success-btn">
            <el-icon><Plus /></el-icon>
            新增
          </el-button>
          <el-button 
            type="danger" 
            size="small"
            :disabled="multiple"
            @click="batchHandleDelete" 
            v-authority="['base:post:delete']"
            class="modern-btn danger-btn">
            <el-icon><Delete /></el-icon>
            删除
          </el-button>
        </div>
      </div>

      <!-- 表格区域 -->
      <div class="table-section">
        <el-table 
        class="modern-table"
        v-loading="loading" 
        :data="postList" 
        @selection-change="handleSelectionChange">
        <el-table-column type="selection" />
        <el-table-column label="ID" v-if="false" prop="id" />
        <el-table-column label="岗位名称" prop="postName" />
        <el-table-column label="岗位编码" prop="postCode" />
        <el-table-column label="岗位状态" prop="postStatus">
          <template v-slot="scope">
            <el-switch 
              v-model="scope.row.postStatus" 
              :active-value="1" 
              :inactive-value="2" 
              style="--el-switch-on-color: var(--ao-success); --el-switch-off-color: var(--ao-info);"
              active-text="启用" 
              inactive-text="停用" 
              class="modern-switch"
              @change="postUpdateStatus(scope.row)">
            </el-switch>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createTime" />
        <el-table-column label="描述" prop="remark" />
        <el-table-column label="操作" width="120" fixed="right">
          <template v-slot="scope">
            <div class="operation-buttons">
              <el-tooltip content="编辑" placement="top">
                <el-button
                  type="warning"
                  size="small"
                  circle
                  @click="handleUpdate(scope.row.id)"
                  v-authority="['base:post:edit']"
                >
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button
                  type="danger"
                  size="small"
                  circle
                  @click="handleDelete(scope.row.id)"
                  v-authority="['base:post:delete']"
                >
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!--分页-->
      <el-pagination 
        class="modern-pagination"
        @size-change="handleSizeChange" 
        @current-change="handleCurrentChange"
        :current-page="queryParams.pageNum" 
        :page-sizes="[10, 50, 100, 500, 1000]" 
        :page-size="queryParams.pageSize"
        layout="total, sizes, prev, pager, next, jumper" 
        :total="total">
      </el-pagination>
      </div>
    </div>

    <!--新增对话框-->
    <el-dialog 
      title="新增岗位" 
      v-model="addPostDialogVisible" 
      width="30%" 
      class="modern-dialog"
      @close="addPostDialogClosed">
      <el-form 
        label-width="80px" 
        ref="addPostFormRefForm" 
        :rules="addPostFormRules" 
        :model="addPostForm"
        class="dialog-form">
        <el-form-item label="岗位名称" prop="postName">
          <el-input 
            placeholder="请输入岗位名称" 
            class="modern-input"
            v-model="addPostForm.postName" />
        </el-form-item>
        <el-form-item label="岗位编码" prop="postCode">
          <el-input 
            placeholder="请输入岗位编码" 
            class="modern-input"
            v-model="addPostForm.postCode" />
        </el-form-item>
        <el-form-item label="岗位状态" prop="postStatus">
          <el-radio-group v-model="addPostForm.postStatus" class="modern-radio-group">
            <el-radio :label="1" class="modern-radio">启用</el-radio>
            <el-radio :label="2" class="modern-radio">停用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="岗位描述" prop="remark">
          <el-input 
            placeholder="请输入岗位描述" 
            class="modern-input"
            v-model="addPostForm.remark" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button 
            type="primary" 
            class="modern-btn primary-btn"
            @click="addPost">确定</el-button>
          <el-button 
            class="modern-btn secondary-btn"
            @click="addPostDialogVisible = false">取消</el-button>
        </div>
      </template>
    </el-dialog>

    <!--编辑对话框-->
    <el-dialog 
      title="编辑岗位" 
      v-model="editPostDialogVisible" 
      width="30%" 
      class="modern-dialog"
      @close="editPostDialogClosed">
      <el-form 
        label-width="80px" 
        ref="editPostFormRefForm" 
        :rules="editPostFormRules" 
        :model="editPostForm"
        class="dialog-form">
        <el-form-item label="岗位名称" prop="postName">
          <el-input 
            placeholder="请输入岗位名称" 
            class="modern-input"
            v-model="editPostForm.postName" />
        </el-form-item>
        <el-form-item label="岗位编码" prop="postCode">
          <el-input 
            placeholder="请输入岗位编码" 
            class="modern-input"
            v-model="editPostForm.postCode" />
        </el-form-item>
        <el-form-item label="岗位状态" prop="postStatus">
          <el-radio-group v-model="editPostForm.postStatus" class="modern-radio-group">
            <el-radio :label="1" class="modern-radio">启用</el-radio>
            <el-radio :label="2" class="modern-radio">停用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="岗位描述" prop="remark">
          <el-input 
            placeholder="请输入岗位描述" 
            class="modern-input"
            v-model="editPostForm.remark" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button 
            type="primary" 
            class="modern-btn primary-btn"
            @click="editPost">确定</el-button>
          <el-button 
            class="modern-btn secondary-btn"
            @click="editPostDialogVisible = false">取消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import {
  Search,
  Refresh,
  Plus,
  Edit,
  Delete
} from '@element-plus/icons-vue'

export default {
  components: {
    Search,
    Refresh,
    Plus,
    Edit,
    Delete
  },
  data() {
    return {
      queryParams: {},
      postStatusList: [{
        value: '1',
        label: '启用'
      }, {
        value: '2',
        label: '停用'
      }],
      loading: true,
      postList: [],
      total: 0,
      addPostDialogVisible: false,
      addPostFormRules: {
        postName: [{ required: true, message: '请输入岗位名称', trigger: 'blur' }],
        postCode: [{ required: true, message: '请输入岗位标识', trigger: 'blur' }],
        postStatus: [{ required: true, message: '请输入岗位状态', trigger: 'blur' }]
      },
      addPostForm: {
        postName: '',
        postCode: '',
        postStatus: 1,
        remark: ''
      },
      editPostDialogVisible: false,
      editPostForm: {},
      editPostFormRules: {
        postName: [{ required: true, message: '请输入岗位名称', trigger: 'blur' }],
        postCode: [{ required: true, message: '请输入岗位标识', trigger: 'blur' }],
        postStatus: [{ required: true, message: '请输入岗位状态', trigger: 'blur' }]
      },
      ids: [],
      single: true,
      multiple: true
    }
  },
  methods: {
    // 新增岗位方法
    handleAddButtonClick() {
      console.log('新增岗位按钮被点击');
      this.addPostDialogVisible = true;
    },
    // 获取列表
    async getPostList() {
      this.loading = true
      const { data: res } = await this.$api.queryPostList(this.queryParams)  // 调用api
      // console.log("res数据:", res)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.postList = res.data.list
        this.total = res.data.total
        this.loading = false
      }
    },
    // 搜索
    handleQuery() {
      this.getPostList()
    },
    // 重置
    resetQuery() {
      this.queryParams = {}
      this.getPostList()
      this.$message.success("重置成功")
    },
    // pageSize
    handleSizeChange(newSize) {
      this.queryParams.pageSize = newSize
      this.getPostList()
    },
    // pageNum
    handleCurrentChange(newPage) {
      this.queryParams.pageNum = newPage
      this.getPostList()
    },
    // 岗位状态修改
    async postUpdateStatus(row) {
      let text = row.postStatus === 2 ? "停用" : "启用"
      const confirmResult = await this.$confirm('确认要"' + text + '""' + row.postName + '"岗位吗?', "警告", {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
      }).catch(err => err)
      if (confirmResult != 'confirm') {
        await this.getPostList()
        return this.$message.info('已取消修改')
      }
      await this.$api.updatePostStatus(row.id, row.postStatus)
      return this.$message.success(text + "成功")
      // eslint-disable-next-line no-unreachable
      await this.getPostList()
    },
    // 监听对话框的关闭
    addPostDialogClosed() {
     // console.log('Add dialog closed');
      this.$refs.addPostFormRefForm.resetFields()
    },
    // 新增操作
    addPost() {
      this.$refs.addPostFormRefForm.validate(async valid => {
        if (!valid) return
        const { data: res } = await this.$api.addPost(this.addPostForm)
        if (res.code !== 200) {
          this.$message.error(res.message)
        } else {
          this.$message.success("新增岗位成功")
          this.addPostDialogVisible = false
          await this.getPostList()
        }
      })
    },
    // 显示编辑对话框
    async handleUpdate(id) {
      const { data: res } = await this.$api.postInfo(id)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.editPostForm = res.data
        this.editPostDialogVisible = true
      }
    },
    // 监听编辑岗位对话框
    editPostDialogClosed() {
      this.$refs.editPostFormRefForm.resetFields()
    },
    // 修改岗位
    editPost() {
      this.$refs.editPostFormRefForm.validate(async valid => {
        if (!valid) return
        const { data: res } = await this.$api.updatePost(this.editPostForm)
        if (res.code !== 200) {
          this.$message.error(res.message)
        } else {
          this.$message.success("修改岗位成功")
          this.editPostDialogVisible = false
          await this.getPostList()
        }
      })
    },
    // 删除岗位
    async handleDelete(id) {
      const confirmResult = await this.$confirm('是否确认删除岗位编号为"' + id + '"的数据项？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).catch(err => err)
      if (confirmResult !== 'confirm') {
        return this.$message.info('已取消删除')
      }
      const { data: res } = await this.$api.deleteSysPost(id)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.$message.success('删除成功')
        await this.getPostList()
      }
    },
    // 多选框选中数据
    handleSelectionChange(selection) {
      this.ids = selection.map(item => item.id);
      this.single = selection.length != 1;
      this.multiple = !selection.length;
    },
    // 批量删除
    async batchHandleDelete() {
      const postIds = this.ids
      const confirmResult = await this.$confirm('是否确认删除岗位编号为"' + postIds + '"的数据项？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).catch(err => err)
      if (confirmResult !== 'confirm') {
        return this.$message.info('已取消删除')
      }
      const { data: res } = await this.$api.batchDeleteSysPost(postIds)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.$message.success('删除成功')
        await this.getPostList()
      }
    }
  },
  created() {
    this.getPostList()
  },
}
</script>

<style lang="less" scoped>
.post-container {
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
  :deep(.el-form-item) { margin-bottom: 0; }
}
.table-section { overflow: hidden; }
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
