<template>
  <div class="holdings-view animate-fade-in">
    <!-- Index Ticker -->
    <IndexTicker :color-convention="colorConvention" />

    <!-- Metric Summary Cards -->
    <MetricCards :holdings="filteredHoldings" :color-convention="colorConvention" :base-currency="baseCurrency" />

    <!-- TreeMap Chart -->
    <TreeMapChart :holdings="filteredHoldings" :color-convention="colorConvention" :theme="theme" :base-currency="baseCurrency" />

    <!-- Holdings Table Section -->
    <div class="table-container glass-panel">
      <div class="table-header">
        <div class="left-actions">
          <h3>持仓明细列表</h3>
          <div class="search-inputs">
            <el-input v-model="searchQuery" placeholder="搜索名称 / 代码" prefix-icon="Search" size="small" clearable style="width: 180px;" />
            
            <el-select v-model="marketFilter" placeholder="过滤市场" size="small" clearable style="width: 120px;">
              <el-option value="A-share" label="A股" />
              <el-option value="HK-stock" label="港股" />
              <el-option value="US-stock" label="美股" />
              <el-option value="Fund" label="基金" />
            </el-select>

            <el-select v-model="comboFilter" placeholder="过滤组合" size="small" clearable style="width: 120px;">
              <el-option v-for="combo in filteredCombos" :key="combo.id" :value="combo.id" :label="combo.name" />
            </el-select>
          </div>
        </div>
        
        <el-button type="primary" icon="Plus" size="small" @click="openAddDialog">
          添加持仓
        </el-button>
      </div>

      <el-table :data="filteredHoldings" v-loading="loading" style="width: 100%">
        <!-- Ticker Name & Symbol -->
        <el-table-column label="标的名称 / 代码" min-width="150">
          <template #default="scope">
            <div class="asset-info">
              <span class="asset-name">{{ scope.row.asset ? scope.row.asset.name : scope.row.assetId }}</span>
              <span class="asset-symbol font-outfit">{{ scope.row.asset ? scope.row.asset.symbol : '' }}</span>
            </div>
          </template>
        </el-table-column>

        <!-- Market Tag -->
        <el-table-column label="市场" width="100">
          <template #default="scope">
            <el-tag :type="getMarketTagType(scope.row.asset ? scope.row.asset.market : '')" size="small" effect="dark">
              {{ getMarketName(scope.row.asset ? scope.row.asset.market : '') }}
            </el-tag>
          </template>
        </el-table-column>

        <!-- Quantity -->
        <el-table-column label="持仓数量" align="right" width="100">
          <template #default="scope">
            <span class="font-outfit">{{ formatFloat(scope.row.quantity, 4) }}</span>
          </template>
        </el-table-column>

        <!-- Cost Price -->
        <el-table-column label="成本价" align="right" width="110">
          <template #default="scope">
            <span class="font-outfit">{{ getCurrencySymbol(scope.row.asset?.currency) }}{{ formatFloat(scope.row.costPrice, 4) }}</span>
          </template>
        </el-table-column>

        <!-- Current Price -->
        <el-table-column label="当前价" align="right" width="110">
          <template #default="scope">
            <span class="font-outfit">{{ getCurrencySymbol(scope.row.asset?.currency) }}{{ formatFloat(scope.row.asset ? scope.row.asset.currentPrice : 0, 4) }}</span>
          </template>
        </el-table-column>

        <!-- Valuation -->
        <el-table-column label="当前市值" align="right" min-width="130" sortable :sort-method="sortValuation">
          <template #default="scope">
            <div class="val-cell">
              <span class="font-outfit text-white font-bold">
                {{ getCurrencySymbol(scope.row.asset?.currency) }}{{ formatMoney(scope.row.quantity * (scope.row.asset ? scope.row.asset.currentPrice : 0)) }}
              </span>
              <span class="sub-cny-val font-outfit" v-if="scope.row.asset && scope.row.asset.currency !== baseCurrency">
                ≈ {{ getCurrencySymbol(baseCurrency) }}{{ formatMoney(scope.row.quantity * scope.row.asset.currentPrice * (scope.row.asset.exchangeRate / baseCurrencyRate)) }}
              </span>
            </div>
          </template>
        </el-table-column>

        <!-- P&L -->
        <el-table-column label="持仓盈亏" align="right" min-width="150" sortable :sort-method="sortPnl">
          <template #default="scope">
            <div class="pnl-cell font-outfit" :class="getPnlClass(getPnlVal(scope.row))">
              <span class="pnl-amt">
                {{ getPnlVal(scope.row) > 0 ? '+' : (getPnlVal(scope.row) < 0 ? '-' : '') }}{{ getCurrencySymbol(scope.row.asset?.currency) }}{{ formatMoney(Math.abs(getPnlVal(scope.row))) }}
              </span>
              <span class="pnl-pct">{{ getPnlVal(scope.row) > 0 ? '+' : '' }}{{ formatFloat(getPnlPct(scope.row), 2) }}%</span>
              <span class="sub-cny-val" v-if="scope.row.asset && scope.row.asset.currency !== baseCurrency">
                ≈ {{ getPnlVal(scope.row) > 0 ? '+' : (getPnlVal(scope.row) < 0 ? '-' : '') }}{{ getCurrencySymbol(baseCurrency) }}{{ formatMoney(Math.abs(getPnlVal(scope.row) * (scope.row.asset.exchangeRate / baseCurrencyRate))) }}
              </span>
            </div>
          </template>
        </el-table-column>

        <!-- Combos (Tags) -->
        <el-table-column label="所属组合" min-width="120">
          <template #default="scope">
            <div class="tag-group">
              <el-tag
                v-for="c in scope.row.combos"
                :key="c.id"
                size="small"
                effect="plain"
                :style="{ color: c.color, borderColor: c.color, backgroundColor: c.color + '10' }"
                class="combo-tag"
              >
                {{ c.name }}
              </el-tag>
            </div>
          </template>
        </el-table-column>

        <!-- Actions -->
        <el-table-column label="操作" width="120" fixed="right" align="center">
          <template #default="scope">
            <el-button link type="primary" icon="Edit" @click="openEditDialog(scope.row)">修改</el-button>
            <el-button link type="danger" icon="Delete" @click="handleDelete(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Add/Edit Holding Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑持仓' : '添加新持仓'"
      width="450px"
      destroy-on-close
      align-center
      append-to-body
    >
      <el-form :model="form" :rules="formRules" ref="formRef" label-position="top">
        <el-form-item label="股票/基金代码" prop="symbol">
          <el-input v-model="form.symbol" placeholder="例如: 600519 或 AAPL" :disabled="isEdit" />
        </el-form-item>

        <el-form-item label="资产市场" prop="market">
          <el-select v-model="form.market" placeholder="选择资产所在市场" :disabled="isEdit" style="width: 100%;">
            <el-option value="A-share" label="A股" />
            <el-option value="HK-stock" label="港股" />
            <el-option value="US-stock" label="美股" />
            <el-option value="Fund" label="基金" />
          </el-select>
        </el-form-item>

        <el-form-item label="持仓数量" prop="quantity">
          <el-input-number v-model="form.quantity" :precision="4" :step="100" :min="0.0001" style="width: 100%;" />
        </el-form-item>

        <el-form-item label="持仓均价" prop="costPrice">
          <el-input-number v-model="form.costPrice" :precision="4" :step="1" :min="0.0001" style="width: 100%;" />
        </el-form-item>

        <el-form-item label="归属组合" prop="comboIds">
          <el-select v-model="form.comboIds" multiple placeholder="可选择多个关联组合" style="width: 100%;">
            <el-option v-for="combo in dialogFilteredCombos" :key="combo.id" :value="combo.id" :label="combo.name" />
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取 消</el-button>
          <el-button type="primary" :loading="submitLoading" @click="submitForm">确 定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../utils/api'
