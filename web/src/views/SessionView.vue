<template>
  <div class="session">
    <div v-if="!session" class="loading">加载中...</div>

    <div v-else class="session-content">
      <div class="session-header">
        <div>
          <h2>任务会话: {{ session.ID.slice(-8) }}</h2>
          <small class="repo-path">{{ session.RepoPath }}</small>
        </div>
        <span class="status" :class="statusClass">{{ session.Status }}</span>
      </div>

      <div class="card">
        <h3>用户任务</h3>
        <p class="task-description">{{ session.UserTask }}</p>
      </div>

      <!-- Task Actions -->
      <div class="actions">
        <button
          v-if="session.Status === 'created'"
          @click="handleDecompose"
          class="btn"
          :disabled="loading"
        >
          {{ loading ? '分析中...' : '分解任务' }}
        </button>

        <button
          v-if="session.Status === 'ready'"
          @click="handleExecute"
          class="btn btn-primary"
          :disabled="loading"
        >
          {{ loading ? '执行中...' : '开始执行' }}
        </button>

        <button
          v-if="session.Status === 'merging'"
          @click="handleMerge"
          class="btn btn-primary"
          :disabled="loading"
        >
          {{ loading ? '合并中...' : '合并结果' }}
        </button>

        <button
          v-if="session.Status === 'completed'"
          class="btn btn-success"
          disabled
        >
          完成 ✓
        </button>

        <button
          v-if="session.Status === 'running'"
          @click="refreshTasks"
          class="btn-small"
        >
          刷新状态
        </button>
      </div>

      <!-- Tasks DAG -->
      <div v-if="tasks.length > 0" class="tasks-section">
        <h3>子任务 ({{ tasks.length }})</h3>
        <div class="task-list">
          <div
            v-for="task in tasks"
            :key="task.ID"
            class="task-item"
            :class="getTaskStatusClass(task.Status)"
            @click="selectedTask = task"
          >
            <div class="task-header">
              <span class="task-status">{{ getTaskStatusIcon(task.Status) }}</span>
              <span class="task-title">{{ task.Title || task.ID }}</span>
              <span v-if="task.BranchName" class="task-branch">{{ task.BranchName }}</span>
            </div>
            <p v-if="task.Description" class="task-desc">{{ task.Description }}</p>
            <div v-if="task.DependsOn && task.DependsOn.length" class="task-deps">
              依赖: {{ task.DependsOn.join(', ') }}
            </div>
          </div>
        </div>
      </div>

      <!-- Agent Output Panel -->
      <div class="output-section">
        <div class="output-header">
          <h3>Agent 输出</h3>
          <button v-if="agentLogs.length > 0" @click="clearLogs" class="btn-small">清空</button>
        </div>

        <div v-if="agentLogs.length === 0" class="output-empty">
          暂无输出，等待任务执行...
        </div>

        <div v-else class="output-content">
          <div
            v-for="(log, idx) in agentLogs"
            :key="idx"
            class="log-entry"
            :class="`log-${log.level}`"
          >
            <span class="log-time">{{ formatLogTime(log.time) }}</span>
            <span class="log-agent">{{ log.agent }}</span>
            <span class="log-message">{{ log.message }}</span>
          </div>
        </div>
      </div>

      <!-- Raw WebSocket Debug -->
      <details v-if="wsData" class="debug-section">
        <summary>WebSocket 数据 (调试)</summary>
        <pre class="output-content">{{ JSON.stringify(wsData, null, 2) }}</pre>
      </details>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useWebSocket } from '../composables/useWebSocket'

const route = useRoute()

const session = ref(null)
const tasks = ref([])
const loading = ref(false)
const error = ref(null)
const selectedTask = ref(null)
const agentLogs = ref([])
let refreshInterval = null

const { connected, data: wsData, connect } = useWebSocket(route.params.id)

const statusClass = computed(() => {
  const status = session.value?.Status || ''
  return `status-${status}`
})

