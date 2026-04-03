<template>
  <div class="table-section">
    <el-table
        v-loading="loading"
        :data="hostListWithMonitor"
        stripe
        style="width: 100%"
        class="host-table"
        @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" width="45" />
      <el-table-column label="ID" prop="id" v-if="false" />
      <el-table-column label="主机名称" width="180" show-overflow-tooltip>
        <template v-slot="scope">
          <div class="host-name-cell" @mouseenter="showCopyIcon($event, 'hostname')" @mouseleave="hideCopyIcon">
            <img 
              src="@/assets/image/linux.svg" 
              alt="linux"
              style="height: 20px; object-fit: contain; flex-shrink: 0;"
            />
            <el-link type="primary" @click="$emit('show-detail', scope.row)">{{ scope.row.hostName }}</el-link>
            <el-icon 
              class="copy-icon" 
              @click="copyToClipboard(scope.row.hostName, '主机名称', $event)"
              style="display: none; margin-left: 5px; cursor: pointer; color: #409EFF;"
            >
              <DocumentCopy />
            </el-icon>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="IP地址" width="150" show-overflow-tooltip>
        <template v-slot="scope">
          <div class="ip-cell" @mouseenter="showCopyIcon($event, 'ip')" @mouseleave="hideCopyIcon">
            <div v-if="scope.row.publicIp" class="ip-row public-ip">
              <img 
                src="@/assets/image/公.svg" 
                alt="公网"
                class="ip-icon"
              />
              <span>{{ scope.row.publicIp || '无公网IP' }}</span>
              <el-icon 
                class="copy-icon" 
                @click="copyToClipboard(scope.row.publicIp, '公网IP', $event)"
                style="display: none; margin-left: 5px; cursor: pointer; color: #409EFF;"
              >
                <DocumentCopy />
              </el-icon>
            </div>
            <div v-if="scope.row.privateIp" class="ip-row private-ip">
              <img 
                src="@/assets/image/内.svg" 
                alt="内网"
                class="ip-icon"
              />
              <span>{{ scope.row.privateIp || '无内网IP' }}</span>
              <el-icon 
                class="copy-icon" 
                @click="copyToClipboard(scope.row.privateIp, '内网IP', $event)"
                style="display: none; margin-left: 5px; cursor: pointer; color: #67C23A;"
              >
                <DocumentCopy />
              </el-icon>
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="CPU使用率">
        <template v-slot="scope">
          <el-progress 
            :percentage="scope.row.cpuUsage || 0" 
            :color="getUsageColor(scope.row.cpuUsage)"
            :show-text="true"
          />
        </template>
      </el-table-column>
      <el-table-column label="内存使用率">
        <template v-slot="scope">
          <el-progress 
            :percentage="scope.row.memoryUsage || 0" 
            :color="getUsageColor(scope.row.memoryUsage)"
            :show-text="true"
          />
        </template>
      </el-table-column>
      <el-table-column label="磁盘使用率">
        <template v-slot="scope">
          <el-progress 
            :percentage="scope.row.diskUsage || 0" 
            :color="getUsageColor(scope.row.diskUsage)"
            :show-text="true"
          />
        </template>
      </el-table-column>
      <el-table-column label="进程" width="70" align="center">
        <template v-slot="scope">
          <el-tooltip class="item" effect="light" content="查看进程监控" placement="top">
            <img
              src="@/assets/image/进程.svg"
              alt="进程"
              style="width: 24px; height: 24px; cursor: pointer;"
              @click="showProcessMonitor(scope.row)"
            />
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column label="端口" width="70" align="center">
        <template v-slot="scope">
          <el-tooltip class="item" effect="light" content="查看TCP端口监控" placement="top">
            <img
              src="@/assets/image/端口.svg"
              alt="端口"
              style="width: 24px; height: 24px; cursor: pointer;"
              @click="showTcpPortMonitor(scope.row)"
            />
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column label="配置信息" show-overflow-tooltip>
        <template v-slot="scope">
          <div class="config-cell">
            <img 
              src="@/assets/image/配置.svg" 
              alt="配置"
              style="width: 16px; height: 16px; margin-right: 6px; flex-shrink: 0;"
            />
            <span>{{ scope.row.cpu }}核{{ scope.row.memory }}G</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="存活状态">
        <template v-slot="scope">
          <div class="status-cell">
            <img 
              :src="scope.row.isAlive ? require('@/assets/image/主机在线.svg') : require('@/assets/image/主机离线.svg')" 
              :alt="scope.row.isAlive ? '在线' : '离线'"
              style="width: 16px; height: 16px; margin-right: 6px; flex-shrink: 0;"
            />
            <el-tag :type="scope.row.isAlive ? 'success' : 'danger'" size="small">
              {{ scope.row.isAlive ? '在线' : '离线' }}
            </el-tag>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="主机状态" width="120">
        <template v-slot="scope">
          <el-tag :type="getStatusTagType(scope.row.status)" size="small">
            {{ getStatusText(scope.row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="主机类型" prop="vendor" show-overflow-tooltip>
        <template v-slot="scope">
          <div class="vendor-cell">
            <template v-if="scope.row.vendor == 1">
              <el-icon :size="18" color="#409EFF"><OfficeBuilding /></el-icon>
              <span>自建</span>
            </template>
            <template v-else-if="scope.row.vendor == 2">
              <img src="@/assets/image/aliyun.png" style="width: 18px; height: 18px"/>
              <span>阿里</span>
            </template>
            <template v-else-if="scope.row.vendor == 3">
              <img src="@/assets/image/tengxun.png" style="width: 18px; height: 18px"/>
              <span>腾讯</span>
            </template>
            <template v-else-if="scope.row.vendor == 4">
              <img src="@/assets/image/baidu.svg" style="width: 18px; height: 18px"/>
              <span>百度</span>
            </template>
            <template v-else>
              {{ scope.row.vendor }}
            </template>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="数据来源" width="100" align="center">
        <template v-slot="scope">
          <el-tag v-if="scope.row.sourceType === 'n9e'" type="" size="small" style="background: #7c3aed; color: #fff; border: none;">N9E</el-tag>
          <el-tag v-else-if="scope.row.sourceType === 'aliyun' || scope.row.sourceType === 'tencent'" type="warning" size="small">云同步</el-tag>
          <el-tag v-else type="success" size="small">手动</el-tag>
        </template>
      </el-table-column>


    

      <el-table-column label="操作" fixed="right" width="280" min-width="280">
        <template v-slot="scope">
          <div class="table-operation">
            <el-button-group>
              <el-tooltip class="item" effect="light" content="编辑" placement="top-end">
                <el-button
                  type="primary"
                  icon="Edit"
                  size="mini"
                  circle
                  plain
                  v-authority="['cmdb:ecs:edit']"
                  @click="$emit('edit-host', scope.row.id)"
                />
              </el-tooltip>
              <el-tooltip class="item" effect="light" content="上传" placement="top-end">
                <el-button
                  type="primary"
                  icon="Upload"
                  size="mini"
                  circle
                  plain
                   v-authority="['cmdb:ecs:upload']"
                  @click="$emit('show-upload', scope.row)"
                />
              </el-tooltip>
              <el-tooltip class="item" effect="light" content="执行命令" placement="top-end">
                <el-button
                  type="success"
                  icon="Position"
                  size="mini"
                  circle
                  plain
                   v-authority="['cmdb:ecs:shell']"
                  @click="$emit('execute-command', scope.row)"
                />
              </el-tooltip>
              <el-tooltip class="item" effect="light" content="删除" placement="top-end">
                <el-button
                  type="danger"
                  icon="Delete"
                  size="mini"
                  circle
                  plain
                   v-authority="['cmdb:ecs:delete']"
                  @click="$emit('delete-host', scope.row)"
                />
              </el-tooltip>
              <el-tooltip class="item" effect="light" content="监控" placement="top-end">
                <el-button
                  type="info"
                  icon="Monitor"
                  size="mini"
                  circle
                  plain
                  v-authority="['cmdb:ecs:monitor']"
                  @click="showMonitor(scope.row)"
                />
              </el-tooltip>
              <el-tooltip class="item" effect="light" content="告警历史" placement="top-end">
                <el-button
                  type="warning"
                  icon="Bell"
                  size="mini"
                  circle
                  plain
                  @click="showAlertHistory(scope.row)"
                />
              </el-tooltip>
              <el-tooltip class="item" effect="light" content="生命周期变更" placement="top-end">
                <el-button
                  type="primary"
                  icon="Promotion"
                  size="mini"
                  circle
                  plain
                  v-authority="['cmdb:ecs:edit']"
                  @click="openLifecycleDialog(scope.row)"
                />
              </el-tooltip>
            </el-button-group>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <!-- 批量操作工具栏（选中时显示） -->
    <div v-if="selectedHosts.length > 0" class="batch-action-bar">
      <span style="font-size:13px;color:#606266;margin-right:12px">
        已选择 <strong>{{ selectedHosts.length }}</strong> 台主机
      </span>
      <el-button type="warning" size="small" icon="Promotion" @click="openBatchLifecycleDialog">
        批量生命周期变更
      </el-button>
      <el-button size="small" @click="clearSelection">取消选择</el-button>
    </div>

    <!-- 生命周期状态变更对话框 -->
    <el-dialog v-model="lifecycleDialog.visible" title="变更主机生命周期状态" width="400px" :append-to-body="true">
      <div v-if="lifecycleDialog.host">
        <el-descriptions :column="1" border size="small" style="margin-bottom:16px">
          <el-descriptions-item label="主机名称">{{ lifecycleDialog.host.hostName }}</el-descriptions-item>
          <el-descriptions-item label="当前状态">
            <el-tag :type="getStatusTagType(lifecycleDialog.host.status)" size="small">
              {{ getStatusText(lifecycleDialog.host.status) }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>
        <el-form label-width="80px">
          <el-form-item label="目标状态" required>
            <el-select v-model="lifecycleDialog.targetStatus" placeholder="请选择目标状态" style="width:100%">
              <el-option
                v-for="s in lifecycleDialog.allowedStatuses"
                :key="s.value"
                :label="s.label"
                :value="s.value"
              />
            </el-select>
          </el-form-item>
        </el-form>
        <div v-if="lifecycleDialog.allowedStatuses.length === 0" style="color:#909399;font-size:13px;text-align:center;padding:8px 0">
          当前状态无可用转换目标（终态或由系统自动管理）
        </div>
      </div>
      <template #footer>
        <el-button @click="lifecycleDialog.visible = false">取消</el-button>
        <el-button type="primary" :loading="lifecycleDialog.loading"
          :disabled="!lifecycleDialog.targetStatus"
          @click="submitLifecycle">确认变更</el-button>
      </template>
    </el-dialog>

    <!-- 批量生命周期变更对话框 -->
    <el-dialog v-model="batchLifecycleDialog.visible" title="批量变更主机生命周期状态" width="420px" :append-to-body="true">
      <div>
        <p style="color:#606266;font-size:13px;margin-bottom:12px">
          已选择 <strong>{{ batchLifecycleDialog.count }}</strong> 台主机，请选择目标状态（不符合转换规则的主机将被跳过）：
        </p>
        <el-form label-width="80px">
          <el-form-item label="目标状态" required>
            <el-select v-model="batchLifecycleDialog.targetStatus" placeholder="请选择目标状态" style="width:100%">
              <el-option v-for="s in batchStatusOptions" :key="s.value" :label="s.label" :value="s.value" />
            </el-select>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="batchLifecycleDialog.visible = false">取消</el-button>
        <el-button type="primary" :loading="batchLifecycleDialog.loading"
          :disabled="!batchLifecycleDialog.targetStatus"
          @click="submitBatchLifecycle">确认变更</el-button>
      </template>
    </el-dialog>

    <monitor-dialog
      v-if="showMonitorDialog"
      v-model="showMonitorDialog"
      :host-id="currentHostId"
      style="z-index: 2001"
    />

    <process-monitor-dialog
      v-if="showProcessDialog"
      v-model="showProcessDialog"
      :host-id="currentProcessHostId"
      style="z-index: 2002"
    />

    <tcp-port-monitor-dialog
      v-if="showTcpPortDialog"
      v-model="showTcpPortDialog"
      :host-id="currentTcpPortHostId"
      style="z-index: 2003"
    />

    <host-alert-history
      v-if="showAlertDialog"
      :visible="showAlertDialog"
      :host-ident="currentAlertIdent"
      :host-name="currentAlertHostName"
      @update:visible="showAlertDialog = $event"
      @close="showAlertDialog = false"
    />
  </div>
</template>

<script>
import MonitorDialog from './MonitorDialog.vue'
import ProcessMonitorDialog from './ProcessMonitorDialog.vue'
import TcpPortMonitorDialog from './TcpPortMonitorDialog.vue'
import HostAlertHistory from './HostAlertHistory.vue'

export default {
  name: 'CmdbHostTable',
  components: {
    MonitorDialog,
    ProcessMonitorDialog,
    TcpPortMonitorDialog,
    HostAlertHistory
  },
  props: {
    hostList: {
      type: Array,
      required: true
    },
    loading: {
      type: Boolean,
      default: false
    }
  },
  data() {
    return {
      monitorData: {},
      refreshInterval: null,
      refreshRate: 10000, // 监控数据和存活状态查询频率，单位毫秒(当前10秒)
      showMonitorDialog: false,
      currentHostId: '',
      showProcessDialog: false,
      currentProcessHostId: '',
      showTcpPortDialog: false,
      currentTcpPortHostId: '',
      showAlertDialog: false,
      currentAlertIdent: '',
      currentAlertHostName: '',
      isFirstOpen: true,
      lifecycleDialog: {
        visible: false,
        host: null,
        targetStatus: null,
        allowedStatuses: [],
        loading: false
      },
      selectedHosts: [],
      batchLifecycleDialog: {
        visible: false,
        count: 0,
        targetStatus: null,
        loading: false
      }
    }
  },
  watch: {
    hostList: {
      immediate: true,
      deep: true,
      handler(newVal) {
        if (newVal && newVal.length > 0) {
          this.stopRefresh() // 先停止之前的刷新
          this.fetchMonitorData() // 立即加载数据
          this.startRefresh() // 重新启动定时刷新
        }
      }
    }
  },
  computed: {
    hostListWithMonitor() {
      
      const result = this.hostList.map(host => {
        const monitor = this.monitorData[host.id] || {}
        const merged = {
          ...host,
          cpuUsage: monitor.cpuUsage,
          memoryUsage: monitor.memoryUsage,
          diskUsage: monitor.diskUsage,
          isAlive: monitor.onlineStatus === 0
        }
        return merged
      })
      
      return result
    },
    batchStatusOptions() {
      // 对批量操作，列出所有可能的手动转换目标状态
      const statusNames = {
        1: '在线', 6: '采购中', 7: '入库', 8: '待上线', 9: '退服申请', 10: '已报废'
      }
      return Object.keys(statusNames).map(v => ({ value: Number(v), label: statusNames[v] }))
    }
  },
  methods: {
    getStatusText(status) {
      const statusMap = {
        1: '在线',
        2: '未认证',
        3: '离线',
        4: '失联',
        5: '降级',
        6: '采购中',
        7: '入库',
        8: '待上线',
        9: '退服申请',
        10: '已报废'
      }
      return statusMap[status] || '未知'
    },
    getStatusTagType(status) {
      const typeMap = {
        1: 'success',
        2: 'warning',
        3: 'danger',
        4: 'danger',
        5: 'warning',
        6: 'info',
        7: 'info',
        8: 'warning',
        9: 'danger',
        10: 'info'
      }
      return typeMap[status] || 'info'
    },
    getUsageColor(usage) {
      if (!usage) return '#909399'
      if (usage > 80) return '#F56C6C'
      if (usage > 60) return '#E6A23C'
      return '#67C23A'
    },
    async fetchMonitorData() {
      if (!this.hostList || this.hostList.length === 0) return

      const validHosts = this.hostList.filter(host => host?.id)
      if (validHosts.length === 0) return

      const ids = validHosts.map(host => host.id).join(',')
      
      // 请求合并后的监控数据接口
      const monitorRes = await this.$api.getHostsMonitorData(ids)

      // 立即更新数据，不等待任何延迟
      if (monitorRes.data.code === 200) {
        this.monitorData = {
          ...this.monitorData,
          ...monitorRes.data.data
        }
      }
    },
    startRefresh() {
      // 立即执行第一次数据加载
      this.fetchMonitorData()
      // 设置定时刷新，但确保第一次加载不等待
      this.refreshInterval = setInterval(() => {
        this.fetchMonitorData()
      }, this.refreshRate)
    },
    stopRefresh() {
      if (this.refreshInterval) {
        clearInterval(this.refreshInterval)
        this.refreshInterval = null
      }
    },
    showMonitor(host) {
      this.currentHostId = host.id
      this.showMonitorDialog = true
      this.isFirstOpen = true
    },
    showProcessMonitor(host) {
      this.currentProcessHostId = host.id
      this.showProcessDialog = true
    },
    showTcpPortMonitor(host) {
      this.currentTcpPortHostId = host.id
      this.showTcpPortDialog = true
    },
    showAlertHistory(host) {
      // 使用 IP 作为 ident 关联 FlashDuty
      this.currentAlertIdent = host.privateIp || host.publicIp || host.sshIp || ''
      this.currentAlertHostName = host.hostName || ''
      this.showAlertDialog = true
    },
    
    // 显示复制图标
    showCopyIcon(event, type) {
      const icons = event.currentTarget.querySelectorAll('.copy-icon')
      icons.forEach(icon => {
        icon.style.display = 'inline-block'
      })
    },
    
    // 隐藏复制图标
    hideCopyIcon(event) {
      const icons = event.currentTarget.querySelectorAll('.copy-icon')
      icons.forEach(icon => {
        icon.style.display = 'none'
      })
    },
    
    // ——— 生命周期变更 ———
    openLifecycleDialog(host) {
      const allowedMap = {
        0:  [{ value: 6, label: '采购中' }],
        6:  [{ value: 7, label: '入库' }],
        7:  [{ value: 8, label: '待上线' }],
        8:  [{ value: 1, label: '在线' }],
        1:  [{ value: 9, label: '退服申请' }, { value: 5, label: '降级' }],
        3:  [{ value: 9, label: '退服申请' }],
        4:  [{ value: 9, label: '退服申请' }],
        5:  [{ value: 9, label: '退服申请' }, { value: 1, label: '在线' }],
        9:  [{ value: 10, label: '已报废' }, { value: 1, label: '撤回在线' }],
        10: []
      }
      this.lifecycleDialog.host = host
      this.lifecycleDialog.targetStatus = null
      this.lifecycleDialog.allowedStatuses = allowedMap[host.status] || []
      this.lifecycleDialog.visible = true
    },
    async submitLifecycle() {
      if (!this.lifecycleDialog.targetStatus) return
      this.lifecycleDialog.loading = true
      try {
        const res = await this.$api.updateHostLifecycle(
          this.lifecycleDialog.host.id,
          this.lifecycleDialog.targetStatus
        )
        if (res.data.code === 200) {
          this.$message.success('状态变更成功')
          this.lifecycleDialog.visible = false
          this.$emit('lifecycle-changed')
        } else {
          this.$message.error(res.data.msg || '变更失败')
        }
      } catch {
        this.$message.error('请求失败')
      } finally {
        this.lifecycleDialog.loading = false
      }
    },

    // ——— 批量生命周期变更 ———
    handleSelectionChange(rows) {
      this.selectedHosts = rows
    },
    clearSelection() {
      this.selectedHosts = []
    },
    openBatchLifecycleDialog() {
      if (this.selectedHosts.length === 0) {
        this.$message.warning('请先选择主机')
        return
      }
      this.batchLifecycleDialog.count = this.selectedHosts.length
      this.batchLifecycleDialog.targetStatus = null
      this.batchLifecycleDialog.visible = true
    },
    async submitBatchLifecycle() {
      if (!this.batchLifecycleDialog.targetStatus) return
      this.batchLifecycleDialog.loading = true
      try {
        const ids = this.selectedHosts.map(h => h.id)
        const res = await this.$api.batchUpdateHostLifecycle(ids, this.batchLifecycleDialog.targetStatus)
        if (res.data.code === 200) {
          const data = res.data.data
          if (data.failed && data.failed.length > 0) {
            this.$message.warning(`成功 ${data.success} 台，跳过 ${data.failed.length} 台（规则不符）`)
          } else {
            this.$message.success(`批量变更成功，共 ${data.success} 台`)
          }
          this.batchLifecycleDialog.visible = false
          this.selectedHosts = []
          this.$emit('lifecycle-changed')
        } else {
          this.$message.error(res.data.msg || '变更失败')
        }
      } catch {
        this.$message.error('请求失败')
      } finally {
        this.batchLifecycleDialog.loading = false
      }
    },

    // 复制到剪贴板
    async copyToClipboard(text, type, event) {
      try {
        await navigator.clipboard.writeText(text)
        this.$message.success(`${type} 已复制: ${text}`)
        
        // 添加复制成功的视觉反馈
        if (event && event.target) {
          const icon = event.target.closest('.copy-icon')
          if (icon) {
            icon.classList.add('copied')
            setTimeout(() => {
              icon.classList.remove('copied')
            }, 1000)
          }
        }
      } catch (error) {
        // 降级方案
        const textArea = document.createElement('textarea')
        textArea.value = text
        document.body.appendChild(textArea)
        textArea.focus()
        textArea.select()
        try {
          document.execCommand('copy')
          this.$message.success(`${type} 已复制: ${text}`)
          
          // 添加复制成功的视觉反馈
          if (event && event.target) {
            const icon = event.target.closest('.copy-icon')
            if (icon) {
              icon.classList.add('copied')
              setTimeout(() => {
                icon.classList.remove('copied')
              }, 1000)
            }
          }
        } catch (fallbackError) {
          this.$message.error('复制失败，请手动复制')
        }
        document.body.removeChild(textArea)
      }
    }
  },
    mounted() {
      // 如果hostList已有数据，立即获取监控数据
      if (this.hostList && this.hostList.length > 0) {
        this.fetchMonitorData()
      }
      // 启动定时刷新
      this.startRefresh()
    },
  beforeUnmount() {
    this.stopRefresh()
  },
    beforeRouteEnter(to, from, next) {
      next(vm => {
        vm.stopRefresh()
        vm.fetchMonitorData()
        vm.startRefresh()
      })
    },
    beforeRouteUpdate(to, from, next) {
      this.stopRefresh()
      this.fetchMonitorData()
      this.startRefresh()
      next()
    },
    activated() {
      this.stopRefresh()
      this.fetchMonitorData()
      this.startRefresh()
    }
}
</script>

<style scoped>
/* 🎨 现代化科技感表格样式 */

.table-section {
  margin-bottom: 15px;
  width: 100%;
}

/* 📊 表格样式 */
.host-table {
  border-radius: var(--ao-radius-lg);
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}

.host-table :deep(.el-table__header) {
  background: var(--ao-bg-page);
}

.host-table :deep(.el-table__header th) {
  background: transparent !important;
  color: #2c3e50 !important;
  font-weight: 700 !important;
  border-bottom: none;
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.8);
}

.host-table :deep(.el-table__header th .cell) {
  color: #2c3e50 !important;
  font-weight: 700 !important;
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.8);
}

.host-table :deep(.el-table__row) {
  transition: all 0.3s ease;
}

.host-table :deep(.el-table__row:hover) {
  background-color: rgba(64, 158, 255, 0.05) !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

/* 🎯 操作按钮样式 */
.table-operation {
  display: flex;
  justify-content: space-around;
  white-space: nowrap;
  min-width: 280px;
}

.table-operation .el-button {
  transition: all 0.3s ease;
}

.table-operation .el-button:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

/* 操作栏按钮组不换行 */
.el-button-group {
  white-space: nowrap;
}

/* 批量操作工具栏 */
.batch-action-bar {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: #f0f9ff;
  border: 1px solid #b3d8ff;
  border-radius: 4px;
  margin-top: 8px;
}

/* 🏷️ 标签样式优化 */
.el-tag {
  font-weight: 500;
  border-radius: 8px;
  border: none;
}

/* 📊 进度条样式 */
.el-progress {
  margin: 2px 0;
}

/* 🔗 链接样式 */
.el-link {
  font-weight: 600;
  transition: all 0.3s ease;
}

.el-link:hover {
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

/* 🔧 表格单元格样式 - 防止换行 */
.host-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
  overflow: hidden;
  position: relative;
}

.ip-cell {
  white-space: nowrap;
  overflow: hidden;
}

.ip-row {
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
  font-size: 12px;
  line-height: 1.2;
  position: relative;
}

.ip-row.public-ip {
  color: #409EFF;
  margin-bottom: 2px;
}

.ip-row.private-ip {
  color: #67C23A;
}

.ip-row span {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.vendor-cell {
  display: flex;
  align-items: center;
  gap: 5px;
  white-space: nowrap;
  overflow: hidden;
}

.vendor-cell span {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.config-cell {
  display: flex;
  align-items: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.status-cell {
  display: flex;
  align-items: center;
  white-space: nowrap;
  overflow: hidden;
}

/* 确保所有表格单元格内容不换行 */
.host-table :deep(.el-table__cell) {
  white-space: nowrap;
  overflow: hidden;
}

.host-table :deep(.cell) {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 📋 复制图标样式 */
.copy-icon {
  opacity: 0;
  transition: all 0.3s ease;
  font-size: 14px !important;
  padding: 2px;
  border-radius: 4px;
}

.copy-icon:hover {
  background-color: rgba(64, 158, 255, 0.1);
}

/* 鼠标悬停时显示复制图标 */
.host-name-cell:hover .copy-icon,
.ip-row:hover .copy-icon {
  opacity: 1;
  display: inline-block !important;
}

/* 复制成功动画效果 */
.copy-icon.copied {
  color: #67C23A !important;
}

/* IP图标样式 */
.ip-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  object-fit: contain;
}
</style>
