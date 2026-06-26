<template>
  <div class="space-y-6">
    <!-- 页面标题 -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">推广链接管理</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">创建和管理您的推广链接，追踪转化效果</p>
      </div>
      <el-button type="primary" @click="showCreateDialog = true">
        <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
        </svg>
        创建链接
      </el-button>
    </div>

    <!-- 主推广链接 -->
    <div class="bg-gradient-to-r from-purple-600 to-pink-500 rounded-2xl p-6 text-white">
      <h2 class="text-lg font-semibold mb-4">主推广链接</h2>
      <div class="flex items-center space-x-4">
        <div class="flex-1 bg-white/20 rounded-lg px-4 py-3 backdrop-blur-sm">
          <code class="text-sm select-all">{{ mainReferralLink }}</code>
        </div>
        <el-button @click="copyLink(mainReferralLink)" class="bg-white/20 border-white/30 hover:bg-white/30">
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
          </svg>
          复制
        </el-button>
        <el-button @click="shareLink(mainReferralLink)" class="bg-white/20 border-white/30 hover:bg-white/30">
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z"/>
          </svg>
          分享
        </el-button>
      </div>
      <div class="mt-4 grid grid-cols-3 gap-4 text-center">
        <div>
          <p class="text-2xl font-bold">{{ linkStats.totalClicks }}</p>
          <p class="text-sm opacity-80">总点击数</p>
        </div>
        <div>
          <p class="text-2xl font-bold">{{ linkStats.totalRegistrations }}</p>
          <p class="text-sm opacity-80">注册数</p>
        </div>
        <div>
          <p class="text-2xl font-bold">{{ linkStats.conversionRate }}%</p>
          <p class="text-sm opacity-80">转化率</p>
        </div>
      </div>
    </div>

    <!-- 自定义链接列表 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700">
      <div class="p-6 border-b border-gray-100 dark:border-gray-700">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">自定义链接</h2>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">为不同渠道创建专属推广链接</p>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead class="bg-gray-50 dark:bg-gray-700/50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">链接名称</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">链接地址</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">点击数</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">注册数</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">创建时间</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
            <tr v-for="link in customLinks" :key="link.id" class="hover:bg-gray-50 dark:hover:bg-gray-700/50">
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="flex items-center">
                  <div class="w-8 h-8 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center mr-3">
                    <svg class="w-4 h-4 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"/>
                    </svg>
                  </div>
                  <span class="text-sm font-medium text-gray-900 dark:text-white">{{ link.name }}</span>
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <code class="text-xs text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">{{ link.url }}</code>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">{{ link.clicks }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">{{ link.registrations }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{{ link.createdAt }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm">
                <el-button text type="primary" size="small" @click="copyLink(link.url)">复制</el-button>
                <el-button text type="primary" size="small" @click="viewStats(link)">统计</el-button>
                <el-button text type="danger" size="small" @click="deleteLink(link.id)">删除</el-button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 创建链接对话框 -->
    <el-dialog v-model="showCreateDialog" title="创建推广链接" width="500px">
      <el-form :model="newLinkForm" label-width="80px">
        <el-form-item label="链接名称">
          <el-input v-model="newLinkForm.name" placeholder="例如：微信群、微博推广" />
        </el-form-item>
        <el-form-item label="目标页面">
          <el-select v-model="newLinkForm.targetPage" placeholder="选择目标页面" class="w-full">
            <el-option label="注册页面" value="register" />
            <el-option label="首页" value="home" />
            <el-option label="产品页" value="product" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="newLinkForm.note" type="textarea" :rows="3" placeholder="可选备注信息" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="createLink">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useUserStore } from '@/stores/user'
import { ElMessage, ElMessageBox } from 'element-plus'

const userStore = useUserStore()

const referralCode = computed(() => userStore.user?.referral_code || 'ABC123')
const mainReferralLink = computed(() => `https://tokenshopai.com/register?ref=${referralCode.value}`)

const showCreateDialog = ref(false)
const newLinkForm = ref({
  name: '',
  targetPage: 'register',
  note: '',
})

const linkStats = ref({
  totalClicks: 1523,
  totalRegistrations: 47,
  conversionRate: 3.1,
})

const customLinks = ref([
  { id: 1, name: '微信群推广', url: 'https://tokenshopai.com/register?ref=ABC123&src=wechat', clicks: 523, registrations: 18, createdAt: '2024-01-15' },
  { id: 2, name: '微博推广', url: 'https://tokenshopai.com/register?ref=ABC123&src=weibo', clicks: 342, registrations: 12, createdAt: '2024-01-18' },
  { id: 3, name: '博客文章', url: 'https://tokenshopai.com/register?ref=ABC123&src=blog', clicks: 189, registrations: 5, createdAt: '2024-01-20' },
])

function copyLink(url: string) {
  navigator.clipboard.writeText(url)
  ElMessage.success('链接已复制到剪贴板')
}

function shareLink(url: string) {
  if (navigator.share) {
    navigator.share({
      title: 'TokenHub - AI API Gateway',
      text: '注册 TokenHub，享受 AI API 服务！',
      url: url,
    })
  } else {
    copyLink(url)
  }
}

function viewStats(link: any) {
  ElMessage.info(`查看 ${link.name} 的详细统计`)
}

function deleteLink(id: number) {
  ElMessageBox.confirm('确定要删除这个推广链接吗？', '确认删除', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning',
  }).then(() => {
    customLinks.value = customLinks.value.filter(l => l.id !== id)
    ElMessage.success('删除成功')
  }).catch(() => {})
}

function createLink() {
  if (!newLinkForm.value.name) {
    ElMessage.warning('请输入链接名称')
    return
  }
  
  const newLink = {
    id: Date.now(),
    name: newLinkForm.value.name,
    url: `${mainReferralLink.value}&src=${newLinkForm.value.name.toLowerCase().replace(/\s+/g, '_')}`,
    clicks: 0,
    registrations: 0,
    createdAt: new Date().toISOString().split('T')[0],
  }
  
  customLinks.value.unshift(newLink)
  showCreateDialog.value = false
  newLinkForm.value = { name: '', targetPage: 'register', note: '' }
  ElMessage.success('创建成功')
}

onMounted(async () => {
  // TODO: 从 API 获取真实数据
})
</script>
