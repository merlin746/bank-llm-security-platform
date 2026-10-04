<template>
  <div class="page-container">
    <PageHeading title="系统管理" description="管理账号、职责权限、安全策略与运行配置。" />
    <el-alert title="职责、数据密级与业务范围分别配置" description="角色决定可执行的操作，数据密级与业务范围共同限定可查看的记录。系统管理员即使具有 L4 密级，也不会自动获得客户数据权限。" type="info" show-icon :closable="false" />

    <section class="card section-gap" aria-label="账号管理">
      <div class="panel-heading">
        <div><h2 class="panel-title">账号管理</h2><p class="panel-description">角色或数据授权修改后，账号需重新登录以使用新权限。</p></div>
        <div class="account-actions"><el-button :loading="accountsPending" :disabled="mutationPending" @click="loadAccounts"><el-icon v-if="!accountsPending"><Refresh /></el-icon><span>刷新账号</span></el-button><el-button type="primary" :disabled="accountsPending || mutationPending || !roles.length" @click="startCreate">新增账号</el-button></div>
      </div>
      <el-alert v-if="accountsError" class="resource-error" :title="users.length ? '账号刷新失败，当前展示上次结果' : '账号加载失败'" :description="accountsError + '。请点击刷新账号重试。'" type="error" show-icon :closable="false" />
      <el-skeleton v-if="accountsPending && !users.length" :rows="6" animated />
      <template v-else>
        <el-input v-model="search" class="account-search" aria-label="搜索账号或角色" placeholder="搜索英文账号或角色" clearable><template #prefix><el-icon><Search /></el-icon></template></el-input>
        <div class="table-scroll" role="region" aria-label="账号列表，可横向滚动" tabindex="0">
          <el-table :data="filteredUsers" row-key="id" empty-text="暂无符合条件的账号">
            <el-table-column prop="username" label="英文账号" min-width="145" />
            <el-table-column prop="role" label="角色" min-width="150" />
            <el-table-column label="数据密级" min-width="155"><template #default="{ row }"><el-tag type="info" size="small" effect="plain">{{ row.dataLevel }}</el-tag><span class="level-description">{{ levelDescription(row.dataLevel) }}</span></template></el-table-column>
            <el-table-column prop="department" label="所属部门" min-width="140" />
            <el-table-column label="授权范围" min-width="200"><template #default="{ row }">{{ scopeLabel(row) }}</template></el-table-column>
            <el-table-column label="操作" min-width="210" fixed="right"><template #default="{ row }">
              <span v-if="row.username === 'admin'" class="muted protected-account">内置管理员 · 已保护</span>
              <div v-else-if="deleteTarget === row.id" class="delete-confirm"><span>删除 {{ row.username }}？</span><el-button link type="danger" :loading="mutationPending" @click="confirmDelete(row)">确认删除</el-button><el-button link :disabled="mutationPending" @click="deleteTarget = null">取消</el-button></div>
              <div v-else class="row-actions"><el-button link type="primary" :disabled="accountsPending || mutationPending" :aria-label="'编辑账号 ' + row.username" @click="startEdit(row)">编辑权限</el-button><el-button link type="danger" :disabled="accountsPending || mutationPending" :aria-label="'删除账号 ' + row.username" @click="requestDelete(row)">删除</el-button></div>
            </template></el-table-column>
          </el-table>
        </div>
      </template>
      <el-alert v-if="mutationError && !editorOpen" class="mutation-error" title="账号操作失败" :description="mutationError + '。请重试该操作。'" type="error" show-icon :closable="false" />
      <div v-if="editorOpen" ref="editorPanel" class="account-editor" tabindex="-1" :aria-label="editingId === null ? '新增账号表单' : '编辑账号权限表单'">
        <div class="panel-heading"><h3 class="panel-title">{{ editingId === null ? '新增账号' : '编辑 ' + accountForm.username + ' 的权限' }}</h3><el-button :disabled="mutationPending" @click="closeEditor">取消编辑</el-button></div>
        <el-form ref="accountFormRef" :model="accountForm" :rules="accountRules" label-position="top" :disabled="mutationPending" @submit.prevent="saveAccount">
          <div class="account-fields">
            <el-form-item label="英文账号" prop="username"><el-input v-model="accountForm.username" :disabled="editingId !== null" name="username" autocomplete="off" placeholder="例如 john" /></el-form-item>
            <el-form-item v-if="editingId === null" label="登录密码" prop="password"><el-input v-model="accountForm.password" type="password" show-password name="password" autocomplete="new-password" placeholder="设置新账号的登录密码" /></el-form-item>
            <el-form-item label="角色" prop="role"><el-select v-model="accountForm.role" aria-label="账号角色" @change="adjustDataLevel"><el-option v-for="role in roles" :key="role.name" :label="role.name" :value="role.name" /></el-select></el-form-item>
            <el-form-item label="数据密级" prop="dataLevel"><el-select v-model="accountForm.dataLevel" aria-label="账号数据密级"><el-option v-for="level in allowedLevels" :key="level.level" :label="level.level + ' · ' + level.desc" :value="level.level" /></el-select></el-form-item>
            <el-form-item label="所属业务部门" prop="department"><el-select v-model="accountForm.department" aria-label="所属业务部门" @change="adjustScope"><el-option v-for="department in departmentOptions" :key="department" :label="department" :value="department" /></el-select></el-form-item>
            <el-form-item v-if="accountForm.role !== '系统管理员'" label="授权业务组织" prop="scopeDepartments"><el-select v-model="accountForm.scopeDepartments" multiple aria-label="授权业务组织" placeholder="选择该角色允许的业务组织"><el-option v-for="department in availableDepartments" :key="department" :label="department" :value="department" /></el-select></el-form-item>
            <el-form-item v-if="accountForm.role !== '系统管理员'" label="仅限本人发起的业务"><el-switch v-model="accountForm.ownerOnly" :disabled="accountForm.role === '柜员／客服'" aria-label="仅限本人发起的业务" /><span class="owner-limit-hint">{{ accountForm.role === '柜员／客服' ? '柜员／客服始终限定本人业务' : '开启后进一步限定本人业务' }}</span></el-form-item>
          </div>
          <p class="authorization-note">{{ selectedRoleDescription }} {{ accountForm.role === '系统管理员' ? '该角色不开放业务组织范围。' : '数据密级仅在选定的业务范围内生效。' }}</p>
          <el-alert v-if="mutationError" class="resource-error" title="账号保存失败" :description="mutationError + '。修改后可再次保存。'" type="error" show-icon :closable="false" />
          <el-button type="primary" native-type="submit" :loading="mutationPending">{{ editingId === null ? '创建账号' : '保存权限' }}</el-button>
        </el-form>
      </div>
    </section>

    <section class="card section-gap" aria-label="角色权限说明">
      <div class="panel-heading"><div><h2 class="panel-title">角色与职责</h2><p class="panel-description">角色定义由服务端统一维护，下表为只读权限说明。</p></div></div>
      <div class="table-scroll" role="region" aria-label="角色职责表，可横向滚动" tabindex="0">
        <el-table :data="roles" row-key="name" empty-text="暂无角色定义，请刷新账号重试">
          <el-table-column prop="name" label="角色" min-width="150" />
          <el-table-column label="开放内容与操作" min-width="380"><template #default="{ row }">{{ roleDescription(row.name) }}</template></el-table-column>
          <el-table-column label="业务授权范围" min-width="220"><template #default="{ row }">{{ roleScopeLabel(row) }}</template></el-table-column>
        </el-table>
      </div>
    </section>

    <section class="card section-gap" aria-label="安全策略与运行配置">
      <div class="panel-heading"><div><h2 class="panel-title">安全策略与运行配置</h2><p class="panel-description">保存后用于后续请求的演示测试与 AI 检测。</p></div><el-button :loading="settingsPending" :disabled="settingsSaving" @click="loadSettings">重新加载配置</el-button></div>
      <el-alert v-if="settingsError" class="resource-error" title="配置加载失败" :description="settingsError + '。请重新加载配置。'" type="error" show-icon :closable="false" />
      <el-skeleton v-if="settingsPending && !savedSettings" :rows="4" animated />
      <el-form v-else-if="savedSettings" label-position="top" :disabled="settingsSaving || settingsPending" @submit.prevent="saveSettings">
        <div class="policy-row"><div><label for="gateway-tests">开放演示攻防测试</label><p class="panel-description">关闭后，具有模拟测试权限的风控审核员不能发起新的演示测试。</p></div><el-switch id="gateway-tests" v-model="settingsForm.allowGatewayTests" aria-label="开放演示攻防测试" /></div>
        <el-form-item label="AI 检测等待时间" class="timeout-field"><div class="timeout-input"><el-input-number v-model="settingsForm.aiTimeoutMs" :min="500" :max="8000" :step="500" :precision="0" controls-position="right" aria-label="AI 检测等待时间，毫秒" /><span>毫秒</span></div><p class="panel-description">允许 500–8000 毫秒；超时后执行备用规则并标注降级原因。</p></el-form-item>
        <el-alert v-if="settingsSaveError" class="resource-error" title="配置保存失败" :description="settingsSaveError + '。当前更改仍保留，可重新保存。'" type="error" show-icon :closable="false" />
        <div class="settings-footer"><el-button type="primary" native-type="submit" :loading="settingsSaving" :disabled="!settingsDirty">{{ settingsSaveError ? '重新保存配置' : '保存配置' }}</el-button><span class="save-status" role="status">{{ settingsDirty ? '有未保存的更改' : settingsSavedAt ? '已保存于 ' + settingsSavedAt : '当前配置已加载' }}</span></div>
      </el-form>
      <p v-else-if="!settingsPending" class="empty-description">配置暂时不可用，重新加载后可编辑。</p>
    </section>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import PageHeading from '@/components/PageHeading.vue'
