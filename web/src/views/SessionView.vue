<template>
  <div class="session">
    <div v-if="!session" class="loading">加载中...</div>

    <div v-else class="session-content">
      <div class="session-header">
        <h2>任务会话: {{ session.ID.slice(-8) }}</h2>
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
      </div>

      <!-- Tasks DAG -->
      <div v-if="tasks.length > 0" class="tasks">
        <h3>子任务 ({{ tasks.length }})</h3>
        <div class="task-list">
          <div
            v-for="task in tasks"
            :key="task.ID"
            class="task-item"
            :class="getTaskStatusClass(task.Status)"
          >
            <div class="task-header">
              <span class="task-status">{{ getTaskStatusIcon(task.Status) }}</span>
              <span class="task-title">{{ task.Title || task.ID }}</span>
            </div>
            <p v-if="task.Description" class="task-desc">{{ task.Description }}</p>
            <div v-if="task.DependsOn && task.DependsOn.length" class="task-deps">
              依赖: {{ task.DependsOn.join(', ') }}
            </div>
          </div>
        </div>
      </div>

      <!-- Agent Output -->
      <div v-if="wsData" class="output">
        <h3>实时输出</h3>
        <pre class="output-content">{{ JSON.stringify(wsData, null, 2) }}</pre>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useSessionStore } from '../stores/session'
import { useWebSocket } from '../composables/useWebSocket'

const route = useRoute()
const sessionStore = useSessionStore()

const session = ref(null)
const tasks = ref([])
const loading = ref(false)
const error = ref(null)

const { connected, data: wsData, connect } = useWebSocket(route.params.id)

const statusClass = computed(() => {
  const status = session.value?.Status || ''
  return `status-${status}`
})

async function loadSession() {
  try {
    session.value = await sessionStore.getSession(route.params.id)
    const taskData = await sessionStore.getTasks(route.params.id)
    tasks.value = taskData || []
  } catch (e) {
    error.value = e.message
  }
}

async function handleDecompose() {
  loading.value = true
  try {
    await sessionStore.decompose(route.params.id)
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
    await sessionStore.execute(route.params.id)
    await loadSession()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function handleMerge() {
  loading.value = true
  try {
    await sessionStore.merge(route.params.id)
    await loadSession()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
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

// Watch WebSocket data for task updates
watch(wsData, (newData) => {
  if (newData?.type === 'session.decomposed') {
    tasks.value = newData.data.tasks || []
  } else if (newData?.type === 'session.executing') {
    if (session.value) session.value.Status = 'running'
    // Reload tasks periodically
    setInterval(() => loadSession(), 2000)
  } else if (newData?.type === 'session.merged') {
    if (session.value) session.value.Status = 'completed'
  }
})

onMounted(() => {
  loadSession()
  connect()
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
  align-items: center;
  justify-content: space-between;
}

.session-header h2 {
  margin: 0;
}

.status {
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.875rem;
  font-weight: 600;
}

.status-created, status-ready {
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
  gap: 1rem;
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

.tasks h3 {
  margin-bottom: 1rem;
  color: #58a6ff;
}

.task-list {
  display: grid;
  gap: 1rem;
}

.task-item {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 1rem;
}

.task-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.task-status {
  font-size: 1.25rem;
}

.task-title {
  font-weight: 600;
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
}

.task-completed {
  border-left: 3px solid #3fb950;
}

.task-failed {
  border-left: 3px solid #f85149;
}

.output {
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 8px;
  padding: 1.5rem;
}

.output h3 {
  margin-top: 0;
  color: #58a6ff;
}

.output-content {
  background: #161b22;
  padding: 1rem;
  border-radius: 6px;
  overflow-x: auto;
  color: #8b949e;
  font-size: 0.875rem;
}

.loading {
  text-align: center;
  padding: 2rem;
  color: #8b949e;
}
</style>
