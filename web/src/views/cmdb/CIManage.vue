<template>
  <div class="ci-management">
    <!-- CI 类型选择区 -->
    <el-card shadow="hover" class="type-card">
      <template #header>
        <div class="card-header">
          <span class="title">CI 模型管理</span>
          <el-button type="primary" size="small" @click="showCreateTypeDialog">
            <el-icon><Plus /></el-icon>
            <span style="margin-left: 4px">新建类型</span>
          </el-button>
        </div>
      </template>

      <div class="type-tabs">
        <div
          v-for="t in ciTypes"
          :key="t.id"
          :class="['type-tab', { active: selectedTypeId === t.id }]"
          @click="selectType(t)"
        >
          <el-icon :size="20"><component :is="t.icon || 'Document'" /></el-icon>
          <span class="tab-name">{{ t.name }}</span>
          <el-badge :value="t.instanceCount" :max="999" type="info" class="tab-badge" />
        </div>
      </div>
    </el-card>

    <!-- 实例列表区 -->
    <el-card shadow="hover" class="instance-card" v-if="selectedType">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-icon :size="20"><component :is="selectedType.icon || 'Document'" /></el-icon>
            <span class="title">{{ selectedType.name }}</span>
            <el-tag size="small" type="info" style="margin-left: 8px">{{ selectedType.code }}</el-tag>
            <el-button text type="primary" size="small" @click="showAttrDialog" style="margin-left: 12px">
              <el-icon><Setting /></el-icon> 属性管理
            </el-button>
            <el-button text type="warning" size="small" @click="showEditTypeDialog(selectedType)" v-if="!selectedType.builtIn">
              <el-icon><Edit /></el-icon> 编辑类型
            </el-button>
          </div>
          <div class="action-section">
            <el-input
              v-model="keyword"
              size="small"
              placeholder="搜索实例..."
              clearable
              @keyup.enter="loadInstances"
              style="width: 200px; margin-right: 12px"
            >
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
            <el-button type="success" size="small" @click="showCreateInstanceDialog">
              <el-icon><Plus /></el-icon>
              <span style="margin-left: 4px">新建实例</span>
            </el-button>
          </div>
        </div>
      </template>

      <!-- 动态列实例表格 -->
      <el-table :data="instances" v-loading="loading" stripe border style="width: 100%">
        <el-table-column type="index" label="#" width="50" />
        <el-table-column prop="name" label="实例名称" min-width="150" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>

        <!-- 动态属性列 -->
        <el-table-column
          v-for="attr in listAttributes"
          :key="attr.code"
          :label="attr.name"
          :prop="'attr_' + attr.code"
          min-width="120"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            {{ getAttrValue(row, attr.code) }}
          </template>
        </el-table-column>

        <el-table-column prop="createTime" label="创建时间" width="170" />
        <el-table-column label="操作" width="150" fixed="right" align="center">
          <template #default="{ row }">
            <el-button text type="primary" size="small" @click="showEditInstanceDialog(row)">
              <el-icon><Edit /></el-icon>
            </el-button>
            <el-popconfirm title="确定删除此实例?" @confirm="deleteInstance(row.id)">
              <template #reference>
                <el-button text type="danger" size="small">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper" v-if="total > 0">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadInstances"
          @current-change="loadInstances"
        />
      </div>
    </el-card>

    <!-- 空状态 -->
    <el-card shadow="hover" class="instance-card" v-if="!selectedType && ciTypes.length > 0">
      <el-empty description="请选择一个 CI 类型" />
    </el-card>

    <!-- ============== 对话框 ============== -->

    <!-- 创建/编辑 CI 类型 -->
    <el-dialog v-model="typeDialogVisible" :title="typeForm.id ? '编辑 CI 类型' : '新建 CI 类型'" width="500px" destroy-on-close>
      <el-form :model="typeForm" label-width="80px">
        <el-form-item label="名称" required>
          <el-input v-model="typeForm.name" placeholder="如: 中间件" />
        </el-form-item>
        <el-form-item label="代码" required v-if="!typeForm.id">
          <el-input v-model="typeForm.code" placeholder="如: middleware (英文标识)" />
        </el-form-item>
        <el-form-item label="图标">
          <el-select v-model="typeForm.icon" placeholder="选择图标" clearable>
            <el-option v-for="ic in iconList" :key="ic" :label="ic" :value="ic">
              <el-icon><component :is="ic" /></el-icon> {{ ic }}
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="typeForm.category" placeholder="选择分类">
            <el-option label="服务器" value="server" />
            <el-option label="数据库" value="database" />
            <el-option label="网络" value="network" />
            <el-option label="中间件" value="middleware" />
            <el-option label="存储" value="storage" />
            <el-option label="云资源" value="cloud" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="typeForm.description" type="textarea" rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="typeDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitType" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <!-- 属性管理 -->
    <el-dialog v-model="attrDialogVisible" :title="'属性管理 — ' + (selectedType ? selectedType.name : '')" width="800px" destroy-on-close>
      <div style="margin-bottom: 12px; text-align: right">
        <el-button type="primary" size="small" @click="showCreateAttrDialog">
          <el-icon><Plus /></el-icon> 新增属性
        </el-button>
      </div>
      <el-table :data="attributes" border stripe size="small">
        <el-table-column prop="name" label="名称" width="120" />
        <el-table-column prop="code" label="代码" width="120" />
        <el-table-column prop="dataType" label="类型" width="80">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.dataType }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="required" label="必填" width="60" align="center">
          <template #default="{ row }">
            <el-icon v-if="row.required" color="#67c23a"><Check /></el-icon>
            <el-icon v-else color="#ccc"><Close /></el-icon>
          </template>
        </el-table-column>
        <el-table-column prop="showInList" label="列表显示" width="80" align="center">
          <template #default="{ row }">
            <el-icon v-if="row.showInList" color="#67c23a"><Check /></el-icon>
            <el-icon v-else color="#ccc"><Close /></el-icon>
          </template>
        </el-table-column>
        <el-table-column prop="sortOrder" label="排序" width="60" align="center" />
        <el-table-column label="操作" width="120" align="center">
          <template #default="{ row }">
            <el-button text type="primary" size="small" @click="showEditAttrDialog(row)">编辑</el-button>
            <el-popconfirm title="确定删除此属性?" @confirm="deleteAttribute(row.id)">
              <template #reference>
                <el-button text type="danger" size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 创建/编辑属性 -->
    <el-dialog v-model="attrFormDialogVisible" :title="attrForm.id ? '编辑属性' : '新增属性'" width="500px" destroy-on-close>
      <el-form :model="attrForm" label-width="90px">
        <el-form-item label="属性名称" required>
          <el-input v-model="attrForm.name" placeholder="如: IP地址" />
        </el-form-item>
        <el-form-item label="属性代码" required v-if="!attrForm.id">
          <el-input v-model="attrForm.code" placeholder="如: ip_address" />
        </el-form-item>
        <el-form-item label="数据类型" required>
          <el-select v-model="attrForm.dataType" placeholder="选择类型">
            <el-option label="字符串" value="string" />
            <el-option label="整数" value="integer" />
            <el-option label="小数" value="float" />
            <el-option label="布尔" value="boolean" />
            <el-option label="枚举" value="enum" />
            <el-option label="日期" value="date" />
            <el-option label="IP地址" value="ip" />
            <el-option label="URL" value="url" />
          </el-select>
        </el-form-item>
        <el-form-item label="枚举选项" v-if="attrForm.dataType === 'enum'">
          <el-input v-model="attrForm.enumOptionsStr" type="textarea" rows="2" placeholder='["选项1","选项2","选项3"]' />
        </el-form-item>
        <el-form-item label="默认值">
          <el-input v-model="attrForm.defaultValue" placeholder="默认值（可选）" />
        </el-form-item>
        <el-form-item label="是否必填">
          <el-switch v-model="attrForm.required" />
        </el-form-item>
        <el-form-item label="列表显示">
          <el-switch v-model="attrForm.showInList" />
        </el-form-item>
        <el-form-item label="可搜索">
          <el-switch v-model="attrForm.searchable" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="attrForm.sortOrder" :min="0" :max="999" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="attrFormDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitAttr" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <!-- 创建/编辑 CI 实例 -->
    <el-dialog v-model="instanceDialogVisible" :title="instanceForm.id ? '编辑实例' : '新建实例'" width="600px" destroy-on-close>
      <el-form :model="instanceForm" label-width="100px">
        <el-form-item label="实例名称" required>
          <el-input v-model="instanceForm.name" placeholder="请输入实例名称" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="instanceForm.status">
            <el-option label="运行中" :value="1" />
            <el-option label="已停机" :value="2" />
            <el-option label="维护中" :value="3" />
            <el-option label="已下线" :value="4" />
          </el-select>
        </el-form-item>

        <!-- 动态属性表单 -->
        <el-divider content-position="left">属性信息</el-divider>
        <el-form-item
          v-for="attr in attributes"
          :key="attr.code"
          :label="attr.name"
          :required="attr.required"
        >
          <!-- string / ip / url -->
          <el-input
            v-if="['string', 'ip', 'url'].includes(attr.dataType)"
            v-model="instanceForm.attributes[attr.code]"
            :placeholder="attr.placeholder || '请输入'"
          />
          <!-- integer -->
          <el-input-number
            v-else-if="attr.dataType === 'integer'"
            v-model="instanceForm.attributes[attr.code]"
            :placeholder="attr.placeholder"
            style="width: 100%"
          />
          <!-- float -->
          <el-input-number
            v-else-if="attr.dataType === 'float'"
            v-model="instanceForm.attributes[attr.code]"
            :precision="2"
            :step="0.1"
            style="width: 100%"
          />
          <!-- boolean -->
          <el-switch
            v-else-if="attr.dataType === 'boolean'"
            v-model="instanceForm.attributes[attr.code]"
          />
          <!-- enum -->
          <el-select
            v-else-if="attr.dataType === 'enum'"
            v-model="instanceForm.attributes[attr.code]"
            :placeholder="attr.placeholder || '请选择'"
            clearable
          >
            <el-option
              v-for="opt in parseEnumOptions(attr.enumOptions)"
              :key="opt"
              :label="opt"
              :value="opt"
            />
          </el-select>
          <!-- date -->
          <el-date-picker
            v-else-if="attr.dataType === 'date'"
            v-model="instanceForm.attributes[attr.code]"
            type="date"
            placeholder="选择日期"
            style="width: 100%"
          />
          <!-- fallback -->
          <el-input v-else v-model="instanceForm.attributes[attr.code]" />
        </el-form-item>

        <el-form-item label="备注">
          <el-input v-model="instanceForm.remark" type="textarea" rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="instanceDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitInstance" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import ciTypeApi from '@/api/ciType'

