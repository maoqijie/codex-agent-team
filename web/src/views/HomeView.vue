<template>
  <div class="home">
    <div class="card">
      <h2>创建新任务</h2>
      <form @submit.prevent="handleSubmit" class="form">
        <div class="form-group">
          <label>工作目录</label>
          <button type="button" @click="showDirPicker = true" class="dir-select-btn">
            <span class="dir-icon">📁</span>
            <span class="dir-path">{{ repoPath || '选择工作目录...' }}</span>
          </button>
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

    <!-- Directory Picker Modal -->
    <div v-if="showDirPicker" class="modal-overlay" @click.self="showDirPicker = false">
      <div class="modal">
        <div class="modal-header">
          <h3>选择工作目录</h3>
          <button @click="showDirPicker = false" class="btn-close">✕</button>
        </div>
        <DirPicker ref="dirPickerRef" @select="selectDir" />
        <div class="modal-footer">
          <button @click="showDirPicker = false" class="btn-small">取消</button>
          <button @click="confirmDir" class="btn btn-primary">确定</button>
        </div>
      </div>
    </div>

    <!-- Active Sessions -->
    <div v-if="activeSessions.length > 0" class="sessions">
      <h3>进行中的任务 ({{ activeSessions.length }})</h3>
      <div class="session-list">
        <div
          v-for="s in activeSessions"
          :key="s.ID"
          @click="router.push(`/session/${s.ID}`)"
          class="session-item running"
        >
          <div class="session-header">
            <span class="session-id">{{ String(s.ID).slice(-8) }}</span>
            <span class="status" :class="`status-${s.Status}`">{{ s.Status }}</span>
          </div>
          <p class="session-task">{{ String(s.UserTask || '').slice(0, 60) }}{{ String(s.UserTask || '').length > 60 ? '...' : '' }}</p>
          <small class="session-repo">{{ s.RepoPath }}</small>
        </div>
      </div>
    </div>

    <!-- Recent Sessions -->
    <div v-if="recentSessions.length > 0" class="sessions">
      <h3>历史会话</h3>
      <div class="session-list">
        <div
          v-for="s in recentSessions"
          :key="s.ID"
          @click="router.push(`/session/${s.ID}`)"
          class="session-item"
        >
          <div class="session-header">
            <span class="session-id">{{ String(s.ID).slice(-8) }}</span>
            <span class="status" :class="`status-${s.Status}`">{{ s.Status }}</span>
          </div>
          <p class="session-task">{{ String(s.UserTask || '').slice(0, 60) }}{{ String(s.UserTask || '').length > 60 ? '...' : '' }}</p>
          <small class="session-repo">{{ s.RepoPath }}</small>
          <small class="session-time">{{ formatTime(s.CreatedAt) }}</small>
        </div>
      </div>
    </div>

    <div v-if="error" class="error">{{ error }}</div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import DirPicker from '../components/DirPicker.vue'

const router = useRouter()

const userTask = ref('')
const repoPath = ref('')
const loading = ref(false)
const error = ref(null)
const sessions = ref([])
const showDirPicker = ref(false)
const dirPickerRef = ref(null)

// Separate active and recent sessions
const activeSessions = computed(() => {
  return sessions.value.filter(s => ['created', 'ready', 'running', 'decomposing', 'merging'].includes(s.Status))
})

const recentSessions = computed(() => {
  return sessions.value
    .filter(s => ['completed', 'failed'].includes(s.Status))
    .slice(0, 10)
})

onMounted(() => {
  loadSessions()
  loadSystemInfo()
  startPolling()
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
      const data = await res.json()
      sessions.value = data
    }
  } catch (e) {
    console.error('Failed to load sessions:', e)
  }
}

function selectDir(path) {
  repoPath.value = path
  showDirPicker.value = false
}

function confirmDir() {
  if (dirPickerRef.value) {
    const path = dirPickerRef.value.confirm()
    if (path) {
      repoPath.value = path
      showDirPicker.value = false
    }
  }
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
    userTask.value = ''
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
  const now = new Date()
  const diff = now - date

  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`
  return date.toLocaleDateString('zh-CN')
}

// Poll for session updates
let pollInterval = null
function startPolling() {
  pollInterval = setInterval(() => {
    loadSessions()
  }, 3000)
}

// Clean up on unmount (in real app would use onUnmounted)
</script>

<style scoped>
.home {
  max-width: 900px;
  margin: 0 auto;
}

.card {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 8px;
  padding: 2rem;
  margin-bottom: 1.5rem;
}

.card h2 {
  margin: 0 0 1.5rem 0;
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

.dir-select-btn {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 0.75rem;
  color: #c9d1d9;
  cursor: pointer;
  transition: border-color 0.2s;
  width: 100%;
}

.dir-select-btn:hover {
  border-color: #58a6ff;
}

.dir-icon {
  font-size: 1.25rem;
}

.dir-path {
  flex: 1;
  font-family: monospace;
  text-align: left;
}

.textarea {
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 6px;
  color: #c9d1d9;
  padding: 1rem;
  font-size: 1rem;
  font-family: inherit;
  resize: vertical;
}

.textarea:focus {
  outline: none;
  border-color: #58a6ff;
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

.btn-small {
  background: #21262d;
  color: #58a6ff;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 0.5rem 1rem;
  cursor: pointer;
}

/* Modal */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 12px;
  padding: 1.5rem;
  min-width: 500px;
  max-width: 90vw;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.modal-header h3 {
  margin: 0;
  color: #58a6ff;
}

.btn-close {
  background: none;
  border: none;
  color: #8b949e;
  font-size: 1.5rem;
  cursor: pointer;
  padding: 0;
  line-height: 1;
}

.btn-close:hover {
  color: #c9d1d9;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid #30363d;
}

/* Sessions */
.sessions {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 8px;
  padding: 1.5rem;
  margin-bottom: 1.5rem;
}

.sessions h3 {
  margin: 0 0 1rem 0;
  color: #58a6ff;
  font-size: 1rem;
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
  cursor: pointer;
}

.session-item:hover {
  border-color: #58a6ff;
}

.session-item.running {
  border-left: 3px solid #e3b341;
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

.status-running, .status-decomposing, .status-merging {
  background: #d2992222;
  color: #e3b341;
  animation: pulse 1.5s infinite;
}

.status-completed {
  background: #23863633;
  color: #3fb950;
}

.status-failed {
  background: #f8514933;
  color: #f85149;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.session-task {
  margin: 0.5rem 0;
  color: #c9d1d9;
  font-size: 0.875rem;
}

.session-repo {
  display: block;
  color: #8b949e;
  font-family: monospace;
  font-size: 0.75rem;
}

.session-time {
  color: #484f58;
  font-size: 0.75rem;
}

.error {
  background: #3d1515;
  border: 1px solid #f85149;
  color: #ffa198;
  padding: 1rem;
  border-radius: 6px;
}
</style>
