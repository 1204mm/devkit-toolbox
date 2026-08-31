<script setup lang="ts">
import { ref, computed } from 'vue'
import { message } from 'ant-design-vue'
import { t } from '../../i18n'

const count = ref(5)
const noDash = ref(false)
const upper = ref(false)
const rawList = ref<string[]>([])

// v4 UUID，用 crypto.getRandomValues 保证随机性
const genUuidV4 = (): string => {
  const b = crypto.getRandomValues(new Uint8Array(16))
  b[6] = (b[6] & 0x0f) | 0x40
  b[8] = (b[8] & 0x3f) | 0x80
  const h = Array.from(b, (x) => x.toString(16).padStart(2, '0')).join('')
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`
}

const generate = () => {
  const n = Math.max(1, Math.min(500, count.value || 1))
  count.value = n
  rawList.value = Array.from({ length: n }, genUuidV4)
}

// 选项变化时直接变换展示，不重新生成
const displayList = computed(() => {
  return rawList.value.map((u) => {
    let v = u
    if (noDash.value) v = v.replace(/-/g, '')
    if (upper.value) v = v.toUpperCase()
    return v
  })
})

const copyOne = (v: string) => {
  navigator.clipboard.writeText(v)
  message.success(t('uuid.copied') + v)
}

const copyAll = () => {
  navigator.clipboard.writeText(displayList.value.join('\n'))
  message.success(t('uuid.copiedN', { n: displayList.value.length }))
}

generate()
</script>

<template>
  <div class="tool">
    <div class="card">
      <div class="card-title">{{ t('uuid.title') }}</div>
      <div class="controls">
        <span class="ctl-label">{{ t('uuid.count') }}</span>
        <a-input-number v-model:value="count" :min="1" :max="500" :precision="0" style="width: 90px" @pressEnter="generate" />
        <a-checkbox v-model:checked="noDash">{{ t('uuid.noDash') }}</a-checkbox>
        <a-checkbox v-model:checked="upper">{{ t('uuid.upper') }}</a-checkbox>
        <div class="spacer"></div>
        <a-button @click="copyAll" :disabled="rawList.length === 0">{{ t('uuid.copyAll') }}</a-button>
        <a-button type="primary" @click="generate">{{ t('uuid.regenerate') }}</a-button>
      </div>
    </div>

    <div class="card" v-if="displayList.length">
      <div class="uuid-row" v-for="(u, i) in displayList" :key="i" @click="copyOne(u)">
        <span class="uuid-idx">{{ i + 1 }}</span>
        <span class="uuid-value mono">{{ u }}</span>
      </div>
      <div class="copy-tip">{{ t('uuid.copyTip') }}</div>
    </div>
  </div>
</template>

<style scoped>
.tool {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.card {
  background: #181825;
  border: 1px solid #313244;
  border-radius: 8px;
  padding: 14px 16px;
}

.card-title {
  font-size: 13px;
  font-weight: 600;
  color: #a6adc8;
  margin-bottom: 10px;
}

.controls {
  display: flex;
  align-items: center;
  gap: 16px;
}
.ctl-label {
  font-size: 13px;
  color: #6c7086;
}
.spacer {
  flex: 1;
}

.uuid-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.12s;
}
.uuid-row:hover {
  background: #11111b;
}
.uuid-idx {
  width: 24px;
  font-size: 11px;
  color: #45475a;
  text-align: right;
  flex-shrink: 0;
}
.uuid-value {
  font-size: 13px;
  color: #a6e3a1;
  word-break: break-all;
}
.mono {
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
}
.copy-tip {
  margin-top: 8px;
  font-size: 11px;
  color: #45475a;
  text-align: center;
}
</style>
