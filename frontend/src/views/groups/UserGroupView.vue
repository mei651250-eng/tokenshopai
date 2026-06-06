<template>
  <div class="group-page">
    <div class="page-header">
      <div>
        <h2>用户分组与倍率</h2>
        <p class="text-sm text-gray-500 mt-1">设置不同用户分组的价格倍率，如 VIP 用户 0.8 倍率（8折优惠）</p>
      </div>
      <el-button type="primary" @click="showDialog = true">
        <el-icon><Plus /></el-icon> 新建分组
      </el-button>
    </div>

    <div class="groups-grid">
      <div v-for="g in groups" :key="g.id" class="group-card">
        <div class="group-header">
          <h3>{{ g.name }}</h3>
          <div class="group-badges">
            <el-tag v-if="g.multiplier < 1" type="success" size="small">{{ (g.multiplier * 10).toFixed(1) }}折</el-tag>
            <el-tag v-else-if="g.multiplier > 1" type="warning" size="small">{{ (g.multiplier * 100).toFixed(0) }}%</el-tag>
            <el-tag v-else type="info" size="small">原价</el-tag>
          </div>
        </div>
        <div class="group-stats">
          <div class="group-stat">
            <span class="stat-label">价格倍率</span>
            <span class="stat-value highlight">{{ g.multiplier }}x</span>
          </div>
          <div class="group-stat">
            <span class="stat-label">RPM 限制</span>
            <span class="stat-value">{{ g.rpm_limit || '默认' }}</span>
          </div>
          <div class="group-stat">
            <span class="stat-label">TPM 限制</span>
            <span class="stat-value">{{ g.tpm_limit ? (g.tpm_limit / 1000).toFixed(0) + 'K' : '默认' }}</span>
          </div>
        </div>
        <div class="group-actions">
          <el-button size="small" @click="openEdit(g)">编辑</el-button>
          <el-button type="danger" size="small" plain @click="deleteGroup(g)">删除</el-button>
        </div>
      </div>
    </div>

    <el-empty v-if="!loading && groups.length === 0" description="暂无用户分组" />

    <el-dialog v-model="showDialog" :title="editingId ? '编辑分组' : '新建分组'" width="460px">
      <el-form :model="form" label-position="top">
        <el-form-item label="分组名称" required>
          <el-input v-model="form.name" placeholder="如 VIP用户、企业用户" />
        </el-form-item>
        <el-form-item label="价格倍率" required>
          <el-input-number v-model="form.multiplier" :min="0.01" :max="10" :step="0.1" :precision="2" />
          <div class="form-tip">0.8 = 8折优惠，1.0 = 原价，1.5 = 1.5倍价格</div>
        </el-form-item>
        <el-form-item label="RPM 限制 (每分钟请求)">
          <el-input-number v-model="form.rpm_limit" :min="0" placeholder="0 表示不限制" />
        </el-form-item>
        <el-form-item label="TPM 限制 (每分钟Token)">
          <el-input-number v-model="form.tpm_limit" :min="0" :step="10000" placeholder="0 表示不限制" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '@/api'

interface GroupItem { id: string; name: string; multiplier: number; rpm_limit: number; tpm_limit: number }

const groups = ref<GroupItem[]>([])
const loading = ref(false)
const showDialog = ref(false)
const saving = ref(false)
const editingId = ref('')

const form = reactive({ name: '', multiplier: 1.0, rpm_limit: 0, tpm_limit: 0 })

async function loadList() {
  loading.value = true
  try { const res: any = await adminApi.getUserGroups(); groups.value = res.data || [] }
  catch (e: any) { ElMessage.error(e.message || '加载失败') }
  finally { loading.value = false }
}

function openEdit(g: GroupItem) {
  editingId.value = g.id
  form.name = g.name; form.multiplier = g.multiplier; form.rpm_limit = g.rpm_limit; form.tpm_limit = g.tpm_limit
  showDialog.value = true
}

async function handleSave() {
  if (!form.name.trim()) { ElMessage.warning('请填写分组名称'); return }
  saving.value = true
  try {
    if (editingId.value) {
      await adminApi.updateUserGroup(editingId.value, { name: form.name, multiplier: form.multiplier, rpm_limit: form.rpm_limit, tpm_limit: form.tpm_limit })
    } else {
      await adminApi.createUserGroup({ name: form.name, multiplier: form.multiplier, rpm_limit: form.rpm_limit, tpm_limit: form.tpm_limit })
    }
    ElMessage.success('保存成功')
    showDialog.value = false
    editingId.value = ''
    form.name = ''; form.multiplier = 1.0; form.rpm_limit = 0; form.tpm_limit = 0
    loadList()
  } catch (e: any) { ElMessage.error(e.message || '保存失败') }
  finally { saving.value = false }
}

async function deleteGroup(g: GroupItem) {
  try {
    await ElMessageBox.confirm(`确定删除分组「${g.name}」？`, '确认', { type: 'warning' })
    await adminApi.deleteUserGroup(g.id); ElMessage.success('已删除'); loadList()
  } catch {}
}

onMounted(loadList)
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { font-size: 20px; font-weight: 700; margin: 0; }
.groups-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 16px; }
.group-card { background: #fff; border: 1px solid #e5e7eb; border-radius: 12px; padding: 24px; transition: all 0.2s; }
.group-card:hover { border-color: #818cf8; box-shadow: 0 4px 12px rgba(99,102,241,0.1); }
.group-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.group-header h3 { font-size: 16px; font-weight: 700; margin: 0; }
.group-stats { display: flex; gap: 20px; margin-bottom: 16px; }
.group-stat { text-align: center; }
.stat-label { display: block; font-size: 11px; color: #94a3b8; margin-bottom: 4px; }
.stat-value { font-size: 16px; font-weight: 700; color: #334155; }
.stat-value.highlight { color: #6366f1; }
.group-actions { display: flex; gap: 8px; }
.form-tip { font-size: 12px; color: #94a3b8; margin-top: 4px; }
</style>
