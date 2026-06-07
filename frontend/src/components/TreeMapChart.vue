<template>
  <div class="chart-container glass-panel">
    <div class="chart-header">
      <h3>持仓云图</h3>
      <div class="chart-controls">
        <el-radio-group v-model="viewMode" size="small" @change="renderChart">
          <el-radio-button value="market">按市场分组</el-radio-button>
          <el-radio-button value="combo">按组合分组</el-radio-button>
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
const viewMode = ref('market') // 'market' or 'combo'

// Format market names
const marketNames = {
  'A-share': '中国A股',
  'HK-stock': '香港港股',
  'US-stock': '美国美股',
  'Fund': '中国基金'
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
      // Assign distinct rotating colors for adjacent items in the same market block
      const colorIndex = groups[market].length
      const color = highContrastPalette[colorIndex % highContrastPalette.length]

      groups[market].push({
        name: `${h.asset ? h.asset.name : h.assetId}\n${pnlPct >= 0 ? '+' : ''}${pnlPct.toFixed(2)}%`,
        value: val,
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

  } else {
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

    // Map each group name to a specific rotating color to keep adjacent combo groups distinct
    const comboNames = Object.keys(comboHoldings).sort()
    const comboColorMap = {}
    comboNames.forEach((cName, idx) => {
      comboColorMap[cName] = highContrastPalette[idx % highContrastPalette.length]
    })

    const chartData = comboNames.map((cName) => {
      const groupColor = comboColorMap[cName]
      return {
        name: cName,
        children: comboHoldings[cName].map((node) => ({
          name: `${node.h.asset ? node.h.asset.name : node.h.assetId}\n${node.pnlPct >= 0 ? '+' : ''}${node.pnlPct.toFixed(2)}%`,
          value: node.value,
          itemStyle: {
            color: groupColor
          },
          label: {
            show: true,
            formatter: `{b}`
          }
        }))
      }
    })

    if (unclassified.length > 0) {
      const unclassifiedColor = highContrastPalette[comboNames.length % highContrastPalette.length]
      chartData.push({
        name: '未分类组合',
        children: unclassified.map((node) => ({
          name: `${node.h.asset ? node.h.asset.name : node.h.assetId}\n${node.pnlPct >= 0 ? '+' : ''}${node.pnlPct.toFixed(2)}%`,
          value: node.value,
          itemStyle: {
            color: unclassifiedColor
          },
          label: {
            show: true,
            formatter: `{b}`
          }
        }))
      })
    }

    return chartData
  }
}

const renderChart = () => {
  if (!chartRef.value) return

  const isLightTheme = props.theme === 'light-ice'
  const chartTheme = isLightTheme ? null : 'dark'

  if (!chartInstance) {
    chartInstance = echarts.init(chartRef.value, chartTheme)
  }

  const data = buildChartData()

  const labelColor = isLightTheme ? '#1f2937' : '#ffffff'
  const borderColor = isLightTheme ? '#ffffff' : '#111622'

  const option = {
    backgroundColor: 'transparent',
    tooltip: {
      formatter: function (info) {
        const value = info.value
        const name = info.name.split('\n')[0]
        const symbol = getCurrencySymbol(props.baseCurrency)
        return [
          `<div style="font-family: 'Outfit', sans-serif; padding: 4px;">`,
          `<strong style="font-size: 14px;">${name}</strong><br/>`,
          `市值: ${symbol} ${value.toLocaleString('zh-CN', { minimumFractionDigits: 2 })}`,
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
  height: 380px;
}
</style>