// ========== State ==========
const ciTypes = ref([])
const selectedTypeId = ref(null)
const selectedType = computed(() => ciTypes.value.find(t => t.id === selectedTypeId.value))
const attributes = ref([])
const listAttributes = computed(() => attributes.value.filter(a => a.showInList))
const instances = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const loading = ref(false)
const submitting = ref(false)

// Dialogs
const typeDialogVisible = ref(false)
const attrDialogVisible = ref(false)
const attrFormDialogVisible = ref(false)
const instanceDialogVisible = ref(false)

// Forms
const typeForm = ref({ name: '', code: '', icon: '', category: 'custom', description: '' })
const attrForm = ref({ name: '', code: '', dataType: 'string', required: false, showInList: true, searchable: false, sortOrder: 0, enumOptionsStr: '' })
const instanceForm = ref({ name: '', status: 1, attributes: {}, remark: '' })

const iconList = ['Monitor', 'Coin', 'Connection', 'SetUp', 'Files', 'Share', 'Document', 'Cpu', 'Box', 'Histogram']

// ========== Methods ==========
async function loadTypes() {
  try {
    const res = await ciTypeApi.getCITypeList()
    if (res.data && res.data.code === 200) {
      ciTypes.value = res.data.data || []
      if (ciTypes.value.length > 0 && !selectedTypeId.value) {
        selectType(ciTypes.value[0])
      }
    }
  } catch (e) {
    console.error('Failed to load CI types', e)
  }
}

