<script setup lang="ts">
import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { t } from '../i18n'

const input = ref('')
const output = ref('')
const indent = ref(2)

const format = () => {
  if (!input.value.trim()) {
    message.warning(t('json.enterJson'))
    return
  }
  try {
    const parsed = JSON.parse(input.value)
    output.value = JSON.stringify(parsed, null, indent.value)
    message.success(t('json.formatOk'))
  } catch (e: unknown) {
    output.value = ''
    message.error(t('json.syntaxErr') + (e instanceof Error ? e.message : String(e)))
  }
}

const minify = () => {
  if (!input.value.trim()) {
    message.warning(t('json.enterJson'))
    return
  }
  try {
    const parsed = JSON.parse(input.value)
    output.value = JSON.stringify(parsed)
    message.success(t('json.minifyOk'))
  } catch (e: unknown) {
    output.value = ''
    message.error(t('json.syntaxErr') + (e instanceof Error ? e.message : String(e)))
  }
}

const validate = () => {
  if (!input.value.trim()) {
    message.warning(t('json.enterJson'))
    return
  }
  try {
    JSON.parse(input.value)
    message.success(t('json.validOk'))
  } catch (e: unknown) {
    message.error(t('json.syntaxErr') + (e instanceof Error ? e.message : String(e)))
  }
}

const copyOutput = () => {
  if (!output.value) return
  navigator.clipboard.writeText(output.value)
  message.success(t('json.copied'))
}

const swapToInput = () => {
  if (!output.value) return
  input.value = output.value
  output.value = ''
}
</script>

<template>
  <div class="page">
    <div class="toolbar">
      <a-select v-model:value="indent" style="width: 110px">
        <a-select-option :value="2">{{ t('json.indent2') }}</a-select-option>
        <a-select-option :value="4">{{ t('json.indent4') }}</a-select-option>
        <a-select-option :value="0">{{ t('json.min') }}</a-select-option>
      </a-select>
      <a-button type="primary" @click="format">{{ t('json.format') }}</a-button>
      <a-button @click="minify">{{ t('json.minify') }}</a-button>
      <a-button @click="validate">{{ t('json.validate') }}</a-button>
    </div>
    <div class="editor-area">
      <div class="editor-col">
        <div class="col-label">{{ t('json.input') }}</div>
        <a-textarea
          v-model:value="input"
          :placeholder="t('json.placeholder')"
          :rows="16"
          class="json-input"
        />
      </div>
      <div class="editor-col">
        <div class="col-label">
          <span>{{ t('json.output') }}</span>
          <span class="col-actions">
            <a-button size="small" type="link" @click="copyOutput" :disabled="!output">{{ t('json.copy') }}</a-button>
            <a-button size="small" type="link" @click="swapToInput" :disabled="!output">{{ t('json.fillInput') }}</a-button>
          </span>
        </div>
        <pre class="json-output">{{ output }}</pre>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.toolbar {
  display: flex;
  gap: 8px;
  padding: 0 0 12px 0;
  align-items: center;
}

.editor-area {
  display: flex;
  gap: 12px;
  flex: 1;
  overflow: hidden;
}

.editor-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.col-label {
  font-size: 12px;
  color: #6c7086;
  margin-bottom: 6px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.col-actions {
  display: flex;
}

.json-input {
  flex: 1;
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  font-size: 13px;
}

.json-output {
  flex: 1;
  background: #11111b;
  border: 1px solid #313244;
  border-radius: 6px;
  padding: 12px;
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  font-size: 13px;
  color: #cdd6f4;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}
</style>
