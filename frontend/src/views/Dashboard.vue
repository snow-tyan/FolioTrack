<template>
  <div class="dashboard-layout">
    <!-- Sidebar -->
    <aside class="sidebar glass-panel">
      <div class="sidebar-brand">
        <el-icon class="brand-icon"><TrendCharts /></el-icon>
        <span class="brand-text">FolioTrack</span>
      </div>

      <nav class="sidebar-menu">
        <router-link :to="{ name: 'Holdings' }" class="menu-item" active-class="active">
          <el-icon><PieChart /></el-icon>
          <span>持仓管理</span>
        </router-link>

        <router-link :to="{ name: 'Combos' }" class="menu-item" active-class="active">
          <el-icon><CollectionTag /></el-icon>
          <span>组合管理</span>
        </router-link>

        <router-link :to="{ name: 'ImportExport' }" class="menu-item" active-class="active">
          <el-icon><UploadFilled /></el-icon>
          <span>导入导出</span>
        </router-link>
      </nav>

      <div class="sidebar-footer">
        <div class="user-profile">
          <el-avatar :size="32" class="user-avatar">{{ username.substring(0, 1).toUpperCase() }}</el-avatar>
          <div class="user-info">
            <span class="username">{{ username }}</span>
          </div>
        </div>
        <button class="logout-btn" @click="handleLogout" title="退出登录">
          <el-icon><SwitchButton /></el-icon>
        </button>
      </div>
    </aside>

    <!-- Main Content Area -->
    <div class="main-area">
      <!-- Header -->
      <header class="header glass-panel">
        <h2 class="page-title">{{ currentRouteTitle }}</h2>
        <div class="header-actions">
          <!-- Base Currency Switch -->
          <div class="setting-item">
            <span class="setting-label">展示货币</span>
            <el-select v-model="baseCurrency" size="small" style="width: 115px;" @change="handleCurrencyChange">
              <el-option value="CNY" label="人民币 (CNY)" />
              <el-option value="USD" label="美元 (USD)" />
              <el-option value="HKD" label="港元 (HKD)" />
            </el-select>
          </div>

          <div class="divider"></div>

          <!-- Theme Switch -->
          <div class="setting-item">
            <span class="setting-label">视觉主题</span>
            <el-select v-model="theme" size="small" style="width: 160px;" @change="handleThemeChange">
              <el-option value="dark-indigo" label="深邃星海 (靛蓝)" />
              <el-option value="dark-emerald" label="翡翠森林 (深绿)" />
              <el-option value="dark-rose" label="赛博霓虹 (玫红)" />
              <el-option value="light-classic" label="温润雅白 (亮色)" />
              <el-option value="dark-amoled" label="曜石纯黑 (暗色)" />
              <el-option value="high-contrast" label="极客黑白 (对比)" />
              <el-option value="nordic-frost" label="北欧极寒 (冷蓝)" />
              <el-option value="coffee-mocha" label="香醇摩卡 (秋棕)" />
            </el-select>
          </div>

          <div class="divider"></div>

          <!-- Color Convention Switch -->
          <div class="setting-item">
            <span class="setting-label">涨跌颜色习惯</span>
            <el-select v-model="colorConvention" size="small" style="width: 130px;" @change="handleColorChange">
              <el-option value="CN" label="红涨绿跌 (国内)" />
              <el-option value="US" label="绿涨红跌 (国际)" />
            </el-select>
          </div>

          <div class="divider"></div>

          <!-- Time display or widget -->
          <span class="time-display">{{ currentTime }}</span>
        </div>
      </header>

      <!-- Content Viewport -->
      <main class="content-viewport">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" :color-convention="colorConvention" :theme="theme" :base-currency="baseCurrency" />
          </transition>
        </router-view>
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'

const router = useRouter()
const route = useRoute()

const username = ref('User')
const colorConvention = ref('CN')
const theme = ref('dark-indigo')
const baseCurrency = ref('CNY')
const currentTime = ref('')
let timer = null

const applyTheme = (newTheme) => {
  const root = document.documentElement
  root.classList.remove(
    'theme-dark-indigo',
    'theme-dark-emerald',
    'theme-dark-rose',
    'theme-light-classic',
    'theme-dark-amoled',
    'theme-high-contrast',
    'theme-nordic-frost',
    'theme-coffee-mocha'
  )
  root.classList.add(`theme-${newTheme}`)

  if (newTheme === 'light-classic') {
    root.classList.remove('dark')
  } else {
    root.classList.add('dark')
  }
}

