<script setup lang="ts">
import { ref, computed } from 'vue'
import { CronExpressionParser } from 'cron-parser'
import cronstrue from 'cronstrue'
import 'cronstrue/locales/zh_CN'

const expr = ref('0 0/30 9-18 * * MON-FRI')

const EXAMPLES: Array<[string, string]> = [
  ['0 0/5 * * * ?', '每 5 分钟'],
  ['0 0 2 * * ?', '每天凌晨 2 点'],
  ['0 0 9-18 * * MON-FRI', '工作日 9-18 点整点'],
  ['0 30 9 ? * 6#3', '每月第 3 个周六 9:30'],
  ['0 0 12 L * ?', '每月最后一天中午 12 点'],
  ['*/10 * * * *', '每 10 分钟（标准5段）'],
]

interface FieldDef {
  label: string
  value: string
}

interface ParseResult {
  fields: FieldDef[]
  description: string
  nextRuns: string[]
  error: string
  yearFiltered: boolean
}

const pad = (n: number) => String(n).padStart(2, '0')
const WEEK = ['日', '一', '二', '三', '四', '五', '六']
const formatRun = (d: Date) =>
  `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())} 周${WEEK[d.getDay()]}`

// 解析 Quartz 年份字段（第7段）：支持 2026 / 2026,2028 / 2026-2028
const parseYears = (field: string): Set<number> | null => {
  const years = new Set<number>()
  for (const part of field.split(',')) {
    const range = part.match(/^(\d{4})-(\d{4})$/)
    if (range) {
      for (let y = Number(range[1]); y <= Number(range[2]); y++) years.add(y)
    } else if (/^\d{4}$/.test(part)) {
      years.add(Number(part))
    } else {
      return null
    }
  }
  return years.size > 0 ? years : null
}

const result = computed<ParseResult>(() => {
  const raw = expr.value.trim().replace(/\s+/g, ' ')
  if (!raw) return { fields: [], description: '', nextRuns: [], error: '', yearFiltered: false }

  let parts = raw.split(' ')
  if (parts.length < 5 || parts.length > 7) {
    return {
      fields: [], description: '', nextRuns: [],
      error: `字段数量错误：${parts.length} 段。标准 cron 为 5 段（分 时 日 月 周），Quartz 为 6~7 段（秒 分 时 日 月 周 [年]）`,
      yearFiltered: false,
    }
  }

  // Quartz 带 7 段时最后一段是年份，cron-parser 不支持，拆出来自己过滤
  let years: Set<number> | null = null
  if (parts.length === 7) {
    years = parseYears(parts[6])
    if (!years) {
      return { fields: [], description: '', nextRuns: [], error: `无法解析年份字段「${parts[6]}」`, yearFiltered: false }
    }
    parts = parts.slice(0, 6)
  }
  const parseExpr = parts.join(' ')

  // 字段说明
  const labels =
    parts.length === 6
      ? ['秒', '分', '时', '日', '月', '周']
      : ['分', '时', '日', '月', '周']
  const fields: FieldDef[] = parts.map((v, i) => ({ label: labels[i], value: v }))

  try {
    const parsed = CronExpressionParser.parse(parseExpr, { currentDate: new Date() })
    const nextRuns: string[] = []
    let yearFiltered = false
    for (let i = 0; i < 5; i++) {
      const d = parsed.next().toDate()
      if (years && !years.has(d.getFullYear())) {
        yearFiltered = true
        break
      }
      nextRuns.push(formatRun(d))
    }
    let description = ''
    try {
      description = cronstrue.toString(parseExpr, { locale: 'zh_CN' })
    } catch {
      description = ''
    }
    return { fields, description, nextRuns, error: '', yearFiltered }
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    return { fields, description: '', nextRuns: [], error: '表达式无效: ' + msg, yearFiltered: false }
  }
})
</script>

<template>
  <div class="tool">
    <div class="card">
      <div class="card-title">
        Cron 表达式
        <span class="sub">支持标准 5 段和 Quartz 6~7 段（秒 分 时 日 月 周 [年]）</span>
      </div>
      <div class="input-row">
        <a-input v-model:value="expr" placeholder="0 0/30 9-18 * * MON-FRI" class="mono big" allow-clear />
      </div>
      <div class="examples">
        <a-tag
          v-for="[e, label] in EXAMPLES"
          :key="e"
          class="example-tag"
          :color="expr.trim().replace(/\s+/g, ' ') === e ? 'blue' : 'default'"
          @click="expr = e"
        >
          {{ label }}
        </a-tag>
      </div>
    </div>

    <div class="card" v-if="result.error">
      <div class="error">{{ result.error }}</div>
    </div>

    <template v-if="!result.error && (result.description || result.nextRuns.length)">
      <div class="card" v-if="result.description">
        <div class="card-title">含义</div>
        <div class="desc">{{ result.description }}</div>
      </div>

      <div class="card">
        <div class="card-title">
          最近 5 次执行时间
          <span class="sub" v-if="result.yearFiltered">（年份字段限制内）</span>
        </div>
        <div class="run-row" v-for="(r, i) in result.nextRuns" :key="i">
          <span class="run-idx">{{ i + 1 }}</span>
          <span class="run-time">{{ r }}</span>
        </div>
      </div>

      <div class="card" v-if="result.fields.length">
        <div class="card-title">字段拆解</div>
        <div class="fields">
          <div class="field" v-for="(f, i) in result.fields" :key="i">
            <div class="field-label">{{ f.label }}</div>
            <div class="field-value mono">{{ f.value }}</div>
          </div>
        </div>
      </div>
    </template>
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

.sub {
  font-weight: 400;
  font-size: 12px;
  color: #6c7086;
}

.mono {
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
}
.big {
  font-size: 15px;
}

.examples {
  margin-top: 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.example-tag {
  cursor: pointer;
  user-select: none;
}

.error {
  color: #f38ba8;
  font-size: 13px;
}

.desc {
  font-size: 14px;
  color: #a6e3a1;
}

.run-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 0;
  border-top: 1px solid #313244;
  font-size: 13px;
}
.run-row:first-of-type {
  border-top: none;
}
.run-idx {
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
}
.run-time {
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  color: #cdd6f4;
}

.fields {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.field {
  background: #11111b;
  border: 1px solid #313244;
  border-radius: 6px;
  padding: 8px 12px;
  min-width: 64px;
  text-align: center;
}
.field-label {
  font-size: 11px;
  color: #6c7086;
  margin-bottom: 4px;
}
.field-value {
  font-size: 14px;
  color: #89b4fa;
  font-weight: 600;
  word-break: break-all;
}
</style>
