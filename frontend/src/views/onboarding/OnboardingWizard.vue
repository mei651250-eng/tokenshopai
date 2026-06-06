<template>
  <div class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center">
    <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl w-full max-w-lg mx-4 overflow-hidden animate-fade-in-up">
      <!-- 步骤指示器 -->
      <div class="px-8 pt-6">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium text-gray-500">{{ currentStep + 1 }} / {{ steps.length }}</span>
          <button class="text-xs text-gray-400 hover:text-gray-600" @click="skip">跳过引导</button>
        </div>
        <div class="flex gap-1.5">
          <div
            v-for="(_, i) in steps"
            :key="i"
            class="h-1.5 flex-1 rounded-full transition-all duration-300"
            :class="i <= currentStep ? 'bg-primary-500' : 'bg-gray-200 dark:bg-gray-700'"
          />
        </div>
      </div>

      <!-- 步骤内容 -->
      <div class="px-8 py-8">
        <div class="text-center mb-6">
          <div class="w-16 h-16 mx-auto rounded-2xl flex items-center justify-center text-3xl mb-4" :style="{ background: steps[currentStep].bg }">
            {{ steps[currentStep].icon }}
          </div>
          <h2 class="text-xl font-bold text-gray-900 dark:text-white">{{ steps[currentStep].title }}</h2>
          <p class="text-sm text-gray-500 mt-2">{{ steps[currentStep].desc }}</p>
        </div>

        <!-- 步骤1: 欢迎 + 邀请码 -->
        <div v-if="currentStep === 0" class="space-y-4">
          <div class="bg-gradient-to-r from-primary-50 to-purple-50 dark:from-primary-900/20 dark:to-purple-900/20 rounded-xl p-4 text-center">
            <p class="text-sm text-gray-600 dark:text-gray-300">注册即送 <span class="text-primary-600 font-bold text-lg">¥1</span> 体验额度</p>
          </div>
          <el-input
            v-model="inviteCode"
            placeholder="有邀请码？填写可获得额外奖励"
            prefix-icon="Present"
            size="large"
          />
        </div>

        <!-- 步骤2: 创建 API Key -->
        <div v-if="currentStep === 1" class="space-y-4">
          <div v-if="!createdKey" class="text-center">
            <p class="text-sm text-gray-500 mb-4">API Key 是你调用 AI 模型的唯一凭证</p>
            <el-button type="primary" size="large" class="w-full" :loading="creating" @click="createFirstKey">
              生成我的第一个 API Key
            </el-button>
          </div>
          <div v-else class="space-y-3">
            <div class="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-3">
              <p class="text-xs text-amber-700 dark:text-amber-300 font-medium mb-1">请立即复制保存，密钥仅显示一次</p>
              <div class="flex items-center gap-2">
                <code class="flex-1 text-sm bg-white dark:bg-gray-800 rounded px-2 py-1.5 font-mono break-all">{{ createdKey }}</code>
                <el-button size="small" @click="copyKey">复制</el-button>
              </div>
            </div>
          </div>
        </div>

        <!-- 步骤3: 快速代码示例 -->
        <div v-if="currentStep === 2" class="space-y-3">
          <el-tabs v-model="codeTab" class="code-tabs">
            <el-tab-pane label="Python" name="python">
              <pre class="bg-gray-900 text-green-400 rounded-lg p-4 text-xs overflow-x-auto"><code>from openai import OpenAI

client = OpenAI(
    base_url="{{ baseUrl }}/v1",
    api_key="{{ createdKey || 'sk-your-key' }}"
)

response = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "你好!"}]
)
print(response.choices[0].message.content)</code></pre>
            </el-tab-pane>
            <el-tab-pane label="cURL" name="curl">
              <pre class="bg-gray-900 text-green-400 rounded-lg p-4 text-xs overflow-x-auto"><code>curl {{ baseUrl }}/v1/chat/completions \
  -H "Authorization: Bearer {{ createdKey || 'sk-your-key' }}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "你好!"}]
  }'</code></pre>
            </el-tab-pane>
            <el-tab-pane label="Node.js" name="node">
              <pre class="bg-gray-900 text-green-400 rounded-lg p-4 text-xs overflow-x-auto"><code>import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: '{{ baseUrl }}/v1',
  apiKey: '{{ createdKey || 'sk-your-key' }}'
});

