<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { ScanPorts, KillPid, KillPort } from '../../wailsjs/go/main/App'
import { t } from '../i18n'
import type { main } from '../../wailsjs/go/models'

type PortInfo = main.PortInfo

const columns = computed(() => [
  { title: t('port.colPort'), dataIndex: 'port', key: 'port', width: 100 },
  { title: t('port.colPid'), dataIndex: 'pid', key: 'pid', width: 90 },
  { title: t('port.colProc'), dataIndex: 'procName', key: 'procName', width: 130 },
  { title: t('port.colProject'), dataIndex: 'project', key: 'project' },
])

const dataSource = ref<PortInfo[]>([])
const loading = ref(false)
const selectedRowKeys = ref<number[]>([])
const selectedRow = ref<PortInfo | null>(null)
const manualPort = ref<number | null>(null)

// 杀死失败时弹出提醒（含权限不足提示）
const showKillError = (msg: string) => {
  Modal.warning({
    title: t('port.errorTitle'),
    content: msg,
    okText: t('port.errorOk'),
  })
}

const fetchPorts = async () => {
  loading.value = true
  try {
    const result = await ScanPorts()
    dataSource.value = result || []
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    message.error(t('port.scanFailed') + msg)
  } finally {
    loading.value = false
  }
}

const killProcess = () => {
  if (!selectedRow.value) {
    message.warning(t('port.selectFirst'))
    return
  }
  Modal.confirm({
    title: t('port.killConfirmTitle'),
    content: t('port.killConfirmContent', { pid: selectedRow.value.pid, proc: selectedRow.value.procName }),
    okText: t('port.killOk'),
    cancelText: t('port.cancel'),
    okButtonProps: { danger: true },
    onOk: async () => {
      try {
        await KillPid(selectedRow.value!.pid)
        message.success(t('port.killed', { pid: selectedRow.value!.pid }))
      } catch (e: unknown) {
        showKillError(e instanceof Error ? e.message : String(e))
      } finally {
        selectedRowKeys.value = []
        selectedRow.value = null
        setTimeout(() => fetchPorts(), 500)
      }
    },
  })
}

const killByPort = () => {
  const port = manualPort.value
  if (!port || port < 1 || port > 65535) {
    message.warning(t('port.invalidPort'))
    return
  }
  Modal.confirm({
    title: t('port.killPortTitle'),
    content: t('port.killPortContent', { port }),
    okText: t('port.killOk'),
    cancelText: t('port.cancel'),
    okButtonProps: { danger: true },
    onOk: async () => {
      try {
        await KillPort(port)
        message.success(t('port.portKilled', { port }))
        manualPort.value = null
      } catch (e: unknown) {
        showKillError(e instanceof Error ? e.message : String(e))
      } finally {
        setTimeout(() => fetchPorts(), 500)
      }
    },
  })
}

const onSelectChange = (keys: (string | number)[], rows: PortInfo[]) => {
  selectedRowKeys.value = keys as number[]
  selectedRow.value = rows.length > 0 ? rows[0] : null
}

onMounted(() => {
  fetchPorts()
})
</script>

<template>
  <div class="page">
    <div class="toolbar">
      <a-button type="primary" :loading="loading" @click="fetchPorts">{{ t('port.scan') }}</a-button>
      <a-button danger @click="killProcess">{{ t('port.kill') }}</a-button>
      <div class="port-kill">
        <a-input-number
          v-model:value="manualPort"
          :placeholder="t('port.portPlaceholder')"
          :min="1"
          :max="65535"
          :precision="0"
          :controls="false"
          class="port-input"
          @pressEnter="killByPort"
        />
        <a-button danger @click="killByPort">{{ t('port.killByPort') }}</a-button>
      </div>
    </div>
    <div class="table-wrap">
      <a-table
        :dataSource="dataSource"
        :columns="columns"
        :loading="loading"
        :pagination="false"
        :rowKey="(record: PortInfo) => record.pid + '-' + record.port"
        size="middle"
        :rowSelection="{
          type: 'radio',
          selectedRowKeys,
          onChange: onSelectChange,
        }"
        :scroll="{ y: 'calc(100vh - 180px)' }"
      />
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

.port-kill {
  display: flex;
  gap: 8px;
  margin-left: 16px;
  align-items: center;
}

.port-input {
  width: 140px;
}

.table-wrap {
  flex: 1;
  overflow: hidden;
}
</style>
