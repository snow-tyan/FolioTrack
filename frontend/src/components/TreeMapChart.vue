<template>
  <div class="chart-container glass-panel">
    <div class="chart-header">
      <h3>持仓云图 (矩形树图)</h3>
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

// Helper to get color based on P&L percentage and convention
const getColorByPnl = (pnlPct, convention) => {
  const maxPct = 15.0 // Cap at +15% / -15% for color scaling
  let factor = Math.min(Math.abs(pnlPct) / maxPct, 1.0) // 0 to 1

  const isUp = pnlPct >= 0
  const isCn = convention === 'CN'

  // HSL colors
  // Rises: CN = Red (Hue 0), US = Green (Hue 140)
  // Falls: CN = Green (Hue 140), US = Red (Hue 0)
  let hue = 0
  if ((isUp && !isCn) || (!isUp && isCn)) {
    hue = 142 // Green
  } else {
    hue = 0 // Red
  }

  // S: Saturation, L: Lightness
  // If factor is near 0 (0% return), we want it to blend with the dark background
  // Low change: low saturation (20%), low lightness (25%)
  // High change: high saturation (80%), medium lightness (45%)
  const s = 30 + factor * 55 // 30% -> 85%
  const l = 20 + factor * 25 // 20% -> 45%

  return `hsl(${hue}, ${s}%, ${l}%)`
}

const buildChartData = () => {
  if (!props.holdings || props.holdings.length === 0) {
    return []
  }

  if (viewMode.value === 'market') {
    // 1. Group by Market
    const groups = {}
    props.holdings.forEach((h) => {
      const market = h.asset ? h.asset.market : 'Other'
      const val = h.quantity * (h.asset ? h.asset.currentPrice : 0)
      if (val <= 0) return

      if (!groups[market]) {
        groups[market] = []
      }

      const pnlPct = getHoldingPnlPct(h)
      const color = getColorByPnl(pnlPct, props.colorConvention)

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
    const comboGroups = {}
    const unclassified = []

    props.holdings.forEach((h) => {
      const val = h.quantity * (h.asset ? h.asset.currentPrice : 0)
      if (val <= 0) return

      const pnlPct = getHoldingPnlPct(h)
      const color = getColorByPnl(pnlPct, props.colorConvention)

      const node = {
        name: `${h.asset ? h.asset.name : h.assetId}\n${pnlPct >= 0 ? '+' : ''}${pnlPct.toFixed(2)}%`,
        value: val,
        itemStyle: {
          color: color
        },
        label: {
          show: true,
          formatter: `{b}`
        }
      }

      if (h.combos && h.combos.length > 0) {
        h.combos.forEach((c) => {
          if (!comboGroups[c.name]) {
            comboGroups[c.name] = []
          }
          // We can duplicate stock under each combo it belongs to
          comboGroups[c.name].push(node)
        })
      } else {
        unclassified.push(node)
      }
    })

    const chartData = Object.keys(comboGroups).map((comboName) => ({
      name: comboName,
      children: comboGroups[comboName]
    }))

    if (unclassified.length > 0) {
      chartData.push({
        name: '未分类组合',
        children: unclassified
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
        return [
          `<div style="font-family: 'Outfit', sans-serif; padding: 4px;">`,
          `<strong style="font-size: 14px;">${name}</strong><br/>`,
          `市值: ¥ ${value.toLocaleString('zh-CN', { minimumFractionDigits: 2 })}`,
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

watch([() => props.holdings, () => props.colorConvention], () => {
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