const resp = await client.chat.completions.create({
  model: 'gpt-4o-mini',
  messages: [{ role: 'user', content: '你好!' }]
});
console.log(resp.choices[0].message.content);</code></pre>
            </el-tab-pane>
          </el-tabs>
        </div>

        <!-- 步骤4: 充值 -->
        <div v-if="currentStep === 3" class="space-y-3">
          <div class="grid grid-cols-3 gap-3">
            <div
              v-for="amt in [10, 50, 100]"
              :key="amt"
              class="border-2 rounded-xl p-4 text-center cursor-pointer transition-all"
              :class="topUpAmount === amt ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20' : 'border-gray-200 dark:border-gray-700 hover:border-primary-300'"
              @click="topUpAmount = amt"
            >
              <span class="text-xl font-bold text-gray-900 dark:text-white">¥{{ amt }}</span>
            </div>
          </div>
          <p class="text-xs text-gray-400 text-center">可稍后在「充值」页面操作</p>
        </div>
      </div>

      <!-- 底部操作 -->
      <div class="px-8 pb-6 flex items-center justify-between">
        <el-button v-if="currentStep > 0" text @click="currentStep--">上一步</el-button>
        <span v-else />
        <el-button type="primary" @click="nextStep">
          {{ currentStep === steps.length - 1 ? '完成' : '下一步' }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { adminApi } from '@/api'
import { ElMessage } from 'element-plus'

const router = useRouter()
const emit = defineEmits(['complete', 'skip'])

const currentStep = ref(0)
const inviteCode = ref('')
const createdKey = ref('')
const creating = ref(false)
const codeTab = ref('python')
const topUpAmount = ref(10)
const baseUrl = window.location.origin

const steps = [
  { title: '欢迎加入 TokenHub', desc: '只需 3 步即可开始使用 AI API', icon: '🚀', bg: 'linear-gradient(135deg, #ede9fe, #ddd6fe)' },
  { title: '获取 API Key', desc: '创建密钥即可开始调用模型', icon: '🔑', bg: 'linear-gradient(135deg, #dbeafe, #bfdbfe)' },
  { title: '快速接入代码', desc: '复制代码即可运行', icon: '💻', bg: 'linear-gradient(135deg, #d1fae5, #a7f3d0)' },
  { title: '充值开始使用', desc: '选择金额充值后即可体验所有模型', icon: '💰', bg: 'linear-gradient(135deg, #fef3c7, #fde68a)' },
]

async function createFirstKey() {
  creating.value = true
  try {
    const res: any = await adminApi.createApiKey({ name: '默认密钥' })
    createdKey.value = res.key || res.api_key || ''
    ElMessage.success('API Key 创建成功')
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    creating.value = false
  }
}

function copyKey() {
  navigator.clipboard.writeText(createdKey.value)
  ElMessage.success('已复制到剪贴板')
}

async function nextStep() {
  if (currentStep.value === 0 && inviteCode.value) {
    try {
      await adminApi.redeemCode(inviteCode.value)
      ElMessage.success('邀请码已使用，获得额外奖励！')
    } catch { /* ignore */ }
  }

  if (currentStep.value < steps.length - 1) {
    currentStep.value++
    // 保存进度
    try {
      adminApi.updateOnboarding({ current_step: currentStep.value })
    } catch { /* ignore */ }
  } else {
    // 完成
    try {
      adminApi.updateOnboarding({ completed: true })
    } catch { /* ignore */ }
    emit('complete')
  }
}

function skip() {
  try {
    adminApi.updateOnboarding({ skipped: true })
  } catch { /* ignore */ }
  emit('skip')
}
</script>

<style scoped>
.code-tabs :deep(.el-tabs__header) {
  margin-bottom: 0;
}
.code-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
}
</style>
