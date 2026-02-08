import { ref, onUnmounted } from 'vue'

export function useWebSocket(sessionId) {
  const connected = ref(false)
  const error = ref(null)
  const data = ref(null)
  let ws = null

  const connect = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    ws = new WebSocket(`${protocol}//${host}/ws/sessions/${sessionId}`)

    ws.onopen = () => {
      connected.value = true
      error.value = null
    }

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        data.value = msg
      } catch (e) {
        console.error('Failed to parse WebSocket message:', e)
      }
    }

    ws.onerror = (err) => {
      error.value = err
    }

    ws.onclose = () => {
      connected.value = false
    }
  }

  const disconnect = () => {
    if (ws) {
      ws.close()
      ws = null
    }
  }

  const send = (message) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message))
    }
  }

  onUnmounted(() => {
    disconnect()
  })

  return {
    connected,
    error,
    data,
    connect,
    disconnect,
    send
  }
}
