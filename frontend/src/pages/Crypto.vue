<script setup lang="ts">
import { ref } from 'vue'
import { message } from 'ant-design-vue'
import {
  Hash, Hmac,
  BcryptHash, BcryptCompare,
  AESEncrypt, AESDecrypt,
  DesEncrypt, DesDecrypt,
  RSAGenerateKey, RSAEncrypt, RSADecrypt,
  Base64Encode, Base64Decode, HexEncode, HexDecode,
  URLEncode, URLDecode,
} from '../../wailsjs/go/main/App'

const activeTab = ref('hash')

// 通用复制
const copyText = (text: string) => {
  navigator.clipboard.writeText(text)
  message.success('已复制')
}

// ========== 哈希 ==========
const hashInput = ref('')
const hashResult = ref('')
const hashAlgo = ref('MD5')

const doHash = async () => {
  if (!hashInput.value) { message.warning('请输入内容'); return }
  try {
    hashResult.value = await Hash(hashAlgo.value, hashInput.value)
  } catch (e: unknown) {
    message.error('计算失败: ' + (e instanceof Error ? e.message : String(e)))
  }
}

// ========== HMAC ==========
const hmacInput = ref('')
const hmacKey = ref('')
const hmacResult = ref('')
const hmacAlgo = ref('SHA256')

const doHmac = async () => {
  if (!hmacInput.value || !hmacKey.value) { message.warning('请输入内容和密钥'); return }
  try {
    hmacResult.value = await Hmac(hmacAlgo.value, hmacInput.value, hmacKey.value)
  } catch (e: unknown) {
    message.error('计算失败: ' + (e instanceof Error ? e.message : String(e)))
  }
}

// ========== Bcrypt ==========
const bcryptPassword = ref('')
const bcryptHashed = ref('')
const bcryptCost = ref(10)
const bcryptResult = ref('')

const doBcryptHash = async () => {
  if (!bcryptPassword.value) { message.warning('请输入密码'); return }
  try {
    bcryptResult.value = await BcryptHash(bcryptPassword.value, bcryptCost.value)
  } catch (e: unknown) {
    message.error('加密失败: ' + (e instanceof Error ? e.message : String(e)))
  }
}

const doBcryptCompare = async () => {
  if (!bcryptPassword.value || !bcryptHashed.value) { message.warning('请输入密码和哈希值'); return }
  try {
    const ok = await BcryptCompare(bcryptPassword.value, bcryptHashed.value)
    bcryptResult.value = ok ? '匹配: 密码正确' : '不匹配: 密码错误'
  } catch (e: unknown) {
    message.error('校验失败: ' + (e instanceof Error ? e.message : String(e)))
  }
}

// ========== AES ==========
const aesInput = ref('')
const aesKey = ref('')
const aesResult = ref('')

const doAESEncrypt = async () => {
  if (!aesInput.value || !aesKey.value) { message.warning('请输入内容和密钥'); return }
  try {
    aesResult.value = await AESEncrypt(aesInput.value, aesKey.value)
  } catch (e: unknown) {
    message.error('加密失败: ' + (e instanceof Error ? e.message : String(e)))
  }
}

const doAESDecrypt = async () => {
  if (!aesInput.value || !aesKey.value) { message.warning('请输入内容和密钥'); return }
  try {
    aesResult.value = await AESDecrypt(aesInput.value, aesKey.value)
  } catch (e: unknown) {
    message.error('解密失败: ' + (e instanceof Error ? e.message : String(e)))
  }
}

// ========== DES ==========
const desInput = ref('')
const desKey = ref('')
const desResult = ref('')

const doDesEncrypt = async () => {
  if (!desInput.value || !desKey.value) { message.warning('请输入内容和密钥'); return }
  try {
    desResult.value = await DesEncrypt(desInput.value, desKey.value)
  } catch (e: unknown) {
    message.error('加密失败: ' + (e instanceof Error ? e.message : String(e)))
  }
}

const doDesDecrypt = async () => {
  if (!desInput.value || !desKey.value) { message.warning('请输入内容和密钥'); return }
  try {
    desResult.value = await DesDecrypt(desInput.value, desKey.value)
  } catch (e: unknown) {
    message.error('解密失败: ' + (e instanceof Error ? e.message : String(e)))
  }
}

// ========== RSA ==========
const rsaInput = ref('')
const rsaKey = ref('')
const rsaResult = ref('')

const doRSAGenKey = async () => {
  try {
    rsaResult.value = await RSAGenerateKey()
    message.success('密钥对已生成')
  } catch (e: unknown) {
    message.error('生成失败: ' + (e instanceof Error ? e.message : String(e)))
  }
}

