<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <div class="max-w-3xl mx-auto px-6 py-16">
      <h1 class="text-3xl font-bold text-gray-900 dark:text-white mb-2">服务条款</h1>
      <p class="text-sm text-gray-500 mb-8">最后更新：2026 年 6 月 1 日</p>

      <div class="bg-white dark:bg-gray-800 rounded-2xl border border-gray-200 dark:border-gray-700 p-8 space-y-6 prose dark:prose-invert max-w-none">
        <section>
          <h2>1. 服务概述</h2>
          <p>TokenHub（以下简称"本平台"）提供 AI API 网关服务，允许用户通过统一接口访问多种 AI 模型。使用本服务即表示您同意遵守以下条款。</p>
        </section>

        <section>
          <h2>2. 账户注册</h2>
          <p>2.1 您需要注册账户才能使用本服务，注册时须提供真实、准确的信息。</p>
          <p>2.2 您有责任保管好自己的账户凭证，因账户凭证泄露导致的损失由您自行承担。</p>
          <p>2.3 每人仅限注册一个账户，禁止出售或转让账户。</p>
        </section>

        <section>
          <h2>3. 服务使用</h2>
          <p>3.1 本服务采用按量计费模式，根据实际使用的 Token 数量计算费用。</p>
          <p>3.2 余额不足时服务将自动暂停，充值后可继续使用。</p>
          <p>3.3 禁止使用本服务从事任何违法活动，包括但不限于生成违法内容、网络攻击等。</p>
          <p>3.4 本平台保留对滥用行为的账户进行限制或封禁的权利。</p>
        </section>

        <section>
          <h2>4. 退款政策</h2>
          <p>4.1 充值后 7 天内未使用的余额可申请全额退款。</p>
          <p>4.2 已使用的服务费用不予退还。</p>
          <p>4.3 退款将在 5-10 个工作日内原路返回。</p>
        </section>

        <section>
          <h2>5. 数据安全</h2>
          <p>5.1 本平台不存储用户的请求和响应内容，数据仅做转发处理。</p>
          <p>5.2 所有通信采用 TLS 加密。</p>
          <p>5.3 API Key 支持权限隔离和过期设置。</p>
        </section>

        <section>
          <h2>6. 服务保障</h2>
          <p>6.1 本平台承诺 99.9% 的服务可用性（企业版 99.99%）。</p>
          <p>6.2 因不可抗力导致的服务中断不在此保障范围内。</p>
          <p>6.3 计划性维护将提前 24 小时通知。</p>
        </section>

        <section>
          <h2>7. 知识产权</h2>
          <p>7.1 通过本平台生成的 AI 输出内容的知识产权归属依据适用法律确定。</p>
          <p>7.2 本平台的服务标识、界面设计等受知识产权法保护。</p>
        </section>

        <section>
          <h2>8. 免责声明</h2>
          <p>8.1 AI 模型的输出可能包含不准确或不当内容，本平台不对模型输出承担责任。</p>
          <p>8.2 因上游 API 服务商的原因导致的服务异常，本平台不承担责任，但会尽力提供替代方案。</p>
        </section>

        <section>
          <h2>9. 条款变更</h2>
          <p>9.1 本平台保留随时修改本条款的权利，重大变更将通过邮件或公告通知。</p>
          <p>9.2 继续使用服务即视为同意修改后的条款。</p>
        </section>

        <section>
          <h2>10. 联系方式</h2>
          <p>如有任何疑问，请通过以下方式联系我们：</p>
          <p>邮箱：support@tokenhub.cc</p>
        </section>
      </div>

      <div class="mt-8 text-center">
        <router-link to="/" class="text-primary-600 hover:underline text-sm">返回首页</router-link>
      </div>
    </div>
  </div>
</template>