const loadUserData = () => {
  const userStr = localStorage.getItem('user')
  if (userStr) {
    try {
      const user = JSON.parse(userStr)
      username.value = user.username || 'User'
    } catch (e) {
      username.value = 'User'
    }
  }

  // Load color preference
  const savedConvention = localStorage.getItem('colorConvention')
  if (savedConvention) {
    colorConvention.value = savedConvention
  }

  // Load theme preference
  const savedTheme = localStorage.getItem('theme') || 'dark-indigo'
  theme.value = savedTheme
  applyTheme(savedTheme)

  // Load base currency preference
  const savedCurrency = localStorage.getItem('baseCurrency') || 'CNY'
  baseCurrency.value = savedCurrency
}

const handleThemeChange = (val) => {
  applyTheme(val)
  localStorage.setItem('theme', val)
  ElMessage.success(`已切换视觉主题`)
}

const handleColorChange = (val) => {
  localStorage.setItem('colorConvention', val)
  ElMessage.success(`已切换为: ${val === 'CN' ? '红涨绿跌' : '绿涨红跌'}`)
}

const handleCurrencyChange = (val) => {
  localStorage.setItem('baseCurrency', val)
  ElMessage.success(`已切换展示货币为: ${val}`)
}

const handleLogout = () => {
  ElMessageBox.confirm('确定退出登录吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning',
    boxType: 'confirm'
  }).then(() => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    ElMessage.success('已安全退出')
    router.push({ name: 'Login' })
  }).catch(() => {})
}

const currentRouteTitle = computed(() => {
  switch (route.name) {
    case 'Holdings':
      return '我的持仓看板'
    case 'Combos':
      return '组合与标签管理'
    case 'ImportExport':
      return '持仓数据导入/导出'
    default:
      return '资产管理系统'
  }
})

const updateTime = () => {
  const now = new Date()
  currentTime.value = now.toLocaleTimeString('zh-CN', { hour12: false })
}

onMounted(() => {
  loadUserData()
  updateTime()
  timer = setInterval(updateTime, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.dashboard-layout {
  display: flex;
  min-height: 100vh;
  background-color: #05070a;
}

/* Sidebar Styling */
.sidebar {
  width: 240px;
  position: fixed;
  top: 15px;
  bottom: 15px;
  left: 15px;
  display: flex;
  flex-direction: column;
  padding: 24px 16px;
  z-index: 10;
  border-radius: 20px;
}

.sidebar-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 35px;
  padding: 0 10px;
}

.brand-icon {
  font-size: 24px;
  color: var(--color-primary);
}

.brand-text {
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 0.05em;
  background: linear-gradient(to right, #ffffff, var(--color-primary));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.sidebar-menu {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 16px;
  color: var(--text-secondary);
  text-decoration: none;
  font-weight: 500;
  font-size: 14px;
  border-radius: 10px;
  border-left: 3px solid transparent;
  transition: var(--transition-smooth);
}

.menu-item:hover {
  background: rgba(255, 255, 255, 0.03);
  color: var(--text-primary);
}

.menu-item.active {
  background: rgba(99, 102, 241, 0.12);
  color: var(--text-primary);
  border-left-color: var(--color-primary);
}

.sidebar-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.user-profile {
  display: flex;
  align-items: center;
  gap: 10px;
}

.user-avatar {
  background-color: var(--color-primary);
  color: #fff;
  font-weight: 600;
}

.user-info {
  display: flex;
  flex-direction: column;
}

.username {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.logout-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 18px;
  cursor: pointer;
  padding: 8px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: var(--transition-smooth);
}

.logout-btn:hover {
  background: rgba(239, 68, 68, 0.1);
  color: var(--color-danger);
}

/* Main Content Area Styling */
.main-area {
  flex: 1;
  min-width: 0; /* 🟢 Prevents flex child from overflowing on window shrink */
  margin-left: 270px; /* sidebar width (240) + spacing (30) */
  padding: 15px 15px 15px 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* Header Styling */
.header {
  height: 64px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 24px;
  border-radius: 16px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 20px;
}

.setting-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.setting-label {
  font-size: 12px;
  color: var(--text-secondary);
}

.divider {
  width: 1px;
  height: 20px;
  background: rgba(255, 255, 255, 0.1);
}

.time-display {
  font-family: 'Outfit', sans-serif;
  font-size: 14px;
  color: var(--text-secondary);
  font-weight: 500;
  min-width: 65px;
}

.content-viewport {
  flex: 1;
}

/* Route transitions */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