const doRSAEncrypt = async () => {
  if (!rsaInput.value || !rsaKey.value) { message.warning('请输入明文和公钥PEM'); return }
  try {
    rsaResult.value = await RSAEncrypt(rsaInput.value, rsaKey.value)
  } catch (e: unknown) {
    message.error('加密失败: ' + (e instanceof Error ? e.message : String(e)))
  }
}

const doRSADecrypt = async () => {
  if (!rsaInput.value || !rsaKey.value) { message.warning('请输入Base64密文和私钥PEM'); return }
  try {
    rsaResult.value = await RSADecrypt(rsaInput.value, rsaKey.value)
  } catch (e: unknown) {
    message.error('解密失败: ' + (e instanceof Error ? e.message : String(e)))
  }
}

// ========== 编解码 ==========
const codecInput = ref('')
const codecResult = ref('')
const codecMode = ref('base64')

const doEncode = async () => {
  if (!codecInput.value) { message.warning('请输入内容'); return }
  try {
    switch (codecMode.value) {
      case 'base64': codecResult.value = await Base64Encode(codecInput.value); break
      case 'hex': codecResult.value = await HexEncode(codecInput.value); break
      case 'url': codecResult.value = await URLEncode(codecInput.value); break
    }
  } catch (e: unknown) {
    message.error('编码失败: ' + (e instanceof Error ? e.message : String(e)))
  }
}

const doDecode = async () => {
  if (!codecInput.value) { message.warning('请输入内容'); return }
  try {
    switch (codecMode.value) {
      case 'base64': codecResult.value = await Base64Decode(codecInput.value); break
      case 'hex': codecResult.value = await HexDecode(codecInput.value); break
      case 'url': codecResult.value = await URLDecode(codecInput.value); break
    }
  } catch (e: unknown) {
    message.error('解码失败: ' + (e instanceof Error ? e.message : String(e)))
  }
}
</script>

