<template>
  <div class="home">
    <div class="card">
      <h2>创建新任务</h2>
      <form @submit.prevent="handleSubmit" class="form">
        <textarea
          v-model="userTask"
          class="textarea"
          placeholder="描述你想要完成的任务..."
          rows="6"
          required
        />
        <button type="submit" class="btn" :disabled="loading">
          {{ loading ? '处理中...' : '创建并分析' }}
        </button>
      </form>
    </div>

    <div v-if="error" class="error">{{ error }}</div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '../stores/session'

const router = useRouter()
const sessionStore = useSessionStore()

const userTask = ref('')
const loading = ref(false)
const error = ref(null)

async function handleSubmit() {
  if (!userTask.value.trim()) return

  loading.value = true
  error.value = null

  try {
    const session = await sessionStore.createSession(userTask.value)
    router.push(`/session/${session.ID}`)
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
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

.card h2 {
  margin-top: 0;
  color: #58a6ff;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
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

.error {
  background: #3d1515;
  border: 1px solid #f85149;
  color: #ffa198;
  padding: 1rem;
  border-radius: 6px;
}
</style>
