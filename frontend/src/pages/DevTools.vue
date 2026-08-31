<script setup lang="ts">
import { ref, computed } from 'vue'
import TimestampTool from './tools/TimestampTool.vue'
import JwtTool from './tools/JwtTool.vue'
import CronTool from './tools/CronTool.vue'
import UuidTool from './tools/UuidTool.vue'
import RegexTool from './tools/RegexTool.vue'
import { t } from '../i18n'

const tools = computed(() => [
  { key: 'timestamp', label: t('tools.timestamp'), desc: t('tools.timestampDesc') },
  { key: 'jwt', label: t('tools.jwt'), desc: t('tools.jwtDesc') },
  { key: 'cron', label: t('tools.cron'), desc: t('tools.cronDesc') },
  { key: 'uuid', label: t('tools.uuid'), desc: t('tools.uuidDesc') },
  { key: 'regex', label: t('tools.regex'), desc: t('tools.regexDesc') },
])

const active = ref('timestamp')
</script>

<template>
  <div class="devtools">
    <aside class="side">
      <button
        v-for="t in tools"
        :key="t.key"
        :class="['side-item', { active: active === t.key }]"
        @click="active = t.key"
      >
        <span class="side-label">{{ t.label }}</span>
        <span class="side-desc">{{ t.desc }}</span>
      </button>
    </aside>
    <div class="content">
      <TimestampTool v-if="active === 'timestamp'" />
      <JwtTool v-else-if="active === 'jwt'" />
      <CronTool v-else-if="active === 'cron'" />
      <UuidTool v-else-if="active === 'uuid'" />
      <RegexTool v-else-if="active === 'regex'" />
    </div>
  </div>
</template>

<style scoped>
.devtools {
  height: 100%;
  display: flex;
  gap: 12px;
  overflow: hidden;
}

.side {
  width: 170px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.side-item {
  background: none;
  border: 1px solid transparent;
  border-radius: 8px;
  padding: 10px 12px;
  text-align: left;
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.side-item:hover {
  background: #181825;
}
.side-item.active {
  background: #181825;
  border-color: #313244;
}
.side-item.active .side-label {
  color: #89b4fa;
}

.side-label {
  font-size: 13px;
  font-weight: 600;
  color: #cdd6f4;
}

.side-desc {
  font-size: 11px;
  color: #6c7086;
}

.content {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
}
</style>
