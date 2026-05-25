<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">{{ t('users.title') }}</h1>

    <!-- Filters & Actions -->
    <div class="bg-white dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700 mb-6">
      <div class="flex flex-wrap items-center gap-3">
        <el-input v-model="search" :placeholder="t('users.searchPlaceholder')" prefix-icon="Search" clearable style="width: 250px" />
        <el-select v-model="roleFilter" :placeholder="t('users.allRoles')" clearable style="width: 140px">
          <el-option :label="t('users.superAdmin')" value="super_admin" />
          <el-option :label="t('users.tenantAdmin')" value="tenant_admin" />
          <el-option :label="t('users.editor')" value="editor" />
          <el-option :label="t('users.viewer')" value="viewer" />
        </el-select>
        <el-select v-model="statusFilter" :placeholder="t('users.allStatus')" clearable style="width: 120px">
          <el-option :label="t('users.active')" value="active" />
          <el-option :label="t('users.inactive')" value="inactive" />
          <el-option :label="t('users.banned')" value="banned" />
        </el-select>
        <el-button type="primary" @click="loadData">{{ t('common.search') }}</el-button>
        <div class="ml-auto">
          <el-button type="primary" @click="showCreateDialog = true">{{ t('users.addUser') }}</el-button>
        </div>
      </div>
    </div>

    <!-- Users Table -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <el-table :data="filteredUsers" stripe>
        <el-table-column type="selection" width="50" />
        <el-table-column prop="email" :label="t('users.email')" min-width="200" />
        <el-table-column prop="display_name" :label="t('users.displayName')" width="140" />
        <el-table-column prop="role" :label="t('users.role')" width="130">
          <template #default="{ row }">
            <el-tag :type="roleTagType[row.role] || 'info'" size="small">{{ t(`users.${row.role}`) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="tenant" :label="t('users.tenant')" width="140" />
        <el-table-column prop="status" :label="t('common.status')" width="100">
          <template #default="{ row }">
            <span class="flex items-center gap-1.5">
              <span class="w-2 h-2 rounded-full" :class="row.status === 'active' ? 'bg-green-500' : row.status === 'banned' ? 'bg-red-500' : 'bg-gray-400'" />
              {{ t(`users.${row.status}`) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="last_login" :label="t('users.lastLogin')" width="170" />
        <el-table-column prop="created_at" :label="t('common.createdAt')" width="170" />
        <el-table-column :label="t('common.actions')" width="180" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="editUser(row)">{{ t('common.edit') }}</el-button>
            <el-button type="warning" link size="small" @click="resetPassword(row)">{{ t('users.resetPwd') }}</el-button>
            <el-button type="danger" link size="small" @click="deleteUser(row)">{{ t('common.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="p-4 flex justify-between items-center">
        <div class="text-sm text-gray-500">{{ t('users.totalCount', { n: users.length }) }}</div>
        <el-pagination v-model:current-page="page" :page-size="20" :total="users.length" layout="prev, pager, next" />
      </div>
    </div>

    <!-- Create/Edit User Dialog -->
    <el-dialog v-model="showCreateDialog" :title="editingUser ? t('users.editUser') : t('users.addUser')" width="500">
      <el-form :model="userForm" label-width="100px">
        <el-form-item :label="t('users.email')" required>
          <el-input v-model="userForm.email" :disabled="!!editingUser" />
        </el-form-item>
        <el-form-item :label="t('users.displayName')">
          <el-input v-model="userForm.display_name" />
        </el-form-item>
        <el-form-item v-if="!editingUser" :label="t('users.password')" required>
          <el-input v-model="userForm.password" type="password" show-password />
        </el-form-item>
        <el-form-item :label="t('users.role')" required>
          <el-select v-model="userForm.role" style="width: 100%">
            <el-option :label="t('users.superAdmin')" value="super_admin" />
            <el-option :label="t('users.tenantAdmin')" value="tenant_admin" />
            <el-option :label="t('users.editor')" value="editor" />
            <el-option :label="t('users.viewer')" value="viewer" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('users.tenant')">
          <el-input v-model="userForm.tenant" />
        </el-form-item>
        <el-form-item :label="t('common.status')">
          <el-select v-model="userForm.status" style="width: 100%">
            <el-option :label="t('users.active')" value="active" />
            <el-option :label="t('users.inactive')" value="inactive" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveUser" :loading="saving">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- Role & Permissions Section -->
    <div class="mt-6 bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
      <div class="flex items-center justify-between mb-4">
        <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('users.rolePermissions') }}</h2>
        <el-button type="primary" size="small" @click="showRoleDialog = true">{{ t('users.addRole') }}</el-button>
      </div>
      <el-table :data="roles" stripe>
        <el-table-column prop="name" :label="t('users.roleName')" width="160" />
        <el-table-column prop="description" :label="t('users.roleDesc')" min-width="200" />
        <el-table-column prop="user_count" :label="t('users.userCount')" width="100" />
        <el-table-column prop="permissions" :label="t('users.permissions')" min-width="300">
          <template #default="{ row }">
            <el-tag v-for="p in row.permissions.slice(0, 5)" :key="p" size="small" class="mr-1 mb-1">{{ p }}</el-tag>
            <el-tag v-if="row.permissions.length > 5" size="small" type="info">+{{ row.permissions.length - 5 }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="140" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="editRole(row)">{{ t('common.edit') }}</el-button>
            <el-button type="danger" link size="small">{{ t('common.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Role Edit Dialog -->
    <el-dialog v-model="showRoleDialog" :title="t('users.rolePermissions')" width="600">
      <el-form label-width="100px">
        <el-form-item :label="t('users.roleName')">
          <el-input v-model="roleForm.name" />
        </el-form-item>
        <el-form-item :label="t('users.roleDesc')">
          <el-input v-model="roleForm.description" />
        </el-form-item>
        <el-form-item :label="t('users.permissions')">
          <el-tree
            ref="permTree"
            :data="permissionTree"
            show-checkbox
            node-key="id"
            :default-checked-keys="roleForm.checkedKeys"
            :props="{ label: 'label', children: 'children' }"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showRoleDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveRole">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()

const search = ref('')
const roleFilter = ref('')
const statusFilter = ref('')
const page = ref(1)
const saving = ref(false)
const showCreateDialog = ref(false)
const showRoleDialog = ref(false)
const editingUser = ref<any>(null)

const roleTagType: Record<string, string> = {
  super_admin: 'danger', tenant_admin: 'warning', editor: '', viewer: 'info',
}

const userForm = reactive({ email: '', display_name: '', password: '', role: 'viewer', tenant: '', status: 'active' })
const roleForm = reactive({ name: '', description: '', checkedKeys: [] as string[] })

const users = ref<any[]>([])
const roles = ref<any[]>([])
const permissionTree = ref<any[]>([])

const filteredUsers = computed(() => {
  return users.value.filter(u => {
    if (search.value && !u.email.includes(search.value) && !u.display_name?.includes(search.value)) return false
    if (roleFilter.value && u.role !== roleFilter.value) return false
    if (statusFilter.value && u.status !== statusFilter.value) return false
    return true
  })
})

async function loadData() {
  const now = Date.now()
  users.value = Array.from({ length: 25 }, (_, i) => ({
    id: `u${i + 1}`,
    email: `user${i + 1}@example.com`,
    display_name: `User ${i + 1}`,
    role: ['super_admin', 'tenant_admin', 'editor', 'viewer'][i % 4],
    tenant: ['Tenant A', 'Tenant B', 'Tenant C'][i % 3],
    status: ['active', 'active', 'inactive', 'banned'][i % 4],
    last_login: new Date(now - i * 3600000).toLocaleString(),
    created_at: new Date(now - i * 86400000 * 30).toLocaleString(),
  }))

  roles.value = [
    { name: 'super_admin', description: t('users.superAdminDesc'), user_count: 2, permissions: ['*'] },
    { name: 'tenant_admin', description: t('users.tenantAdminDesc'), user_count: 5, permissions: ['models:read', 'models:write', 'billing:read', 'users:read', 'settings:read'] },
    { name: 'editor', description: t('users.editorDesc'), user_count: 10, permissions: ['models:read', 'models:write', 'billing:read'] },
    { name: 'viewer', description: t('users.viewerDesc'), user_count: 8, permissions: ['models:read', 'billing:read'] },
  ]

  permissionTree.value = [
    { id: 'models', label: t('users.permModels'), children: [
      { id: 'models:read', label: t('users.permRead') },
      { id: 'models:write', label: t('users.permWrite') },
      { id: 'models:delete', label: t('users.permDelete') },
    ]},
    { id: 'billing', label: t('users.permBilling'), children: [
      { id: 'billing:read', label: t('users.permRead') },
      { id: 'billing:write', label: t('users.permWrite') },
    ]},
    { id: 'users', label: t('users.permUsers'), children: [
      { id: 'users:read', label: t('users.permRead') },
      { id: 'users:write', label: t('users.permWrite') },
      { id: 'users:delete', label: t('users.permDelete') },
    ]},
    { id: 'settings', label: t('users.permSettings'), children: [
      { id: 'settings:read', label: t('users.permRead') },
      { id: 'settings:write', label: t('users.permWrite') },
    ]},
    { id: 'security', label: t('users.permSecurity'), children: [
      { id: 'security:read', label: t('users.permRead') },
      { id: 'security:write', label: t('users.permWrite') },
    ]},
  ]
}

function editUser(user: any) {
  editingUser.value = user
  Object.assign(userForm, { email: user.email, display_name: user.display_name, password: '', role: user.role, tenant: user.tenant, status: user.status })
  showCreateDialog.value = true
}

async function saveUser() {
  saving.value = true
  try {
    ElMessage.success(editingUser.value ? t('users.userUpdated') : t('users.userCreated'))
    showCreateDialog.value = false
    editingUser.value = null
  } finally {
    saving.value = false
  }
}

async function resetPassword(user: any) {
  await ElMessageBox.confirm(t('users.resetPwdConfirm', { email: user.email }), t('users.resetPwd'), { type: 'warning' })
  ElMessage.success(t('users.resetPwdSent'))
}

async function deleteUser(user: any) {
  await ElMessageBox.confirm(t('users.deleteConfirm', { email: user.email }), t('common.confirm'), { type: 'danger' })
  users.value = users.value.filter(u => u.id !== user.id)
  ElMessage.success(t('common.deleted'))
}

function editRole(role: any) {
  roleForm.name = role.name
  roleForm.description = role.description
  roleForm.checkedKeys = role.permissions.filter(p => p !== '*')
  showRoleDialog.value = true
}

function saveRole() {
  ElMessage.success(t('users.roleSaved'))
  showRoleDialog.value = false
}

onMounted(loadData)
</script>
