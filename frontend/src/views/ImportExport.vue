<template>
  <div class="import-export-view animate-fade-in">
    <div class="cards-grid">
      <!-- Import Card -->
      <div class="glass-panel action-card">
        <div class="card-icon">
          <el-icon><Upload /></el-icon>
        </div>
        <div class="card-body">
          <h3>导入持仓数据</h3>
          <p>上传已备好的 CSV 文件导入持仓。若包含系统内未有的股票/基金，后台会自动拉取其名称和最新价格。同一账户内重复制的标的会被覆盖。</p>
          
          <el-upload
            class="csv-uploader"
            drag
            action="/api/holdings/import"
            :headers="uploadHeaders"
            :on-success="handleUploadSuccess"
            :on-error="handleUploadError"
            :before-upload="beforeUpload"
            accept=".csv"
            :show-file-list="false"
          >
            <el-icon class="el-icon--upload"><document-add /></el-icon>
            <div class="el-upload__text">
              将 CSV 文件拖到此处，或<em>点击上传</em>
            </div>
            <template #tip>
              <div class="el-upload__tip text-muted">
                仅支持 .csv 格式的文件，文件大小不超过 5MB
              </div>
            </template>
          </el-upload>

          <div class="template-action">
            <el-button type="info" link icon="Download" @click="downloadTemplate">
              下载 CSV 模板文件
            </el-button>
          </div>
        </div>
      </div>

      <!-- Export Card -->
      <div class="glass-panel action-card">
        <div class="card-icon">
          <el-icon><Download /></el-icon>
        </div>
        <div class="card-body">
          <h3>导出持仓数据</h3>
          <p>将您当前的全部持仓标的、账户、数量、均价以及所属的组合标签导出为 CSV 备份文件，方便在其他设备或 Excel 中分析和查看。</p>
          
          <div class="export-actions">
            <el-button type="primary" size="large" icon="Download" :loading="exporting" @click="handleExport">
              开始导出 CSV
            </el-button>
          </div>

          <div class="template-info">
            <h4>CSV 文件说明：</h4>
            <ul>
              <li><strong>symbol</strong>: 证券或基金代码 (如 600519, AAPL)</li>
              <li><strong>market</strong>: 对应市场 (A-share, HK-stock, US-stock, Fund)</li>
              <li><strong>account_name</strong>: 所属账户；为空时自动归入对应市场的默认账户</li>
              <li><strong>name</strong>: 证券或基金名称 (可选，导入时将被忽略)</li>
              <li><strong>quantity</strong>: 持股数量</li>
              <li><strong>cost_price</strong>: 持仓均价</li>
              <li><strong>combos</strong>: 关联的组合标签，多个用分号 <code>;</code> 隔开</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import axios from 'axios'

const exporting = ref(false)

// Auth header for upload requests since we bypass Axios interceptor by using standard Action parameter in el-upload
const uploadHeaders = computed(() => {
  const token = localStorage.getItem('token')
  return {
    Authorization: token ? `Bearer ${token}` : ''
  }
})

const beforeUpload = (file) => {
  const isCSV = file.type === 'text/csv' || file.name.endsWith('.csv')
  const isLt5M = file.size / 1024 / 1024 < 5

  if (!isCSV) {
    ElMessage.error('上传文件只能是 CSV 格式!')
  }
  if (!isLt5M) {
    ElMessage.error('上传文件大小不能超过 5MB!')
  }
  return isCSV && isLt5M
}

const handleUploadSuccess = (response) => {
  if (response && response.code === 0) {
    ElMessage.success('导入持仓数据成功！')
  } else {
    ElMessage.error(response.message || '导入失败，请检查文件格式')
  }
}

const handleUploadError = (err) => {
  try {
    const errorData = JSON.parse(err.message)
    ElMessage.error(errorData.message || '导入失败')
  } catch (e) {
    ElMessage.error('网络或服务器异常，导入失败')
  }
}

const downloadTemplate = () => {
  // Generate sample CSV text
  const csvContent = 'symbol,market,account_name,name,quantity,cost_price,combos\n600519,A-share,默认账户,贵州茅台,100,1750.50,白酒组合;核心资产\nAAPL,US-stock,默认账户,苹果,50,180.20,美股科技;核心资产\n00700,HK-stock,默认账户,腾讯控股,200,310.00,港股科技\n110011,Fund,默认账户,易方达优质精选,10000,1.854,消费基金\n'
  
  // Write BOM for Excel auto-detection
  const blob = new Blob([new Uint8Array([0xEF, 0xBB, 0xBF]), csvContent], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  
  const link = document.createElement('a')
  link.href = url
  link.setAttribute('download', 'foliotrack_template.csv')
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

const handleExport = async () => {
  exporting.value = true
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get('/api/holdings/export', {
      headers: {
        Authorization: `Bearer ${token}`
      },
      responseType: 'blob'
    })
    
    const blob = new Blob([response.data], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', 'foliotrack_holdings.csv')
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    ElMessage.success('导出数据成功')
  } catch (err) {
    ElMessage.error('导出持仓数据失败')
  } finally {
    exporting.value = false
  }
}
</script>

<style scoped>
.cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 25px;
}

.action-card {
  padding: 30px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.card-icon {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  background: rgba(99, 102, 241, 0.15);
  color: var(--color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  margin-bottom: 20px;
}

.card-body {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.card-body h3 {
  font-size: 16px;
  color: var(--text-primary);
  margin-bottom: 10px;
}

.card-body p {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.6;
  margin-bottom: 25px;
}

.csv-uploader {
  width: 100%;
  max-width: 320px;
  margin-bottom: 20px;
}

/* Override element-plus drag-uploader background */
:deep(.el-upload-dragger) {
  background-color: rgba(255, 255, 255, 0.02) !important;
  border: 1px dashed rgba(255, 255, 255, 0.15) !important;
  border-radius: 12px !important;
  transition: var(--transition-smooth) !important;
}

:deep(.el-upload-dragger:hover) {
  border-color: var(--color-primary) !important;
  background-color: rgba(99, 102, 241, 0.05) !important;
}

.template-action {
  margin-top: 10px;
}

.export-actions {
  margin: 30px 0;
  width: 100%;
}

.export-actions .el-button {
  width: 100%;
  max-width: 260px;
  height: 46px;
}

.template-info {
  width: 100%;
  text-align: left;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 12px;
  padding: 16px;
  margin-top: 15px;
}

.template-info h4 {
  font-size: 13px;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.template-info ul {
  list-style: none;
  font-size: 12px;
  color: var(--text-secondary);
  padding: 0;
}

.template-info li {
  margin-bottom: 6px;
  line-height: 1.5;
}

.template-info code {
  background: rgba(255, 255, 255, 0.1);
  padding: 2px 4px;
  border-radius: 4px;
  font-family: monospace;
}

.text-muted {
  color: var(--text-muted);
}
</style>
