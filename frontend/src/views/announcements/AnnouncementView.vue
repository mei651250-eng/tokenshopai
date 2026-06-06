<template>
  <div class="announcement-page">
    <div class="page-header">
      <div>
        <h2>公告管理</h2>
        <p class="text-sm text-gray-500 mt-1">发布和管理系统公告，置顶重要通知</p>
      </div>
      <el-button type="primary" @click="openCreateDialog">
        <el-icon><Plus /></el-icon> 发布公告
      </el-button>
    </div>

    <el-table :data="announcements" v-loading="loading" stripe border>
      <el-table-column prop="title" label="标题" min-width="200" />
      <el-table-column prop="type" label="类型" width="100" align="center">
        <template #default="{ row }">
          <el-tag :type="typeTagMap[row.type] || 'info'" size="small">{{ typeLabels[row.type] || row.type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="pinned" label="置顶" width="70" align="center">
        <template #default="{ row }">
          <span v-if="row.pinned">📌</span>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="90" align="center">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">{{ row.status === 'active' ? '生效' : '归档' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="发布时间" width="180">
        <template #default="{ row }">{{ new Date(row.created_at).toLocaleString('zh-CN') }}</template>
      </el-table-column>
      <el-table-column label="操作" width="200" align="center">
        <template #default="{ row }">
          <el-button size="small" @click="openEditDialog(row)">编辑</el-button>
          <el-button v-if="row.status === 'active'" type="warning" size="small" plain @click="archiveAnn(row)">归档</el-button>
          <el-button type="danger" size="small" plain @click="deleteAnn(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="showDialog" :title="editingId ? '编辑公告' : '发布公告'" width="560px">
      <el-form :model="form" label-position="top">
        <el-form-item label="标题" required>
          <el-input v-model="form.title" placeholder="公告标题" />
        </el-form-item>
        <el-form-item label="内容" required>
          <el-input v-model="form.content" type="textarea" :rows="4" placeholder="公告内容" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type">
            <el-option label="通知" value="info" />
            <el-option label="成功" value="success" />
            <el-option label="警告" value="warning" />
            <el-option label="错误" value="error" />
          </el-select>
        </el-form-item>
        <el-form-item label="置顶">
          <el-switch v-model="form.pinned" />
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

interface AnnItem { id: string; title: string; content: string; type: string; status: string; pinned: boolean; created_at: string }

const typeTagMap: Record<string, string> = { info: '', success: 'success', warning: 'warning', error: 'danger' }
const typeLabels: Record<string, string> = { info: '通知', success: '成功', warning: '警告', error: '错误' }

const announcements = ref<AnnItem[]>([])
const loading = ref(false)
const showDialog = ref(false)
const saving = ref(false)
const editingId = ref('')

const form = reactive({ title: '', content: '', type: 'info', pinned: false })

async function loadList() {
  loading.value = true
  try { const res: any = await adminApi.getAnnouncements(); announcements.value = res.data || [] }
  catch (e: any) { ElMessage.error(e.message || '加载失败') }
  finally { loading.value = false }
}

function openCreateDialog() {
  editingId.value = ''
  form.title = ''; form.content = ''; form.type = 'info'; form.pinned = false
  showDialog.value = true
}

function openEditDialog(row: AnnItem) {
  editingId.value = row.id
  form.title = row.title; form.content = row.content; form.type = row.type; form.pinned = row.pinned
  showDialog.value = true
}

async function handleSave() {
  if (!form.title.trim() || !form.content.trim()) { ElMessage.warning('请填写标题和内容'); return }
  saving.value = true
  try {
    if (editingId.value) {
      await adminApi.updateAnnouncement(editingId.value, { title: form.title, content: form.content, type: form.type, pinned: form.pinned })
    } else {
      await adminApi.createAnnouncement({ title: form.title, content: form.content, type: form.type, pinned: form.pinned })
    }
    ElMessage.success('保存成功')
    showDialog.value = false
    loadList()
  } catch (e: any) { ElMessage.error(e.message || '保存失败') }
  finally { saving.value = false }
}

async function archiveAnn(row: AnnItem) {
  try { await adminApi.updateAnnouncement(row.id, { status: 'archived' }); ElMessage.success('已归档'); loadList() }
  catch (e: any) { ElMessage.error(e.message || '操作失败') }
}

async function deleteAnn(row: AnnItem) {
  try {
    await ElMessageBox.confirm(`确定删除公告「${row.title}」？`, '确认', { type: 'warning' })
    await adminApi.deleteAnnouncement(row.id)
    ElMessage.success('已删除')
    loadList()
  } catch {}
}

onMounted(loadList)
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { font-size: 20px; font-weight: 700; margin: 0; }
</style>
