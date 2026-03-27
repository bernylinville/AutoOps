<template>
  <div class="modern-menu-container">
    <!-- 主卡片 -->
    <div class="main-card">
      <!-- 卡片标题 -->
      <div class="card-header">
        <h1 class="gradient-title">菜单管理</h1>
      </div>

      <!-- 搜索区域 -->
      <div class="search-section">
        <el-form :inline="true" :model="queryParams" class="search-form">
          <el-form-item prop="menuName" label="菜单名称">
            <el-input 
              placeholder="请输入菜单名称" 
              clearable 
              size="small" 
              @keyup.enter="handleQuery"
              v-model="queryParams.menuName" 
              class="modern-input" />
          </el-form-item>
          <el-form-item prop="menuStatus" label="菜单状态">
            <el-select 
              size="small" 
              placeholder="菜单状态" 
              v-model="queryParams.menuStatus" 
              class="modern-select"
              style="width: 150px;">
              <el-option v-for="item in menuStatusList" :key="item.value" :label="item.label" :value="item.value">
              </el-option>
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="small" @click="handleQuery" class="modern-btn primary-btn">
              <el-icon><Search /></el-icon>
              搜索
            </el-button>
            <el-button size="small" @click="resetQuery" class="modern-btn reset-btn">
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
            @click="addMenuDialogVisible = true" 
            v-authority="['base:menu:add']"
            class="modern-btn success-btn">
            <el-icon><Plus /></el-icon>
            新增菜单
          </el-button>
          <el-button 
            size="small" 
            @click="toggleExpandAll"
            class="modern-btn secondary-btn">
            <el-icon><Sort /></el-icon>
            展开/折叠
          </el-button>
        </div>
      </div>

      <!-- 数据表格区域 -->
      <div class="table-section">
      <el-table 
        v-if="refreshTable"
        v-loading="loading" 
        :data="menuList" 
        row-key="id" 
        :default-expand-all="isExpandAll"
        :tree-props="{ children: 'children' }"
        class="modern-table"
        :header-cell-style="{ background: 'transparent', color: '#2c3e50', fontWeight: '600' }"
        :row-style="{ background: 'rgba(255, 255, 255, 0.05)' }"
        stripe>
        <el-table-column prop="menuName" label="菜单名称" min-width="150">
          <template v-slot="scope">
            <span class="menu-name">{{ scope.row.menuName }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="icon" label="图标" width="80">
          <template v-slot="scope">
            <div class="icon-wrapper">
              <el-icon class="menu-icon">
                <component :is="scope.row.icon" />
              </el-icon>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="80" />
        <el-table-column prop="value" label="权限标识" min-width="150" />
        <el-table-column prop="url" label="组件路径" min-width="200" />
        <el-table-column prop="menuType" label="菜单类型" width="100">
          <template v-slot="scope">
            <el-tag 
              v-if="scope.row.menuType === 1" 
              class="modern-tag modern-tag-directory">
              目录
            </el-tag>
            <el-tag 
              v-else-if="scope.row.menuType === 2" 
              class="modern-tag modern-tag-menu">
              菜单
            </el-tag>
            <el-tag 
              v-else-if="scope.row.menuType === 3" 
              class="modern-tag modern-tag-button">
              按钮
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="menuStatus" label="状态" width="100">
          <template v-slot="scope">
            <el-tag 
              v-if="scope.row.menuStatus === 2" 
              class="modern-tag modern-tag-active">
              启用
            </el-tag>
            <el-tag 
              v-else-if="scope.row.menuStatus === 1" 
              class="modern-tag modern-tag-inactive">
              禁用
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createTime" min-width="160" />
        <el-table-column label="操作" width="190" fixed="right">
          <template v-slot="scope">
            <div class="operation-buttons">
              <el-tooltip content="修改" placement="top">
                <el-button
                  type="warning"
                  size="small"
                  circle
                  @click="showEditMenuDialog(scope.row.id)"
                  v-authority="['base:menu:edit']"
                >
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="复制" placement="top">
                <el-button
                  type="primary"
                  size="small"
                  circle
                  @click="handleCopyMenu(scope.row)"
                  v-authority="['base:menu:add']"
                >
                  <el-icon><DocumentCopy /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button
                  type="danger"
                  size="small"
                  circle
                  @click="handleMenuDelete(scope.row)"
                  v-authority="['base:admin:delete']"
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

    <!-- 新增菜单对话框 -->
    <el-dialog 
      title="新增菜单" 
      v-model="addMenuDialogVisible" 
      width="600px" 
      @close="addMenuDialogClosed"
      class="modern-dialog"
      :modal-class="'modern-modal'">
      <div class="dialog-content">
        <el-form :model="menuForm" :rules="addMenuFormRules" ref="addMenuFormRefForm" label-width="100px" class="modern-form">
          <el-row :gutter="20">
            <el-col :span="24">
              <el-form-item label="菜单类型" prop="menuType">
                <el-radio-group v-model="menuForm.menuType" class="modern-radio-group">
                  <el-radio :label="1" class="modern-radio">目录</el-radio>
                  <el-radio :label="2" class="modern-radio">菜单</el-radio>
                  <el-radio :label="3" class="modern-radio">按钮</el-radio>
                </el-radio-group>
              </el-form-item>
            </el-col>
            <el-col :span="24" v-if="menuForm.menuType != 1">
              <el-form-item label="上级菜单" prop="parentId">
                <treeselect :options="treeList" placeholder="请选择上级菜单" v-model="menuForm.parentId" class="modern-treeselect" />
              </el-form-item>
            </el-col>
            <el-col :span="24" v-if="menuForm.menuType != 3">
              <el-form-item label="菜单图标" prop="icon">
                <el-select v-model="menuForm.icon" class="modern-select" placeholder="请选择图标">
                  <el-option v-for="item in iconList" :key="item.value" :label="item.label" :value="item.value">
                      <span style="display: flex; align-items: center;">
                        <el-icon style="font-size: 20px;">
                          <component :is="item.value" />
                        </el-icon>
                        <span style="margin-left: 8px;">{{ item.label }}</span>
                      </span>
                  </el-option>
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="菜单名称" prop="menuName">
                <el-input v-model="menuForm.menuName" placeholder="请输入菜单名称" class="modern-input" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="显示排序" prop="sort">
                <el-input-number v-model="menuForm.sort" controls-position="right" :min="0" class="modern-input-number" />
              </el-form-item>
            </el-col>
            <el-col :span="24" v-if="menuForm.menuType != 3">
              <el-form-item label="菜单路径" prop="url">
                <el-input v-model="menuForm.url" placeholder="请输入菜单路径" class="modern-input" />
              </el-form-item>
            </el-col>
            <el-col :span="24" v-if="menuForm.menuType != 1">
              <el-form-item label="权限标识" prop="value">
                <el-input v-model="menuForm.value" placeholder="请输入权限标识" maxlength="50" class="modern-input" />
              </el-form-item>
            </el-col>
            <el-col :span="24" v-if="menuForm.menuType != 3">
              <el-form-item label="显示状态" prop="menuStatus">
                <el-radio-group v-model="menuForm.menuStatus" class="modern-radio-group">
                  <el-radio :label="1" class="modern-radio">停用</el-radio>
                  <el-radio :label="2" class="modern-radio">启用</el-radio>
                </el-radio-group>
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="addMenuDialogVisible = false" class="modern-btn modern-btn-cancel">取消</el-button>
          <el-button type="primary" @click="addMenu" class="modern-btn modern-btn-primary">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 修改菜单对话框 -->
    <el-dialog 
      title="修改菜单" 
      v-model="editMenuDialogVisible" 
      width="600px" 
      @close="editMenuDialogClosed"
      class="modern-dialog"
      :modal-class="'modern-modal'">
      <div class="dialog-content">
        <el-form :model="menuInfo" :rules="editMenuFormRules" ref="editMenuFormRefForm" label-width="100px" class="modern-form">
          <el-row :gutter="20">
            <el-col :span="24">
              <el-form-item label="菜单类型" prop="menuType">
                <el-radio-group v-model="menuInfo.menuType" class="modern-radio-group">
                  <el-radio :label="1" class="modern-radio">目录</el-radio>
                  <el-radio :label="2" class="modern-radio">菜单</el-radio>
                  <el-radio :label="3" class="modern-radio">按钮</el-radio>
                </el-radio-group>
              </el-form-item>
            </el-col>
            <el-col :span="24" v-if="menuInfo.menuType != 1">
              <el-form-item label="上级菜单" prop="parentId">
                <treeselect :options="treeList" placeholder="请选择上级菜单" v-model="menuInfo.parentId" class="modern-treeselect" />
              </el-form-item>
            </el-col>
            <el-col :span="24" v-if="menuInfo.menuType != 3">
              <el-form-item label="菜单图标" prop="icon">
                <el-select v-model="menuInfo.icon" class="modern-select" placeholder="请选择图标">
                  <el-option v-for="item in iconList" :key="item.value" :label="item.label" :value="item.value">
                      <span style="display: flex; align-items: center;">
                        <el-icon style="font-size: 20px;">
                          <component :is="item.value" />
                        </el-icon>
                        <span style="margin-left: 8px;">{{ item.label }}</span>
                      </span>
                  </el-option>
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="菜单名称" prop="menuName">
                <el-input v-model="menuInfo.menuName" placeholder="请输入菜单名称" class="modern-input" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="显示排序" prop="sort">
                <el-input-number v-model="menuInfo.sort" controls-position="right" :min="0" class="modern-input-number" />
              </el-form-item>
            </el-col>
            <el-col :span="24" v-if="menuInfo.menuType != 3">
              <el-form-item label="菜单路径" prop="url">
                <el-input v-model="menuInfo.url" placeholder="请输入菜单路径" class="modern-input" />
              </el-form-item>
            </el-col>
            <el-col :span="24" v-if="menuInfo.menuType != 1">
              <el-form-item label="权限标识" prop="value">
                <el-input v-model="menuInfo.value" placeholder="请输入权限标识" maxlength="50" class="modern-input" />
              </el-form-item>
            </el-col>
            <el-col :span="24" v-if="menuInfo.menuType != 3">
              <el-form-item label="显示状态" prop="menuStatus">
                <el-radio-group v-model="menuInfo.menuStatus" class="modern-radio-group">
                  <el-radio :label="1" class="modern-radio">停用</el-radio>
                  <el-radio :label="2" class="modern-radio">启用</el-radio>
                </el-radio-group>
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="editMenuDialogVisible = false" class="modern-btn modern-btn-cancel">取消</el-button>
          <el-button type="primary" @click="editMenu" class="modern-btn modern-btn-primary">确定</el-button>
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
  Delete,
  DocumentCopy
} from '@element-plus/icons-vue'