import IndexTicker from '../components/IndexTicker.vue'
import MetricCards from '../components/MetricCards.vue'
import TreeMapChart from '../components/TreeMapChart.vue'

const props = defineProps({
  colorConvention: {
    type: String,
    default: 'CN'
  },
  baseCurrency: {
    type: String,
    default: 'CNY'
  },
  theme: {
    type: String,
    default: 'dark-indigo'
  }
})

const holdings = ref([])
const combos = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitLoading = ref(false)
const currentHoldingId = ref(null)

const searchQuery = ref('')
const marketFilter = ref('')
const comboFilter = ref('')

const formRef = ref(null)
const form = reactive({
  symbol: '',
  market: 'A-share',
  quantity: 100,
  costPrice: 10.0,
  comboIds: []
})

const formRules = {
  symbol: [{ required: true, message: '请输入代码', trigger: 'blur' }],
  market: [{ required: true, message: '请选择市场', trigger: 'change' }],
  quantity: [{ required: true, message: '请输入数量', trigger: 'blur' }],
  costPrice: [{ required: true, message: '请输入持仓均价', trigger: 'blur' }]
}

// Filtered combos based on selected market filter
const filteredCombos = computed(() => {
  if (!marketFilter.value) {
    return combos.value
  }
  return combos.value.filter((c) => c.market === marketFilter.value)
})

