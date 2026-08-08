<script setup lang="ts">
import { ref } from 'vue'
import { message } from 'ant-design-vue'

const input = ref('')
const output = ref('')
const indent = ref(2)

const format = () => {
  if (!input.value.trim()) {
    message.warning('请输入 JSON')
    return
  }
  try {
    const parsed = JSON.parse(input.value)
    output.value = JSON.stringify(parsed, null, indent.value)
    message.success('格式化成功')
  } catch (e: unknown) {
    output.value = ''
    message.error('JSON 语法错误: ' + (e instanceof Error ? e.message : String(e)))
  }
}

const minify = () => {
  if (!input.value.trim()) {
    message.warning('请输入 JSON')
    return
  }
  try {
    const parsed = JSON.parse(input.value)
    output.value = JSON.stringify(parsed)
    message.success('压缩成功')
  } catch (e: unknown) {
    output.value = ''
    message.error('JSON 语法错误: ' + (e instanceof Error ? e.message : String(e)))
  }
}

const validate = () => {
  if (!input.value.trim()) {
    message.warning('请输入 JSON')
    return
  }
  try {
    JSON.parse(input.value)
    message.success('JSON 语法正确')
  } catch (e: unknown) {
    message.error('JSON 语法错误: ' + (e instanceof Error ? e.message : String(e)))
  }
}

const copyOutput = () => {
  if (!output.value) return
  navigator.clipboard.writeText(output.value)
  message.success('已复制')
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
        <a-select-option :value="2">2 空格缩进</a-select-option>
        <a-select-option :value="4">4 空格缩进</a-select-option>
        <a-select-option :value="0">压缩(无缩进)</a-select-option>
      </a-select>
      <a-button type="primary" @click="format">格式化</a-button>
      <a-button @click="minify">压缩</a-button>
      <a-button @click="validate">校验</a-button>
    </div>
    <div class="editor-area">
      <div class="editor-col">
        <div class="col-label">输入</div>
        <a-textarea
          v-model:value="input"
          placeholder='粘贴 JSON，如 {"name":"test","value":123}'
          :rows="16"
          class="json-input"
        />
      </div>
      <div class="editor-col">
        <div class="col-label">
          <span>输出</span>
          <span class="col-actions">
            <a-button size="small" type="link" @click="copyOutput" :disabled="!output">复制</a-button>
            <a-button size="small" type="link" @click="swapToInput" :disabled="!output">填入输入</a-button>
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
