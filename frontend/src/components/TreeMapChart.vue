<template>
  <div class="chart-container glass-panel">
    <div class="chart-header">
      <h3>持仓云图</h3>
      <div class="chart-controls">
        <el-radio-group v-model="viewMode" size="small" @change="renderChart">
          <el-radio-button value="market">按市场分组</el-radio-button>
          <el-radio-button value="combo">按组合分组</el-radio-button>
          <el-radio-button value="account">按账户分组</el-radio-button>
        </el-radio-group>
      </div>
    </div>
    <div ref="chartRef" class="treemap-chart"></div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import * as echarts from 'echarts'

const props = defineProps({
  holdings: {
    type: Array,
    required: true
  },
  colorConvention: {
    type: String,
    default: 'CN'
  },
  theme: {
    type: String,
    default: 'dark-indigo'
  },
  baseCurrency: {
    type: String,
    default: 'CNY'
  }
})

const chartRef = ref(null)
let chartInstance = null
const viewMode = ref('market') // 'market', 'combo', or 'account'

// Format market names
const marketNames = {
  'A-share': 'A股',
  'HK-stock': '港股',
  'US-stock': '美股',
  'Fund': '基金'
}

// Compute P&L percentage for a holding
const getHoldingPnlPct = (h) => {
  const cost = h.costPrice || 0
  const curr = h.asset ? (h.asset.currentPrice || 0) : 0
  if (cost === 0) return 0
  return ((curr - cost) / cost) * 100
}

// Predefined high-contrast, distinct HSL-based palette for adjacent elements
const highContrastPalette = [
  '#4f46e5', // Indigo
  '#10b981', // Emerald Green
  '#f59e0b', // Amber/Orange
  '#ec4899', // Hot Pink
  '#06b6d4', // Cyan
  '#f43f5e', // Rose
  '#8b5cf6', // Violet
  '#14b8a6', // Teal
  '#eab308', // Yellow
  '#3b82f6', // Blue
  '#a855f7', // Purple
  '#84cc16'  // Lime
]

const getBaseCurrencyRate = (currency, holdings) => {
  const rates = { CNY: 1.0, USD: 7.20, HKD: 0.92 }
  holdings.forEach((h) => {
    if (h.asset && h.asset.currency && h.asset.exchangeRate) {
      rates[h.asset.currency] = h.asset.exchangeRate
    }
  })
  return rates[currency] || 1.0
}

const getCurrencySymbol = (currency) => {
  const symbols = {
    'CNY': '¥',
    'USD': '$',
    'HKD': 'HK$'
  }
  return symbols[currency] || '¥'
}

