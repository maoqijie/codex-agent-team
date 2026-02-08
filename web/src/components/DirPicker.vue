<template>
  <div class="dir-picker">
    <div class="picker-header">
      <button @click="goUp" class="btn-icon" title="上级目录">⬆️</button>
      <button @click="goHome" class="btn-icon" title="主目录">🏠</button>
      <button @click="refresh" class="btn-icon" title="刷新">🔄</button>
      <span class="current-path">{{ currentPath }}</span>
    </div>

    <div class="path-breadcrumbs">
      <span
        v-for="(part, idx) in pathParts"
        :key="idx"
        class="breadcrumb"
        @click="navigateTo(idx)"
      >
        {{ part }} <span v-if="idx < pathParts.length - 1">/</span>
      </span>
    </div>

    <div class="dir-list">
      <div v-if="loading" class="loading">加载中...</div>
      <div
        v-for="dir in directories"
        :key="dir.path"
        class="dir-item"
        :class="{ selected: selectedPath === dir.path, 'is-git': dir.isGit }"
        @click="selectDir(dir.path)"
        @dblclick="enterDir(dir.path)"
      >
        <span class="dir-icon">{{ dir.isGit ? '📂' : '📁' }}</span>
        <span class="dir-name">{{ dir.name }}</span>
      </div>

      <div v-if="!loading && directories.length === 0" class="empty">
        此目录为空或无法访问
      </div>
    </div>

    <!-- Recent directories -->
    <div v-if="recentDirs.length > 0" class="recent-section">
      <h4>最近使用的目录</h4>
      <div class="recent-list">
        <div
          v-for="dir in recentDirs"
          :key="dir"
          class="recent-item"
          @click="navigateToPath(dir)"
        >
          {{ dir }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'

const emit = defineEmits(['select'])

const currentPath = ref('.')
const selectedPath = ref('')
const directories = ref([])
const recentDirs = ref([])
const loading = ref(false)

const pathParts = computed(() => {
  if (currentPath.value === '.') return ['主目录']
  const parts = []
  let path = currentPath.value
  while (path !== '/' && path !== '.') {
    parts.unshift(basename(path))
    path = dirname(path)
  }
  return parts.length > 0 ? parts : ['主目录']
})

// Simulated filepath functions
function basename(p) {
  const parts = p.split('/')
  return parts[parts.length - 1] || p
}

function dirname(p) {
  const parts = p.split('/')
  parts.pop()
  const result = parts.join('/')
  return result || '.'
}

onMounted(() => {
  loadRecentDirs()
  loadPath('.')
})

async function loadPath(path) {
  loading.value = true
  try {
    const url = path ? `/api/dirs?path=${encodeURIComponent(path)}` : '/api/dirs'
    const res = await fetch(url)
    if (res.ok) {
      const data = await res.json()
      currentPath.value = data.path
      directories.value = data.dirs || []
    }
  } catch (e) {
    console.error('Failed to load directories:', e)
  } finally {
    loading.value = false
  }
}

function selectDir(path) {
  selectedPath.value = path
}

function enterDir(path) {
  selectedPath.value = path
  loadPath(path)
}

function goUp() {
  const parent = filepath.dirname(currentPath.value)
  loadPath(parent)
}

function goHome() {
  loadPath('.')
}

function refresh() {
  loadPath(currentPath.value)
}

function navigateTo(idx) {
  const parts = pathParts.value
  if (idx === 0 || parts[0] === '主目录') {
    loadPath('.')
  } else {
    // Reconstruct path up to this index
    const targetPath = currentPath.value.split('/').slice(0, idx + 2).join('/') || '.'
    loadPath(targetPath)
  }
}

function navigateToPath(path) {
  loadPath(path)
  selectedPath.value = path
}

function loadRecentDirs() {
  const saved = localStorage.getItem('recent-dirs')
  if (saved) {
    try {
      recentDirs.value = JSON.parse(saved)
    } catch (e) {
      recentDirs.value = []
    }
  }
}

function saveRecentDir(path) {
  const dirs = recentDirs.value.filter(d => d !== path)
  dirs.unshift(path)
  recentDirs.value = dirs.slice(0, 10)
  localStorage.setItem('recent-dirs', JSON.stringify(dirs))
}

// Expose confirm method
defineExpose({
  confirm: () => {
    if (selectedPath.value) {
      saveRecentDir(selectedPath.value)
      emit('select', selectedPath.value)
    }
    return selectedPath.value
  }
})
</script>

<style scoped>
.dir-picker {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 8px;
  padding: 1rem;
}

.picker-header {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-bottom: 0.75rem;
}

.btn-icon {
  background: #21262d;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 0.5rem;
  cursor: pointer;
  font-size: 1rem;
}

.btn-icon:hover {
  background: #30363d;
}

.current-path {
  flex: 1;
  font-family: monospace;
  color: #8b949e;
  font-size: 0.875rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.path-breadcrumbs {
  display: flex;
  gap: 0.25rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.breadcrumb {
  color: #58a6ff;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.875rem;
}

.breadcrumb:hover {
  background: #1f6feb22;
}

.dir-list {
  max-height: 300px;
  overflow-y: auto;
  border: 1px solid #30363d;
  border-radius: 6px;
  margin-bottom: 1rem;
}

.loading {
  padding: 2rem;
  text-align: center;
  color: #8b949e;
}

.dir-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0.75rem;
  cursor: pointer;
  border-bottom: 1px solid #21262d;
}

.dir-item:hover {
  background: #21262d;
}

.dir-item.selected {
  background: #1f6feb33;
}

.dir-item.is-git {
  border-left: 2px solid #238636;
}

.dir-icon {
  font-size: 1.25rem;
}

.dir-name {
  color: #c9d1d9;
  flex: 1;
}

.empty {
  padding: 2rem;
  text-align: center;
  color: #484f58;
}

.recent-section h4 {
  margin: 0 0 0.75rem 0;
  color: #8b949e;
  font-size: 0.875rem;
}

.recent-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.recent-item {
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 4px;
  padding: 0.25rem 0.75rem;
  font-family: monospace;
  font-size: 0.75rem;
  color: #58a6ff;
  cursor: pointer;
}

.recent-item:hover {
  border-color: #58a6ff;
}
</style>
