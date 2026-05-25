import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  const sidebarCollapsed = ref(localStorage.getItem('sidebar_collapsed') === 'true')
  const darkMode = ref(localStorage.getItem('dark_mode') === 'true')
  const openedTabs = ref<Array<{ path: string; name: string; label: string }>>(
    JSON.parse(localStorage.getItem('opened_tabs') || '[]')
  )
  const activeTab = ref('/')

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
    localStorage.setItem('sidebar_collapsed', String(sidebarCollapsed.value))
  }

  function toggleDarkMode() {
    darkMode.value = !darkMode.value
    localStorage.setItem('dark_mode', String(darkMode.value))
    applyDarkMode()
  }

  function applyDarkMode() {
    if (darkMode.value) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }

  function addTab(tab: { path: string; name: string; label: string }) {
    if (!openedTabs.value.find(t => t.path === tab.path)) {
      openedTabs.value.push(tab)
      localStorage.setItem('opened_tabs', JSON.stringify(openedTabs.value))
    }
    activeTab.value = tab.path
  }

  function removeTab(path: string) {
    openedTabs.value = openedTabs.value.filter(t => t.path !== path)
    localStorage.setItem('opened_tabs', JSON.stringify(openedTabs.value))
  }

  function clearTabs() {
    openedTabs.value = []
    localStorage.setItem('opened_tabs', '[]')
  }

  // Initialize dark mode on load
  applyDarkMode()

  return {
    sidebarCollapsed, darkMode, openedTabs, activeTab,
    toggleSidebar, toggleDarkMode, applyDarkMode,
    addTab, removeTab, clearTabs,
  }
})
