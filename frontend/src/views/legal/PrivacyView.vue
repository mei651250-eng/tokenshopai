<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <div class="max-w-3xl mx-auto px-6 py-16">
      <h1 class="text-3xl font-bold text-gray-900 dark:text-white mb-2">隐私政策</h1>
      <p class="text-sm text-gray-500 mb-8">最后更新：2026 年 6 月 1 日</p>

      <div class="bg-white dark:bg-gray-800 rounded-2xl border border-gray-200 dark:border-gray-700 p-8 space-y-6 prose dark:prose-invert max-w-none">
        <section>
          <h2>1. 信息收集</h2>
          <p>我们收集以下类型的信息以提供和改善服务：</p>
          <ul>
            <li><strong>账户信息</strong>：邮箱、用户名（注册时提供）</li>
            <li><strong>使用数据</strong>：API 调用量、Token 消耗、模型选择（用于计费和统计）</li>
            <li><strong>技术数据</strong>：浏览器类型、IP 地址（用于安全防护）</li>
            <li><strong>支付数据</strong>：充值和消费记录（用于财务对账）</li>
          </ul>
        </section>

        <section>
          <h2>2. 信息使用</h2>
          <p>我们仅将收集的信息用于以下目的：</p>
          <ul>
            <li>提供、维护和改善服务</li>
            <li>处理计费和退款</li>
            <li>发送服务通知（如余额不足、系统维护）</li>
            <li>安全防护和反欺诈</li>
            <li>遵守法律法规要求</li>
          </ul>
        </section>

        <section>
          <h2>3. 数据安全</h2>
          <p>3.1 所有通信采用 TLS 1.2+ 加密传输。</p>
          <p>3.2 密码使用 bcrypt 加密存储，永远不以明文形式保存。</p>
          <p>3.3 API Key 使用加密存储，仅在使用时解密。</p>
          <p>3.4 我们不存储用户通过 API 发送的请求和响应内容。</p>
          <p>3.5 实施严格的访问控制，仅授权人员可访问用户数据。</p>
        </section>

        <section>
          <h2>4. 数据共享</h2>
          <p>我们不会出售您的个人信息。仅在以下情况下可能共享数据：</p>
          <ul>
            <li>经您明确同意</li>
            <li>法律法规要求</li>
            <li>保护本平台及用户的合法权益</li>
          </ul>
        </section>

        <section>
          <h2>5. 数据保留</h2>
          <p>5.1 账户数据保留至您删除账户。</p>
          <p>5.2 计费记录保留 5 年（税务合规要求）。</p>
          <p>5.3 操作审计日志保留 1 年。</p>
          <p>5.4 您可以随时请求删除个人数据（法律要求保留的除外）。</p>
        </section>

        <section>
          <h2>6. Cookie 使用</h2>
          <p>我们使用 Cookie 和类似技术来：</p>
          <ul>
            <li>维持登录状态</li>
            <li>记住您的偏好设置（如深色模式、语言）</li>
            <li>分析服务使用情况</li>
          </ul>
        </section>

        <section>
          <h2>7. 您的权利</h2>
          <p>根据适用法律，您有权：</p>
          <ul>
            <li>访问您的个人数据</li>
            <li>更正不准确的数据</li>
            <li>请求删除数据</li>
            <li>撤回同意</li>
            <li>数据可携带</li>
          </ul>
        </section>

        <section>
          <h2>8. 儿童隐私</h2>
          <p>本服务不面向 16 岁以下未成年人。如果我们发现未成年人的数据，将立即删除。</p>
        </section>

        <section>
          <h2>9. 政策更新</h2>
          <p>我们可能会定期更新本隐私政策。重大变更将通过邮件或站内通知告知。</p>
        </section>

        <section>
          <h2>10. 联系我们</h2>
          <p>如有隐私相关问题，请联系：privacy@tokenhub.cc</p>
        </section>
      </div>

      <div class="mt-8 text-center">
        <router-link to="/" class="text-primary-600 hover:underline text-sm">返回首页</router-link>
      </div>
    </div>
  </div>
</template>
