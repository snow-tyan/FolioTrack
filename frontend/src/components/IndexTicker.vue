<template>
  <div class="index-ticker-container glass-panel">
    <div class="ticker-wrapper">
      <div class="ticker-track">
        <div v-for="index in indexes" :key="index.symbol" class="ticker-item">
          <span class="index-name">{{ index.name }}</span>
          <span class="index-value">{{ formatNum(index.current, 2) }}</span>
          <span :class="['index-change', getChangeClass(index.change)]">
            {{ index.change >= 0 ? '+' : '' }}{{ formatNum(index.change, 2) }} 
            ({{ index.changePct >= 0 ? '+' : '' }}{{ formatNum(index.changePct, 2) }}%)
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import api from '../utils/api'

const props = defineProps({
  colorConvention: {
    type: String,
    default: 'CN'
  }
})

const indexes = ref([])
let timer = null

const fetchIndexes = async () => {
  try {
    const data = await api.get('/market/indexes')
    indexes.value = data
  } catch (err) {
    console.error('Failed to fetch indexes:', err)
  }
}

const formatNum = (val, decimals) => {
  if (val === undefined || val === null || isNaN(val)) return '0.00'
  return val.toFixed(decimals)
}

const getChangeClass = (change) => {
  if (change > 0) {
    return props.colorConvention === 'CN' ? 'up-cn' : 'up-us'
  } else if (change < 0) {
    return props.colorConvention === 'CN' ? 'down-cn' : 'down-us'
  }
  return 'flat'
}

onMounted(() => {
  fetchIndexes()
  // Refresh every 30 seconds
  timer = setInterval(fetchIndexes, 30000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.index-ticker-container {
  padding: 8px 16px;
  margin-bottom: 20px;
  overflow: hidden;
  border-radius: 12px;
  background: var(--card-bg);
  height: 38px;
  display: flex;
  align-items: center;
}

.ticker-wrapper {
  overflow: hidden;
  width: 100%;
}

.ticker-track {
  display: flex;
  gap: 40px;
  animation: scroll-ticker 25s linear infinite;
  white-space: nowrap;
  width: max-content;
}

.ticker-track:hover {
  animation-play-state: paused;
}

.ticker-item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 500;
}

.index-name {
  color: var(--text-secondary);
}

.index-value {
  font-family: 'Outfit', sans-serif;
  color: var(--text-primary);
  font-weight: 600;
}

.index-change {
  font-family: 'Outfit', sans-serif;
  font-size: 12px;
  font-weight: 500;
  padding: 1px 6px;
  border-radius: 4px;
}

/* Colors based on convention */
.up-cn, .down-us {
  color: var(--color-danger); /* Red rises in CN / Red falls in US */
  background: rgba(239, 68, 68, 0.15);
}

.down-cn, .up-us {
  color: var(--color-success); /* Green falls in CN / Green rises in US */
  background: rgba(16, 185, 129, 0.15);
}

.flat {
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.05);
}

@keyframes scroll-ticker {
  0% {
    transform: translate3d(0, 0, 0);
  }
  100% {
    transform: translate3d(-50%, 0, 0); /* We double the items if we want a continuous loop, but for dashboard a simple float works */
  }
}
</style>
