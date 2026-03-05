<template>
  <div class="login-pending-page">
    <div class="container">
      <div class="pending-icon">
        <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/>
        </svg>
      </div>
      <h1>账号审核中</h1>
      <div class="message">
        您的账号正在审核中，请等待管理员审核通过后再登录
      </div>
      <div class="info-box">
        <p>⏳ 账号正在等待管理员审核</p>
        <p>📧 审核通过后会收到通知</p>
        <p>🔒 请耐心等待审核结果</p>
      </div>
      <div class="close-tip">
        <strong>提示：</strong>您可以关闭此页面，审核通过后重新扫码登录
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.login-pending-page {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.container {
  background: white;
  border-radius: 20px;
  padding: 40px 30px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  text-align: center;
  max-width: 400px;
  width: 100%;
}

.pending-icon {
  width: 80px;
  height: 80px;
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  border-radius: 50%;
  margin: 0 auto 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  animation: pulse 2s ease-in-out infinite;
}

.pending-icon svg {
  width: 40px;
  height: 40px;
  fill: white;
}

@keyframes pulse {
  0%, 100% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.1);
  }
}

h1 {
  color: #333;
  font-size: 28px;
  margin-bottom: 15px;
  font-weight: 600;
}

.message {
  color: #666;
  font-size: 16px;
  line-height: 1.6;
  margin-bottom: 30px;
}

.info-box {
  background: #fff3f3;
  border-radius: 10px;
  padding: 20px;
  margin-bottom: 20px;
  border-left: 4px solid #f5576c;
}

.info-box p {
  color: #555;
  font-size: 14px;
  margin: 5px 0;
}

.close-tip {
  color: #999;
  font-size: 14px;
  margin-top: 20px;
}

.close-tip strong {
  color: #f5576c;
}
</style>
