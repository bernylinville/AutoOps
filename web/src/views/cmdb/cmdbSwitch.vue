<template>
  <div class="cmdb-switch-management">
    <el-card shadow="hover" class="switch-card">
      <template #header>
        <div class="card-header">
          <span class="title">网络设备管理</span>
        </div>
      </template>

      <!-- 搜索区域 -->
      <div class="search-section">
        <el-form :inline="true" :model="queryParams" class="demo-form-inline">
          <el-form-item label="设备名称" prop="name">
            <el-input
              size="small"
              placeholder="请输入设备名称"
              clearable
              v-model="queryParams.name"
              @keyup.enter="handleQuery"
              style="width: 180px;"
            />
          </el-form-item>
          <el-form-item label="IP地址" prop="ip">
            <el-input
              size="small"
              placeholder="请输入IP地址"
              clearable
              v-model="queryParams.ip"
              @keyup.enter="handleQuery"
              style="width: 150px;"
            />
          </el-form-item>
          <el-form-item label="设备类型" prop="deviceType">
            <el-select size="small" placeholder="请选择类型" v-model="queryParams.deviceType" style="width: 140px;" clearable>
              <el-option label="交换机" value="switch" />
              <el-option label="路由器" value="router" />
              <el-option label="防火墙" value="firewall" />
              <el-option label="负载均衡" value="loadbalancer" />
            </el-select>
          </el-form-item>
          <div class="action-section">
            <el-button type="primary" size="small" @click="handleQuery">
              <el-icon><Search /></el-icon>
              <span style="margin-left: 4px">搜索</span>
            </el-button>
            <el-button type="warning" size="small" @click="resetQuery">
              <el-icon><Refresh /></el-icon>
              <span style="margin-left: 4px">重置</span>
            </el-button>
            <el-button type="success" size="small" @click="handleCreate" v-authority="['cmdb:switch:add']">
              <el-icon><Plus /></el-icon>
              <span style="margin-left: 4px">新增设备</span>
            </el-button>
          </div>
        </el-form>
      </div>

      <!-- 设备表格 -->
      <el-table
        :data="deviceList"
        v-loading="loading"
        stripe
        border
        style="width: 100%"
        empty-text=" "
      >
        <el-table-column prop="name" label="设备名称" min-width="150" show-overflow-tooltip />
        <el-table-column prop="ip" label="管理IP" min-width="130" />
        <el-table-column prop="deviceType" label="设备类型" min-width="100">
          <template #default="{ row }">
            <el-tag v-if="row.deviceType === 'switch'" type="primary">交换机</el-tag>
            <el-tag v-else-if="row.deviceType === 'router'" type="success">路由器</el-tag>
            <el-tag v-else-if="row.deviceType === 'firewall'" type="danger">防火墙</el-tag>
            <el-tag v-else-if="row.deviceType === 'loadbalancer'" type="warning">负载均衡</el-tag>
            <el-tag v-else type="info">{{ row.deviceType || '未知' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="brand" label="品牌" min-width="100" />
        <el-table-column prop="model" label="型号" min-width="120" />
        <el-table-column prop="status" label="状态" min-width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
              {{ row.status === 1 ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="location" label="位置" min-width="150" show-overflow-tooltip />
        <el-table-column prop="remark" label="备注" min-width="150" show-overflow-tooltip />
        <el-table-column label="操作" width="180" fixed="right" align="center">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="handleEdit(row)">
              <el-icon><Edit /></el-icon>
            </el-button>
            <el-button type="danger" link size="small" @click="handleDelete(row)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>

        <!-- 空状态插槽 -->
        <template #empty>
          <div class="empty-state">
            <el-empty description="暂无网络设备数据">
              <template #image>
                <svg viewBox="0 0 128 128" width="120" height="120" xmlns="http://www.w3.org/2000/svg">
                  <rect x="20" y="35" width="88" height="20" rx="4" fill="#e6e8eb" stroke="#c0c4cc" stroke-width="1.5"/>
                  <circle cx="32" cy="45" r="4" fill="#67c23a"/>
                  <circle cx="44" cy="45" r="4" fill="#409eff"/>
                  <rect x="54" y="42" width="6" height="6" rx="1" fill="#c0c4cc"/>
                  <rect x="64" y="42" width="6" height="6" rx="1" fill="#c0c4cc"/>
                  <rect x="74" y="42" width="6" height="6" rx="1" fill="#c0c4cc"/>
                  <rect x="84" y="42" width="6" height="6" rx="1" fill="#c0c4cc"/>
                  <rect x="94" y="42" width="6" height="6" rx="1" fill="#c0c4cc"/>
                  <rect x="20" y="65" width="88" height="20" rx="4" fill="#e6e8eb" stroke="#c0c4cc" stroke-width="1.5"/>
                  <circle cx="32" cy="75" r="4" fill="#e6a23c"/>
                  <circle cx="44" cy="75" r="4" fill="#909399"/>
                  <rect x="54" y="72" width="6" height="6" rx="1" fill="#c0c4cc"/>
                  <rect x="64" y="72" width="6" height="6" rx="1" fill="#c0c4cc"/>
                  <rect x="74" y="72" width="6" height="6" rx="1" fill="#c0c4cc"/>
                  <rect x="84" y="72" width="6" height="6" rx="1" fill="#c0c4cc"/>
                  <rect x="94" y="72" width="6" height="6" rx="1" fill="#c0c4cc"/>
                  <line x1="64" y1="55" x2="64" y2="65" stroke="#c0c4cc" stroke-width="2" stroke-dasharray="3,2"/>
                </svg>
              </template>
              <div class="empty-tips">
                <p style="color: #909399; font-size: 13px; margin: 8px 0;">
                  您可以通过「新增设备」按钮手动添加交换机、路由器等网络设备
                </p>
              </div>
            </el-empty>
          </div>
        </template>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-section" v-if="total > 0">
        <el-pagination
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
          :current-page="queryParams.pageNum"
          :page-sizes="[10, 50, 100]"
          :page-size="queryParams.pageSize"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
        />
      </div>
    </el-card>
  </div>
</template>

<script>
export default {
  name: 'CmdbSwitch',
  data() {
    return {
      loading: false,
      queryParams: {
        pageNum: 1,
        pageSize: 10,
        name: '',
        ip: '',
        deviceType: ''
      },
      deviceList: [],
      total: 0
    }
  },
  created() {
    this.getDeviceList()
  },
  methods: {
    async getDeviceList() {
      // TODO: 接入后端 API 后启用
      // this.loading = true
      // try {
      //   const { data: res } = await this.$api.getNetworkDevices(this.queryParams)
      //   if (res.code === 200) {
      //     this.deviceList = res.data?.list || []
      //     this.total = res.data?.total || 0
      //   }
      // } finally {
      //   this.loading = false
      // }
    },
    handleQuery() {
      this.queryParams.pageNum = 1
      this.getDeviceList()
    },
    resetQuery() {
      this.queryParams = {
        pageNum: 1,
        pageSize: 10,
        name: '',
        ip: '',
        deviceType: ''
      }
      this.getDeviceList()
    },
    handleCreate() {
      this.$message.info('网络设备新增功能开发中')
    },
    handleEdit(row) {
      this.$message.info('网络设备编辑功能开发中')
    },
    handleDelete(row) {
      this.$message.info('网络设备删除功能开发中')
    },
    handleSizeChange(val) {
      this.queryParams.pageSize = val
      this.getDeviceList()
    },
    handleCurrentChange(val) {
      this.queryParams.pageNum = val
      this.getDeviceList()
    }
  }
}
</script>

<style scoped>
.cmdb-switch-management {
  padding: 0;
}

.switch-card {
  border-radius: 12px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header .title {
  font-size: 18px;
  font-weight: 600;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.search-section {
  margin-bottom: 16px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
}

.action-section {
  margin-top: 8px;
}

.pagination-section {
  margin-top: 16px;
  display: flex;
  justify-content: center;
}

.empty-state {
  padding: 40px 0;
}

.empty-tips {
  text-align: center;
}
</style>
