<script setup lang="ts">
import { ref } from 'vue'
import PortManager from './pages/PortManager.vue'
import Crypto from './pages/Crypto.vue'
import JsonFormatter from './pages/JsonFormatter.vue'
import Totp from './pages/Totp.vue'

const activeMenu = ref('port')

const menuItems = [
  { key: 'port', label: '端口管理' },
  { key: 'crypto', label: '加密解密' },
  { key: 'totp', label: '2FA验证码' },
  { key: 'json', label: 'JSON格式化' },
]
</script>

<template>
  <div class="app">
    <header class="header">
      <div class="header-left">
        <span class="logo">DevKit</span>
        <nav class="nav">
          <button
            v-for="item in menuItems"
            :key="item.key"
            :class="['nav-item', { active: activeMenu === item.key }]"
            @click="activeMenu = item.key"
          >
            {{ item.label }}
          </button>
        </nav>
      </div>
    </header>
    <main class="main">
      <PortManager v-if="activeMenu === 'port'" />
      <Crypto v-else-if="activeMenu === 'crypto'" />
      <Totp v-else-if="activeMenu === 'totp'" />
      <JsonFormatter v-else-if="activeMenu === 'json'" />
    </main>
  </div>
</template>

<style scoped>
.app {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #1e1e2e;
}

.header {
  height: 44px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  padding: 0 16px;
  background: #181825;
  border-bottom: 1px solid #313244;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 24px;
}

.logo {
  font-size: 15px;
  font-weight: 700;
  color: #89b4fa;
  letter-spacing: 0.5px;
}

.nav {
  display: flex;
  gap: 2px;
}

.nav-item {
  background: none;
  border: none;
  color: #6c7086;
  font-size: 13px;
  padding: 6px 14px;
  cursor: pointer;
  border-radius: 6px;
  transition: all 0.15s;
}

.nav-item:hover {
  color: #cdd6f4;
  background: #313244;
}

.nav-item.active {
  color: #89b4fa;
  background: #313244;
  font-weight: 500;
}

.main {
  flex: 1;
  padding: 16px;
  overflow: hidden;
}
</style>
