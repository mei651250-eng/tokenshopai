<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">{{ t('security.waf') }}</h1>

    <!-- Security Modules -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-semibold text-gray-900 dark:text-white">WAF 防火墙</h3>
          <el-switch v-model="wafEnabled" />
        </div>
        <p class="text-sm text-gray-500 mb-4">IP黑白名单、请求限流、恶意Prompt拦截</p>
        <div class="space-y-2">
          <div class="flex justify-between text-sm">
            <span class="text-gray-500">已拦截IP</span>
            <span class="font-medium">12</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-gray-500">今日拦截</span>
            <span class="font-medium text-red-500">234</span>
          </div>
        </div>
      </div>

      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-semibold text-gray-900 dark:text-white">数据脱敏</h3>
          <el-switch v-model="desensitizeEnabled" />
        </div>
        <p class="text-sm text-gray-500 mb-4">自动识别并脱敏PII（邮箱、手机号、身份证等）</p>
        <div class="space-y-2">
          <div class="flex justify-between text-sm">
            <span class="text-gray-500">脱敏规则</span>
            <span class="font-medium">7</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-gray-500">今日脱敏</span>
            <span class="font-medium text-blue-500">1,234</span>
          </div>
        </div>
      </div>

      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <h3 class="font-semibold text-gray-900 dark:text-white mb-4">安全统计</h3>
        <div class="space-y-3">
          <div class="flex justify-between text-sm">
            <span class="text-gray-500">Prompt注入拦截</span>
            <span class="font-medium text-red-500">89</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-gray-500">SQL注入拦截</span>
            <span class="font-medium text-red-500">12</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-gray-500">限流触发</span>
            <span class="font-medium text-yellow-500">456</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-gray-500">IP黑名单命中</span>
            <span class="font-medium text-orange-500">234</span>
          </div>
        </div>
      </div>
    </div>

    <!-- IP Blacklist -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 mb-6">
      <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex justify-between items-center">
        <h3 class="font-semibold text-gray-900 dark:text-white">IP 黑名单</h3>
        <el-button type="danger" size="small" plain>添加IP</el-button>
      </div>
      <el-table :data="blockedIPs" stripe size="small">
        <el-table-column prop="ip" label="IP 地址" width="200" />
        <el-table-column prop="reason" label="原因" />
        <el-table-column prop="blocked_at" label="封禁时间" width="180" />
        <el-table-column :label="t('common.actions')" width="100">
          <template #default>
            <el-button size="small" text type="primary">解封</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Prompt Rules -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex justify-between items-center">
        <h3 class="font-semibold text-gray-900 dark:text-white">Prompt 安全规则</h3>
        <el-button type="primary" size="small" plain>添加规则</el-button>
      </div>
      <el-table :data="promptRules" stripe size="small">
        <el-table-column prop="name" label="规则名称" width="250" />
        <el-table-column prop="action" label="动作" width="100">
          <template #default="{ row }">
            <el-tag :type="row.action === 'block' ? 'danger' : row.action === 'warn' ? 'warning' : 'info'" size="small">
              {{ row.action }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="severity" label="严重级别" width="100">
          <template #default="{ row }">
            <el-tag :type="row.severity === 'critical' ? 'danger' : row.severity === 'high' ? 'warning' : 'info'" size="small">
              {{ row.severity }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="hits" label="今日命中" width="100" />
        <el-table-column :label="t('common.status')" width="80">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" size="small" />
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const wafEnabled = ref(true)
const desensitizeEnabled = ref(true)

const blockedIPs = [
  { ip: '185.233.xx.xx', reason: '暴力破解API Key', blocked_at: '2025-05-23 04:12' },
  { ip: '45.155.xx.xx', reason: 'SQL注入攻击', blocked_at: '2025-05-22 18:30' },
  { ip: '103.75.xx.xx', reason: '恶意Prompt注入', blocked_at: '2025-05-22 12:45' },
]

const promptRules = ref([
  { name: 'Prompt注入-忽略指令', action: 'block', severity: 'high', hits: 45, enabled: true },
  { name: 'Prompt注入-角色覆盖', action: 'warn', severity: 'medium', hits: 23, enabled: true },
  { name: 'Prompt注入-数据提取', action: 'block', severity: 'critical', hits: 12, enabled: true },
  { name: 'DAN模式/Jailbreak', action: 'block', severity: 'critical', hits: 9, enabled: true },
  { name: 'SQL注入', action: 'block', severity: 'high', hits: 12, enabled: true },
  { name: '代码执行', action: 'warn', severity: 'medium', hits: 8, enabled: false },
])
</script>