const buildChartData = () => {
  if (!props.holdings || props.holdings.length === 0) {
    return []
  }

  const baseRate = getBaseCurrencyRate(props.baseCurrency, props.holdings)

  if (viewMode.value === 'market') {
    // 1. Group by Market
    const groups = {}
    props.holdings.forEach((h) => {
      const market = h.asset ? h.asset.market : 'Other'
      const assetRate = h.asset ? (h.asset.exchangeRate || 1.0) : 1.0
      const val = h.quantity * (h.asset ? h.asset.currentPrice : 0) * (assetRate / baseRate)
      if (val <= 0) return

      if (!groups[market]) {
        groups[market] = []
      }

      const pnlPct = getHoldingPnlPct(h)
      const colorIndex = groups[market].length
      const color = highContrastPalette[colorIndex % highContrastPalette.length]
      const comboNames = h.combos ? h.combos.map((c) => c.name) : []
      const accountName = h.account ? h.account.name : '默认账户'

      groups[market].push({
        name: `${h.asset ? h.asset.name : h.assetId}\n${pnlPct >= 0 ? '+' : ''}${pnlPct.toFixed(2)}%`,
        value: val,
        combos: comboNames,
        account: accountName,
        itemStyle: {
          color: color
        },
        label: {
          show: true,
          formatter: `{b}`
        }
      })
    })

    return Object.keys(groups).map((market) => ({
      name: marketNames[market] || market,
      children: groups[market]
    }))

  } else if (viewMode.value === 'combo') {
    // 2. Group by Combo
    const comboHoldings = {}
    const unclassified = []

    props.holdings.forEach((h) => {
      const assetRate = h.asset ? (h.asset.exchangeRate || 1.0) : 1.0
      const val = h.quantity * (h.asset ? h.asset.currentPrice : 0) * (assetRate / baseRate)
      if (val <= 0) return

      const pnlPct = getHoldingPnlPct(h)
      const node = {
        h: h,
        pnlPct: pnlPct,
        value: val
      }

      if (h.combos && h.combos.length > 0) {
        h.combos.forEach((c) => {
          if (!comboHoldings[c.name]) {
            comboHoldings[c.name] = []
          }
          comboHoldings[c.name].push(node)
        })
      } else {
        unclassified.push(node)
      }
    })

    const comboNames = Object.keys(comboHoldings).sort()
    const comboColorMap = {}
    comboNames.forEach((cName, idx) => {
      comboColorMap[cName] = highContrastPalette[idx % highContrastPalette.length]
    })

    const chartData = comboNames.map((cName) => {
      const groupColor = comboColorMap[cName]
      return {
        name: cName,
        children: comboHoldings[cName].map((node) => {
          const comboNames = node.h.combos ? node.h.combos.map((c) => c.name) : []
          const accountName = node.h.account ? node.h.account.name : '默认账户'
          return {
            name: `${node.h.asset ? node.h.asset.name : node.h.assetId}\n${node.pnlPct >= 0 ? '+' : ''}${node.pnlPct.toFixed(2)}%`,
            value: node.value,
            combos: comboNames,
            account: accountName,
            itemStyle: {
              color: groupColor
            },
            label: {
              show: true,
              formatter: `{b}`
            }
          }
        })
      }
    })

    if (unclassified.length > 0) {
      const unclassifiedColor = highContrastPalette[comboNames.length % highContrastPalette.length]
      chartData.push({
        name: '未分类组合',
        children: unclassified.map((node) => {
          const comboNames = node.h.combos ? node.h.combos.map((c) => c.name) : []
          const accountName = node.h.account ? node.h.account.name : '默认账户'
          return {
            name: `${node.h.asset ? node.h.asset.name : node.h.assetId}\n${node.pnlPct >= 0 ? '+' : ''}${node.pnlPct.toFixed(2)}%`,
            value: node.value,
            combos: comboNames,
            account: accountName,
            itemStyle: {
              color: unclassifiedColor
            },
            label: {
              show: true,
              formatter: `{b}`
            }
          }
        })
      })
    }

    return chartData
  }

  // 3. Group by Account
  const accountHoldings = {}
  props.holdings.forEach((h) => {
    const assetRate = h.asset ? (h.asset.exchangeRate || 1.0) : 1.0
    const val = h.quantity * (h.asset ? h.asset.currentPrice : 0) * (assetRate / baseRate)
    if (val <= 0) return

    const marketName = h.asset ? (marketNames[h.asset.market] || h.asset.market) : '其他'
    const accountName = h.account ? h.account.name : '默认账户'
    const groupName = `${marketName} / ${accountName}`
    if (!accountHoldings[groupName]) {
      accountHoldings[groupName] = []
    }
    accountHoldings[groupName].push({
      h,
      pnlPct: getHoldingPnlPct(h),
      value: val
    })
  })

  const accountNames = Object.keys(accountHoldings).sort()
  return accountNames.map((accountName, idx) => {
    const groupColor = highContrastPalette[idx % highContrastPalette.length]
    return {
      name: accountName,
      children: accountHoldings[accountName].map((node) => {
        const comboNames = node.h.combos ? node.h.combos.map((c) => c.name) : []
        const holdingAccountName = node.h.account ? node.h.account.name : '默认账户'
        return {
          name: `${node.h.asset ? node.h.asset.name : node.h.assetId}\n${node.pnlPct >= 0 ? '+' : ''}${node.pnlPct.toFixed(2)}%`,
          value: node.value,
          combos: comboNames,
          account: holdingAccountName,
          itemStyle: {
            color: groupColor
          },
          label: {
            show: true,
            formatter: `{b}`
          }
        }
      })
    }
  })
}