export default {
  components: { Treeselect },
  data() {
    return {
      queryParams: {},
      menuStatusList: [{
        value: '2',
        label: '启用'
      }, {
        value: '1',
        label: '停用'
      }],
      loading: true,
      menuList: [],
      isExpandAll: false,
      refreshTable: true,
      iconList: [
        {value: 'HomeFilled', label: 'HomeFilled'},
        {value: 'UploadFilled', label: 'UploadFilled'},
        {value: 'Menu', label: 'Menu'},
        {value: 'Search', label: 'Search'},
        {value: 'Edit', label: 'Edit'},
        {value: 'Delete', label: 'Delete'},
        {value: 'More', label: 'More'},
        {value: 'Star', label: 'Star'},
        {value: 'StarFilled', label: 'StarFilled'},
        {value: 'Platform', label: 'Platform'},
        {value: 'TrendCharts', label: 'TrendCharts'},
        {value: 'Document', label: 'Document'},
        {value: 'Eleme', label: 'Eleme'},
        {value: 'Delete', label: 'Delete'},
        {value: 'Tools', label: 'Tools'},
        {value: 'Setting', label: 'Setting'},
        {value: 'User', label: 'User'},
        {value: 'Phone', label: 'Phone'},
        {value: 'Goods', label: 'Goods'},
        {value: 'Help', label: 'Help'},
        {value: 'Picture', label: 'Picture'},
        {value: 'Upload', label: 'Upload'},
        {value: 'Download', label: 'Download'},
        {value: 'Promotion', label: 'Promotion'},
        {value: 'Shop', label: 'Shop'},
        {value: 'menu', label: 'Menu'},
        {value: 'share', label: 'hare'},
        {value: 'bottom', label: 'Bottom'},
        {value: 'top', label: 'Top'},
        {value: 'key', label: 'Key'},
        {value: 'unlock', label: 'Unlock'},
        {value: 'shopping-cart-full', label: 'ShoppingCartFull'},
        {value: 'Coin', label: 'Coin'},
        {value: 'present', label: 'Present'},
        {value: 'box', label: 'Box'},
        {value: 'wallet', label: 'Wallet'},
        {value: 'discount', label: 'Discount'},
        {value: 'price-tag', label: 'PriceTag'},
        {value: 'guide', label: 'Guide'},
        {value: 'connection', label: 'Connection'},
        {value: 'chat-dot-round', label: 'ChatDotRound'}
      ],
      addMenuDialogVisible: false,
      menuForm: {
        menuStatus: 2
      },
      addMenuFormRules: {
        menuType: [{ required: true, message: "菜单类型不能为空", trigger: "blur" }],
        menuName: [{ required: true, message: "菜单名称不能为空", trigger: "blur" }],
        sort: [{ required: true, message: "菜单顺序不能为空", trigger: "blur" }],
        value: [{ required: true, message: "权限标识不能为空", trigger: "blur" }]
      },
      treeList: [],
      editMenuDialogVisible: false,
      menuInfo: [],
      editMenuFormRules: {
        menuType: [{ required: true, message: "菜单类型不能为空", trigger: "blur" }],
        menuName: [{ required: true, message: "菜单名称不能为空", trigger: "blur" }],
        sort: [{ required: true, message: "菜单顺序不能为空", trigger: "blur" }],
        value: [{ required: true, message: "权限标识不能为空", trigger: "blur" }]
      },
    }
  },
  methods: {
    // 列表
    async getMenuList() {
      this.loading = true;
      const { data: res } = await this.$api.queryMenuList(this.queryParams)
      // console.log(res)
      if (res.code !== 200) {
        this.$message.error(res.message);
      } else {
        this.menuList = this.$handleTree.handleTree(res.data, "id");
        this.loading = false;
      }
    },
    // 搜索
    handleQuery() {
      this.getMenuList();
    },
    // 重置
    resetQuery() {
      this.queryParams = {}
      this.getMenuList();
      this.$message.success("重置成功")
    },
    // 展开/折叠
    toggleExpandAll() {
      this.refreshTable = false
      this.isExpandAll = !this.isExpandAll
      this.$nextTick(() => {
        this.refreshTable = true
      })
    },
    // 新增菜单关闭事件
    addMenuDialogClosed() {
      this.$refs.addMenuFormRefForm.resetFields()
    },
    // 按sort字段对菜单数据排序
    sortMenusBySort(menuList) {
      if (!menuList || !Array.isArray(menuList)) {
        return [];
      }

      // 对当前层级按sort字段排序
      const sortedList = [...menuList].sort((a, b) => {
        const sortA = a.sort || 0;
        const sortB = b.sort || 0;
        return sortA - sortB;
      });

      return sortedList;
    },

    // 新增下拉列表
    async getMenuVoList() {
      try {
        // 同时获取菜单数据和带sort的菜单数据
        const [menuVoRes, menuWithSortRes] = await Promise.all([
          this.$api.querySysMenuVoList(),
          this.$api.queryMenuList({})
        ]);

        if (menuVoRes.data.code !== 200) {
          this.$message.error(menuVoRes.data.message);
          return;
        }

        const menuVoData = menuVoRes.data.data;
        const menuWithSort = menuWithSortRes.data.data;

        // 创建id到sort值的映射
        const sortMap = {};
        if (menuWithSort && Array.isArray(menuWithSort)) {
          menuWithSort.forEach(menu => {
            sortMap[menu.id] = menu.sort || 0;
          });
        }

        // 为菜单数据添加sort字段
        const menuWithSortField = menuVoData.map(menu => ({
          ...menu,
          sort: sortMap[menu.id] || 999
        }));

        // 按sort字段排序
        const sortedMenus = this.sortMenusBySort(menuWithSortField);

        // 构建树形结构
        this.treeList = this.$handleTree.handleTree(sortedMenus, "id");

        console.log('上级菜单排序结果:', this.treeList);
      } catch (error) {
        console.error('获取菜单列表失败:', error);
        this.$message.error('获取菜单列表失败');
      }
    },
    // 新增操作
    addMenu() {
      this.$refs.addMenuFormRefForm.validate(async valid => {
        if (!valid) return

        // 清理复制时可能产生的无效字段
        const submitData = { ...this.menuForm }
        delete submitData.children
        delete submitData.createTime
        delete submitData.updateTime

        console.log('提交的菜单数据:', submitData)

        try {
          const { data: res } = await this.$api.addMenu(submitData);
          if (res.code === 200) {
            this.$message.success("新增菜单成功")
            this.addMenuDialogVisible = false
            await this.getMenuList()
            await this.getMenuVoList()
          } else {
            this.$message.error(res.message || "新增菜单失败")
          }
        } catch (error) {
          console.error('新增菜单错误:', error)
          // 如果是400错误但实际可能创建成功，先关闭对话框再刷新列表
          if (error.response?.status === 400) {
            this.$message.warning("操作完成，正在刷新数据...")
            this.addMenuDialogVisible = false
            await this.getMenuList()
            await this.getMenuVoList()
          } else {
            this.$message.error("新增菜单失败: " + (error.message || "未知错误"))
          }
        }
      })
    },
    // 监听修改菜单关闭事件
    editMenuDialogClosed() {
      this.$refs.editMenuFormRefForm.resetFields()
    },
    // 打开菜单
    async showEditMenuDialog(id) {
      const { data: res } = await this.$api.menuInfo(id)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.menuInfo = res.data
        this.editMenuDialogVisible = true
      }
    },
    // 修改菜单
    editMenu() {
      this.$refs.editMenuFormRefForm.validate(async valid => {
        if (!valid) return
        const { data: res } = await this.$api.menuUpdate(this.menuInfo)
        if (res.code !== 200) {
          this.$message.error(res.message)
        } else {
          this.editMenuDialogVisible = false
          await this.getMenuList()
          this.$message.success("修改菜单成功")
        }
      })
    },
    // 复制菜单
    handleCopyMenu(menuData) {
      // 复制数据并清除不需要的字段
      const copyData = {
        parentId: menuData.parentId,
        menuName: `${menuData.menuName}_副本`,
        icon: menuData.icon || '',
        value: menuData.value ? `${menuData.value}_copy` : '',
        menuType: menuData.menuType,
        url: menuData.url ? `${menuData.url}_copy` : '',
        sort: menuData.sort,
        menuStatus: menuData.menuStatus || 2
      }

      // 重置表单并填充复制的数据
      this.$nextTick(() => {
        if (this.$refs.addMenuFormRefForm) {
          this.$refs.addMenuFormRefForm.resetFields()
        }
      })

      this.menuForm = { ...copyData }
      this.addMenuDialogVisible = true

      // 调试输出
      console.log('复制的菜单数据:', copyData)
    },
    // 删除菜单
    async handleMenuDelete(row) {
      const confirmResult = await this.$confirm('是否确认删除菜单为"' + row.menuName + '"的数据项？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).catch(err => err)
      if (confirmResult !== 'confirm') {
        return this.$message.info('已取消删除')
      }
      const { data: res } = await this.$api.menuDelete(row.id)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.$message.success('删除成功')
        await this.getMenuList()
      }
    },
  },
  created() {
    this.getMenuList()
    this.getMenuVoList()
  }
}
</script>

<style lang="less" scoped>
.menu-container {
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
.action-section { margin-bottom: 8px; }
.table-section { overflow: hidden; }
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
