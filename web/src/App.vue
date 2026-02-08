<template>
  <div class="app">
    <header class="header">
      <h1>Codex Agent Team</h1>
      <nav class="nav">
        <router-link to="/">Home</router>
      </nav>
    </header>
    <main class="main">
      <router-view />
    </main>
  </div>
</template>

<script setup>
</script>

<style scoped>
.app {
  min-height: 100vh;
  background: #0d1117;
  color: #c9d1d9;
}

.header {
  padding: 1rem 2rem;
  background: #161b22;
  border-bottom: 1px solid #30363d;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header h1 {
  margin: 0;
  font-size: 1.5rem;
  color: #58a6ff;
}

.nav a {
  color: #58a6ff;
  text-decoration: none;
  margin-left: 1rem;
}

.main {
  padding: 2rem;
  max-width: 1200px;
  margin: 0 auto;
}
</style>

<style>
* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
}
</style>
