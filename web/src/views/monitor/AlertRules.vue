<template>
  <div class="alert-rules-container">
    <!-- 告警规则 Tab -->
    <el-tabs v-model="activeTab" type="border-card">
      <el-tab-pane label="告警规则" name="rules">
        <div class="toolbar">
          <el-button type="primary" @click="showRuleDialog('create')">
            <el-icon><Plus /></el-icon> 新建规则
          </el-button>
          <el-button @click="fetchRules">
            <el-icon><Refresh /></el-icon> 刷新
          </el-button>
        </div>

        <el-table :data="rules" v-loading="rulesLoading" stripe>
          <el-table-column prop="name" label="规则名称" min-width="160" />
          <el-table-column prop="severity" label="级别" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="severityType(row.severity)" size="small">{{ row.severity }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="source" label="来源" width="100" />
          <el-table-column prop="notifyChannels" label="通知渠道" width="160">
            <template #default="{ row }">
              <span v-for="ch in parseJSON(row.notifyChannels)" :key="ch">
                <el-tag size="small" style="margin-right:4px">{{ ch }}</el-tag>
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="enabled" label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" link @click="showRuleDialog('edit', row)">编辑</el-button>
              <el-popconfirm title="确定删除此规则？" @confirm="deleteRule(row.id)">
                <template #reference>
                  <el-button size="small" type="danger" link>删除</el-button>
                </template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>

        <el-pagination
          v-model:current-page="rulesPage"
          :page-size="10"
          :total="rulesTotal"
          layout="total, prev, pager, next"
          @current-change="fetchRules"
          style="margin-top: 16px; justify-content: flex-end;"
        />
      </el-tab-pane>

      <!-- 通知渠道 Tab -->
      <el-tab-pane label="通知渠道" name="channels">
        <div class="toolbar">
          <el-button type="primary" @click="showChannelDialog('create')">
            <el-icon><Plus /></el-icon> 新建渠道
          </el-button>
          <el-button @click="fetchChannels">
            <el-icon><Refresh /></el-icon> 刷新
          </el-button>
        </div>

        <el-table :data="channels" v-loading="channelsLoading" stripe>
          <el-table-column prop="name" label="渠道名称" min-width="160" />
          <el-table-column prop="type" label="类型" width="120" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="channelTypeColor(row.type)">
                {{ channelTypeLabel(row.type) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="enabled" label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="success" link @click="testChannel(row.id)">测试</el-button>
              <el-button size="small" type="primary" link @click="showChannelDialog('edit', row)">编辑</el-button>
              <el-popconfirm title="确定删除此渠道？" @confirm="deleteChannel(row.id)">
                <template #reference>
                  <el-button size="small" type="danger" link>删除</el-button>
                </template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 规则 Dialog -->
    <el-dialog v-model="ruleDialogVisible" :title="ruleDialogTitle" width="600px">
      <el-form :model="ruleForm" label-width="100px">
        <el-form-item label="规则名称" required>
          <el-input v-model="ruleForm.name" />
        </el-form-item>
        <el-form-item label="严重级别">
          <el-select v-model="ruleForm.severity" style="width:100%">
            <el-option label="Critical" value="critical" />
            <el-option label="Warning" value="warning" />
            <el-option label="Info" value="info" />
          </el-select>
        </el-form-item>
        <el-form-item label="告警来源">
          <el-select v-model="ruleForm.source" style="width:100%">
            <el-option label="N9E 夜莺" value="n9e" />
            <el-option label="Prometheus" value="prometheus" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="通知渠道">
          <el-checkbox-group v-model="ruleNotifyChannels">
            <el-checkbox label="wechat">企业微信</el-checkbox>
            <el-checkbox label="dingtalk">钉钉</el-checkbox>
            <el-checkbox label="email">邮件</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="ruleForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="ruleForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRule" :loading="saving">保存</el-button>
      </template>
    </el-dialog>

    <!-- 渠道 Dialog -->
    <el-dialog v-model="channelDialogVisible" :title="channelDialogTitle" width="600px">
      <el-form :model="channelForm" label-width="100px">
        <el-form-item label="渠道名称" required>
          <el-input v-model="channelForm.name" />
        </el-form-item>
        <el-form-item label="渠道类型" required>
          <el-select v-model="channelForm.type" @change="onChannelTypeChange" style="width:100%">
            <el-option label="企业微信" value="wechat" />
            <el-option label="钉钉" value="dingtalk" />
            <el-option label="邮件" value="email" />
          </el-select>
        </el-form-item>
        <template v-if="channelForm.type === 'wechat' || channelForm.type === 'dingtalk'">
          <el-form-item label="Webhook URL" required>
            <el-input v-model="channelConfig.webhookUrl" placeholder="https://qyapi.weixin.qq.com/..." />
          </el-form-item>
        </template>
        <template v-if="channelForm.type === 'email'">
          <el-form-item label="SMTP Host" required>
            <el-input v-model="channelConfig.host" placeholder="smtp.example.com" />
          </el-form-item>
          <el-form-item label="SMTP Port">
            <el-input-number v-model="channelConfig.port" :min="1" :max="65535" />
          </el-form-item>
          <el-form-item label="用户名">
            <el-input v-model="channelConfig.username" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="channelConfig.password" type="password" show-password />
          </el-form-item>
          <el-form-item label="发件人">
            <el-input v-model="channelConfig.from" />
          </el-form-item>
          <el-form-item label="收件人">
            <el-input v-model="channelConfig.toStr" placeholder="多个邮箱用逗号分隔" />
          </el-form-item>
        </template>
        <el-form-item label="启用">
          <el-switch v-model="channelForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="channelDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveChannel" :loading="saving">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Refresh } from '@element-plus/icons-vue'
import request from '@/utils/request'

const activeTab = ref('rules')

// ---- 告警规则 ----
const rules = ref([])
const rulesLoading = ref(false)
const rulesPage = ref(1)
const rulesTotal = ref(0)
const ruleDialogVisible = ref(false)
const ruleDialogTitle = ref('新建告警规则')
const ruleMode = ref('create')
const ruleNotifyChannels = ref([])
const saving = ref(false)

const ruleForm = reactive({
  id: 0, name: '', severity: 'warning', source: 'n9e',
  matchLabels: '', notifyChannels: '', notifyTarget: '',
  enabled: true, description: ''
})

const fetchRules = async () => {
  rulesLoading.value = true
  try {
    const res = await request({ url: 'n9e/alert/rules', method: 'get', params: { page: rulesPage.value, pageSize: 10 } })
    const d = res.data || res
    if (d.code === 200 && d.data) {
      rules.value = d.data.list || []
      rulesTotal.value = d.data.total || 0
    }
  } catch (e) { console.error(e) }
  rulesLoading.value = false
}

const showRuleDialog = (mode, row) => {
  ruleMode.value = mode
  ruleDialogTitle.value = mode === 'create' ? '新建告警规则' : '编辑告警规则'
  if (row) {
    Object.assign(ruleForm, row)
    ruleNotifyChannels.value = parseJSON(row.notifyChannels) || []
  } else {
    Object.assign(ruleForm, { id: 0, name: '', severity: 'warning', source: 'n9e', matchLabels: '', notifyChannels: '', notifyTarget: '', enabled: true, description: '' })
    ruleNotifyChannels.value = []
  }
  ruleDialogVisible.value = true
}

const saveRule = async () => {
  ruleForm.notifyChannels = JSON.stringify(ruleNotifyChannels.value)
  saving.value = true
  try {
    const method = ruleMode.value === 'create' ? 'post' : 'put'
    await request({ url: 'n9e/alert/rules', method, data: ruleForm })
    ElMessage.success(ruleMode.value === 'create' ? '创建成功' : '更新成功')
    ruleDialogVisible.value = false
    fetchRules()
  } catch (e) { ElMessage.error('操作失败') }
  saving.value = false
}

const deleteRule = async (id) => {
  try {
    await request({ url: `n9e/alert/rules/${id}`, method: 'delete' })
    ElMessage.success('删除成功')
    fetchRules()
  } catch (e) { ElMessage.error('删除失败') }
}

// ---- 通知渠道 ----
const channels = ref([])
const channelsLoading = ref(false)
const channelDialogVisible = ref(false)
const channelDialogTitle = ref('新建通知渠道')
const channelMode = ref('create')
const channelForm = reactive({ id: 0, name: '', type: 'wechat', config: '', enabled: true })
const channelConfig = reactive({ webhookUrl: '', host: '', port: 587, username: '', password: '', from: '', toStr: '' })

const fetchChannels = async () => {
  channelsLoading.value = true
  try {
    const res = await request({ url: 'n9e/alert/channels', method: 'get' })
    const d = res.data || res
    if (d.code === 200) channels.value = d.data || []
  } catch (e) { console.error(e) }
  channelsLoading.value = false
}

const showChannelDialog = (mode, row) => {
  channelMode.value = mode
  channelDialogTitle.value = mode === 'create' ? '新建通知渠道' : '编辑通知渠道'
  if (row) {
    Object.assign(channelForm, row)
    const cfg = parseJSON(row.config) || {}
    Object.assign(channelConfig, { webhookUrl: cfg.webhookUrl || '', host: cfg.host || '', port: cfg.port || 587, username: cfg.username || '', password: cfg.password || '', from: cfg.from || '', toStr: (cfg.to || []).join(',') })
  } else {
    Object.assign(channelForm, { id: 0, name: '', type: 'wechat', config: '', enabled: true })
    Object.assign(channelConfig, { webhookUrl: '', host: '', port: 587, username: '', password: '', from: '', toStr: '' })
  }
  channelDialogVisible.value = true
}

const onChannelTypeChange = () => {
  Object.assign(channelConfig, { webhookUrl: '', host: '', port: 587, username: '', password: '', from: '', toStr: '' })
}

const saveChannel = async () => {
  let config = {}
  if (channelForm.type === 'wechat' || channelForm.type === 'dingtalk') {
    config = { webhookUrl: channelConfig.webhookUrl }
  } else if (channelForm.type === 'email') {
    config = { host: channelConfig.host, port: channelConfig.port, username: channelConfig.username, password: channelConfig.password, from: channelConfig.from, to: channelConfig.toStr.split(',').map(s => s.trim()).filter(Boolean) }
  }
  channelForm.config = JSON.stringify(config)
  saving.value = true
  try {
    const method = channelMode.value === 'create' ? 'post' : 'put'
    await request({ url: 'n9e/alert/channels', method, data: channelForm })
    ElMessage.success(channelMode.value === 'create' ? '创建成功' : '更新成功')
    channelDialogVisible.value = false
    fetchChannels()
  } catch (e) { ElMessage.error('操作失败') }
  saving.value = false
}

const deleteChannel = async (id) => {
  try {
    await request({ url: `n9e/alert/channels/${id}`, method: 'delete' })
    ElMessage.success('删除成功')
    fetchChannels()
  } catch (e) { ElMessage.error('删除失败') }
}

const testChannel = async (id) => {
  try {
    const res = await request({ url: `n9e/alert/channels/${id}/test`, method: 'post' })
    const d = res.data || res
    if (d.code === 200) ElMessage.success('测试通知已发送')
    else ElMessage.error(d.message || '发送失败')
  } catch (e) { ElMessage.error('测试失败') }
}

// ---- Helpers ----
const parseJSON = (str) => { try { return JSON.parse(str) } catch { return null } }
const severityType = (s) => ({ critical: 'danger', warning: 'warning', info: 'info' }[s] || '')
const channelTypeColor = (t) => ({ wechat: 'success', dingtalk: 'primary', email: 'warning' }[t] || '')
const channelTypeLabel = (t) => ({ wechat: '企业微信', dingtalk: '钉钉', email: '邮件' }[t] || t)

onMounted(() => { fetchRules(); fetchChannels() })
</script>

<style scoped>
.alert-rules-container { padding: 20px; }
.toolbar { margin-bottom: 16px; display: flex; gap: 8px; }
</style>
