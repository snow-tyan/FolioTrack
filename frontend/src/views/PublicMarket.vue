<template>
  <div class="public-market animate-fade-in">
    <div class="market-header glass-panel animate-slide-down">
      <h3>公开共享市场</h3>
      <p>在此查看其他用户主动公开分享的资产配置。您可以点击“查看持仓”来查看其公开的个股树状热力图、市值构成及明细。</p>
    </div>

    <div class="market-content glass-panel animate-fade-in" style="animation-delay: 0.1s;">
      <el-table :data="groupedUsers" v-loading="loading" style="width: 100%" class="custom-table">
        <!-- Username -->
        <el-table-column label="分享者" min-width="140">
          <template #default="scope">
            <div class="user-cell">
              <el-avatar :size="32" class="user-avatar">{{ scope.row.username.charAt(0).toUpperCase() }}</el-avatar>
              <span class="user-name">{{ scope.row.username }}</span>
            </div>
          </template>
        </el-table-column>

        <!-- Role -->
        <el-table-column label="系统角色" width="140">
          <template #default="scope">
            <el-tag :type="getRoleTagType(scope.row.role)" size="small" effect="dark" round>
              {{ getRoleName(scope.row.role) }}
            </el-tag>
          </template>
        </el-table-column>

        <!-- Public Holdings Count -->
        <el-table-column label="公开持仓只数" width="120" align="center">
          <template #default="scope">
            <span class="font-outfit count-badge">{{ scope.row.holdings.length }} 只</span>
          </template>
        </el-table-column>

        <!-- Public Combos -->
        <el-table-column label="涉及公开投资组合" min-width="200">
          <template #default="scope">
            <div class="combos-wrapper">
              <el-tag 
                v-for="name in scope.row.comboNames" 
                :key="name" 
                size="small" 
                type="primary" 
                effect="plain"
                class="market-combo-tag"
              >
                {{ name }}
              </el-tag>
              <span v-if="scope.row.comboNames.length === 0" class="no-combos">无组合关联</span>
            </div>
          </template>
        </el-table-column>

        <!-- Actions -->
        <el-table-column label="操作" width="140" align="center" fixed="right">
          <template #default="scope">
            <el-button type="primary" size="small" icon="View" class="view-btn" @click="viewUserHoldings(scope.row)">
              查看持仓
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- View User Public Holdings Drawer -->
    <el-drawer
      v-model="drawerVisible"
      :title="'用户 ' + selectedUser?.username + ' 的公开持仓看板'"
      size="65%"
      destroy-on-close
      append-to-body
      :custom-class="'public-drawer ' + props.theme"
      @opened="onDrawerOpened"
      @closed="onDrawerClosed"
    >
      <div style="min-height: 200px; padding: 10px 20px;" class="drawer-content">
        <template v-if="isDrawerOpened && selectedUser?.holdings && selectedUser.holdings.length > 0">
          <!-- Metric summary of their public holdings -->
          <MetricCards 
            :holdings="sortedDrawerHoldings" 
            :color-convention="colorConvention" 
            :base-currency="baseCurrency" 
            style="margin-bottom: 20px;" 
          />

          <!-- TreeMap of their public holdings -->
          <TreeMapChart 
            :holdings="sortedDrawerHoldings" 
            :color-convention="colorConvention" 
            :theme="theme" 
            :base-currency="baseCurrency" 
            style="height: 420px; margin-bottom: 24px; border-radius: 12px; overflow: hidden;" 
          />

          <!-- Table of their public holdings -->
          <el-table :data="sortedDrawerHoldings" style="width: 100%" border size="small" class="drawer-table">
            <el-table-column label="标的名称 / 代码" min-width="150">
              <template #default="scope">
                <div class="asset-info-small">
                  <span class="asset-name-small">{{ scope.row.asset ? scope.row.asset.name : scope.row.assetId }}</span>
                  <span class="asset-symbol-small font-outfit">{{ scope.row.asset ? scope.row.asset.symbol : '' }}</span>
                </div>
              </template>
            </el-table-column>
            
            <el-table-column label="所属市场" width="100" align="center">
              <template #default="scope">
                <el-tag :type="getMarketTagType(scope.row.asset?.market)" size="small" effect="dark">
                  {{ getMarketName(scope.row.asset?.market) }}
                </el-tag>
              </template>
            </el-table-column>

            <el-table-column label="账户" min-width="100">
              <template #default="scope">
                <span>{{ scope.row.account ? scope.row.account.name : '默认账户' }}</span>
              </template>
            </el-table-column>
            
            <el-table-column label="持仓数量" align="right" width="100">
              <template #default="scope">
                <span class="font-outfit">{{ formatFloat(scope.row.quantity, 4) }}</span>
              </template>
            </el-table-column>
            
            <el-table-column label="均价" align="right" width="110">
              <template #default="scope">
                <span class="font-outfit">{{ getCurrencySymbol(scope.row.asset?.currency) }}{{ formatFloat(scope.row.costPrice, 4) }}</span>
              </template>
            </el-table-column>
            
            <el-table-column label="当前价" align="right" width="110">
              <template #default="scope">
                <span class="font-outfit">{{ getCurrencySymbol(scope.row.asset?.currency) }}{{ formatFloat(scope.row.asset ? scope.row.asset.currentPrice : 0, 4) }}</span>
              </template>
            </el-table-column>
            
            <el-table-column label="当前市值" align="right" width="130">
              <template #default="scope">
                <span class="font-outfit text-white font-bold">
                  {{ getCurrencySymbol(scope.row.asset?.currency) }}{{ formatMoney(scope.row.quantity * (scope.row.asset ? scope.row.asset.currentPrice : 0)) }}
                </span>
              </template>
            </el-table-column>
            
            <el-table-column label="估算回报" align="right" width="110">
              <template #default="scope">
                <span class="font-outfit font-bold" :class="getPnlClass(getPnlVal(scope.row))">
                  {{ getPnlVal(scope.row) > 0 ? '+' : '' }}{{ formatFloat(getPnlPct(scope.row), 2) }}%
                </span>
              </template>
            </el-table-column>
          </el-table>
        </template>
        <el-empty v-else-if="isDrawerOpened" description="该用户暂无公开持仓数据" />
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../utils/api'
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

