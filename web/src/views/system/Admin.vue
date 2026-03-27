<template>
  <div class="admin-container">
    <!-- 左右布局容器 -->
    <div class="admin-layout">
      <!-- 左侧部门树 -->
      <div class="dept-tree-section">
        <div class="section-header">部门选择</div>
        <el-tree
            :data="deptList"
            :props="defaultProps"
            node-key="id"
            default-expand-all
            :highlight-current="true"
            @node-click="handleDeptClick"
            @node-expand="handleNodeExpand"
            @node-collapse="handleNodeCollapse"
        >
          <template v-slot="{ node, data }">
            <span :class="{ 'font-weight-bold': !data.parentId }">
              <template v-if="!data.parentId">
                <el-icon v-if="expandedKeys.includes(node.key)" style="margin-right: 5px"><FolderOpened /></el-icon>
                <el-icon v-else style="margin-right: 5px"><Folder /></el-icon>
              </template>
              <template v-else>
                <el-icon style="margin-right: 5px"><Folder /></el-icon>
              </template>
              {{ node.label }}
            </span>
          </template>
        </el-tree>
      </div>
      
      <!-- 右侧用户管理区域 -->
      <div class="user-table-section">
        <!-- 搜索表单 -->
        <div class="search-bar">
          <el-form :inline="true" :model="queryParams">
            <el-form-item label="用户账号" prop="username">
              <el-input size="small" placeholder="请输入用户账号" clearable v-model="queryParams.username" @keyup.enter="handleQuery" />
            </el-form-item>
            <el-form-item label="账号状态" prop="status" style="width: 150px;">
              <el-select size="small" placeholder="账号状态" v-model="queryParams.status">
                <el-option v-for="item in statusList" :key="item.value" :label="item.label" :value="item.value"></el-option>
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" size="small" @click="handleQuery"><el-icon><Search /></el-icon> 搜索</el-button>
              <el-button size="small" @click="resetQuery"><el-icon><Refresh /></el-icon> 重置</el-button>
              <el-button type="primary" size="small" @click="addDialogVisible = true" v-authority="['base:admin:add']"><el-icon><Plus /></el-icon> 新增</el-button>
            </el-form-item>
          </el-form>
        </div>

        <!-- 用户表格 -->
        <el-table v-loading="Loading" :data="adminList" :row-class-name="tableRowClassName">
          <el-table-column label="ID" prop="id" v-if="false" />
          <el-table-column label="用户账号" prop="username">
            <template v-slot="scope"><span class="mono-font">{{ scope.row.username }}</span></template>
          </el-table-column>
          <el-table-column label="用户头像" prop="icon" width="80">
            <template v-slot="scope">
              <el-avatar shape="circle" :size="32" :src="scope.row.icon"></el-avatar>
            </template>
          </el-table-column>
          <el-table-column label="用户昵称" prop="nickname" />
          <el-table-column label="角色名称" prop="roleName">
            <template v-slot="scope">
              <el-tag size="small" type="info" disable-transitions>{{ scope.row.roleName }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="岗位" prop="postName" />
          <el-table-column label="账号状态" width="100">
            <template v-slot="scope">
              <el-switch
                  v-model="scope.row.status"
                  :active-value="1"
                  :inactive-value="2"
                  style="--el-switch-on-color: var(--ao-success); --el-switch-off-color: var(--ao-info);"
                  @change="adminUpdateStatus(scope.row)"
              ></el-switch>
            </template>
          </el-table-column>
          <el-table-column label="操作" fixed="right" width="200">
            <template v-slot="scope">
              <el-button type="primary" link size="small" @click="showEditAdminDialog(scope.row.id)" v-authority="['base:admin:edit']">编辑</el-button>
              <el-button type="danger" link size="small" @click="handleAdminDelete(scope.row)" v-authority="['base:admin:delete']">删除</el-button>
              <el-button type="info" link size="small" @click="handleResetPwd(scope.row)" v-authority="['base:admin:reset']">重置密码</el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 分页 -->
        <el-pagination
            @size-change="handleSizeChange"
            @current-change="handleCurrentChange"
            :current-page="queryParams.pageNum"
            :page-sizes="[10, 50, 100, 500, 1000]"
            :page-size="queryParams.pageSize"
            layout="total, sizes, prev, pager, next, jumper"
            :total="total"
            style="margin-top: 12px; justify-content: flex-end;"
        ></el-pagination>
      </div>
    </div>

    <!-- 新增用户对话框 -->
    <el-dialog title="新增用户" v-model="addDialogVisible" width="640px" @close="addDialogClosed">
      <el-form :model="addForm" :rules="addFormRules" ref="addFormRefForm" label-width="80px">
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="用户名称" prop="username"><el-input v-model="addForm.username" placeholder="请输入英文名称" maxlength="30" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="用户密码" prop="password"><el-input v-model="addForm.password" placeholder="请输入用户密码" type="password" maxlength="20" show-password /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="用户昵称" prop="nickname"><el-input v-model="addForm.nickname" placeholder="请输入中文昵称" maxlength="30" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="归属部门" prop="deptId"><treeselect v-model="addForm.deptId" :options="deptList" :show-count="true" placeholder="请选择归属部门" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="手机号码" prop="phone"><el-input v-model="addForm.phone" placeholder="请输入手机号码" maxlength="11" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="用户邮箱" prop="email"><el-input v-model="addForm.email" placeholder="请输入邮箱" maxlength="50" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="用户状态" prop="status">
              <el-radio-group v-model="addForm.status">
                <el-radio :label="1">正常</el-radio>
                <el-radio :label="2">停用</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="用户岗位" prop="postId"><el-select placeholder="请选择岗位" v-model="addForm.postId" style="width: 100%"><el-option v-for="item in postList" :key="item.id" :label="item.postName" :value="item.id"></el-option></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="用户角色" prop="roleId"><el-select placeholder="请选择角色" v-model="addForm.roleId" style="width: 100%"><el-option v-for="item in roleList" :key="item.id" :label="item.roleName" :value="item.id"></el-option></el-select></el-form-item></el-col>
        </el-row>
        <el-form-item label="个人简介" prop="note">
          <el-input v-model="addForm.note" type="textarea" placeholder="请输入内容"></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addDialogVisible = false">取 消</el-button>
        <el-button type="primary" @click="addAdmin">确 定</el-button>
      </template>
    </el-dialog>

    <!-- 编辑用户对话框 -->
    <el-dialog title="修改用户" v-model="editDialogVisible" width="640px" @close="editDialogClosed">
      <el-form :model="adminInfo" :rules="editFormRules" ref="editFormRefForm" label-width="80px">
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="用户昵称" prop="nickname"><el-input v-model="adminInfo.nickname" placeholder="请输入用户昵称" maxlength="30" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="归属部门" prop="deptId"><treeselect v-model="adminInfo.deptId" :options="deptList" :show-count="true" placeholder="请选择归属部门" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="手机号码" prop="phone"><el-input v-model="adminInfo.phone" placeholder="请输入手机号码" maxlength="11" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="用户邮箱" prop="email"><el-input v-model="adminInfo.email" placeholder="请输入邮箱" maxlength="50" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="用户名称" prop="username"><el-input v-model="adminInfo.username" placeholder="请输入用户名称" maxlength="30" /></el-form-item></el-col>
          <el-col :span="12">
            <el-form-item label="用户状态" prop="status">
              <el-radio-group v-model="adminInfo.status">
                <el-radio :label="1">正常</el-radio>
                <el-radio :label="2">停用</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="用户岗位" prop="postId"><el-select placeholder="请选择岗位" v-model="adminInfo.postId" style="width: 100%"><el-option v-for="item in postList" :key="item.id" :label="item.postName" :value="item.id"></el-option></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="用户角色" prop="roleId"><el-select placeholder="请选择角色" v-model="adminInfo.roleId" style="width: 100%"><el-option v-for="item in roleList" :key="item.id" :label="item.roleName" :value="item.id"></el-option></el-select></el-form-item></el-col>
        </el-row>
        <el-form-item label="个人简介" prop="note">
          <el-input v-model="adminInfo.note" type="textarea" placeholder="请输入内容"></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取 消</el-button>
        <el-button type="primary" @click="editAdminInfo">确 定</el-button>
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
  Edit,
  Delete,
  Key,
  Folder,
  FolderOpened
} from '@element-plus/icons-vue'

