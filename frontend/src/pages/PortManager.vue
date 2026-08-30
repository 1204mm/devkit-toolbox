<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { ScanPorts, KillPid, KillPort } from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'

type PortInfo = main.PortInfo

const columns = [
  { title: '端口', dataIndex: 'port', key: 'port', width: 100 },
  { title: 'PID', dataIndex: 'pid', key: 'pid', width: 90 },
  { title: '进程', dataIndex: 'procName', key: 'procName', width: 130 },
  { title: '项目识别', dataIndex: 'project', key: 'project' },
]

const dataSource = ref<PortInfo[]>([])
const loading = ref(false)
const selectedRowKeys = ref<number[]>([])
const selectedRow = ref<PortInfo | null>(null)
const manualPort = ref<number | null>(null)

// 杀死失败时弹出提醒（含权限不足提示）
const showKillError = (msg: string) => {
  Modal.warning({
    title: '杀死失败',
    content: msg,
    okText: '知道了',
  })
}

const fetchPorts = async () => {
  loading.value = true
  try {
    const result = await ScanPorts()
    dataSource.value = result || []
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    message.error('扫描失败: ' + msg)
  } finally {
    loading.value = false
  }
}

const killProcess = () => {
  if (!selectedRow.value) {
    message.warning('请先选择一行')
    return
  }
  Modal.confirm({
    title: '确认杀死进程',
    content: `确定要杀死 PID ${selectedRow.value.pid}（${selectedRow.value.procName}）吗？`,
    okText: '杀死',
    cancelText: '取消',
    okButtonProps: { danger: true },
    onOk: async () => {
      try {
        await KillPid(selectedRow.value!.pid)
        message.success(`PID ${selectedRow.value!.pid} 已杀死`)
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
    message.warning('请输入有效的端口号（1-65535）')
    return
  }
  Modal.confirm({
    title: '确认按端口杀死',
    content: `确定要杀死占用端口 ${port} 的进程吗？`,
    okText: '杀死',
    cancelText: '取消',
    okButtonProps: { danger: true },
    onOk: async () => {
      try {
        await KillPort(port)
        message.success(`端口 ${port} 的进程已杀死`)
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
      <a-button type="primary" :loading="loading" @click="fetchPorts">刷新扫描</a-button>
      <a-button danger @click="killProcess">强制杀死</a-button>
      <div class="port-kill">
        <a-input-number
          v-model:value="manualPort"
          placeholder="输入端口号"
          :min="1"
          :max="65535"
          :precision="0"
          :controls="false"
          class="port-input"
          @pressEnter="killByPort"
        />
        <a-button danger @click="killByPort">按端口杀死</a-button>
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