const loading = ref(false)
const holdings = ref([])
const drawerVisible = ref(false)
const isDrawerOpened = ref(false)
const selectedUser = ref(null)

const onDrawerOpened = () => {
  isDrawerOpened.value = true
}

const onDrawerClosed = () => {
  isDrawerOpened.value = false
}

const fetchPublicHoldings = async () => {
  loading.value = true
  try {
    const data = await api.get('/market/public/holdings')
    holdings.value = data
  } catch (err) {
    ElMessage.error(err.message || '获取公开持仓数据失败')
  } finally {
    loading.value = false
  }
}

// Group public holdings by User
const groupedUsers = computed(() => {
  const userMap = {}
  holdings.value.forEach(h => {
    if (!h.user) return
    const userId = h.user.id
    if (!userMap[userId]) {
      userMap[userId] = {
        user: h.user,
        holdings: [],
        combos: new Set()
      }
    }
    userMap[userId].holdings.push(h)
    if (h.combos) {
      h.combos.forEach(c => {
        userMap[userId].combos.add(c.name)
      })
    }
  })
  
  return Object.values(userMap).map(u => ({
    ...u.user,
    holdings: u.holdings,
    comboNames: Array.from(u.combos)
  }))
})

// Sort drawer holdings by combo and individual market value
const sortedDrawerHoldings = computed(() => {
  if (!selectedUser.value || !selectedUser.value.holdings) return []
  
  const list = [...selectedUser.value.holdings]
  
  const getValCNY = (h) => {
    const price = h.asset ? h.asset.currentPrice : 0
    const rate = h.asset ? (h.asset.exchangeRate || 1.0) : 1.0
    return h.quantity * price * rate
  }
  
  const processed = list.map(h => {
    const valCNY = getValCNY(h)
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
  processed.forEach(item => {
    if (item.primaryCombo) {
      comboValuations[item.primaryCombo] = (comboValuations[item.primaryCombo] || 0) + item.valCNY
    }
  })
  
  processed.sort((a, b) => {
    if (a.primaryCombo && b.primaryCombo) {
      if (a.primaryCombo === b.primaryCombo) {
        return b.valCNY - a.valCNY
      }
      const valA = comboValuations[a.primaryCombo]
      const valB = comboValuations[b.primaryCombo]
      if (valB !== valA) {
        return valB - valA
      }
      return a.primaryCombo.localeCompare(b.primaryCombo)
    }
    
    if (a.primaryCombo && !b.primaryCombo) return -1
    if (!a.primaryCombo && b.primaryCombo) return 1
    
    return b.valCNY - a.valCNY
  })
  
  return processed.map(item => item.holding)
})

const viewUserHoldings = (userRow) => {
  selectedUser.value = userRow
  drawerVisible.value = true
}

const getRoleName = (role) => {
  const roles = {
    'admin': '超级管理员',
    'manager': '管理员',
    'advanced': '高级用户',
    'user': '普通用户'
  }
  return roles[role] || '普通用户'
}

const getRoleTagType = (role) => {
  const types = {
    'admin': 'danger',
    'manager': 'warning',
    'advanced': 'success',
    'user': 'info'
  }
  return types[role] || 'info'
}

const getPnlVal = (h) => {
  const currPrice = h.asset ? h.asset.currentPrice : 0
  return h.quantity * (currPrice - h.costPrice)
}

const getPnlPct = (h) => {
  if (h.costPrice === 0) return 0
  const currPrice = h.asset ? h.asset.currentPrice : 0
  return ((currPrice - h.costPrice) / h.costPrice) * 100
}

const getCurrencySymbol = (currency) => {
  const symbols = {
    'CNY': '¥',
    'USD': '$',
    'HKD': 'HK$'
  }
  return symbols[currency] || '¥'
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

const formatMoney = (val) => {
  if (val === undefined || val === null || isNaN(val)) return '0.00'
  return val.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

const formatFloat = (val, dec) => {
  if (val === undefined || val === null || isNaN(val)) return '0.00'
  return val.toFixed(dec)
}

onMounted(() => {
  fetchPublicHoldings()
})
</script>

<style scoped>
.public-market {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.market-header {
  padding: 24px;
}

.market-header h3 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 6px;
}

.market-header p {
  font-size: 13px;
  color: var(--text-secondary);
}

.market-content {
  padding: 24px;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar {
  background: var(--color-primary-light);
  color: var(--color-primary);
  font-weight: 600;
}

.user-name {
  font-weight: 600;
  color: var(--text-primary);
}

.count-badge {
  font-weight: 600;
  color: var(--color-primary);
  background: var(--color-primary-light);
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
}

.combos-wrapper {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.market-combo-tag {
  border-radius: 6px;
}

.no-combos {
  color: var(--text-muted);
  font-size: 13px;
}

.view-btn {
  border-radius: 8px;
  font-weight: 500;
  padding: 6px 16px;
  transition: all 0.2s;
}

.view-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px var(--color-primary-light);
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

.asset-info-small {
  display: flex;
  flex-direction: column;
}

.asset-name-small {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.asset-symbol-small {
  font-size: 11px;
  color: var(--text-secondary);
}

.drawer-content {
  display: flex;
  flex-direction: column;
}
</style>