export default {
  components: {
    Treeselect
  },
  data() {
    return {
      expandedKeys: [], // 用于跟踪展开的节点key
      statusList: [{
        value: '1',
        label: '启用',
      }, {
        value: '2',
        label: '停用',
      }],
      Loading: false,
      queryParams: {},
      adminList: [],
      total: 0,
      addDialogVisible: false,
      deptList: [],// 确保初始化为空数组
      roleList: [],
      postList: [],
      addForm: {
        username: '',
        password: '',
        deptId: undefined,
        postId: undefined,
        roleId: undefined,
        email: '',
        nickname: '',
        status: 1,
        phone: '',
        note: ''
      },
      addFormRules: {
        deptId: [{required: true, message: '请选择部门', trigger: 'blur'}],
        postId: [{required: true, message: '请选择岗位', trigger: 'blur'}],
        roleId: [{required: true, message: '请选择角色', trigger: 'blur'}],
        username: [{required: true, message: '请输入用户账号', trigger: 'blur'}],
        password: [{required: true, message: '请输入用户密码', trigger: 'blur'}],
        status: [{required: true, message: '请选择状态', trigger: 'blur'}],
        email: [{required: true, message: '请输入用户邮箱', trigger: 'blur'}],
        nickname: [{required: true, message: '请输入用户昵称', trigger: 'blur'}],
        phone: [{required: true, message: '请输入用户手机', trigger: 'blur'}]
      },
      editDialogVisible: false,
      adminInfo: {},
      editFormRules: {
        deptId: [{required: true, message: '请选择部门', trigger: 'blur'}],
        postId: [{required: true, message: '请选择岗位', trigger: 'blur'}],
        roleId: [{required: true, message: '请选择角色', trigger: 'blur'}],
        username: [{required: true, message: '请输入用户账号', trigger: 'blur'}],
        status: [{required: true, message: '请选择状态', trigger: 'blur'}],
        email: [{required: true, message: '请输入用户邮箱', trigger: 'blur'}],
        nickname: [{required: true, message: '请输入用户昵称', trigger: 'blur'}],
        phone: [{required: true, message: '请输入用户手机', trigger: 'blur'}]
      },
      defaultProps: {
        children: 'children',
        label: 'label'
      },
      currentDeptId: null
    }
  },
  created() {
    this.getDeptVoList()
    this.getRoleVoList()
    this.getPostVoList()
    this.getAdminList()
  },
  methods: {
    // 表格行样式
    tableRowClassName({row, rowIndex}) {
      return 'modern-table-row';
    },
    
    // 查询列表
    async getAdminList() {
      this.Loading = true
      const {data: res} = await this.$api.queryAdminList(this.queryParams)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.adminList = res.data.list
        this.total = res.data.total
        this.Loading = false
      }
    },

    // 根据部门查询用户
    async loadAdminsByDept(deptId) {
      console.log('开始加载部门 ID:', deptId);
      this.Loading = true;
      const { data: res } = await this.$api.deptUsers(deptId);
      console.log('接口返回结果:', res);

      if (res.code !== 200) {
        this.$message.error(res.message);
      } else {
        if (!res.data || res.data.length === 0) {
          // 🔥 显式处理无数据的情况
          this.adminList = [];
          this.total = 0;
          this.queryParams.pageNum = 1;
          this.Loading = false;
          return;
        }

        // 添加字段映射逻辑
        const users = res.data.map(item => ({
          id: item.id,
          username: item.username,
          nickname: item.nickname,
          status: item.status,
          icon: item.icon,
          email: item.email,
          phone: item.phone,
          note: item.note,
          createTime: item.create_time || item.createTime,
          deptName: item.dept_name || item.deptName,
          postName: item.post_name || item.postName,
          roleName: item.role_name || item.roleName
        }));
        this.adminList = users;
        this.total = users.length;
        this.queryParams.pageNum = 1; // 回到第一页
      }
      this.Loading = false;
    }

    ,

    // 搜索按钮操作
    handleQuery() {
      this.getAdminList();
    },

    // 重置按钮操作
    resetQuery() {
      this.queryParams = {}
      this.getAdminList();
      this.$message.success("重置成功")
    },

    // pageSize
    handleSizeChange(newSize) {
      this.queryParams.pageSize = newSize
      this.getAdminList()
    },

    // pageNum
    handleCurrentChange(newPage) {
      this.queryParams.pageNum = newPage
      this.getAdminList()
    },

    // 修改用户状态
    async adminUpdateStatus(row) {
      let text = row.status === 2 ? "停用" : "启用";
      const confirmResult = await this.$confirm('确认要"' + text + '""' + row.username + '"用户吗?', "警告", {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
      }).catch(err => err);

      if (confirmResult !== 'confirm') {
        await this.getAdminList();
        return this.$message.info('已取消删除');
      }

      // 更新状态
      const { data: res } = await this.$api.updateAdminStatus(row.id, row.status);
      if (res.code === 200) {
        this.$message.success(text + "成功");

        // 手动更新当前行状态，避免重新加载整个列表
        const index = this.adminList.findIndex(item => item.id === row.id);
        if (index > -1) {
          this.$set(this.adminList, index, {...row}); // 强制触发 Vue 响应式更新
        }
      } else {
        this.$message.error(res.message);
      }
    },

    // 部门下拉列表
    async getDeptVoList() {
      const { data: res } = await this.$api.querySysDeptVoList()
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        const treeData = this.$handleTree.handleTree(res.data, "id")
        this.deptList = treeData
      }
    }
    ,

    // 角色下拉列表
    async getRoleVoList() {
      const {data: res} = await this.$api.querySysRoleVoList()
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.roleList = res.data
      }
    },

    // 岗位下拉列表
    async getPostVoList() {
      const {data: res} = await this.$api.querySysPostVoList()
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.postList = res.data
      }
    },

    // 监听添加用户对话框关闭
    addDialogClosed() {
      this.$refs.addFormRefForm.resetFields()
    },

    // 新增用户
    addAdmin() {
      this.$refs.addFormRefForm.validate(async valid => {
        if (!valid) return
        const {data: res} = await this.$api.addAdmin(this.addForm);
        if (res.code !== 200) {
          this.$message.error(res.message)
        } else {
          this.$message.success('新增用户成功')
          this.addDialogVisible = false
          await this.getAdminList()
        }
      })
    },

    // 展示编辑用户的对话框
    async showEditAdminDialog(id) {
      const {data: res} = await this.$api.adminInfo(id) // [adminInfo](file:///Users/apple/Desktop/zhangfan/2025-06-30/gin-web/src/api/index.js#L333-L342)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.adminInfo = res.data
        
        // 如果用户有部门ID，获取部门信息以确保treeselect正确显示
        if (this.adminInfo.deptId) {
          try {
            const {data: deptRes} = await this.$api.deptInfo(this.adminInfo.deptId)
            if (deptRes.code === 200) {
              // 确保部门信息在treeselect选项中可用
              console.log('Department info loaded:', deptRes.data)
            }
          } catch (error) {
            console.warn('Failed to load department info:', error)
          }
        }
        
        this.editDialogVisible = true
      }
    },

    // 监听修改用户对话框的关闭事件
    editDialogClosed() {
      this.$refs.editFormRefForm.resetFields()
    },

    // 修改用户信息并提交
    editAdminInfo() {
      this.$refs.editFormRefForm.validate(async valid => {
        if (!valid) return
        const {data: res} = await this.$api.adminUpdate(this.adminInfo) // [adminUpdate](file:///Users/apple/Desktop/zhangfan/2025-06-30/gin-web/src/api/index.js#L352-L361)
        if (res.code !== 200) {
          this.$message.error(res.message)
        } else {
          this.editDialogVisible = false
          await this.getAdminList()
          this.$message.success('修改用户成功')
        }
      })
    },

    // 删除
    async handleAdminDelete(row) {
      const confirmResult = await this.$confirm('是否确认删除用户为"' + row.username + '"的数据项？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).catch(err => err)
      if (confirmResult !== 'confirm') {
        return this.$message.info('已取消删除')
      }
      const {data: res} = await this.$api.deleteAdmin(row.id) // [deleteAdmin](file:///Users/apple/Desktop/zhangfan/2025-06-30/gin-web/src/api/index.js#L373-L382)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.$message.success('删除成功')
        await this.getAdminList()
      }
    },

    // 重置密码
    handleResetPwd(row) {
      this.$prompt('请输入"' + row.username + '"的新密码', "提示", {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        closeOnClickModal: false,
        inputPattern: /^.{5,20}$/,
        inputErrorMessage: "用户密码长度必须介于 5 和 20 之间"
      }).then(({value}) => {
        this.$api.resetPassword(row.id, value).then(() => { // [resetPassword](file:///Users/apple/Desktop/zhangfan/2025-06-30/gin-web/src/api/index.js#L363-L371)
          this.$message.success("修改成功，新密码是：" + value);
        });
      }).catch(() => {
      });
    },

