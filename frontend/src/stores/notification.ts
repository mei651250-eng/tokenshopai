import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface Notification {
  id: string
  type: 'system' | 'alert' | 'billing' | 'security' | 'model'
  level: 'info' | 'warning' | 'error' | 'success'
  title: string
  message: string
  read: boolean
  createdAt: number
  link?: string
}

export const useNotificationStore = defineStore('notification', () => {
  const notifications = ref<Notification[]>([])
  const ws = ref<WebSocket | null>(null)
  const connected = ref(false)

  const unreadCount = computed(() => notifications.value.filter(n => !n.read).length)

  function addNotification(n: Notification) {
    notifications.value.unshift(n)
    if (notifications.value.length > 100) {
      notifications.value = notifications.value.slice(0, 100)
    }
  }

  function markAsRead(id: string) {
    const n = notifications.value.find(item => item.id === id)
    if (n) n.read = true
  }

  function markAllAsRead() {
    notifications.value.forEach(n => { n.read = true })
  }

  function removeNotification(id: string) {
    notifications.value = notifications.value.filter(n => n.id !== id)
  }

  function clearAll() {
    notifications.value = []
  }

  function connectWs(baseUrl: string) {
    if (ws.value?.readyState === WebSocket.OPEN) return
    const wsUrl = baseUrl.replace(/^http/, 'ws').replace(/\/$/, '')
    ws.value = new WebSocket(`${wsUrl}/ws/notifications`)
    ws.value.onopen = () => { connected.value = true }
    ws.value.onclose = () => { connected.value = false; setTimeout(() => connectWs(baseUrl), 5000) }
    ws.value.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        addNotification({
          id: data.id || Date.now().toString(),
          type: data.type || 'system',
          level: data.level || 'info',
          title: data.title || '',
          message: data.message || '',
          read: false,
          createdAt: data.created_at || Date.now(),
          link: data.link,
        })
      } catch { /* ignore */ }
    }
  }

  function disconnectWs() {
    ws.value?.close()
    ws.value = null
    connected.value = false
  }

  return {
    notifications, connected, unreadCount,
    addNotification, markAsRead, markAllAsRead, removeNotification, clearAll,
    connectWs, disconnectWs,
  }
})