const renderChart = () => {
  if (!chartRef.value) return

  const isLightTheme = props.theme === 'light-classic'
  const chartTheme = isLightTheme ? null : 'dark'

  if (!chartInstance) {
    chartInstance = echarts.init(chartRef.value, chartTheme)
  }

  const data = buildChartData()

  // Calculate total valuation of currently displayed holdings
  const baseRate = getBaseCurrencyRate(props.baseCurrency, props.holdings)
  let totalValue = 0
  props.holdings.forEach((h) => {
    const assetRate = h.asset ? (h.asset.exchangeRate || 1.0) : 1.0
    const val = h.quantity * (h.asset ? h.asset.currentPrice : 0) * (assetRate / baseRate)
    if (val > 0) {
      totalValue += val
    }
  })

  const labelColor = isLightTheme ? '#1f2937' : '#ffffff'
  const borderColor = isLightTheme ? '#ffffff' : '#111622'

  const option = {
    backgroundColor: 'transparent',
    tooltip: {
      formatter: function (info) {
        if (!info.data || !info.data.combos) {
          return `<div style="font-family: 'Outfit', sans-serif; padding: 4px;"><strong>${info.name}</strong></div>`
        }

        const value = info.value
        const name = info.name.split('\n')[0]
        const symbol = getCurrencySymbol(props.baseCurrency)
        
        // Calculate holding percentage
        const pct = totalValue > 0 ? (value / totalValue) * 100 : 0
        
        // Calculate group total percentage of the entire portfolio when grouping by combo or account
        let groupPctStr = ''
        if ((viewMode.value === 'combo' || viewMode.value === 'account') && info.treePathInfo && info.treePathInfo.length > 2) {
          const totalVal = info.treePathInfo[0].value
          const groupVal = info.treePathInfo[1].value
          if (totalVal > 0) {
            const groupPct = (groupVal / totalVal) * 100
            const groupLabel = viewMode.value === 'account' ? '当前账户占比' : '当前组合占比'
            groupPctStr = `<span style="color: #9ca3af;">${groupLabel}:</span> <span style="font-weight: 600;">${groupPct.toFixed(2)}%</span><br/>`
          }
        }

        const combosStr = info.data.combos.length > 0 ? info.data.combos.join(', ') : '无'
        const accountStr = info.data.account || '默认账户'

        return [
          `<div style="font-family: 'Outfit', sans-serif; padding: 6px; line-height: 1.6;">`,
          `<strong style="font-size: 14px; color: #ffffff;">${name}</strong><br/>`,
          `<span style="color: #9ca3af;">当前市值:</span> <span style="font-weight: 600;">${symbol} ${value.toLocaleString('zh-CN', { minimumFractionDigits: 2 })}</span><br/>`,
          `<span style="color: #9ca3af;">总持仓占比:</span> <span style="font-weight: 600;">${pct.toFixed(2)}%</span><br/>`,
          groupPctStr,
          `<span style="color: #9ca3af;">所属账户:</span> <span style="font-weight: 600;">${accountStr}</span><br/>`,
          `<span style="color: #9ca3af;">所属组合:</span> <span style="font-weight: 600;">${combosStr}</span>`,
          `</div>`
        ].join('')
      }
    },
    series: [
      {
        name: '持仓占比',
        type: 'treemap',
        visibleMin: 300,
        data: data,
        leafDepth: 2,
        roam: false,
        label: {
          show: true,
          fontFamily: 'Inter, sans-serif',
          fontSize: 12,
          fontWeight: 'bold',
          lineHeight: 18,
          align: 'center',
          verticalAlign: 'middle'
        },
        upperLabel: {
          show: true,
          height: 25,
          fontFamily: 'Outfit, sans-serif',
          color: labelColor,
          fontWeight: '600',
          fontSize: 12
        },
        itemStyle: {
          borderColor: borderColor,
          borderWidth: 2,
          gapWidth: 1
        },
        levels: [
          {
            itemStyle: {
              borderColor: borderColor,
              borderWidth: 4,
              gapWidth: 4
            }
          },
          {
            imageStyle: {
              borderColor: borderColor,
              borderWidth: 2,
              gapWidth: 2
            }
          }
        ]
      }
    ]
  }

  chartInstance.setOption(option)
}

const handleResize = () => {
  if (chartInstance) {
    chartInstance.resize()
  }
}

watch([() => props.holdings, () => props.colorConvention, () => props.baseCurrency], () => {
  renderChart()
}, { deep: true })

watch(() => props.theme, () => {
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }
  renderChart()
})

onMounted(() => {
  renderChart()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }
})
</script>

<style scoped>
.chart-container {
  padding: 20px;
  margin-bottom: 25px;
  display: flex;
  flex-direction: column;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.chart-header h3 {
  color: var(--text-primary);
  font-size: 16px;
  font-weight: 600;
}

.treemap-chart {
  width: 100%;
  height: 520px;
  transition: height 0.3s ease;
}

@media (max-width: 768px) {
  .treemap-chart {
    height: 380px;
  }
}
</style>