import { createUser, deleteUser, getDataLevels, getRoles, getUsers, updateUser } from '@/api/admin'
import { getSystemSettings, updateSystemSettings } from '@/api/system'
import { demoAccounts } from '@/config/demoAccounts'

const users = ref([])
const roles = ref([])
const levels = ref([])
const search = ref('')
const accountsPending = ref(false)
const accountsError = ref('')
const mutationPending = ref(false)
const mutationError = ref('')
const deleteTarget = ref(null)
const editorOpen = ref(false)
const editingId = ref(null)
const editorPanel = ref()
const accountFormRef = ref()
const accountForm = reactive({ username: '', password: '', role: '', dataLevel: 'L1', department: '', scopeDepartments: [], ownerOnly: false })
const roleDescriptions = {
  '柜员／客服': '查看自身授权业务的处理结果和风险提示，不开放全局安全日志。',
  '风控审核员': '查看授权范围内的态势、告警详情，并执行风险复核。',
  '审计人员': '只读查看授权审计记录、证据与对账结果。',
  '系统管理员': '管理账号、角色、策略与运行配置，不自动获得客户数据权限。'
}
const accountRules = {
  username: [{ required: true, whitespace: true, message: '请输入英文账号名称', trigger: 'blur' }, { pattern: /^[A-Za-z][A-Za-z_-]*$/, message: '使用英文名称，可包含下划线或连字符', trigger: 'blur' }],
  password: [{ required: true, whitespace: true, message: '请设置登录密码', trigger: 'blur' }],
  role: [{ required: true, message: '请选择角色', trigger: 'change' }],
  dataLevel: [{ required: true, message: '请选择账号数据密级', trigger: 'change' }],
  department: [{ required: true, message: '请选择账号所属业务部门', trigger: 'change' }],
  scopeDepartments: [{ validator: (_rule, value, callback) => callback(accountForm.role !== '系统管理员' && !value.length ? new Error('请选择至少一个授权业务组织') : undefined), trigger: 'change' }]
}
const filteredUsers = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  return users.value.filter(user => !keyword || [user.username, user.role].some(value => String(value).toLowerCase().includes(keyword)))
})
const selectedRole = computed(() => roles.value.find(role => role.name === accountForm.role))
const selectedRoleLimit = computed(() => roleLimit(selectedRole.value))
const selectedRoleDescription = computed(() => roleDescription(accountForm.role))
const allowedLevels = computed(() => levels.value.filter(level => (level.rank || rank(level.level)) <= rank(selectedRole.value?.maxAccessLevel || 'L4')))
const departmentOptions = computed(() => accountForm.role === '系统管理员' ? ['平台运维部'] : ['零售业务部', '信贷业务部'])
const availableDepartments = computed(() => {
  if (accountForm.role === '系统管理员') return []
  if (accountForm.role === '柜员／客服' || accountForm.role === '风控审核员') return accountForm.department ? [accountForm.department] : []
  return ['零售业务部', '信贷业务部']
})
function rank(level) { return Number(String(level ?? '').replace(/^L/, '')) || 1 }
function roleLimit(role) { return 'L' + rank(role?.maxAccessLevel) }
function roleDescription(role) { return roleDescriptions[role] || '请按该角色的授权范围执行操作。' }
function levelDescription(level) { return levels.value.find(item => item.level === level)?.desc || '' }
function roleScope(role) {
  const scope = role?.scope && typeof role.scope === 'object' ? role.scope : demoAccounts.find(account => account.role === role?.name)?.scope
  return { departments: Array.isArray(scope?.departments) ? scope.departments : [], ownerOnly: !!scope?.ownerOnly }
}
function scopeLabel(user) {
  const departments = user.scope?.departments || []
  return departments.length ? (user.scope?.ownerOnly ? '本人业务 · ' : '') + departments.join('、') : '无客户业务范围'
}
function roleScopeLabel(role) {
  return { '柜员／客服': '本人业务 · 所属业务部门', '风控审核员': '所属业务部门', '审计人员': '授权零售与信贷业务记录', '系统管理员': '平台配置 · 无客户业务范围' }[role.name] || scopeLabel({ scope: roleScope(role) })
}

