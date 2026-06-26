<template>
  <div class="space-y-6">
    <!-- 页面标题 -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">推广素材</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">下载和使用推广素材，提高转化率</p>
      </div>
    </div>

    <!-- 素材分类 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-100 dark:border-gray-700">
      <div class="flex flex-wrap gap-2">
        <el-button
          v-for="cat in categories"
          :key="cat.id"
          :type="selectedCategory === cat.id ? 'primary' : 'default'"
          @click="selectedCategory = cat.id"
        >
          {{ cat.name }}
        </el-button>
      </div>
    </div>

    <!-- 素材列表 -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <div
        v-for="material in filteredMaterials"
        :key="material.id"
        class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 overflow-hidden"
      >
        <!-- 素材预览 -->
        <div class="aspect-video bg-gray-100 dark:bg-gray-700 flex items-center justify-center">
          <img v-if="material.type === 'image'" :src="material.preview" :alt="material.title" class="w-full h-full object-cover" />
          <div v-else-if="material.type === 'video'" class="text-center">
            <svg class="w-16 h-16 mx-auto text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/>
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
            <p class="text-sm text-gray-500 dark:text-gray-400 mt-2">视频素材</p>
          </div>
          <div v-else class="text-center">
            <svg class="w-16 h-16 mx-auto text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
            <p class="text-sm text-gray-500 dark:text-gray-400 mt-2">文案素材</p>
          </div>
        </div>

        <!-- 素材信息 -->
        <div class="p-4">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-2">{{ material.title }}</h3>
          <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">{{ material.description }}</p>
          <div class="flex items-center justify-between text-xs text-gray-400 mb-3">
            <span>{{ material.size }}</span>
            <span>{{ material.downloads }} 次下载</span>
          </div>
          <div class="flex space-x-2">
            <el-button size="small" type="primary" class="flex-1" @click="downloadMaterial(material)">
              <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
              </svg>
              下载
            </el-button>
            <el-button size="small" @click="previewMaterial(material)">
              预览
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 推广文案 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700">
      <div class="p-6 border-b border-gray-100 dark:border-gray-700">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">推广文案</h2>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">复制文案到社交媒体进行推广</p>
      </div>
      <div class="p-6 space-y-4">
        <div v-for="copy in copywriting" :key="copy.id" class="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
          <div class="flex items-start justify-between mb-2">
            <h3 class="text-sm font-medium text-gray-900 dark:text-white">{{ copy.title }}</h3>
            <el-button text type="primary" size="small" @click="copyText(copy.content)">
              复制
            </el-button>
          </div>
          <p class="text-sm text-gray-600 dark:text-gray-400 whitespace-pre-line">{{ copy.content }}</p>
        </div>
      </div>
    </div>

    <!-- 预览对话框 -->
    <el-dialog v-model="showPreviewDialog" :title="previewMaterial?.title" width="80%">
      <div v-if="previewMaterial?.type === 'image'" class="text-center">
        <img :src="previewMaterial?.preview" :alt="previewMaterial?.title" class="max-w-full mx-auto" />
      </div>
      <div v-else-if="previewMaterial?.type === 'video'" class="aspect-video bg-black rounded-lg flex items-center justify-center">
        <p class="text-white">视频预览区域</p>
      </div>
      <div v-else class="p-8 bg-gray-50 dark:bg-gray-700 rounded-lg">
        <p class="text-gray-600 dark:text-gray-400">{{ previewMaterial?.content }}</p>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

const selectedCategory = ref('all')
const showPreviewDialog = ref(false)
const previewMaterial = ref<any>(null)

const categories = ref([
  { id: 'all', name: '全部' },
  { id: 'banner', name: '横幅广告' },
  { id: 'poster', name: '海报' },
  { id: 'video', name: '视频' },
  { id: 'copy', name: '文案' },
])

const materials = ref([
  { id: 1, title: '产品介绍横幅', description: '适合网站顶部展示', type: 'image', category: 'banner', size: '1920x600', downloads: 256, preview: 'https://via.placeholder.com/1920x600/6366f1/ffffff?text=Product+Banner' },
  { id: 2, title: '促销活动海报', description: '适合社交媒体分享', type: 'image', category: 'poster', size: '1080x1920', downloads: 189, preview: 'https://via.placeholder.com/1080x1920/ec4899/ffffff?text=Promotion+Poster' },
  { id: 3, title: '产品演示视频', description: '展示产品核心功能', type: 'video', category: 'video', size: 'MP4 50MB', downloads: 123, preview: '' },
  { id: 4, title: '用户案例海报', description: '真实用户使用案例', type: 'image', category: 'poster', size: '1080x1920', downloads: 145, preview: 'https://via.placeholder.com/1080x1920/8b5cf6/ffffff?text=Case+Study' },
  { id: 5, title: '功能介绍视频', description: '详细功能使用教程', type: 'video', category: 'video', size: 'MP4 80MB', downloads: 98, preview: '' },
  { id: 6, title: '品牌宣传横幅', description: '品牌形象展示', type: 'image', category: 'banner', size: '1920x600', downloads: 167, preview: 'https://via.placeholder.com/1920x600/10b981/ffffff?text=Brand+Banner' },
])

const copywriting = ref([
  {
    id: 1,
    title: '微信朋友圈文案',
    content: `🚀 发现一个超好用的AI API平台！

✅ 支持多种AI模型
✅ 按量付费，价格透明
✅ 稳定可靠，响应快速

注册即送免费额度，快来试试吧！
链接：https://tokenshopai.com/register?ref=YOUR_CODE`,
  },
  {
    id: 2,
    title: '微博推广文案',
    content: `【推荐】TokenHub - 您的AI API管家

🔥 支持GPT-4、Claude等多种主流模型
💰 按量付费，无最低消费
⚡️ 稳定高速，99.9%可用性

新用户注册送$10体验金
立即体验：https://tokenshopai.com/register?ref=YOUR_CODE

#AI #API #人工智能`,
  },
  {
    id: 3,
    title: '博客文章模板',
    content: `标题：TokenHub使用体验分享

正文：
最近在使用TokenHub这个AI API平台，体验非常不错。它支持多种AI模型，包括GPT-4、Claude等，而且价格透明，按量付费。

最让我满意的是它的稳定性，响应速度很快，基本没有遇到过故障。对于开发者来说，API文档也很完善，集成起来很方便。

如果你也需要AI API服务，推荐试试TokenHub，新用户注册还有免费额度。

注册链接：https://tokenshopai.com/register?ref=YOUR_CODE`,
  },
])

const filteredMaterials = computed(() => {
  if (selectedCategory.value === 'all') {
    return materials.value
  }
  return materials.value.filter(m => m.category === selectedCategory.value)
})

function downloadMaterial(material: any) {
  ElMessage.success(`开始下载：${material.title}`)
  // TODO: 实现真实下载逻辑
}

function previewMaterial(material: any) {
  previewMaterial.value = material
  showPreviewDialog.value = true
}

function copyText(text: string) {
  navigator.clipboard.writeText(text)
  ElMessage.success('文案已复制到剪贴板')
}

onMounted(async () => {
  // TODO: 从 API 获取真实数据
})
</script>