// 点击部门节点
    handleNodeExpand(data, node) {
      if (!this.expandedKeys.includes(node.key)) {
        this.expandedKeys.push(node.key)
      }
    },

    handleNodeCollapse(data, node) {
      this.expandedKeys = this.expandedKeys.filter(key => key !== node.key)
    },

    handleDeptClick(node, element) {
      let deptId;

      // 多种方式尝试获取 deptId
      if (element && element.data && element.data.id) {
        deptId = element.data.id;
      } else if (element && element.id) {
        deptId = element.id;
      } else if (node && node.key) {
        deptId = node.key;
      }
      console.log('最终获取到的 deptId:', deptId);
      if (!deptId) {
        this.$message.warning("无法获取部门ID");
        return;
      }
      this.currentDeptId = deptId;
      this.loadAdminsByDept(deptId);
    }





  }
}
</script>

<style scoped>
/* 主容器 — 无渐变，使用页面背景色 */
.admin-container {
  padding: var(--ao-page-padding);
}

.admin-layout {
  display: flex;
  gap: var(--ao-spacing-md);
  min-height: 600px;
}

.dept-tree-section {
  width: 260px;
  flex-shrink: 0;
  background: var(--ao-bg-white);
  border-radius: var(--ao-radius);
  border: 1px solid var(--ao-border-lighter);
  padding: 16px;
}

.section-header {
  font-size: 14px;
  font-weight: 600;
  color: var(--ao-text-primary);
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--ao-border-lighter);
}

.user-table-section {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--ao-spacing-md);
}

.search-bar {
  background: var(--ao-bg-white);
  border-radius: var(--ao-radius);
  border: 1px solid var(--ao-border-lighter);
  padding: 16px;
}

.search-bar .el-form-item {
  margin-bottom: 0;
}

.font-weight-bold {
  font-weight: 600;
}

@media (max-width: 1200px) {
  .admin-layout {
    flex-direction: column;
  }
  .dept-tree-section {
    width: 100%;
  }
}
</style>
