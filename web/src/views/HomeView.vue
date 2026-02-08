<template>
  <div class="home">
    <div class="card">
      <h2>创建新任务</h2>
      <form @submit.prevent="handleSubmit" class="form">
        <div class="form-group">
          <label>工作目录</label>
          <div class="path-input">
            <input
              v-model="repoPath"
              class="input"
              placeholder="/path/to/your/repo"
              required
            />
            <button type="button" @click="selectDefaultPath" class="btn-small">使用当前目录</button>
          </div>
        </div>
        <div class="form-group">
          <label>任务描述</label>
          <textarea
            v-model="userTask"
            class="textarea"
            placeholder="描述你想要完成的任务..."
            rows="6"
            required
          />
        </div>
        <button type="submit" class="btn" :disabled="loading">
          {{ loading ? '处理中...' : '创建并分析' }}
        </button>
      </form>
    </div>

    <!-- Sessions List -->
    <div v-if="sessions.length > 0" class="sessions">
      <h3>历史会话</h3>
      <div class="session-list">
        <router-link
          v-for="s in sessions"
          :key="s.ID"
          :to="`/session/${s.ID}`"
          class="session-item"
        >
          <div class="session-header">
            <span class="session-id">{{ s.ID.slice(-8) }}</span>
            <span class="status" :class="`status-${s.Status}`">{{ s.Status }}</span>
          </div>
          <p class="session-task">{{ s.UserTask.slice(0, 80) }}{{ s.UserTask.length > 80 ? '...' : '' }}</p>
          <small class="session-time">{{ formatTime(s.CreatedAt) }}</small>
        </router-link>
      </div>
    </div>

    <div v-if="error" class="error">{{ error }}</div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const userTask = ref('')
const repoPath = ref('')
const loading = ref(false)
const error = ref(null)
const sessions = ref([])

onMounted(() => {
  loadSessions()
  loadSystemInfo()
})

async function loadSystemInfo() {
  try {
    const res = await fetch('/api/info')
    if (res.ok) {
      const info = await res.json()
      repoPath.value = info.defaultRepo || ''
    }
  } catch (e) {
    console.error('Failed to load system info:', e)
  }
}

async function loadSessions() {
  try {
    const res = await fetch('/api/sessions')
    if (res.ok) {
      sessions.value = await res.json()
    }
  } catch (e) {
    console.error('Failed to load sessions:', e)
  }
}

function selectDefaultPath() {
  repoPath.value = window.location.pathname.split('/').slice(0, -1).join('/') || '.'
}

async function handleSubmit() {
  if (!userTask.value.trim() || !repoPath.value.trim()) return

  loading.value = true
  error.value = null

  try {
    const res = await fetch('/api/sessions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        userTask: userTask.value,
        repoPath: repoPath.value
      })
    })

    if (!res.ok) {
      throw new Error('Failed to create session')
    }

    const session = await res.json()
    router.push(`/session/${session.ID}`)
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function formatTime(timeStr) {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN')
}
</script>

<style scoped>
.home {
  max-width: 800px;
  margin: 0 auto;
}

.card {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 8px;
  padding: 2rem;
  margin-bottom: 2rem;
}

.card h2, .sessions h3 {
  margin-top: 0;
  color: #58a6ff;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-group label {
  font-size: 0.875rem;
  color: #8b949e;
}

.path-input {
  display: flex;
  gap: 0.5rem;
}

.input, .textarea {
  flex: 1;
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 6px;
  color: #c9d1d9;
  padding: 0.75rem;
  font-size: 1rem;
  font-family: inherit;
}

.input:focus, .textarea:focus {
  outline: none;
  border-color: #58a6ff;
}

.textarea {
  resize: vertical;
}

.btn-small {
  background: #21262d;
  color: #58a6ff;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 0 1rem;
  cursor: pointer;
  white-space: nowrap;
}

.btn-small:hover {
  background: #30363d;
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

.error {
  background: #3d1515;
  border: 1px solid #f85149;
  color: #ffa198;
  padding: 1rem;
  border-radius: 6px;
}

.sessions {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 8px;
  padding: 1.5rem;
}

.session-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.session-item {
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 1rem;
  text-decoration: none;
  color: inherit;
  transition: border-color 0.2s;
}

.session-item:hover {
  border-color: #58a6ff;
}

.session-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.session-id {
  font-family: monospace;
  color: #58a6ff;
}

.status {
  padding: 0.125rem 0.5rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
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

.session-task {
  margin: 0.5rem 0;
  color: #8b949e;
}

.session-time {
  color: #484f58;
  font-size: 0.75rem;
}
</style>
