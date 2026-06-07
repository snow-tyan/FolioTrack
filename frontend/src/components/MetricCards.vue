<template>
  <div class="metrics-grid">
    <!-- Card 1: Total Assets -->
    <div class="glass-panel metric-card animate-fade-in" style="animation-delay: 0.1s;">
      <div class="card-icon-wrapper wallet">
        <el-icon><Wallet /></el-icon>
      </div>
      <div class="card-content">
        <span class="card-title">总资产市值</span>
        <span class="card-value font-outfit">¥ {{ formatMoney(metrics.totalValue) }}</span>
      </div>
    </div>

    <!-- Card 2: Total Cost -->
    <div class="glass-panel metric-card animate-fade-in" style="animation-delay: 0.2s;">
      <div class="card-icon-wrapper cost">
        <el-icon><Money /></el-icon>
      </div>
      <div class="card-content">
        <span class="card-title">总持仓成本</span>
        <span class="card-value font-outfit">¥ {{ formatMoney(metrics.totalCost) }}</span>
      </div>
    </div>

    <!-- Card 3: Cumulative P&L -->
    <div class="glass-panel metric-card animate-fade-in" style="animation-delay: 0.3s;">
      <div class="card-icon-wrapper" :class="getPnlIconClass(metrics.cumulativePnl)">
        <el-icon>
          <CaretTop v-if="metrics.cumulativePnl >= 0" />
          <CaretBottom v-else />
        </el-icon>
      </div>
      <div class="card-content">
        <span class="card-title">累计盈亏</span>
        <span class="card-value font-outfit" :class="getPnlTextClass(metrics.cumulativePnl)">
          {{ metrics.cumulativePnl >= 0 ? '+' : '' }}{{ formatMoney(metrics.cumulativePnl) }}
        </span>
        <span class="card-sub-info font-outfit" :class="getPnlTextClass(metrics.cumulativePnl)">
          {{ metrics.cumulativePnl >= 0 ? '+' : '' }}{{ formatPercent(metrics.cumulativePnlRatio) }}%
        </span>
      </div>
    </div>

    <!-- Card 4: Daily P&L -->
    <div class="glass-panel metric-card animate-fade-in" style="animation-delay: 0.4s;">
      <div class="card-icon-wrapper" :class="getPnlIconClass(metrics.dailyPnl)">
        <el-icon>
          <CaretTop v-if="metrics.dailyPnl >= 0" />
          <CaretBottom v-else />
        </el-icon>
      </div>
      <div class="card-content">
        <span class="card-title">今日盈亏</span>
        <span class="card-value font-outfit" :class="getPnlTextClass(metrics.dailyPnl)">
          {{ metrics.dailyPnl >= 0 ? '+' : '' }}{{ formatMoney(metrics.dailyPnl) }}
        </span>
        <span class="card-sub-info font-outfit" :class="getPnlTextClass(metrics.dailyPnl)">
          {{ metrics.dailyPnl >= 0 ? '+' : '' }}{{ formatPercent(metrics.dailyPnlRatio) }}%
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  holdings: {
    type: Array,
    required: true
  },
  colorConvention: {
    type: String,
    default: 'CN'
  }
})

const metrics = computed(() => {
  let totalValue = 0
  let totalCost = 0
  let dailyPnl = 0

  props.holdings.forEach((h) => {
    const qty = h.quantity || 0
    const cost = h.costPrice || 0
    const currPrice = h.asset ? (h.asset.currentPrice || 0) : 0
    const prevClose = h.asset ? (h.asset.prevClose || currPrice) : currPrice

    totalValue += qty * currPrice
    totalCost += qty * cost
    dailyPnl += qty * (currPrice - prevClose)
  })

  const cumulativePnl = totalValue - totalCost
  const cumulativePnlRatio = totalCost > 0 ? (cumulativePnl / totalCost) * 100 : 0

  // Daily change relative to yesterday's closing value
  const yesterdayValue = totalValue - dailyPnl
  const dailyPnlRatio = yesterdayValue > 0 ? (dailyPnl / yesterdayValue) * 100 : 0

  return {
    totalValue,
    totalCost,
    cumulativePnl,
    cumulativePnlRatio,
    dailyPnl,
    dailyPnlRatio
  }
})

const formatMoney = (val) => {
  if (val === undefined || val === null || isNaN(val)) return '0.00'
  return val.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

const formatPercent = (val) => {
  if (val === undefined || val === null || isNaN(val)) return '0.00'
  return val.toFixed(2)
}

const getPnlIconClass = (val) => {
  if (val >= 0) {
    return props.colorConvention === 'CN' ? 'up-cn-bg' : 'up-us-bg'
  } else {
    return props.colorConvention === 'CN' ? 'down-cn-bg' : 'down-us-bg'
  }
}

const getPnlTextClass = (val) => {
  if (val >= 0) {
    return props.colorConvention === 'CN' ? 'up-cn-text' : 'up-us-text'
  } else {
    return props.colorConvention === 'CN' ? 'down-cn-text' : 'down-us-text'
  }
}
</script>

<style scoped>
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 20px;
  margin-bottom: 25px;
}

.metric-card {
  display: flex;
  align-items: center;
  padding: 20px;
  position: relative;
  overflow: hidden;
}

.card-icon-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 12px;
  font-size: 24px;
  margin-right: 16px;
}

.wallet {
  background: rgba(99, 102, 241, 0.15);
  color: var(--color-primary);
}

.cost {
  background: rgba(245, 158, 11, 0.15);
  color: var(--color-warning);
}

/* Red rises in CN / Red falls in US */
.up-cn-bg, .down-us-bg {
  background: rgba(239, 68, 68, 0.15);
  color: var(--color-danger);
}

/* Green falls in CN / Green rises in US */
.down-cn-bg, .up-us-bg {
  background: rgba(16, 185, 129, 0.15);
  color: var(--color-success);
}

.card-content {
  display: flex;
  flex-direction: column;
}

.card-title {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 4px;
}

.card-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
}

.font-outfit {
  font-family: 'Outfit', sans-serif;
}

.card-sub-info {
  font-size: 12px;
  margin-top: 2px;
  font-weight: 500;
}

/* Colors based on convention */
.up-cn-text, .down-us-text {
  color: var(--color-danger) !important;
}

.down-cn-text, .up-us-text {
  color: var(--color-success) !important;
}
</style>