async function loadAccounts() {
  if (accountsPending.value || mutationPending.value) return
  accountsPending.value = true
  accountsError.value = ''
  try {
    const [userResponse, roleResponse, levelResponse] = await Promise.all([getUsers(), getRoles(), getDataLevels()])
    if (![userResponse, roleResponse, levelResponse].every(response => Array.isArray(response?.data))) throw new Error('服务返回的账号或权限定义不完整')
    users.value = userResponse.data
    roles.value = roleResponse.data.map(role => ({ ...role, name: role.name || role.role }))
    levels.value = levelResponse.data
  } catch (reason) {
    accountsError.value = reason.message || '无法获取账号与权限定义'
  } finally {
    accountsPending.value = false
  }
}
async function focusEditor() {
  await nextTick()
  accountFormRef.value?.clearValidate()
  editorPanel.value?.scrollIntoView({ block: 'nearest' })
  editorPanel.value?.focus({ preventScroll: true })
}
function startCreate() {
  editingId.value = null
  const role = roles.value.find(item => item.name === '柜员／客服')?.name || roles.value[0]?.name || ''
  Object.assign(accountForm, { username: '', password: '', role, dataLevel: 'L1', department: '', scopeDepartments: [], ownerOnly: false })
  adjustDataLevel()
  mutationError.value = ''
  deleteTarget.value = null
  editorOpen.value = true
  focusEditor()
}
function startEdit(user) {
  if (user.username === 'admin') return
  editingId.value = user.id
  Object.assign(accountForm, { username: user.username, password: '', role: user.role, dataLevel: user.dataLevel, department: user.department || '', scopeDepartments: [...(user.scope?.departments || [])], ownerOnly: !!user.scope?.ownerOnly })
  mutationError.value = ''
  deleteTarget.value = null
  editorOpen.value = true
  focusEditor()
}
function closeEditor() { editorOpen.value = false; mutationError.value = '' }
function adjustDataLevel() {
  if (!allowedLevels.value.some(level => level.level === accountForm.dataLevel)) accountForm.dataLevel = selectedRoleLimit.value
  const defaults = demoAccounts.find(account => account.role === accountForm.role)
  accountForm.department = defaults?.department || departmentOptions.value[0]
  accountForm.ownerOnly = roleScope(selectedRole.value).ownerOnly
  accountForm.scopeDepartments = [...availableDepartments.value]
  adjustScope()
  accountFormRef.value?.clearValidate('dataLevel')
}
function adjustScope() {
  if (accountForm.role === '柜员／客服' || accountForm.role === '风控审核员' || accountForm.role === '系统管理员') accountForm.scopeDepartments = [...availableDepartments.value]
  if (accountForm.role === '柜员／客服') accountForm.ownerOnly = true
  if (accountForm.role === '系统管理员') accountForm.ownerOnly = false
  accountFormRef.value?.clearValidate('scopeDepartments')
}
async function saveAccount() {
  if (mutationPending.value) return
  const valid = await accountFormRef.value.validate().catch(() => false)
  if (!valid) return
  mutationPending.value = true
  mutationError.value = ''
  try {
    const payload = {
      role: accountForm.role, dataLevel: accountForm.dataLevel, department: accountForm.department.trim(),
      scope: { departments: accountForm.role === '系统管理员' ? [] : [...accountForm.scopeDepartments], ownerOnly: accountForm.role === '柜员／客服' || (accountForm.role !== '系统管理员' && accountForm.ownerOnly) }
    }
    const response = editingId.value === null
      ? await createUser({ ...payload, username: accountForm.username.trim(), password: accountForm.password })
      : await updateUser(editingId.value, payload)
    if (!response?.data?.id) throw new Error('服务未返回保存后的账号，请刷新确认')
    const index = users.value.findIndex(user => user.id === response.data.id)
    if (index < 0) users.value.push(response.data)
    else users.value.splice(index, 1, response.data)
    ElMessage.success(editingId.value === null ? '账号已创建' : '账号权限已保存，该账号需重新登录')
    closeEditor()
  } catch (reason) {
    mutationError.value = reason.message || '账号未能保存'
  } finally {
    mutationPending.value = false
  }
}
function requestDelete(user) {
  if (user.username === 'admin') return
  editorOpen.value = false
  mutationError.value = ''
  deleteTarget.value = user.id
}
async function confirmDelete(user) {
  if (mutationPending.value || user.username === 'admin') return
  mutationPending.value = true
  mutationError.value = ''
  try {
    await deleteUser(user.id)
    users.value = users.value.filter(item => item.id !== user.id)
    deleteTarget.value = null
    ElMessage.success('账号已删除')
  } catch (reason) {
    mutationError.value = reason.message || '账号未能删除'
  } finally {
    mutationPending.value = false
  }
}

