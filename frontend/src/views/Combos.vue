<template>
  <div class="combos-view glass-panel animate-fade-in">
    <div class="view-header">
      <div class="header-text">
        <h3>组合列表与属性配置</h3>
        <p>自定义投资组合（标签），用于将不同股票和基金归类。云图支持切换到以组合分组的模式查看占比。</p>
      </div>
      <div class="header-actions-row">
        <el-select v-model="marketFilter" placeholder="过滤市场" size="small" clearable style="width: 120px;">
          <el-option value="A-share" label="A股" />
          <el-option value="HK-stock" label="港股" />
          <el-option value="US-stock" label="美股" />
          <el-option value="Fund" label="基金" />
        </el-select>
        <el-button type="primary" icon="Plus" size="small" @click="openAddDialog">
          新建组合
        </el-button>
      </div>
    </div>

    <el-table :data="filteredCombos" v-loading="loading" style="width: 100%">
      <!-- Color dot -->
      <el-table-column label="标识颜色" width="100" align="center">
        <template #default="scope">
          <div class="color-dot-wrapper">
            <span class="color-dot" :style="{ backgroundColor: scope.row.color }"></span>
            <span class="color-hex font-outfit">{{ scope.row.color }}</span>
          </div>
        </template>
      </el-table-column>

      <!-- Combo Name -->
      <el-table-column label="组合名称" min-width="150">
        <template #default="scope">
          <span class="combo-name">{{ scope.row.name }}</span>
        </template>
      </el-table-column>

      <!-- Market Tag -->
      <el-table-column label="所属市场" width="120">
        <template #default="scope">
          <el-tag :type="getMarketTagType(scope.row.market)" size="small" effect="dark">
            {{ getMarketName(scope.row.market) }}
          </el-tag>
        </template>
      </el-table-column>

      <!-- Creation Date -->
      <el-table-column label="创建时间" min-width="150">
        <template #default="scope">
          <span class="created-at font-outfit">{{ formatDate(scope.row.createdAt) }}</span>
        </template>
      </el-table-column>

      <!-- Public column -->
      <el-table-column label="共享" width="100" align="center">
        <template #default="scope">
          <el-tag size="small" :type="scope.row.isPublic ? 'success' : 'info'" effect="dark">
            {{ scope.row.isPublic ? '公开' : '私有' }}
          </el-tag>
        </template>
      </el-table-column>

      <!-- Actions -->
      <el-table-column label="操作" width="180" align="center">
        <template #default="scope">
          <el-button link type="primary" icon="Edit" @click="openEditDialog(scope.row)">编辑</el-button>
          <el-button link type="danger" icon="Delete" @click="handleDelete(scope.row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Add/Edit Combo Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑组合' : '创建新组合'"
      width="400px"
      destroy-on-close
      align-center
      append-to-body
    >
      <el-form :model="form" :rules="formRules" ref="formRef" label-position="top">
        <el-form-item label="所属市场" prop="market">
          <el-select v-model="form.market" placeholder="请选择组合所属市场" :disabled="isEdit" style="width: 100%;">
            <el-option value="A-share" label="A股" />
            <el-option value="HK-stock" label="港股" />
            <el-option value="US-stock" label="美股" />
            <el-option value="Fund" label="基金" />
          </el-select>
        </el-form-item>

        <el-form-item label="组合名称" prop="name">
          <el-input v-model="form.name" placeholder="例如: 科技板块, 股息组合, 核心资产" />
        </el-form-item>

        <el-form-item label="标识颜色" prop="color">
          <div class="color-picker-row">
            <el-color-picker v-model="form.color" predefine="predefinedColors" />
            <span class="color-preview font-outfit">{{ form.color }}</span>
          </div>
        </el-form-item>

        <el-form-item label="公开共享" prop="isPublic">
          <el-switch v-model="form.isPublic" active-text="公开此组合" inactive-text="仅私有" />
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
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../utils/api'

const combos = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitLoading = ref(false)
const currentComboId = ref(null)

const marketFilter = ref('')

const filteredCombos = computed(() => {
  if (!marketFilter.value) {
    return combos.value
  }
  return combos.value.filter((c) => c.market === marketFilter.value)
})

const formRef = ref(null)
const form = reactive({
  name: '',
  color: '#409EFF',
  market: 'A-share',
  isPublic: false
})

const formRules = {
  name: [
    { required: true, message: '请输入组合名称', trigger: 'blur' },
    { max: 50, message: '不能超过 50 个字符', trigger: 'blur' }
  ],
  market: [
    { required: true, message: '请选择所属市场', trigger: 'change' }
  ]
}

const predefinedColors = ref([
  '#409EFF',
  '#67C23A',
  '#E6A23C',
  '#F56C6C',
  '#909399',
  '#9b59b6',
  '#1abc9c',
  '#e67e22',
  '#2ecc71',
  '#34495e',
  '#1abc9c'
])

const fetchCombos = async () => {
  loading.value = true
  try {
    const data = await api.get('/combos')
    combos.value = data
  } catch (err) {
    ElMessage.error(err.message || '获取组合失败')
  } finally {
    loading.value = false
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN', { hour12: false })
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
  form.name = ''
  form.color = '#409EFF'
  form.market = 'A-share'
  form.isPublic = false
}

const openEditDialog = (row) => {
  isEdit.value = true
  currentComboId.value = row.id
  dialogVisible.value = true
  form.name = row.name
  form.color = row.color
  form.market = row.market || 'A-share'
  form.isPublic = row.isPublic || false
}

const submitForm = () => {
  if (!formRef.value) return
  formRef.value.validate(async (valid) => {
    if (valid) {
      submitLoading.value = true
      try {
        if (isEdit.value) {
          await api.put(`/combos/${currentComboId.value}`, form)
          ElMessage.success('更新组合成功')
        } else {
          await api.post('/combos', form)
          ElMessage.success('创建组合成功')
        }
        dialogVisible.value = false
        fetchCombos()
      } catch (err) {
        ElMessage.error(err.message || '操作失败')
      } finally {
        submitLoading.value = false
      }
    }
  })
}

const handleDelete = (row) => {
  ElMessageBox.confirm(`确定删除组合 "${row.name}" 吗？注意：此操作不会删除组合中的持仓标的。`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'danger'
  }).then(async () => {
    try {
      await api.delete(`/combos/${row.id}`)
      ElMessage.success('删除成功')
      fetchCombos()
    } catch (err) {
      ElMessage.error(err.message || '删除失败')
    }
  }).catch(() => {})
}

onMounted(() => {
  fetchCombos()
})
</script>

<style scoped>
.combos-view {
  padding: 24px;
}

.view-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.header-text h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 6px;
}

.header-text p {
  font-size: 13px;
  color: var(--text-secondary);
}

.color-dot-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.color-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  display: inline-block;
  box-shadow: 0 0 8px rgba(0, 0, 0, 0.4);
}

.color-hex {
  font-size: 11px;
  color: var(--text-muted);
}

.combo-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.created-at {
  font-size: 13px;
  color: var(--text-secondary);
}

.color-picker-row {
  display: flex;
  align-items: center;
  gap: 15px;
}

.color-preview {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.font-outfit {
  font-family: 'Outfit', sans-serif;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.header-actions-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
</style>
