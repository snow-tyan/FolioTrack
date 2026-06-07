<template>
  <div class="users-view glass-panel animate-fade-in">
    <div class="view-header">
      <div class="header-text">
        <h3>人员与权限管理</h3>
        <p>超级管理员和管理员可在此管理所有注册账户、变更角色权限、清零连续登录错误计数（解锁）或重置用户密码。</p>
      </div>
    </div>

    <!-- Users Table -->
    <el-table :data="users" v-loading="loading" style="width: 100%">
      <!-- ID -->
      <el-table-column label="用户 ID" width="90" align="center">
        <template #default="scope">
          <span class="font-outfit font-bold">{{ scope.row.id }}</span>
        </template>
      </el-table-column>

      <!-- Username -->
      <el-table-column label="用户名" min-width="150">
        <template #default="scope">
          <div class="username-cell">
            <span class="username-text">{{ scope.row.username }}</span>
            <el-tag v-if="scope.row.id === currentUserID" size="small" type="success" effect="plain" class="self-tag">自己</el-tag>
          </div>
        </template>
      </el-table-column>

      <!-- Role -->
      <el-table-column label="系统角色" width="160">
        <template #default="scope">
          <el-select
            v-model="scope.row.role"
            size="small"
            style="width: 130px;"
            :disabled="isActionDisabled(scope.row)"
            @change="(val) => handleRoleChange(scope.row, val)"
          >
            <el-option value="user" label="普通用户" />
            <el-option value="advanced" label="高级用户" />
            <el-option value="manager" label="管理员" />
            <el-option value="admin" label="超级管理员" :disabled="currentUserRole !== 'admin'" />
          </el-select>
        </template>
      </el-table-column>

      <!-- Status -->
      <el-table-column label="状态" width="130" align="center">
        <template #default="scope">
          <el-tag size="small" :type="scope.row.isLocked ? 'danger' : 'success'" effect="dark">
            {{ scope.row.isLocked ? '已锁定' : '正常' }}
          </el-tag>
        </template>
      </el-table-column>

      <!-- Failed Attempts -->
      <el-table-column label="登录错误次数" width="120" align="center">
        <template #default="scope">
          <span class="font-outfit" :class="{ 'text-danger': scope.row.loginAttempts > 0 }">
            {{ scope.row.loginAttempts }} / 5
          </span>
        </template>
      </el-table-column>

      <!-- Creation Date -->
      <el-table-column label="注册时间" min-width="180">
        <template #default="scope">
          <span class="created-at font-outfit">{{ formatDate(scope.row.createdAt) }}</span>
        </template>
      </el-table-column>

      <!-- Actions -->
      <el-table-column label="操作" width="280" align="center" fixed="right">
        <template #default="scope">
          <div class="action-buttons">
            <el-button
              v-if="canViewHoldings(scope.row)"
              link
              type="success"
              icon="View"
              @click="viewUserHoldings(scope.row)"
            >
              查看持仓
            </el-button>

            <template v-if="!isActionDisabled(scope.row)">
              <el-button
                v-if="scope.row.isLocked"
                link
                type="warning"
                icon="Unlock"
                @click="handleUnlock(scope.row)"
              >
                解锁
              </el-button>
              <el-button link type="primary" icon="Key" @click="openPasswordDialog(scope.row)">重置密码</el-button>
              <el-button
                link
                type="danger"
                icon="Delete"
                :disabled="scope.row.id === currentUserID"
                @click="handleDelete(scope.row)"
              >
                删除
              </el-button>
            </template>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <!-- Reset Password Dialog -->
    <el-dialog
      v-model="passwordDialogVisible"
      title="重置用户密码"
      width="400px"
      destroy-on-close
      align-center
    >
      <div style="margin-bottom: 15px; color: var(--text-secondary); font-size: 14px;">
        正在为用户 <strong style="color: var(--text-primary);">{{ selectedUser?.username }}</strong> 重置密码：
      </div>
      <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef" label-position="top">
        <el-form-item label="新密码" prop="password">
          <el-input
            v-model="passwordForm.password"
            type="password"
            placeholder="请输入新密码（至少 6 位）"
            show-password
            prefix-icon="Lock"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="passwordDialogVisible = false">取 消</el-button>
          <el-button type="primary" :loading="resetLoading" @click="submitPasswordReset">确 定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- View User Holdings Drawer -->
    <el-drawer
      v-model="drawerVisible"
      :title="'用户 ' + selectedUserForHoldings?.username + ' 的资产持仓看板'"
      size="65%"
      destroy-on-close
      append-to-body
      :custom-class="'user-drawer ' + props.theme"
      @opened="onDrawerOpened"
      @closed="onDrawerClosed"
    >
      <div v-loading="drawerLoading" style="min-height: 200px; padding: 10px 20px;" class="drawer-content">
        <template v-if="isDrawerOpened && sortedUserHoldings.length > 0">
          <!-- User Holdings Metrics Summary -->
          <MetricCards :holdings="sortedUserHoldings" :color-convention="props.colorConvention" :base-currency="props.baseCurrency" style="margin-bottom: 20px;" />

          <!-- User Holdings TreeMap Cloud -->
          <TreeMapChart :holdings="sortedUserHoldings" :color-convention="props.colorConvention" :theme="props.theme" :base-currency="props.baseCurrency" style="height: 380px; margin-bottom: 20px;" />

          <!-- Holdings Table -->
          <el-table :data="sortedUserHoldings" style="width: 100%" border size="small">
            <el-table-column label="标的名称 / 代码" min-width="140">
              <template #default="scope">
                <div class="asset-info-small">
                  <span class="asset-name-small">{{ scope.row.asset ? scope.row.asset.name : scope.row.assetId }}</span>
                  <span class="asset-symbol-small font-outfit">{{ scope.row.asset ? scope.row.asset.symbol : '' }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="数量" align="right" width="90">
              <template #default="scope">
                <span class="font-outfit">{{ formatFloat(scope.row.quantity, 4) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="成本价" align="right" width="100">
              <template #default="scope">
                <span class="font-outfit">{{ getCurrencySymbol(scope.row.asset?.currency) }}{{ formatFloat(scope.row.costPrice, 4) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="当前价" align="right" width="100">
              <template #default="scope">
                <span class="font-outfit">{{ getCurrencySymbol(scope.row.asset?.currency) }}{{ formatFloat(scope.row.asset ? scope.row.asset.currentPrice : 0, 4) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="当前市值" align="right" width="120">
              <template #default="scope">
                <span class="font-outfit text-white font-bold">
                  {{ getCurrencySymbol(scope.row.asset?.currency) }}{{ formatMoney(scope.row.quantity * (scope.row.asset ? scope.row.asset.currentPrice : 0)) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="共享" width="90" align="center">
              <template #default="scope">
                <el-tag size="small" :type="getPublicStatus(scope.row).type" effect="dark">
                  {{ getPublicStatus(scope.row).label }}
                </el-tag>
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
        <el-empty v-else-if="isDrawerOpened && !drawerLoading" description="该用户暂无持仓资产数据" />
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../utils/api'
import MetricCards from '../components/MetricCards.vue'
import TreeMapChart from '../components/TreeMapChart.vue'

const props = defineProps({
  colorConvention: { type: String, default: 'CN' },
  baseCurrency: { type: String, default: 'CNY' },
  theme: { type: String, default: 'dark-indigo' }
})

const users = ref([])
const loading = ref(false)

const currentUserID = ref(0)
const currentUserRole = ref('user')

const passwordDialogVisible = ref(false)
const resetLoading = ref(false)
const selectedUser = ref(null)
const passwordFormRef = ref(null)

const passwordForm = reactive({
  password: ''
})

const passwordRules = {
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于 6 个字符', trigger: 'blur' }
  ]
}

// removed local theme/color refs; using props

const drawerVisible = ref(false)
const isDrawerOpened = ref(false)
const drawerLoading = ref(false)
const selectedUserForHoldings = ref(null)
const userHoldings = ref([])

const onDrawerOpened = () => {
  isDrawerOpened.value = true
}

const onDrawerClosed = () => {
  isDrawerOpened.value = false
}

const sortedUserHoldings = computed(() => {
  if (!userHoldings.value || userHoldings.value.length === 0) return []
  
  const list = [...userHoldings.value]
  
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

const loadCurrentUser = () => {
  const userStr = localStorage.getItem('user')
  if (userStr) {
    try {
      const user = JSON.parse(userStr)
      currentUserID.value = user.id || 0
      currentUserRole.value = user.role || 'user'
    } catch (e) {
      console.error(e)
    }
  }
}

const fetchUsers = async () => {
  loading.value = true
  try {
    const data = await api.get('/users')
    users.value = data
  } catch (err) {
    ElMessage.error(err.message || '获取用户列表失败')
  } finally {
    loading.value = false
  }
}

const isActionDisabled = (row) => {
  if (currentUserRole.value === 'admin') {
    return false
  }
  if (currentUserRole.value === 'manager') {
    return row.role === 'admin' || row.role === 'manager'
  }
  return true
}

const canViewHoldings = (row) => {
  if (currentUserRole.value === 'admin') {
    return true
  }
  if (currentUserRole.value === 'manager') {
    return row.role === 'advanced' || row.role === 'user'
  }
  return false
}

const viewUserHoldings = async (row) => {
  selectedUserForHoldings.value = row
  userHoldings.value = []
  drawerVisible.value = true
  drawerLoading.value = true
  try {
    const data = await api.get(`/users/${row.id}/holdings`)
    userHoldings.value = data
  } catch (err) {
    ElMessage.error(err.message || '获取用户持仓数据失败')
    drawerVisible.value = false
  } finally {
    drawerLoading.value = false
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN', { hour12: false })
}

const handleRoleChange = async (row, newRole) => {
  try {
    await api.put(`/users/${row.id}`, {
      role: newRole,
      isLocked: row.isLocked
    })
    ElMessage.success(`用户 ${row.username} 的角色已更新为 ${newRole}`)
    fetchUsers()
  } catch (err) {
    ElMessage.error(err.message || '更改角色失败')
    fetchUsers()
  }
}

const handleUnlock = async (row) => {
  try {
    await api.post(`/users/${row.id}/unlock`)
    ElMessage.success(`用户 ${row.username} 已成功解锁`)
    fetchUsers()
  } catch (err) {
    ElMessage.error(err.message || '解锁用户失败')
  }
}

const openPasswordDialog = (row) => {
  selectedUser.value = row
  passwordForm.password = ''
  passwordDialogVisible.value = true
}

const submitPasswordReset = () => {
  if (!passwordFormRef.value) return
  passwordFormRef.value.validate(async (valid) => {
    if (valid) {
      resetLoading.value = true
      try {
        await api.post(`/users/${selectedUser.value.id}/reset-password`, {
          password: passwordForm.password
        })
        ElMessage.success(`用户 ${selectedUser.value.username} 密码重置成功`)
        passwordDialogVisible.value = false
        fetchUsers()
      } catch (err) {
        ElMessage.error(err.message || '密码重置失败')
      } finally {
        resetLoading.value = false
      }
    }
  })
}

const handleDelete = (row) => {
  ElMessageBox.confirm(`警告：确定要彻底删除用户 ${row.username} 吗？此操作将级联删除其所有的资产持仓与组合，且不可恢复！`, '删除确认', {
    confirmButtonText: '确定删除',
    cancelButtonText: '取消',
    type: 'danger'
  }).then(async () => {
    try {
      await api.delete(`/users/${row.id}`)
      ElMessage.success('用户已被彻底删除')
      fetchUsers()
    } catch (err) {
      ElMessage.error(err.message || '删除用户失败')
    }
  }).catch(() => {})
}

const formatMoney = (val) => {
  if (val === undefined || val === null || isNaN(val)) return '0.00'
  return val.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

const formatFloat = (val, dec) => {
  if (val === undefined || val === null || isNaN(val)) return '0.00'
  return val.toFixed(dec)
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

onMounted(() => {
  loadCurrentUser()
  fetchUsers()
})
</script>

<style scoped>
.users-view {
  padding: 24px;
}

.view-header {
  margin-bottom: 25px;
}

.header-text h3 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.header-text p {
  font-size: 13px;
  color: var(--text-secondary);
}

.username-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.username-text {
  font-weight: 600;
  color: var(--text-primary);
}

.self-tag {
  font-weight: 500;
}

.font-outfit {
  font-family: 'Outfit', sans-serif;
}

.font-bold {
  font-weight: 600;
}

.created-at {
  color: var(--text-secondary);
  font-size: 13px;
}

.text-danger {
  color: var(--color-danger);
  font-weight: 600;
}

.action-buttons {
  display: flex;
  justify-content: center;
  gap: 12px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.drawer-content {
  overflow-y: auto;
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

.up-cn-text, .down-us-text {
  color: var(--color-danger);
}

.down-cn-text, .up-us-text {
  color: var(--color-success);
}

.flat-text {
  color: var(--text-muted);
}
</style>
