<template>
  <div class="login-success-page">
    <div class="container">
      <div class="success-icon">
        <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
          <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
        </svg>
      </div>
      <h1>登录成功</h1>
      <div class="message">
        PC端已自动登录成功！
      </div>
      <div class="info-box">
        <p>✓ 您的账号已安全登录</p>
        <p>✓ PC端已自动跳转到首页</p>
        <p>✓ 可以开始使用系统了</p>
      </div>
      <div class="close-tip">
        <strong>提示：</strong>您可以关闭此页面
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.login-success-page {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
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

.success-icon {
  width: 80px;
  height: 80px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 50%;
  margin: 0 auto 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  animation: scaleIn 0.5s ease-out;
}

.success-icon svg {
  width: 40px;
  height: 40px;
  fill: white;
}

@keyframes scaleIn {
  from {
    transform: scale(0);
    opacity: 0;
  }
  to {
    transform: scale(1);
    opacity: 1;
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
  background: #f8f9fa;
  border-radius: 10px;
  padding: 20px;
  margin-bottom: 20px;
  border-left: 4px solid #667eea;
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
  color: #667eea;
}
</style>
