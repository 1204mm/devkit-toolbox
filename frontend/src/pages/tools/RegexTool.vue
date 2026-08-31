<script setup lang="ts">
import { ref, computed } from 'vue'
import { t } from '../../i18n'

const patternInput = ref('\\d{4}-\\d{2}-\\d{2}')
const testText = ref('上线日期: 2026-08-29, 截止日期: 2026-09-15, 版本 v2.3.1')
const flags = ref({ g: true, i: false, m: false, s: false, u: false })

// 兼容粘贴 /pattern/flags 形式
const cleanPattern = computed(() => {
  const p = patternInput.value.trim()
  const m = p.match(/^\/(.+)\/([gimsuy]*)$/s)
  return m ? m[1] : p
})

const flagStr = computed(() => {
  let f = ''
  for (const [k, on] of Object.entries(flags.value)) {
    if (on) f += k
  }
  return f
})

interface CompileResult {
  re: RegExp | null
  error: string
}

const compiled = computed<CompileResult>(() => {
  if (!cleanPattern.value) return { re: null, error: '' }
  try {
    return { re: new RegExp(cleanPattern.value, flagStr.value), error: '' }
  } catch (e: unknown) {
    return { re: null, error: t('regex.syntaxErr') + (e instanceof Error ? e.message : String(e)) }
  }
})

interface MatchItem {
  text: string
  start: number
  end: number
  groups: string[]
}

const matches = computed<MatchItem[]>(() => {
  const { re } = compiled.value
  const text = testText.value
  if (!re || !text) return []
  const out: MatchItem[] = []
  try {
    if (flags.value.g) {
      for (const m of text.matchAll(re)) {
        if (m[0] === '') continue // 跳过零长度匹配
        out.push({
          text: m[0],
          start: m.index ?? 0,
          end: (m.index ?? 0) + m[0].length,
          groups: m.slice(1).map((g) => (g === undefined ? '∅' : g)),
        })
        if (out.length >= 500) break
      }
    } else {
      const m = re.exec(text)
      if (m && m[0] !== '') {
        out.push({
          text: m[0],
          start: m.index,
          end: m.index + m[0].length,
          groups: m.slice(1).map((g) => (g === undefined ? '∅' : g)),
        })
      }
    }
  } catch {
    // u 标志下非法序列等运行时错误，忽略
  }
  return out
})

// 高亮分段：匹配部分与普通文本交替
interface Segment {
  text: string
  isMatch: boolean
  first: boolean
}

const segments = computed<Segment[]>(() => {
  const text = testText.value
  if (!text) return []
  if (matches.value.length === 0) return [{ text, isMatch: false, first: false }]
  const segs: Segment[] = []
  let pos = 0
  for (const m of matches.value) {
    if (m.start > pos) segs.push({ text: text.slice(pos, m.start), isMatch: false, first: false })
    segs.push({ text: m.text, isMatch: true, first: false })
    pos = m.end
  }
  if (pos < text.length) segs.push({ text: text.slice(pos), isMatch: false, first: false })
  return segs
})
</script>

<template>
  <div class="tool">
    <div class="card">
      <div class="card-title">{{ t('regex.title') }}</div>
      <div class="pattern-row">
        <span class="slash">/</span>
        <input v-model="patternInput" class="pattern-input mono" spellcheck="false" />
        <span class="slash">/{{ flagStr }}</span>
      </div>
      <div class="flags">
        <a-checkbox v-model:checked="flags.g">{{ t('regex.g') }}</a-checkbox>
        <a-checkbox v-model:checked="flags.i">{{ t('regex.i') }}</a-checkbox>
        <a-checkbox v-model:checked="flags.m">{{ t('regex.m') }}</a-checkbox>
        <a-checkbox v-model:checked="flags.s">{{ t('regex.s') }}</a-checkbox>
        <a-checkbox v-model:checked="flags.u">{{ t('regex.u') }}</a-checkbox>
        <span class="match-count" v-if="compiled.re">{{ t('regex.count', { n: matches.length }) }}</span>
      </div>
      <div class="error" v-if="compiled.error">{{ compiled.error }}</div>
    </div>

    <div class="card">
      <div class="card-title">{{ t('regex.testText') }}</div>
      <a-textarea v-model:value="testText" :rows="5" class="mono" />
      <div class="highlight-box mono" v-if="testText">
        <template v-for="(seg, i) in segments" :key="i">
          <mark v-if="seg.isMatch" class="hl">{{ seg.text }}</mark>
          <span v-else>{{ seg.text }}</span>
        </template>
      </div>
    </div>

    <div class="card" v-if="matches.length">
      <div class="card-title">{{ t('regex.details') }}</div>
      <div class="match-row" v-for="(m, i) in matches" :key="i">
        <span class="m-idx">{{ i + 1 }}</span>
        <span class="m-text mono">{{ m.text }}</span>
        <span class="m-pos">[{{ m.start }}, {{ m.end }})</span>
        <span class="m-groups" v-if="m.groups.length">
          {{ t('regex.group') }}<code v-for="(g, gi) in m.groups" :key="gi" class="mono">${{ gi + 1 }}={{ g }}</code>
        </span>
      </div>
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

.mono {
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
}

.pattern-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.slash {
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  font-size: 16px;
  color: #f5c2e7;
  font-weight: 700;
}
.pattern-input {
  flex: 1;
  background: #11111b;
  border: 1px solid #313244;
  border-radius: 6px;
  padding: 8px 10px;
  font-size: 14px;
  color: #cdd6f4;
  outline: none;
  transition: border-color 0.15s;
}
.pattern-input:focus {
  border-color: #89b4fa;
}

.flags {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 10px;
}
.match-count {
  margin-left: auto;
  font-size: 12px;
  color: #a6e3a1;
}

.error {
  margin-top: 8px;
  font-size: 12px;
  color: #f38ba8;
}

.highlight-box {
  margin-top: 10px;
  background: #11111b;
  border: 1px solid #313244;
  border-radius: 6px;
  padding: 10px 12px;
  font-size: 13px;
  line-height: 1.8;
  color: #cdd6f4;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 180px;
  overflow: auto;
}
.hl {
  background: rgba(137, 180, 250, 0.25);
  color: #89b4fa;
  border-radius: 3px;
  padding: 0 1px;
}

.match-row {
  display: flex;
  align-items: baseline;
  gap: 10px;
  padding: 6px 0;
  border-top: 1px solid #313244;
  font-size: 13px;
}
.match-row:first-of-type {
  border-top: none;
}
.m-idx {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #313244;
  color: #a6adc8;
  font-size: 11px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  align-self: center;
}
.m-text {
  color: #89b4fa;
  word-break: break-all;
}
.m-pos {
  font-size: 11px;
  color: #45475a;
  flex-shrink: 0;
}
.m-groups {
  font-size: 12px;
  color: #6c7086;
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.m-groups code {
  color: #f9e2af;
}
</style>