// Filtered combos in add/edit dialog based on form.market
const dialogFilteredCombos = computed(() => {
  return combos.value.filter((c) => c.market === form.market)
})

// Watchers for resetting / filtering combo choices when market changes
watch(marketFilter, (newMarket) => {
  if (newMarket && comboFilter.value) {
    const selectedCombo = combos.value.find((c) => c.id === comboFilter.value)
    if (selectedCombo && selectedCombo.market !== newMarket) {
      comboFilter.value = ''
    }
  }
})

watch(() => form.market, (newMarket) => {
  if (form.comboIds && form.comboIds.length > 0) {
    form.comboIds = form.comboIds.filter((id) => {
      const combo = combos.value.find((c) => c.id === id)
      return combo && combo.market === newMarket
    })
  }
})

const fetchHoldings = async () => {
  loading.value = true
  try {
    const data = await api.get('/holdings')
    holdings.value = data
  } catch (err) {
    ElMessage.error(err.message || '获取持仓失败')
  } finally {
    loading.value = false
  }
}

const fetchCombos = async () => {
  try {
    const data = await api.get('/combos')
    combos.value = data
  } catch (err) {
    console.error('Failed to load combos:', err)
  }
}

// Filtered Holdings computed property
const filteredHoldings = computed(() => {
  const list = holdings.value.filter((h) => {
    // 1. Search Query filter (match symbol or name)
    const matchSearch = searchQuery.value
      ? (h.asset && (
          h.asset.symbol.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
          h.asset.name.toLowerCase().includes(searchQuery.value.toLowerCase())
        ))
      : true

    // 2. Market filter
    const matchMarket = marketFilter.value
      ? (h.asset && h.asset.market === marketFilter.value)
      : true

    // 3. Combo filter
    const matchCombo = comboFilter.value
      ? h.combos && h.combos.some((c) => c.id === comboFilter.value)
      : true

    return matchSearch && matchMarket && matchCombo
  })

  // Sort by combos: keep identical combos together, with larger combo total valuation first
  const items = list.map((h) => {
    const qty = h.quantity || 0
    const price = h.asset ? (h.asset.currentPrice || 0) : 0
    const rate = h.asset ? (h.asset.exchangeRate || 1.0) : 1.0
    const valCNY = qty * price * rate

    let primaryCombo = ""
    if (h.combos && h.combos.length > 0) {
      const sortedCombos = [...h.combos].sort((c1, c2) => c1.name.localeCompare(c2.name))
      primaryCombo = sortedCombos[0].name
    }

    return {
      holding: h,
      valCNY,
      primaryCombo
    }
  })

  const comboValuations = {}
  items.forEach((item) => {
    if (item.primaryCombo) {
      comboValuations[item.primaryCombo] = (comboValuations[item.primaryCombo] || 0) + item.valCNY
    }
  })

  items.sort((a, b) => {
    if (a.primaryCombo && b.primaryCombo) {
      if (a.primaryCombo === b.primaryCombo) {
        return b.valCNY - a.valCNY
      }
      const valA = comboValuations[a.primaryCombo]
      const valB = comboValuations[b.primaryCombo]
      if (valA !== valB) {
        return valB - valA
      }
      return a.primaryCombo.localeCompare(b.primaryCombo)
    }

    if (a.primaryCombo && !b.primaryCombo) return -1
    if (!a.primaryCombo && b.primaryCombo) return 1

    return b.valCNY - a.valCNY
  })

  return items.map(item => item.holding)
})

const getPnlVal = (h) => {
  const currPrice = h.asset ? h.asset.currentPrice : 0
  return h.quantity * (currPrice - h.costPrice)
}

const getPnlPct = (h) => {
  if (h.costPrice === 0) return 0
  const currPrice = h.asset ? h.asset.currentPrice : 0
  return ((currPrice - h.costPrice) / h.costPrice) * 100
}

const baseCurrencyRate = computed(() => {
  const rates = { CNY: 1.0, USD: 7.20, HKD: 0.92 }
  holdings.value.forEach((h) => {
    if (h.asset && h.asset.currency && h.asset.exchangeRate) {
      rates[h.asset.currency] = h.asset.exchangeRate
    }
  })
  return rates[props.baseCurrency] || 1.0
})

const getCurrencySymbol = (currency) => {
  const symbols = {
    'CNY': '¥',
    'USD': '$',
    'HKD': 'HK$'
  }
  return symbols[currency] || '¥'
}

const sortValuation = (a, b) => {
  const valA = a.quantity * (a.asset ? a.asset.currentPrice : 0) * (a.asset ? (a.asset.exchangeRate || 1.0) : 1.0)
  const valB = b.quantity * (b.asset ? b.asset.currentPrice : 0) * (b.asset ? (b.asset.exchangeRate || 1.0) : 1.0)
  return valA - valB
}

