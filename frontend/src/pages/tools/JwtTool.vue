<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

const token = ref('')
const nowMs = ref(Date.now())
let timer: number | null = null

onMounted(() => {
  timer = window.setInterval(() => {
    nowMs.value = Date.now()
  }, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

// base64url 解码（JWT 不带 padding，需补齐）
const b64urlDecode = (s: string): string => {
  let t = s.replace(/-/g, '+').replace(/_/g, '/')
  while (t.length % 4) t += '='
  const bytes = Uint8Array.from(atob(t), (c) => c.charCodeAt(0))
  return new TextDecoder().decode(bytes)
}

interface DecodeResult {
  header: object | null
  payload: object | null
  signature: string
  error: string
}

const decodeResult = computed<DecodeResult>(() => {
  const raw = token.value.trim().replace(/^Bearer\s+/i, '')
  if (!raw) return { header: null, payload: null, signature: '', error: '' }
  const parts = raw.split('.')
  if (parts.length < 2 || parts.length > 3 || parts[0] === '' || parts[1] === '') {
    return { header: null, payload: null, signature: '', error: '不是有效的 JWT（格式应为 header.payload.signature）' }
  }
  try {
    const header = JSON.parse(b64urlDecode(parts[0]))
    const payload = JSON.parse(b64urlDecode(parts[1]))
    return { header, payload, signature: parts[2] || '', error: '' }
  } catch {
    return { header: null, payload: null, signature: '', error: '解码失败：header/payload 不是有效的 Base64URL JSON' }
  }
})

const headerJson = computed(() =>
  decodeResult.value.header ? JSON.stringify(decodeResult.value.header, null, 2) : ''
)
const payloadJson = computed(() =>
  decodeResult.value.payload ? JSON.stringify(decodeResult.value.payload, null, 2) : ''
)

const pad = (n: number) => String(n).padStart(2, '0')
const formatLocal = (ms: number) => {
  const d = new Date(ms)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

// 时间声明（exp/iat/nbf）所在行高亮
const isClaimLine = (line: string) => /"(exp|iat|nbf)"\s*:/.test(line)

interface ClaimInfo {
  key: string
  label: string
  ms: number
  desc: string
  state: 'ok' | 'expired' | 'pending'
}

const humanDelta = (deltaMs: number): string => {
  const abs = Math.abs(deltaMs)
  const day = Math.floor(abs / 86400000)
  const hour = Math.floor((abs % 86400000) / 3600000)
  const min = Math.floor((abs % 3600000) / 60000)
  const sec = Math.floor((abs % 60000) / 1000)
  if (day > 0) return `${day} 天 ${hour} 小时`
  if (hour > 0) return `${hour} 小时 ${min} 分`
  if (min > 0) return `${min} 分 ${sec} 秒`
  return `${sec} 秒`
}

const claims = computed<ClaimInfo[]>(() => {
  const p = decodeResult.value.payload as Record<string, unknown> | null
  if (!p) return []
  const out: ClaimInfo[] = []
  const defs: Array<[string, string, ClaimInfo['state'] | 'auto']> = [
    ['nbf', '生效时间 (nbf)', 'pending'],
    ['iat', '签发时间 (iat)', 'ok'],
    ['exp', '过期时间 (exp)', 'auto'],
  ]
  for (const [key, label, defaultState] of defs) {
    const v = p[key]
    if (typeof v !== 'number') continue
    const ms = v * 1000
    let state: ClaimInfo['state'] = defaultState === 'auto' ? 'ok' : defaultState
    let desc = ''
    if (key === 'exp') {
      if (ms > nowMs.value) {
        state = 'ok'
        desc = `有效 · 剩余 ${humanDelta(ms - nowMs.value)}`
      } else {
        state = 'expired'
        desc = `已过期 ${humanDelta(nowMs.value - ms)}`
      }
    } else if (key === 'nbf') {
      state = ms <= nowMs.value ? 'ok' : 'pending'
      desc = ms <= nowMs.value ? '已生效' : `${humanDelta(ms - nowMs.value)}后生效`
    }
    out.push({ key, label, ms, desc, state })
  }
  return out
})

const expState = computed(() => claims.value.find((c) => c.key === 'exp'))
const alg = computed(() => {
  const h = decodeResult.value.header as Record<string, unknown> | null
  return h && typeof h.alg === 'string' ? h.alg : ''
})

const sigShort = computed(() => {
  const s = decodeResult.value.signature
  if (!s) return '无签名（alg: none）'
  return s.length > 28 ? s.slice(0, 28) + '…' : s
})
</script>

<template>
  <div class="tool">
    <div class="card">
      <div class="card-title">粘贴 JWT（自动解码，支持 Bearer 前缀）</div>
      <a-textarea
        v-model:value="token"
        placeholder="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NSIsIm5hbWUiOiJ0ZXN0In0.xxx"
        :rows="4"
        class="mono"
      />
      <div class="tip err" v-if="decodeResult.error">{{ decodeResult.error }}</div>
    </div>

    <template v-if="decodeResult.header">
      <!-- 过期状态卡 -->
      <div class="card" v-if="expState">
        <div class="exp-banner" :class="expState.state">
          <span class="exp-state">{{ expState.state === 'ok' ? '✓ Token 有效' : '✗ Token 已过期' }}</span>
          <span class="exp-desc">{{ expState.desc }}</span>
        </div>
        <div class="result-row" v-for="c in claims" :key="c.key">
          <span class="rk">{{ c.label }}</span>
          <span class="rv">
            {{ formatLocal(c.ms) }}
            <em v-if="c.desc" :class="c.state">{{ c.desc }}</em>
          </span>
        </div>
      </div>

      <div class="pair">
        <div class="card half">
          <div class="card-title">
            Header
            <a-tag v-if="alg" color="blue" class="alg-tag">{{ alg }}</a-tag>
          </div>
          <pre class="code-block"><span
            v-for="(line, i) in headerJson.split('\n')"
            :key="i"
            :class="{ 'claim-line': isClaimLine(line) }"
          >{{ line }}
</span></pre>
        </div>
        <div class="card half">
          <div class="card-title">Payload</div>
          <pre class="code-block"><span
            v-for="(line, i) in payloadJson.split('\n')"
            :key="i"
            :class="{ 'claim-line': isClaimLine(line) }"
          >{{ line }}
</span></pre>
        </div>
      </div>

      <div class="card">
        <div class="card-title">Signature（不校验签名，仅展示）</div>
        <div class="sig mono">{{ sigShort }}</div>
      </div>
    </template>
    <div class="empty" v-else-if="!decodeResult.error">
      <p>粘贴任意 JWT 后自动解码</p>
      <p class="tip2">解码在本地完成，不会发送任何数据</p>
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

.alg-tag {
  margin-left: auto;
}

.mono {
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
}

.tip {
  font-size: 12px;
  margin-top: 6px;
}
.tip.err {
  color: #f38ba8;
}

.pair {
  display: flex;
  gap: 12px;
}
.half {
  flex: 1;
  min-width: 0;
}

.code-block {
  margin: 0;
  background: #11111b;
  border: 1px solid #313244;
  border-radius: 6px;
  padding: 12px;
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.7;
  color: #cdd6f4;
  overflow: auto;
  max-height: 320px;
  white-space: pre-wrap;
  word-break: break-all;
}

:deep(.claim-line) {
  display: inline;
  background: rgba(249, 226, 175, 0.12);
  color: #f9e2af;
  border-radius: 3px;
}

.exp-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 6px;
  margin-bottom: 10px;
  font-size: 13px;
}
.exp-banner.ok {
  background: rgba(166, 227, 161, 0.1);
  color: #a6e3a1;
}
.exp-banner.expired {
  background: rgba(243, 139, 168, 0.1);
  color: #f38ba8;
}
.exp-state {
  font-weight: 700;
}
.exp-desc {
  font-size: 12px;
  opacity: 0.85;
}

.result-row {
  display: flex;
  padding: 6px 0;
  font-size: 13px;
}
.rk {
  width: 130px;
  flex-shrink: 0;
  color: #6c7086;
}
.rv {
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  color: #cdd6f4;
}
.rv em {
  font-style: normal;
  margin-left: 10px;
  font-size: 12px;
}
.rv em.ok {
  color: #a6e3a1;
}
.rv em.expired {
  color: #f38ba8;
}
.rv em.pending {
  color: #f9e2af;
}

.sig {
  background: #11111b;
  border: 1px solid #313244;
  border-radius: 6px;
  padding: 10px 12px;
  font-size: 13px;
  color: #6c7086;
  word-break: break-all;
}

.empty {
  padding: 60px 0;
  text-align: center;
  color: #6c7086;
}
.empty p {
  margin: 4px 0;
}
.tip2 {
  font-size: 12px;
  color: #45475a;
}
</style>
