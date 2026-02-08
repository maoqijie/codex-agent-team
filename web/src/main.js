import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia } from 'pinia'
import App from './App.vue'
import HomeView from './views/HomeView.vue'
import SessionView from './views/SessionView.vue'

const routes = [
  { path: '/', component: HomeView },
  { path: '/session/:id', component: SessionView, props: true }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

const app = createApp(App)
app.use(router)
app.use(createPinia())
app.mount('#app')