const settingsForm = reactive({ allowGatewayTests: true, aiTimeoutMs: 4000 })
const savedSettings = ref(null)
const settingsPending = ref(false)
const settingsSaving = ref(false)
const settingsError = ref('')
const settingsSaveError = ref('')
const settingsSavedAt = ref('')
const settingsDirty = computed(() => savedSettings.value && (settingsForm.allowGatewayTests !== savedSettings.value.policy.allowGatewayTests || settingsForm.aiTimeoutMs !== savedSettings.value.runtime.aiTimeoutMs))
function applySettings(data) {
  if (typeof data?.policy?.allowGatewayTests !== 'boolean' || !Number.isInteger(data?.runtime?.aiTimeoutMs)) throw new Error('服务返回的运行配置不完整')
  savedSettings.value = data
  Object.assign(settingsForm, { allowGatewayTests: data.policy.allowGatewayTests, aiTimeoutMs: data.runtime.aiTimeoutMs })
}
async function loadSettings() {
  if (settingsPending.value || settingsSaving.value) return
  settingsPending.value = true
  settingsError.value = ''
  try {
    const response = await getSystemSettings()
    applySettings(response?.data)
    settingsSaveError.value = ''
    settingsSavedAt.value = ''
  } catch (reason) {
    settingsError.value = reason.message || '无法获取运行配置'
  } finally {
    settingsPending.value = false
  }
}
async function saveSettings() {
  if (settingsSaving.value || !settingsDirty.value) return
  settingsSaveError.value = ''
  if (!Number.isInteger(settingsForm.aiTimeoutMs) || settingsForm.aiTimeoutMs < 500 || settingsForm.aiTimeoutMs > 8000) {
    settingsSaveError.value = 'AI 检测等待时间需为 500–8000 毫秒的整数'
    return
  }
  settingsSaving.value = true
  try {
    const response = await updateSystemSettings({ policy: { allowGatewayTests: settingsForm.allowGatewayTests }, runtime: { aiTimeoutMs: settingsForm.aiTimeoutMs } })
    applySettings(response?.data)
    settingsError.value = ''
    settingsSavedAt.value = new Date().toLocaleTimeString('zh-CN', { hour12: false, timeZone: 'Asia/Shanghai' })
    ElMessage.success('策略与运行配置已保存')
  } catch (reason) {
    settingsSaveError.value = reason.message || '配置未能保存'
  } finally {
    settingsSaving.value = false
  }
}
onMounted(() => { loadAccounts(); loadSettings() })
</script>