async function selectType(t) {
  selectedTypeId.value = t.id
  await loadAttributes()
  await loadInstances()
}

async function loadAttributes() {
  if (!selectedTypeId.value) return
  try {
    const res = await ciTypeApi.getCITypeAttributes(selectedTypeId.value)
    if (res.data && res.data.code === 200) {
      attributes.value = res.data.data || []
    }
  } catch (e) {
    console.error('Failed to load attributes', e)
  }
}

async function loadInstances() {
  if (!selectedTypeId.value) return
  loading.value = true
  try {
    const res = await ciTypeApi.getCIInstanceList({
      ciTypeId: selectedTypeId.value,
      page: page.value,
      pageSize: pageSize.value,
      keyword: keyword.value
    })
    if (res.data && res.data.code === 200) {
      const pageData = res.data.data || {}
      instances.value = pageData.list || []
      total.value = pageData.total || 0
    }
  } catch (e) {
    console.error('Failed to load instances', e)
  } finally {
    loading.value = false
  }
}

function getAttrValue(row, code) {
  if (!row.attributes) return '-'
  const v = row.attributes[code]
  return v !== undefined && v !== null && v !== '' ? v : '-'
}

function statusText(s) {
  const map = { 1: '运行中', 2: '已停机', 3: '维护中', 4: '已下线' }
  return map[s] || '未知'
}

