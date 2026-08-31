<script setup lang="ts">
import { ref, computed } from 'vue'
import { ConfigProvider } from 'ant-design-vue'
import { lang, setLang, antdLocale, isZh, t } from './i18n'
import PortManager from './pages/PortManager.vue'
import Crypto from './pages/Crypto.vue'
import JsonFormatter from './pages/JsonFormatter.vue'
import Totp from './pages/Totp.vue'
import DevTools from './pages/DevTools.vue'
import IconForge from './pages/IconForge.vue'

const activeMenu = ref('port')

const menuItems = computed(() => [
  { key: 'port', label: t('nav.port') },
  { key: 'crypto', label: t('nav.crypto') },
  { key: 'totp', label: t('nav.totp') },
  { key: 'json', label: t('nav.json') },
  { key: 'tools', label: t('nav.tools') },
  { key: 'iconforge', label: t('nav.iconforge') },
])

const toggleLang = () => setLang(isZh.value ? 'en' : 'zh')
</script>

<template>
  <a-config-provider :locale="antdLocale">
    <div class="app">
      <header class="header">
        <button class="lang-btn" @click="toggleLang" :title="isZh ? 'Switch language' : '切换语言'">
          {{ isZh ? t('switch.en') : t('switch.zh') }}
        </button>
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
        <DevTools v-else-if="activeMenu === 'tools'" />
        <IconForge v-else-if="activeMenu === 'iconforge'" />
      </main>
    </div>
  </a-config-provider>
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

.lang-btn {
  margin-right: 24px;
  border: 1px solid #313244;
  background: transparent;
  color: #a6adc8;
  font-size: 12px;
  line-height: 1;
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}

.lang-btn:hover {
  color: #89b4fa;
  border-color: #89b4fa;
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