<style scoped>
.account-actions, .row-actions, .delete-confirm, .settings-footer { display: flex; align-items: center; flex-wrap: wrap; gap: 12px; }
.account-actions .el-button + .el-button, .row-actions .el-button + .el-button, .delete-confirm .el-button + .el-button { margin-left: 0; }
.account-search { width: min(100%, 340px); margin-bottom: 18px; }
.level-description { color: var(--text-sub); font-size: 12px; margin-left: 8px; }
.protected-account { font-size: 12px; }
.delete-confirm { column-gap: 10px; font-size: 12px; }
.delete-confirm > span { width: 100%; }
.mutation-error { margin-top: 18px; }
.account-editor { padding-top: 24px; margin-top: 24px; border-top: 1px solid var(--border); scroll-margin-top: 100px; }
.account-fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 20px; }
.account-fields :deep(.el-select) { width: 100%; }
.owner-limit-hint { margin-left: 12px; color: var(--text-sub); font-size: 12px; }
.authorization-note { color: var(--text-sub); font-size: 12px; margin: 0 0 20px; max-width: 75ch; }
.policy-row { display: flex; justify-content: space-between; align-items: center; gap: 24px; padding-bottom: 24px; margin-bottom: 24px; border-bottom: 1px solid var(--border); }
.policy-row label { font-weight: 500; }
.policy-row .el-switch { flex-shrink: 0; }
.timeout-field :deep(.el-form-item__content) { display: block; }
.timeout-input { display: flex; align-items: center; gap: 12px; }
.timeout-input .el-input-number { width: 190px; }
.timeout-input > span, .save-status { color: var(--text-sub); font-size: 12px; }
.timeout-field .panel-description { margin-top: 8px; }
.settings-footer { margin-top: 24px; }
@media (max-width: 640px) {
  .account-fields { grid-template-columns: minmax(0, 1fr); }
  .account-actions { width: 100%; }
  .account-actions .el-button { flex: 1; }
  .policy-row { align-items: flex-start; gap: 16px; }
}
</style>
