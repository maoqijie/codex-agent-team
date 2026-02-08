import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useSessionStore = defineStore('session', () => {
  const currentSession = ref(null)
  const tasks = ref([])
  const loading = ref(false)
  const error = ref(null)

  const API_BASE = '/api'

  async function createSession(userTask) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${API_BASE}/sessions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ userTask })
      })
      if (!response.ok) throw new Error('Failed to create session')
      const session = await response.json()
      currentSession.value = session
      return session
    } catch (e) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  async function getSession(id) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${API_BASE}/sessions/${id}`)
      if (!response.ok) throw new Error('Failed to get session')
      const session = await response.json()
      currentSession.value = session
      return session
    } catch (e) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  async function decompose(id) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${API_BASE}/sessions/${id}/decompose`, {
        method: 'POST'
      })
      if (!response.ok) throw new Error('Failed to decompose')
      return await response.json()
    } catch (e) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  async function execute(id) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${API_BASE}/sessions/${id}/execute`, {
        method: 'POST'
      })
      if (!response.ok) throw new Error('Failed to execute')
      return await response.json()
    } catch (e) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  async function merge(id) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${API_BASE}/sessions/${id}/merge`, {
        method: 'POST'
      })
      if (!response.ok) throw new Error('Failed to merge')
      return await response.json()
    } catch (e) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  async function getTasks(id) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${API_BASE}/sessions/${id}/tasks`)
      if (!response.ok) throw new Error('Failed to get tasks')
      const tasksData = await response.json()
      tasks.value = tasksData
      return tasksData
    } catch (e) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  function setTasks(newTasks) {
    tasks.value = newTasks
  }

  return {
    currentSession,
    tasks,
    loading,
    error,
    createSession,
    getSession,
    decompose,
    execute,
    merge,
    getTasks,
    setTasks
  }
})
