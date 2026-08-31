<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { t } from '../i18n'
import {
  TOTPGenerateAll, TOTPAddSecret, TOTPDeleteSecret,
  TOTPIsPasswordSet, TOTPSetupPassword, TOTPUnlock, TOTPIsUnlocked,
} from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'

type TOTPCode = main.TOTPCode

const codes = ref<TOTPCode[]>([])
const remain = ref(30)
const timer = ref<number | null>(null)
const unlocked = ref(false)

// 密码弹窗
const pwdVisible = ref(false)
const pwdMode = ref<'setup' | 'unlock'>('unlock')
const pwdInput = ref('')
const pwdConfirm = ref('')

// 添加密钥弹窗
const addVisible = ref(false)
const addName = ref('')
const addSecret = ref('')
const addIssuer = ref('')

const checkAuth = async () => {
  try {
    const isSet = await TOTPIsPasswordSet()
    if (!isSet) {
      pwdMode.value = 'setup'
      pwdInput.value = ''
      pwdConfirm.value = ''
      pwdVisible.value = true
    } else {
      const isUnlocked = await TOTPIsUnlocked()
      if (isUnlocked) {
        unlocked.value = true
        await fetchCodes()
        startTimer()
      } else {
        pwdMode.value = 'unlock'
        pwdInput.value = ''
        pwdVisible.value = true
      }
    }
  } catch (e: unknown) {
    message.error(t('totp.initFailed') + (e instanceof Error ? e.message : String(e)))
  }
}

const doPwdSubmit = async () => {
  if (!pwdInput.value) {
    message.warning(t('totp.enterPwd'))
    return
  }
  try {
    if (pwdMode.value === 'setup') {
      if (pwdInput.value !== pwdConfirm.value) {
        message.warning(t('totp.pwdMismatch'))
        return
      }
      if (pwdInput.value.length < 4) {
        message.warning(t('totp.pwdTooShort'))
        return
      }
      await TOTPSetupPassword(pwdInput.value)
      message.success(t('totp.pwdSetOk'))
    } else {
      const ok = await TOTPUnlock(pwdInput.value)
      if (!ok) {
        message.error(t('totp.pwdWrong'))
        return
      }
      message.success(t('totp.unlockedOk'))
    }
    pwdVisible.value = false
    unlocked.value = true
    await fetchCodes()
    startTimer()
  } catch (e: unknown) {
    message.error(t('totp.opFailed') + (e instanceof Error ? e.message : String(e)))
  }
}

const fetchCodes = async () => {
  if (!unlocked.value) return
  try {
    const result = await TOTPGenerateAll()
    codes.value = result || []
    if (codes.value.length > 0) {
      remain.value = codes.value[0].remain
    }
  } catch (e: unknown) {
    message.error(t('totp.fetchFailed') + (e instanceof Error ? e.message : String(e)))
  }
}

const startTimer = () => {
  if (timer.value) clearInterval(timer.value)
  timer.value = window.setInterval(async () => {
    remain.value--
    if (remain.value <= 0) {
      await fetchCodes()
    }
  }, 1000)
}

const showAdd = () => {
  addName.value = ''
  addSecret.value = ''
  addIssuer.value = ''
  addVisible.value = true
}

const doAdd = async () => {
  if (!addName.value.trim() || !addSecret.value.trim()) {
    message.warning(t('totp.nameSecretRequired'))
    return
  }
  try {
    await TOTPAddSecret(addName.value.trim(), addSecret.value.trim(), addIssuer.value.trim())
    message.success(t('totp.addedOk'))
    addVisible.value = false
    await fetchCodes()
  } catch (e: unknown) {
    message.error(t('totp.addFailed') + (e instanceof Error ? e.message : String(e)))
  }
}

const doDelete = (name: string) => {
  Modal.confirm({
    title: t('totp.deleteTitle'),
    content: t('totp.deleteContent', { name }),
    okText: t('totp.deleteOk'),
    cancelText: t('port.cancel'),
    okButtonProps: { danger: true },
    onOk: async () => {
      try {
        await TOTPDeleteSecret(name)
        message.success(t('totp.deletedOk'))
        await fetchCodes()
      } catch (e: unknown) {
        message.error(t('totp.deleteFailed') + (e instanceof Error ? e.message : String(e)))
      }
    },
  })
}

const copyCode = (code: string) => {
  navigator.clipboard.writeText(code)
  message.success(t('totp.copied') + code)
}

const remainPercent = () => {
  return (remain.value / 30) * 100
}

onMounted(() => {
  checkAuth()
})

onUnmounted(() => {
  if (timer.value) clearInterval(timer.value)
})
</script>

