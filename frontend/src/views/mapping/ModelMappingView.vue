<template>
  <div class="mapping-page">
    <div class="page-header">
      <div>
        <h2>模型映射</h2>
        <p class="text-sm text-gray-500 mt-1">将用户请求的模型 A 自动路由到模型 B，用于降级替代或成本优化</p>
      </div>
      <el-button type="primary" @click="showDialog = true">
        <el-icon><Plus /></el-icon> 新建映射
      </el-button>
    </div>

    <el-table :data="mappings" v-loading="loading" stripe border>
      <el-table-column prop="source_model" label="源模型" min-width="180">
        <template #default="{ row }"><code class="model-code">{{ row.source_model }}</code></template>
      </el-table-column>
      <el-table-column label="" width="60" align="center">
        <template #default><span class="arrow">→</span></template>
      </el-table-column>
      <el-table-column prop="target_model" label="目标模型" min-width="180">
        <template #default="{ row }"><code class="model-code">{{ row.target_model }}</code></template>
      </el-table-column>
      <el-table-column prop="tenant_id" label="租户" width="140">
        <template #default="{ row }">{{ row.tenant_id || '全部' }}</template>
      </el-table-column>
      <el-table-column prop="priority" label="优先级" width="80" align="center" />
      <el-table-column prop="enabled" label="状态" width="80" align="center">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" align="center">
        <template #default="{ row }">
          <el-button size="small" @click="toggleMapping(row)">{{ row.enabled ? '禁用' : '启用' }}</el-button>
          <el-button type="danger" size="small" plain @click="deleteMapping(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="showDialog" title="新建模型映射" width="480px">
      <el-form :model="form" label-position="top">
        <el-form-item label="源模型 (用户请求的)" required>
          <el-input v-model="form.source_model" placeholder="如 gpt-4" />
        </el-form-item>
        <el-form-item label="目标模型 (实际调用的)" required>
          <el-input v-model="form.target_model" placeholder="如 deepseek-v3" />
        </el-form-item>
        <el-form-item label="租户ID (留空为全局)">
          <el-input v-model="form.tenant_id" placeholder="留空表示所有租户适用" />
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="form.priority" :min="0" :max="100" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '@/api'

interface MappingItem { id: string; source_model: string; target_model: string; tenant_id: string; priority: number; enabled: boolean }

const mappings = ref<MappingItem[]>([])
const loading = ref(false)
const showDialog = ref(false)
const creating = ref(false)

const form = reactive({ source_model: '', target_model: '', tenant_id: '', priority: 0 })

async function loadList() {
  loading.value = true
  try { const res: any = await adminApi.getModelMappings(); mappings.value = res.data || [] }
  catch (e: any) { ElMessage.error(e.message || '加载失败') }
  finally { loading.value = false }
}

async function handleCreate() {
  if (!form.source_model || !form.target_model) { ElMessage.warning('请填写源模型和目标模型'); return }
  creating.value = true
  try {
    await adminApi.createModelMapping({ source_model: form.source_model, target_model: form.target_model, tenant_id: form.tenant_id || undefined, priority: form.priority })
    ElMessage.success('映射已创建')
    showDialog.value = false
    form.source_model = ''; form.target_model = ''; form.tenant_id = ''; form.priority = 0
    loadList()
  } catch (e: any) { ElMessage.error(e.message || '创建失败') }
  finally { creating.value = false }
}

async function toggleMapping(row: MappingItem) {
  try { await adminApi.toggleModelMapping(row.id); loadList() } catch (e: any) { ElMessage.error(e.message || '操作失败') }
}

async function deleteMapping(row: MappingItem) {
  try {
    await ElMessageBox.confirm(`确定删除映射 ${row.source_model} → ${row.target_model}？`, '确认', { type: 'warning' })
    await adminApi.deleteModelMapping(row.id); ElMessage.success('已删除'); loadList()
  } catch {}
}

onMounted(loadList)
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { font-size: 20px; font-weight: 700; margin: 0; }
.model-code { font-family: monospace; font-size: 13px; background: #f1f5f9; padding: 2px 8px; border-radius: 4px; }
.arrow { font-size: 18px; color: #6366f1; font-weight: 700; }
</style>