<template>
  <div class="page">
    <a-tabs v-model:activeKey="activeTab" size="small">

      <!-- 哈希 -->
      <a-tab-pane key="hash" tab="哈希">
        <div class="form-row">
          <a-select v-model:value="hashAlgo" style="width: 120px">
            <a-select-option value="MD5">MD5</a-select-option>
            <a-select-option value="SHA1">SHA1</a-select-option>
            <a-select-option value="SHA256">SHA256</a-select-option>
            <a-select-option value="SHA512">SHA512</a-select-option>
          </a-select>
          <a-button type="primary" @click="doHash">计算</a-button>
        </div>
        <a-textarea v-model:value="hashInput" placeholder="输入要计算哈希的文本" :rows="5" class="input-area" />
        <div class="result-wrap" v-if="hashResult">
          <div class="result-label"><span>结果</span><a-button size="small" type="link" @click="copyText(hashResult)">复制</a-button></div>
          <pre class="result-box">{{ hashResult }}</pre>
        </div>
      </a-tab-pane>

      <!-- HMAC -->
      <a-tab-pane key="hmac" tab="HMAC">
        <div class="form-row">
          <a-select v-model:value="hmacAlgo" style="width: 140px">
            <a-select-option value="SHA256">HMAC-SHA256</a-select-option>
            <a-select-option value="SHA512">HMAC-SHA512</a-select-option>
            <a-select-option value="MD5">HMAC-MD5</a-select-option>
          </a-select>
          <a-button type="primary" @click="doHmac">计算</a-button>
        </div>
        <a-input v-model:value="hmacKey" placeholder="密钥" class="input-area" />
        <a-textarea v-model:value="hmacInput" placeholder="输入要计算HMAC的文本" :rows="4" class="input-area" />
        <div class="result-wrap" v-if="hmacResult">
          <div class="result-label"><span>结果</span><a-button size="small" type="link" @click="copyText(hmacResult)">复制</a-button></div>
          <pre class="result-box">{{ hmacResult }}</pre>
        </div>
      </a-tab-pane>

      <!-- Bcrypt -->
      <a-tab-pane key="bcrypt" tab="Bcrypt">
        <div class="form-row">
          <span class="form-label">强度：</span>
          <a-input-number v-model:value="bcryptCost" :min="4" :max="31" style="width: 80px" />
          <a-button type="primary" @click="doBcryptHash">加密</a-button>
          <a-button @click="doBcryptCompare">校验</a-button>
        </div>
        <a-input-password v-model:value="bcryptPassword" placeholder="密码" class="input-area" />
        <a-input v-model:value="bcryptHashed" placeholder="bcrypt 哈希值（校验时填写）" class="input-area" />
        <div class="result-wrap" v-if="bcryptResult">
          <div class="result-label"><span>结果</span><a-button size="small" type="link" @click="copyText(bcryptResult)">复制</a-button></div>
          <pre class="result-box">{{ bcryptResult }}</pre>
        </div>
      </a-tab-pane>

      <!-- AES -->
      <a-tab-pane key="aes" tab="AES">
        <div class="form-row">
          <a-button type="primary" @click="doAESEncrypt">加密</a-button>
          <a-button @click="doAESDecrypt">解密</a-button>
        </div>
        <a-input-password v-model:value="aesKey" placeholder="密钥（任意长度）" class="input-area" />
        <a-textarea v-model:value="aesInput" placeholder="加密输入明文 / 解密输入Base64密文" :rows="4" class="input-area" />
        <div class="result-wrap" v-if="aesResult">
          <div class="result-label"><span>结果</span><a-button size="small" type="link" @click="copyText(aesResult)">复制</a-button></div>
          <pre class="result-box">{{ aesResult }}</pre>
        </div>
      </a-tab-pane>

      <!-- DES -->
      <a-tab-pane key="des" tab="DES">
        <div class="form-row">
          <a-button type="primary" @click="doDesEncrypt">加密</a-button>
          <a-button @click="doDesDecrypt">解密</a-button>
        </div>
        <a-input-password v-model:value="desKey" placeholder="密钥（至少8位，不足自动补0）" class="input-area" />
        <a-textarea v-model:value="desInput" placeholder="加密输入明文 / 解密输入Base64密文" :rows="4" class="input-area" />
        <div class="result-wrap" v-if="desResult">
          <div class="result-label"><span>结果</span><a-button size="small" type="link" @click="copyText(desResult)">复制</a-button></div>
          <pre class="result-box">{{ desResult }}</pre>
        </div>
      </a-tab-pane>

      <!-- RSA -->
      <a-tab-pane key="rsa" tab="RSA">
        <div class="form-row">
          <a-button @click="doRSAGenKey">生成密钥对</a-button>
          <a-button type="primary" @click="doRSAEncrypt">公钥加密</a-button>
          <a-button @click="doRSADecrypt">私钥解密</a-button>
        </div>
        <a-textarea v-model:value="rsaKey" placeholder="公钥PEM（加密时）或 私钥PEM（解密时）" :rows="4" class="input-area" />
        <a-textarea v-model:value="rsaInput" placeholder="加密输入明文 / 解密输入Base64密文" :rows="3" class="input-area" />
        <div class="result-wrap" v-if="rsaResult">
          <div class="result-label"><span>结果</span><a-button size="small" type="link" @click="copyText(rsaResult)">复制</a-button></div>
          <pre class="result-box">{{ rsaResult }}</pre>
        </div>
      </a-tab-pane>

      <!-- 编解码 -->
      <a-tab-pane key="codec" tab="编解码">
        <div class="form-row">
          <a-radio-group v-model:value="codecMode">
            <a-radio-button value="base64">Base64</a-radio-button>
            <a-radio-button value="hex">Hex</a-radio-button>
            <a-radio-button value="url">URL</a-radio-button>
          </a-radio-group>
          <a-button type="primary" @click="doEncode">编码</a-button>
          <a-button @click="doDecode">解码</a-button>
        </div>
        <a-textarea v-model:value="codecInput" placeholder="输入要编码/解码的文本" :rows="5" class="input-area" />
        <div class="result-wrap" v-if="codecResult">
          <div class="result-label"><span>结果</span><a-button size="small" type="link" @click="copyText(codecResult)">复制</a-button></div>
          <pre class="result-box">{{ codecResult }}</pre>
        </div>
      </a-tab-pane>

    </a-tabs>
  </div>
</template>

<style scoped>
.page {
  height: 100%;
  overflow-y: auto;
  padding: 0 4px;
}

.form-row {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  align-items: center;
}

.form-label {
  color: #a6adc8;
  font-size: 13px;
  white-space: nowrap;
}

.input-area {
  margin-bottom: 12px;
}

.result-wrap {
  margin-top: 4px;
}

.result-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
  color: #a6adc8;
  font-size: 12px;
}

.result-box {
  background: #11111b;
  border: 1px solid #313244;
  border-radius: 6px;
  padding: 12px;
  font-size: 13px;
  color: #a6e3a1;
  word-break: break-all;
  white-space: pre-wrap;
  max-height: 240px;
  overflow-y: auto;
}
</style>