<template>
  <div class="page">
    <!-- 未解锁时显示锁屏 -->
    <div class="lock-screen" v-if="!unlocked">
      <div class="lock-icon">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
          <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
      </div>
      <p class="lock-text">{{ t('totp.locked') }}</p>
      <p class="lock-tip">{{ t('totp.lockTip') }}</p>
    </div>

    <!-- 已解锁显示列表 -->
    <template v-if="unlocked">
      <div class="toolbar">
        <a-button type="primary" @click="showAdd">{{ t('totp.add') }}</a-button>
        <a-button @click="fetchCodes">{{ t('totp.refresh') }}</a-button>
        <div class="timer" v-if="codes.length > 0">
          <a-progress
            :percent="remainPercent()"
            :show-info="false"
            size="small"
            :stroke-color="remain < 5 ? '#f38ba8' : '#89b4fa'"
            style="width: 60px"
          />
          <span class="remain-text">{{ remain }}s</span>
        </div>
      </div>

      <div class="code-list" v-if="codes.length > 0">
        <div v-for="item in codes" :key="item.name" class="code-card">
          <div class="code-info">
            <div class="code-issuer" v-if="item.issuer">{{ item.issuer }}</div>
            <div class="code-name">{{ item.name }}</div>
          </div>
          <div class="code-value" @click="copyCode(item.code)">{{ item.code }}</div>
          <div class="code-actions">
            <a-button size="small" type="text" danger @click="doDelete(item.name)">{{ t('totp.delete') }}</a-button>
          </div>
        </div>
      </div>

      <div class="empty" v-else>
        <p>{{ t('totp.empty') }}</p>
        <p class="tip">{{ t('totp.tip') }}</p>
      </div>
    </template>

    <!-- 密码弹窗 -->
    <a-modal
      v-model:open="pwdVisible"
      :title="pwdMode === 'setup' ? t('totp.setPwd') : t('totp.inputPwd')"
      :okText="pwdMode === 'setup' ? t('totp.setting') : t('totp.unlock')"
      :cancelText="t('port.cancel')"
      @ok="doPwdSubmit"
      :maskClosable="false"
      :closable="false"
      :keyboard="false"
    >
      <div class="form-field">
        <label>{{ pwdMode === 'setup' ? t('totp.setPwdLabel') : t('totp.pwdLabel') }}</label>
        <a-input-password v-model:value="pwdInput" :placeholder="pwdMode === 'setup' ? t('totp.min4') : t('totp.accessPwd')" @pressEnter="doPwdSubmit" />
      </div>
      <div class="form-field" v-if="pwdMode === 'setup'">
        <label>{{ t('totp.confirmPwd') }}</label>
        <a-input-password v-model:value="pwdConfirm" :placeholder="t('totp.confirmPwdPh')" @pressEnter="doPwdSubmit" />
      </div>
      <div class="pwd-tip" v-if="pwdMode === 'setup'">
        {{ t('totp.pwdTip') }}
      </div>
    </a-modal>

    <!-- 添加密钥弹窗 -->
    <a-modal v-model:open="addVisible" :title="t('totp.addTitle')" @ok="doAdd" :okText="t('totp.addOk')" :cancelText="t('port.cancel')">
      <div class="form-field">
        <label>{{ t('totp.nameLabel') }}</label>
        <a-input v-model:value="addName" :placeholder="t('totp.namePh')" />
      </div>
      <div class="form-field">
        <label>{{ t('totp.issuerLabel') }}</label>
        <a-input v-model:value="addIssuer" :placeholder="t('totp.issuerPh')" />
      </div>
      <div class="form-field">
        <label>{{ t('totp.secretLabel') }}</label>
        <a-input v-model:value="addSecret" :placeholder="t('totp.secretPh')" />
      </div>
    </a-modal>
  </div>
</template>

<style scoped>
.page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.lock-screen {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #6c7086;
}

.lock-icon {
  color: #585b70;
  margin-bottom: 16px;
}

.lock-text {
  font-size: 16px;
  color: #a6adc8;
  margin-bottom: 4px;
}

.lock-tip {
  font-size: 12px;
  color: #45475a;
}

.toolbar {
  display: flex;
  gap: 8px;
  padding: 0 0 12px 0;
  align-items: center;
}

.timer {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
}

.remain-text {
  font-size: 12px;
  color: #a6adc8;
  min-width: 28px;
}

.code-list {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.code-card {
  display: grid;
  grid-template-columns: 1fr 200px 60px;
  align-items: center;
  padding: 18px 16px;
  background: #181825;
  border: 1px solid #313244;
  border-radius: 8px;
  transition: border-color 0.15s;
}

.code-card:hover {
  border-color: #45475a;
}

.code-info {
  overflow: hidden;
}

.code-issuer {
  font-size: 11px;
  color: #6c7086;
  margin-bottom: 2px;
}

.code-name {
  font-size: 14px;
  font-weight: 500;
  color: #cdd6f4;
}

.code-value {
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  font-size: 28px;
  font-weight: 700;
  line-height: 1.8;
  color: #a6e3a1;
  letter-spacing: 4px;
  cursor: pointer;
  text-align: center;
  user-select: all;
  transition: color 0.15s;
}

.code-value:hover {
  color: #b4befe;
}

.code-actions {
  text-align: right;
}

.empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #6c7086;
}

.empty p {
  margin: 4px 0;
}

.empty .tip {
  font-size: 12px;
  color: #45475a;
}

.form-field {
  margin-bottom: 12px;
}

.form-field label {
  display: block;
  font-size: 12px;
  color: #a6adc8;
  margin-bottom: 4px;
}

.pwd-tip {
  font-size: 11px;
  color: #6c7086;
  margin-top: 8px;
}

:deep(.ant-progress-inner) {
  background: #313244;
}
</style>