const sortPnl = (a, b) => {
  const pnlA = a.quantity * ((a.asset ? a.asset.currentPrice : 0) - a.costPrice) * (a.asset ? (a.asset.exchangeRate || 1.0) : 1.0)
  const pnlB = b.quantity * ((b.asset ? b.asset.currentPrice : 0) - b.costPrice) * (b.asset ? (b.asset.exchangeRate || 1.0) : 1.0)
  return pnlA - pnlB
}

const formatMoney = (val) => {
  if (val === undefined || val === null || isNaN(val)) return '0.00'
  return val.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

const formatFloat = (val, dec) => {
  if (val === undefined || val === null || isNaN(val)) return '0.00'
  return val.toFixed(dec)
}

const getPnlClass = (val) => {
  if (val > 0) {
    return props.colorConvention === 'CN' ? 'up-cn-text' : 'up-us-text'
  } else if (val < 0) {
    return props.colorConvention === 'CN' ? 'down-cn-text' : 'down-us-text'
  }
  return 'flat-text'
}

const getMarketName = (market) => {
  const names = {
    'A-share': 'A股',
    'HK-stock': '港股',
    'US-stock': '美股',
    'Fund': '基金'
  }
  return names[market] || market
}

const getMarketTagType = (market) => {
  const types = {
    'A-share': 'warning',
    'HK-stock': 'primary',
    'US-stock': 'danger',
    'Fund': 'success'
  }
  return types[market] || 'info'
}

const openAddDialog = () => {
  isEdit.value = false
  dialogVisible.value = true
  form.symbol = ''
  form.market = 'A-share'
  form.quantity = 100
  form.costPrice = 10.0
  form.comboIds = []
}

const openEditDialog = (row) => {
  isEdit.value = true
  currentHoldingId.value = row.id
  dialogVisible.value = true
  
  form.symbol = row.asset ? row.asset.symbol : ''
  form.market = row.asset ? row.asset.market : 'A-share'
  form.quantity = row.quantity
  form.costPrice = row.costPrice
  form.comboIds = row.combos ? row.combos.map((c) => c.id) : []
}

const submitForm = () => {
  if (!formRef.value) return
  formRef.value.validate(async (valid) => {
    if (valid) {
      submitLoading.value = true
      try {
        if (isEdit.value) {
          // Update
          await api.put(`/holdings/${currentHoldingId.value}`, form)
          ElMessage.success('更新成功')
        } else {
          // Create
          await api.post('/holdings', form)
          ElMessage.success('添加成功')
        }
        dialogVisible.value = false
        fetchHoldings()
      } catch (err) {
        ElMessage.error(err.message || '操作失败')
      } finally {
        submitLoading.value = false
      }
    }
  })
}

const handleDelete = (row) => {
  ElMessageBox.confirm(`确定删除 ${row.asset ? row.asset.name : '该持仓'} 吗？`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'danger'
  }).then(async () => {
    try {
      await api.delete(`/holdings/${row.id}`)
      ElMessage.success('删除成功')
      fetchHoldings()
    } catch (err) {
      ElMessage.error(err.message || '删除失败')
    }
  }).catch(() => {})
}

onMounted(() => {
  fetchHoldings()
  fetchCombos()
})
</script>

<style scoped>
.holdings-view {
  display: flex;
  flex-direction: column;
}

.table-container {
  padding: 24px;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.left-actions {
  display: flex;
  align-items: center;
  gap: 20px;
}

.left-actions h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.search-inputs {
  display: flex;
  gap: 10px;
  align-items: center;
}

.asset-info {
  display: flex;
  flex-direction: column;
}

.asset-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.asset-symbol {
  font-size: 12px;
  color: var(--text-secondary);
}

.pnl-cell {
  display: flex;
  flex-direction: column;
}

.pnl-amt {
  font-weight: 600;
  font-size: 14px;
}

.pnl-pct {
  font-size: 12px;
  margin-top: 1px;
}

.val-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.sub-cny-val {
  font-size: 11px;
  color: var(--text-secondary);
  margin-top: 2px;
  font-weight: normal;
}

.tag-group {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.combo-tag {
  border-radius: 4px;
  font-weight: 500;
}

.font-outfit {
  font-family: 'Outfit', sans-serif;
}

.font-bold {
  font-weight: 600;
}

.text-white {
  color: var(--text-primary);
}

/* Up/Down colors mapping */
.up-cn-text, .down-us-text {
  color: var(--color-danger);
}

.down-cn-text, .up-us-text {
  color: var(--color-success);
}

.flat-text {
  color: var(--text-muted);
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