function statusTagType(s) {
  const map = { 1: 'success', 2: 'danger', 3: 'warning', 4: 'info' }
  return map[s] || 'info'
}

function parseEnumOptions(options) {
  if (!options) return []
  try {
    if (typeof options === 'string') return JSON.parse(options)
    return options
  } catch { return [] }
}

// === CI Type ===
function showCreateTypeDialog() {
  typeForm.value = { name: '', code: '', icon: '', category: 'custom', description: '' }
  typeDialogVisible.value = true
}

function showEditTypeDialog(t) {
  typeForm.value = { ...t }
  typeDialogVisible.value = true
}

async function submitType() {
  submitting.value = true
  try {
    if (typeForm.value.id) {
      await ciTypeApi.updateCIType(typeForm.value)
      ElMessage.success('更新成功')
    } else {
      await ciTypeApi.createCIType(typeForm.value)
      ElMessage.success('创建成功')
    }
    typeDialogVisible.value = false
    await loadTypes()
  } catch (e) {
    ElMessage.error('操作失败')
  } finally {
    submitting.value = false
  }
}

// === Attribute ===
function showAttrDialog() {
  loadAttributes()
  attrDialogVisible.value = true
}

function showCreateAttrDialog() {
  attrForm.value = { name: '', code: '', dataType: 'string', required: false, showInList: true, searchable: false, sortOrder: 0, enumOptionsStr: '' }
  attrFormDialogVisible.value = true
}

function showEditAttrDialog(row) {
  attrForm.value = { ...row, enumOptionsStr: row.enumOptions ? JSON.stringify(row.enumOptions) : '' }
  attrFormDialogVisible.value = true
}

async function submitAttr() {
  submitting.value = true
  try {
    const data = { ...attrForm.value }
    if (data.enumOptionsStr) {
      data.enumOptions = data.enumOptionsStr
    }
    delete data.enumOptionsStr

    if (data.id) {
      await ciTypeApi.updateCITypeAttribute(data)
      ElMessage.success('更新成功')
    } else {
      data.ciTypeId = selectedTypeId.value
      await ciTypeApi.createCITypeAttribute(data)
      ElMessage.success('创建成功')
    }
    attrFormDialogVisible.value = false
    await loadAttributes()
  } catch (e) {
    ElMessage.error('操作失败')
  } finally {
    submitting.value = false
  }
}

async function deleteAttribute(id) {
  try {
    await ciTypeApi.deleteCITypeAttribute(id)
    ElMessage.success('删除成功')
    await loadAttributes()
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

// === Instance ===
function showCreateInstanceDialog() {
  const defaults = {}
  attributes.value.forEach(a => {
    if (a.defaultValue) defaults[a.code] = a.defaultValue
  })
  instanceForm.value = { name: '', status: 1, attributes: defaults, remark: '', ciTypeId: selectedTypeId.value }
  instanceDialogVisible.value = true
}

function showEditInstanceDialog(row) {
  instanceForm.value = {
    id: row.id,
    name: row.name,
    status: row.status,
    attributes: { ...(row.attributes || {}) },
    remark: row.remark || '',
    ciTypeId: row.ciTypeId
  }
  instanceDialogVisible.value = true
}

async function submitInstance() {
  submitting.value = true
  try {
    if (instanceForm.value.id) {
      await ciTypeApi.updateCIInstance(instanceForm.value)
      ElMessage.success('更新成功')
    } else {
      instanceForm.value.ciTypeId = selectedTypeId.value
      await ciTypeApi.createCIInstance(instanceForm.value)
      ElMessage.success('创建成功')
    }
    instanceDialogVisible.value = false
    await loadInstances()
    await loadTypes() // refresh instance count
  } catch (e) {
    ElMessage.error('操作失败')
  } finally {
    submitting.value = false
  }
}

async function deleteInstance(id) {
  try {
    await ciTypeApi.deleteCIInstance(id)
    ElMessage.success('删除成功')
    await loadInstances()
    await loadTypes()
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

// ========== Init ==========
onMounted(() => {
  loadTypes()
})
</script>

<style scoped>
.ci-management {
  padding: 0;
}

.type-card {
  margin-bottom: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header .header-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.card-header .title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.type-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.type-tab {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  background: var(--el-fill-color-lighter);
}

.type-tab:hover {
  border-color: var(--el-color-primary-light-5);
  background: var(--el-color-primary-light-9);
}

.type-tab.active {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-weight: 500;
}

.tab-name {
  font-size: 14px;
}

.tab-badge {
  margin-left: 4px;
}

.instance-card {
  min-height: 300px;
}

.action-section {
  display: flex;
  align-items: center;
}

.pagination-wrapper {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