async function loadSession() {
  try {
    const res = await fetch(`/api/sessions/${route.params.id}`)
    if (res.ok) {
      session.value = await res.json()
    }
    const taskRes = await fetch(`/api/sessions/${route.params.id}/tasks`)
    if (taskRes.ok) {
      tasks.value = await taskRes.json() || []
    }
  } catch (e) {
    error.value = e.message
  }
}

async function handleDecompose() {
  loading.value = true
  try {
    const res = await fetch(`/api/sessions/${route.params.id}/decompose`, { method: 'POST' })
    if (!res.ok) throw new Error('Decompose failed')
    await loadSession()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function handleExecute() {
  loading.value = true
  try {
    const res = await fetch(`/api/sessions/${route.params.id}/execute`, { method: 'POST' })
    if (!res.ok) throw new Error('Execute failed')

    // Start polling for task updates
    startPolling()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function handleMerge() {
  loading.value = true
  try {
    const res = await fetch(`/api/sessions/${route.params.id}/merge`, { method: 'POST' })
    if (!res.ok) throw new Error('Merge failed')
    await loadSession()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function refreshTasks() {
  await loadSession()
}

function startPolling() {
  if (refreshInterval) clearInterval(refreshInterval)
  refreshInterval = setInterval(() => {
    loadSession()
  }, 2000)
}

function stopPolling() {
  if (refreshInterval) {
    clearInterval(refreshInterval)
    refreshInterval = null
  }
}

function addLog(level, agent, message) {
  agentLogs.value.push({
    time: new Date(),
    level,
    agent,
    message
  })
  // Keep only last 100 logs
  if (agentLogs.value.length > 100) {
    agentLogs.value = agentLogs.value.slice(-100)
  }
}

function clearLogs() {
  agentLogs.value = []
}

function formatLogTime(date) {
  return date.toLocaleTimeString('zh-CN')
}

function getTaskStatusClass(status) {
  return `task-${status}`
}

function getTaskStatusIcon(status) {
  const icons = {
    pending: '⏳',
    running: '▶️',
    completed: '✅',
    failed: '❌',
    cancelled: '⏹️'
  }
  return icons[status] || '⏳'
}

// Watch WebSocket data
watch(wsData, (newData) => {
  if (!newData) return

  switch (newData.type) {
    case 'session.created':
      addLog('info', 'system', '会话已创建')
      break
    case 'session.decomposed':
      tasks.value = newData.data.tasks || []
      addLog('info', 'orchestrator', `任务分解完成，生成 ${tasks.value.length} 个子任务`)
      loadSession()
      break
    case 'session.executing':
      addLog('info', 'executor', '开始执行任务')
      startPolling()
      if (session.value) session.value.Status = 'running'
      break
    case 'session.error':
      addLog('error', 'system', newData.data.error || '发生错误')
      stopPolling()
      break
    case 'session.merged':
      addLog('info', 'merger', '合并完成')
      stopPolling()
      if (session.value) session.value.Status = 'completed'
      break
    case 'task.started':
      addLog('info', newData.data.agent || 'agent', `开始任务: ${newData.data.task}`)
      break
    case 'task.completed':
      addLog('success', newData.data.agent || 'agent', `完成任务: ${newData.data.task}`)
      loadSession()
      break
    case 'task.failed':
      addLog('error', newData.data.agent || 'agent', `任务失败: ${newData.data.task}`)
      stopPolling()
      loadSession()
      break
  }
})

// Watch tasks for status changes
watch(tasks, (newTasks, oldTasks) => {
  if (!oldTasks || oldTasks.length === 0) return

  newTasks.forEach((task, idx) => {
    const oldTask = oldTasks[idx]
    if (!oldTask) return

    if (task.Status !== oldTask.Status) {
      switch (task.Status) {
        case 'running':
          addLog('info', `worker-${task.ID}`, `开始执行: ${task.Title || task.ID}`)
          break
        case 'completed':
          addLog('success', `worker-${task.ID}`, `完成: ${task.Title || task.ID}`)
          break
        case 'failed':
          addLog('error', `worker-${task.ID}`, `失败: ${task.Title || task.ID}`)
          break
      }
    }
  })
}, { deep: true })

onMounted(() => {
  loadSession()
  connect()
})

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.session-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.session-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.session-header h2 {
  margin: 0;
}

.repo-path {
  display: block;
  color: #8b949e;
  font-family: monospace;
  margin-top: 0.25rem;
}

.status {
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.875rem;
  font-weight: 600;
  flex-shrink: 0;
}

.status-created, .status-ready {
  background: #1f6feb33;
  color: #58a6ff;
}

.status-running {
  background: #d2992222;
  color: #e3b341;
}

.status-completed {
  background: #23863633;
  color: #3fb950;
}

.status-failed {
  background: #f8514933;
  color: #f85149;
}

.card {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 8px;
  padding: 1.5rem;
}

.card h3 {
  margin-top: 0;
  color: #58a6ff;
}

.task-description {
  color: #8b949e;
  line-height: 1.6;
}

.actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.btn {
  background: #238636;
  color: white;
  border: none;
  border-radius: 6px;
  padding: 0.75rem 1.5rem;
  font-size: 1rem;
  cursor: pointer;
  transition: background 0.2s;
}

.btn:hover:not(:disabled) {
  background: #2ea043;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: #1f6feb;
}

.btn-primary:hover:not(:disabled) {
  background: #58a6ff;
}

.btn-success {
  background: #238636;
}

.btn-small {
  background: #21262d;
  color: #58a6ff;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 0.5rem 1rem;
  cursor: pointer;
}

.tasks-section h3 {
  margin-bottom: 1rem;
  color: #58a6ff;
}

.task-list {
  display: grid;
  gap: 0.75rem;
}

.task-item {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 1rem;
  cursor: pointer;
  transition: border-color 0.2s;
}

.task-item:hover {
  border-color: #58a6ff;
}

.task-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.task-status {
  font-size: 1.25rem;
}

.task-title {
  font-weight: 600;
}

.task-branch {
  font-family: monospace;
  font-size: 0.75rem;
  color: #58a6ff;
  background: #1f6feb22;
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
}

.task-desc {
  margin: 0.5rem 0;
  color: #8b949e;
}

.task-deps {
  font-size: 0.875rem;
  color: #58a6ff;
}

.task-pending {
  border-left: 3px solid #8b949e;
}

.task-running {
  border-left: 3px solid #e3b341;
  animation: pulse 1.5s infinite;
}

.task-completed {
  border-left: 3px solid #3fb950;
}

.task-failed {
  border-left: 3px solid #f85149;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.output-section {
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 8px;
  padding: 1rem;
}

.output-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.output-header h3 {
  margin: 0;
  color: #58a6ff;
}

.output-empty {
  text-align: center;
  padding: 2rem;
  color: #484f58;
}

.output-content {
  max-height: 400px;
  overflow-y: auto;
  font-family: monospace;
  font-size: 0.875rem;
}

.log-entry {
  padding: 0.5rem;
  border-bottom: 1px solid #21262d;
  display: flex;
  gap: 0.75rem;
  align-items: flex-start;
}

.log-time {
  color: #484f58;
  flex-shrink: 0;
}

.log-agent {
  color: #58a6ff;
  flex-shrink: 0;
  min-width: 100px;
}

.log-message {
  color: #c9d1d9;
  word-break: break-word;
}

.log-info .log-message {
  color: #c9d1d9;
}

.log-success .log-message {
  color: #3fb950;
}

.log-error .log-message {
  color: #f85149;
}

.debug-section {
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 8px;
  padding: 1rem;
}

.debug-section summary {
  cursor: pointer;
  color: #8b949e;
}

.debug-section summary:hover {
  color: #58a6ff;
}

.loading {
  text-align: center;
  padding: 2rem;
  color: #8b949e;
}
</style>
