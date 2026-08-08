<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { ScanPorts, KillPid, IsAdmin } from '../../wailsjs/go/main/App'
import type { PortInfo } from '../../wailsjs/go/main/App'

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
        const msg = e instanceof Error ? e.message : String(e)
        message.error('杀死失败: ' + msg)
      } finally {
        selectedRowKeys.value = []
        selectedRow.value = null
        setTimeout(() => fetchPorts(), 500)
      }
    },
  })
}

const onSelectChange = (keys: (string | number)[], rows: PortInfo[]) => {
  selectedRowKeys.value = keys as number[]
  selectedRow.value = rows.length > 0 ? rows[0] : null
}

onMounted(async () => {
  try {
    const admin = await IsAdmin()
    if (!admin) {
      Modal.warning({
        title: '权限不足',
        content: '未以管理员身份运行，部分进程可能无法杀死。建议以管理员身份重新运行。',
        okText: '知道了',
      })
    }
  } catch {
    // ignore
  }
  await fetchPorts()
})
</script>

<template>
  <div class="page">
    <div class="toolbar">
      <a-button type="primary" :loading="loading" @click="fetchPorts">刷新扫描</a-button>
      <a-button danger :disabled="!selectedRow" @click="killProcess">强制杀死</a-button>
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
}

.table-wrap {
  flex: 1;
  overflow: hidden;
}
</style>
