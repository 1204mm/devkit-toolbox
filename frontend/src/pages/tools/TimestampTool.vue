<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { message } from 'ant-design-vue'

// ===================== 当前时间戳 =====================
const nowSec = ref(0)
const nowMs = ref(0)
let timer: number | null = null

const tick = () => {
  nowMs.value = Date.now()
  nowSec.value = Math.floor(nowMs.value / 1000)
}

onMounted(() => {
  tick()
  timer = window.setInterval(tick, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const copyText = (v: string | number) => {
  navigator.clipboard.writeText(String(v))
  message.success('已复制: ' + v)
}

// ===================== 通用格式化 =====================
const pad = (n: number) => String(n).padStart(2, '0')
const formatLocal = (d: Date) =>
  `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
const formatUtc = (d: Date) =>
  `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())}`
const formatIso = (d: Date) => {
  const off = -d.getTimezoneOffset()
  const sign = off >= 0 ? '+' : '-'
  return `${formatLocal(d).replace(' ', 'T')}${sign}${pad(Math.floor(Math.abs(off) / 60))}:${pad(Math.abs(off) % 60)}`
}

// ===================== 时间戳 -> 日期 =====================
const tsInput = ref('')
const tsUnit = ref<'auto' | 's' | 'ms'>('auto')

const detectedUnit = computed<'s' | 'ms' | null>(() => {
  const raw = tsInput.value.trim()
  if (!/^\d{1,16}$/.test(raw)) return null
  if (tsUnit.value === 's') return 's'
  if (tsUnit.value === 'ms') return 'ms'
  // 13 位以上按毫秒，10 位左右按秒
  return raw.length >= 13 ? 'ms' : 's'
})

const tsResult = computed<Date | null>(() => {
  const unit = detectedUnit.value
  if (!unit) return null
  let n = Number(tsInput.value.trim())
  if (unit === 's') n *= 1000
  const d = new Date(n)
  return isNaN(d.getTime()) ? null : d
})

const unitTip = computed(() => {
  if (tsInput.value.trim() === '') return ''
  if (detectedUnit.value === null) return '请输入纯数字时间戳'
  return `已按 ${detectedUnit.value === 's' ? '秒' : '毫秒'} 解析`
})

// ===================== 日期 -> 时间戳 =====================
const dateInput = ref('')

const parsedDate = computed<Date | null>(() => {
  const s = dateInput.value.trim()
  if (!s) return null
  let t = s.replace(/\//g, '-').replace(' ', 'T')
  // 纯日期（YYYY-MM-DD）按本地时区零点解析，而不是 UTC
  if (/^\d{4}-\d{2}-\d{2}$/.test(t)) t += 'T00:00:00'
  const ms = Date.parse(t)
  if (isNaN(ms)) return null
  return new Date(ms)
})

const dateTip = computed(() => {
  if (dateInput.value.trim() === '') return ''
  return parsedDate.value ? '' : '无法解析，支持 2026-08-29 15:30:00 / 2026-08-29T15:30:00+08:00 等格式'
})

const fillNow = () => {
  dateInput.value = formatLocal(new Date())
}
const fillTodayStart = () => {
  const d = new Date()
  d.setHours(0, 0, 0, 0)
  dateInput.value = formatLocal(d)
}
</script>

<template>
  <div class="tool">
    <!-- 当前时间戳 -->
    <div class="card">
      <div class="card-title">
        当前时间戳
        <span class="live-dot"></span>
      </div>
      <div class="now-row">
        <div class="now-box" @click="copyText(nowSec)">
          <div class="now-label">秒（10位）</div>
          <div class="now-value">{{ nowSec }}</div>
        </div>
        <div class="now-box" @click="copyText(nowMs)">
          <div class="now-label">毫秒（13位）</div>
          <div class="now-value">{{ nowMs }}</div>
        </div>
      </div>
      <div class="now-time">{{ formatLocal(new Date(nowMs)) }} · 点击复制</div>
    </div>

    <!-- 时间戳 -> 日期 -->
    <div class="card">
      <div class="card-title">时间戳 → 日期</div>
      <div class="input-row">
        <a-input
          v-model:value="tsInput"
          placeholder="粘贴时间戳，如 1787000000 或 1787000000000"
          class="mono"
          allow-clear
        />
        <a-radio-group v-model:value="tsUnit" size="small" class="unit-group">
          <a-radio-button value="auto">自动</a-radio-button>
          <a-radio-button value="s">秒</a-radio-button>
          <a-radio-button value="ms">毫秒</a-radio-button>
        </a-radio-group>
      </div>
      <div class="tip" :class="{ err: detectedUnit === null && tsInput.trim() !== '' }">{{ unitTip }}</div>
      <template v-if="tsResult">
        <div class="result-row"><span class="rk">本地时间</span><span class="rv hl" @click="copyText(formatLocal(tsResult))">{{ formatLocal(tsResult) }}</span></div>
        <div class="result-row"><span class="rk">星期</span><span class="rv">周{{ '日一二三四五六'[tsResult.getDay()] }}</span></div>
        <div class="result-row"><span class="rk">ISO 8601</span><span class="rv" @click="copyText(formatIso(tsResult))">{{ formatIso(tsResult) }}</span></div>
        <div class="result-row"><span class="rk">UTC 时间</span><span class="rv">{{ formatUtc(tsResult) }}</span></div>
      </template>
    </div>

    <!-- 日期 -> 时间戳 -->
    <div class="card">
      <div class="card-title">
        日期 → 时间戳
        <span class="quick">
          <a @click="fillNow">现在</a>
          <a @click="fillTodayStart">今天零点</a>
        </span>
      </div>
      <a-input
        v-model:value="dateInput"
        placeholder="如 2026-08-29 15:30:00 或 2026-08-29T15:30:00+08:00"
        class="mono"
        allow-clear
      />
      <div class="tip" :class="{ err: dateTip !== '' }">{{ dateTip || '支持带/不带时区的格式，不带时区按本地时间解析' }}</div>
      <template v-if="parsedDate">
        <div class="result-row"><span class="rk">秒（10位）</span><span class="rv hl" @click="copyText(Math.floor(parsedDate.getTime() / 1000))">{{ Math.floor(parsedDate.getTime() / 1000) }}</span></div>
        <div class="result-row"><span class="rk">毫秒（13位）</span><span class="rv hl" @click="copyText(parsedDate.getTime())">{{ parsedDate.getTime() }}</span></div>
      </template>
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
  display: flex;
  align-items: center;
  gap: 8px;
}

.quick {
  margin-left: auto;
  font-weight: 400;
}
.quick a {
  margin-left: 10px;
  font-size: 12px;
}

.live-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #a6e3a1;
  animation: blink 2s infinite;
}
@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.now-row {
  display: flex;
  gap: 12px;
}
.now-box {
  flex: 1;
  background: #11111b;
  border: 1px solid #313244;
  border-radius: 6px;
  padding: 10px 12px;
  cursor: pointer;
  transition: border-color 0.15s;
}
.now-box:hover {
  border-color: #89b4fa;
}
.now-label {
  font-size: 11px;
  color: #6c7086;
  margin-bottom: 4px;
}
.now-value {
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  font-size: 22px;
  font-weight: 700;
  color: #a6e3a1;
}
.now-time {
  margin-top: 8px;
  font-size: 12px;
  color: #6c7086;
}

.input-row {
  display: flex;
  gap: 8px;
  align-items: center;
}
.unit-group {
  flex-shrink: 0;
}
.mono {
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
}

.tip {
  font-size: 12px;
  color: #6c7086;
  margin: 6px 0 2px;
  min-height: 18px;
}
.tip.err {
  color: #f38ba8;
}

.result-row {
  display: flex;
  padding: 6px 0;
  border-top: 1px solid #313244;
  font-size: 13px;
}
.result-row:first-of-type {
  border-top: none;
}
.rk {
  width: 90px;
  flex-shrink: 0;
  color: #6c7086;
}
.rv {
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  color: #cdd6f4;
  word-break: break-all;
}
.rv.hl {
  color: #89b4fa;
  cursor: pointer;
}
.rv.hl:hover {
  color: #b4befe;
}
</style>
