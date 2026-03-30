<template>
  <div class="n9e-config-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <h3>N9E 夜莺监控配置</h3>
      <p class="page-desc">配置 N9E (夜莺) 监控系统的连接信息，用于同步业务组、主机和数据源。</p>
    </div>

    <el-row :gutter="20">
      <!-- 左侧：配置表单和 Webhook 引导 -->
      <el-col :span="14">
        <el-card shadow="hover" class="config-card">
          <template #header>
            <div class="card-header">
              <span><el-icon><Setting /></el-icon> 连接配置</span>
              <el-tag :type="configForm.enabled ? 'success' : 'info'" size="small">
                {{ configForm.enabled ? '已启用' : '未启用' }}
              </el-tag>
            </div>
          </template>

          <el-form ref="configFormRef" :model="configForm" :rules="formRules" label-width="120px" label-position="right">
            <el-form-item label="N9E 地址" prop="endpoint">
              <el-input v-model="configForm.endpoint" placeholder="http://n9e-server:17000" clearable>
                <template #prefix><el-icon><Link /></el-icon></template>
              </el-input>
            </el-form-item>

            <el-form-item label="用户 Token" prop="token">
              <el-input v-model="configForm.token" type="password" placeholder="X-User-Token" show-password clearable>
                <template #prefix><el-icon><Key /></el-icon></template>
              </el-input>
            </el-form-item>

            <el-form-item label="超时时间">
              <el-input-number v-model="configForm.timeout" :min="5" :max="120" :step="5" />
              <span class="form-tip">秒</span>
            </el-form-item>

            <el-form-item label="自动同步">
              <el-input v-model="configForm.syncCron" placeholder="例: 0 */30 * * * (每30分钟)" clearable style="width: 280px" />
              <span class="form-tip">Cron 表达式</span>
            </el-form-item>

            <el-form-item label="启用集成">
              <el-switch v-model="configForm.enabled" active-text="启用" inactive-text="禁用" />
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="handleSaveConfig" :loading="saving" :icon="Check">保存配置</el-button>
              <el-button @click="handleTestConnection" :loading="testing" :icon="Connection">测试连接</el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <!-- Webhook 配置指引 -->
        <el-card shadow="hover" class="webhook-card">
          <template #header>
            <div class="card-header webhook-header">
              <span><el-icon><Bell /></el-icon> 告警 Webhook 接入指引</span>
              <el-tooltip content="配置在 N9E 的推板或配置中，将告警通过 Webhook 推送到 AutoOps 统一处理留痕" placement="top">
                <el-icon class="info-icon"><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
          </template>
          
          <el-alert
            title="通过配置 Webhook, 可使 N9E 发生告警时自动将事件推送到 AutoOps, 并交由内置规则分析。"
            type="info"
            show-icon
            :closable="false"
            style="margin-bottom: 20px"
          />

          <el-form label-position="top">
            <el-form-item label="Webhook URL (推送端点)">
              <div class="copy-box">
                <span class="code-text">{{ webhookUrl }}</span>
                <el-button type="primary" link :icon="DocumentCopy" @click="copyText(webhookUrl)">复制</el-button>
              </div>
            </el-form-item>
            
            <el-form-item label="X-Webhook-Token (安全校验 Header)">
              <div class="copy-box">
                <span class="code-text">webhook-notify-token-2024</span>
                <el-button type="primary" link :icon="DocumentCopy" @click="copyText('webhook-notify-token-2024')">复制</el-button>
              </div>
            </el-form-item>
          </el-form>
          
          <div class="webhook-note">
            <h4><el-icon><Warning /></el-icon> 注意事项：</h4>
            <p>1. 请在 N9E "告警规则" 的 "回调地址" 中填入上述 URL，并在请求头中添加鉴权 Token。</p>
            <p>2. 如您在 N9E 已配置 <strong>FlashDuty 官方集成通道</strong>, 我们建议您保留原生通道用于故障闭环，AutoOps 的这一通道纯用于「运维变更与告警强关联分析」与控制台监控。</p>
          </div>
        </el-card>

      </el-col>

      <!-- 右侧：同步状态 -->
      <el-col :span="10">
        <el-card shadow="hover" class="sync-card">
          <template #header>
            <div class="card-header">
              <span><el-icon><Refresh /></el-icon> 同步状态</span>
              <el-button type="primary" size="small" @click="handleSync" :loading="syncing" :icon="Refresh">
                立即同步
              </el-button>
            </div>
          </template>

          <div class="sync-info">
            <div class="sync-item">
              <span class="sync-label">最后同步时间</span>
              <span class="sync-value">{{ lastSyncTime || '从未同步' }}</span>
            </div>

            <el-divider />

            <div v-if="syncResult" class="sync-stats">
              <h4>同步统计</h4>
              <el-row :gutter="12">
                <el-col :span="8">
                  <div class="stat-box">
                    <div class="stat-number">{{ syncResult.busiGroups?.created + syncResult.busiGroups?.updated || 0 }}</div>
                    <div class="stat-label">业务组</div>
                  </div>
                </el-col>
                <el-col :span="8">
                  <div class="stat-box">
                    <div class="stat-number">{{ syncResult.hosts?.created + syncResult.hosts?.updated || 0 }}</div>
                    <div class="stat-label">主机</div>
                  </div>
                </el-col>
                <el-col :span="8">
                  <div class="stat-box">
                    <div class="stat-number">{{ syncResult.datasources?.created + syncResult.datasources?.updated || 0 }}</div>
                    <div class="stat-label">数据源</div>
                  </div>
                </el-col>
              </el-row>

              <el-divider />

              <div class="sync-detail">
                <el-descriptions :column="3" size="small" border>
                  <el-descriptions-item label="业务组新增">{{ syncResult.busiGroups?.created || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="业务组更新">{{ syncResult.busiGroups?.updated || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="业务组跳过">{{ syncResult.busiGroups?.skipped || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="主机新增">{{ syncResult.hosts?.created || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="主机更新">{{ syncResult.hosts?.updated || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="主机跳过">{{ syncResult.hosts?.skipped || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="数据源新增">{{ syncResult.datasources?.created || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="数据源更新">{{ syncResult.datasources?.updated || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="数据源跳过">{{ syncResult.datasources?.skipped || 0 }}</el-descriptions-item>
                </el-descriptions>
              </div>
            </div>

            <el-empty v-else description="暂无同步记录" :image-size="80" />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Setting, Link, Key, Check, Connection, Refresh, Bell, DocumentCopy, QuestionFilled, Warning } from '@element-plus/icons-vue'
import n9eApi from '@/api/n9e'

// 生成当前环境下的 Webhook API
const webhookUrl = computed(() => {
  const host = window.location.host // eg. localhost:18088
  const protocol = window.location.protocol
  return `${protocol}//${host}/api/v1/n9e/alert/webhook`
})

// 剪贴板复制方法
const copyText = async (text) => {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
    } else {
      const textArea = document.createElement("textarea")
      textArea.value = text
      // 使文本框不在视口内
      textArea.style.position = "absolute"
      textArea.style.left = "-999999px"
      document.body.prepend(textArea)
      textArea.select()
      document.execCommand('copy')
      textArea.remove()
    }
    ElMessage.success('复制成功')
  } catch (err) {
    ElMessage.error('复制失败，请手动选取复制')
  }
}


// 表单数据
const configForm = reactive({
  endpoint: '',
  token: '',
  timeout: 30,
  syncCron: '',
  enabled: false
})

// 表单校验
const formRules = {
  endpoint: [{ required: true, message: '请输入 N9E 地址', trigger: 'blur' }],
  token: [{ required: true, message: '请输入用户 Token', trigger: 'blur' }]
}

const configFormRef = ref(null)
const saving = ref(false)
const testing = ref(false)
const syncing = ref(false)
const lastSyncTime = ref('')
const syncResult = ref(null)

// 加载配置
const loadConfig = async () => {
  try {
    const res = await n9eApi.getConfig()
    if (res.data?.code === 200 && res.data.data) {
      const data = res.data.data
      configForm.endpoint = data.endpoint || ''
      configForm.token = data.token || ''
      configForm.timeout = data.timeout || 30
      configForm.syncCron = data.syncCron || ''
      configForm.enabled = data.enabled || false
      lastSyncTime.value = data.lastSyncTime || ''
      if (data.lastSyncResult) {
        try {
          syncResult.value = typeof data.lastSyncResult === 'string'
            ? JSON.parse(data.lastSyncResult)
            : data.lastSyncResult
        } catch (e) {
          syncResult.value = null
        }
      }
    }
  } catch (err) {
    console.log('N9E config not found, using defaults')
  }
}

// 保存配置
const handleSaveConfig = async () => {
  try {
    await configFormRef.value?.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    const res = await n9eApi.saveConfig({
      endpoint: configForm.endpoint,
      token: configForm.token,
      timeout: configForm.timeout,
      syncCron: configForm.syncCron,
      enabled: configForm.enabled
    })
    if (res.data?.code === 200) {
      ElMessage.success('配置保存成功')
    } else {
      ElMessage.error(res.data?.message || '保存失败')
    }
  } catch (err) {
    ElMessage.error('保存失败: ' + (err.message || '未知错误'))
  } finally {
    saving.value = false
  }
}

// 测试连接
const handleTestConnection = async () => {
  if (!configForm.endpoint || !configForm.token) {
    ElMessage.warning('请先填写 N9E 地址和 Token')
    return
  }

  testing.value = true
  try {
    const res = await n9eApi.testConnection({
      endpoint: configForm.endpoint,
      token: configForm.token,
      timeout: configForm.timeout
    })
    if (res.data?.code === 200) {
      ElMessage.success('连接成功！N9E 服务器响应正常')
    } else {
      ElMessage.error(res.data?.message || '连接失败')
    }
  } catch (err) {
    ElMessage.error('连接失败: ' + (err.message || '未知错误'))
  } finally {
    testing.value = false
  }
}

// 触发同步
const handleSync = async () => {
  syncing.value = true
  try {
    const res = await n9eApi.triggerSync()
    if (res.data?.code === 200) {
      syncResult.value = res.data.data
      lastSyncTime.value = new Date().toLocaleString()
      ElMessage.success('同步完成')
    } else {
      ElMessage.error(res.data?.message || '同步失败')
    }
  } catch (err) {
    ElMessage.error('同步失败: ' + (err.message || '未知错误'))
  } finally {
    syncing.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.n9e-config-container {
  padding: 20px;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h3 {
  font-size: 18px;
  font-weight: 600;
  margin: 0 0 8px 0;
  color: #303133;
}

.page-desc {
  color: #909399;
  font-size: 13px;
  margin: 0;
}

.config-card,
.sync-card {
  border-radius: 8px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
}

.card-header span {
  display: flex;
  align-items: center;
  gap: 6px;
}

.form-tip {
  margin-left: 8px;
  color: #909399;
  font-size: 12px;
}

.sync-info {
  padding: 10px 0;
}

.sync-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
}

.sync-label {
  color: #606266;
  font-size: 14px;
}

.sync-value {
  color: #303133;
  font-weight: 500;
}

.sync-stats h4 {
  margin: 0 0 12px 0;
  color: #303133;
  font-size: 14px;
}

.stat-box {
  text-align: center;
  padding: 12px 0;
  background: #f5f7fa;
  border-radius: 6px;
}

.stat-number {
  font-size: 24px;
  font-weight: 700;
  color: #409eff;
}

.stat-label {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.sync-detail {
  margin-top: 10px;
}

/* Webhook 设计样式 */
.webhook-card {
  margin-top: 20px;
  border-radius: 8px;
}

.webhook-header {
  color: #E6A23C;
}

.info-icon {
  color: #909399;
  cursor: help;
  font-size: 16px;
}

.copy-box {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: #f8f9fa;
  border: 1px solid #EBEEF5;
  border-radius: 4px;
  padding: 8px 16px;
  width: 100%;
}

.code-text {
  font-family: Consolas, Monaco, monospace;
  color: #e83e8c;
  font-size: 14px;
  word-break: break-all;
}

.webhook-note {
  margin-top: 24px;
  padding: 12px 16px;
  background-color: #fdf6ec;
  border-left: 4px solid #e6a23c;
  border-radius: 4px;
}

.webhook-note h4 {
  margin: 0 0 8px 0;
  display: flex;
  align-items: center;
  gap: 6px;
  color: #e6a23c;
  font-size: 14px;
}

.webhook-note p {
  margin: 4px 0;
  font-size: 13px;
  color: #606266;
  line-height: 1.6;
}
</style>
