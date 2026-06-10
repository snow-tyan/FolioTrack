<template>
  <div class="login-wrapper">
    <!-- Glowing background elements -->
    <div class="glow-sphere sphere-1"></div>
    <div class="glow-sphere sphere-2"></div>

    <div class="login-container glass-panel animate-fade-in">
      <div class="login-header">
        <h2>FolioTrack</h2>
        <p>{{ isRegister ? '注册新账号' : '登录您的资产账户' }}</p>
      </div>

      <el-form :model="form" :rules="rules" ref="formRef" label-position="top" @keyup.enter="handleSubmit">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="请输入用户名" prefix-icon="User" />
        </el-form-item>
        
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" placeholder="请输入密码" show-password prefix-icon="Lock" />
        </el-form-item>

        <el-form-item v-if="isRegister" label="确认密码" prop="confirmPassword">
          <el-input v-model="form.confirmPassword" type="password" placeholder="请再次输入密码" show-password prefix-icon="Lock" />
        </el-form-item>

        <el-form-item v-if="isRegister && requireInviteCode" label="邀请码" prop="inviteCode">
          <el-input v-model="form.inviteCode" placeholder="请输入管理员提供的邀请码" prefix-icon="Key" />
        </el-form-item>

        <div class="action-buttons">
          <el-button type="primary" :loading="loading" @click="handleSubmit" class="submit-btn">
            {{ isRegister ? '注 册' : '登 录' }}
          </el-button>
        </div>
      </el-form>

      <div v-if="allowRegistration" class="toggle-mode">
        <span>{{ isRegister ? '已有账号？' : '还没有账号？' }}</span>
        <a href="#" @click.prevent="toggleMode">{{ isRegister ? '立即登录' : '立即注册' }}</a>
      </div>
      <div v-else class="toggle-mode">
        <span>本站未开放公开注册，请联系管理员开通账号</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '../utils/api'

const router = useRouter()
const formRef = ref(null)
const isRegister = ref(false)
const loading = ref(false)
const allowRegistration = ref(false)
const requireInviteCode = ref(false)

const form = reactive({
  username: '',
  password: '',
  confirmPassword: '',
  inviteCode: ''
})

const validatePass2 = (rule, value, callback) => {
  if (value === '') {
    callback(new Error('请再次输入密码'))
  } else if (value !== form.password) {
    callback(new Error('两次输入密码不一致!'))
  } else {
    callback()
  }
}

const validateInviteCode = (rule, value, callback) => {
  if (isRegister.value && requireInviteCode.value && !value) {
    callback(new Error('请输入邀请码'))
  } else {
    callback()
  }
}

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '长度在 3 到 20 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码不能少于 6 个字符', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: validatePass2, trigger: 'blur' }
  ],
  inviteCode: [
    { validator: validateInviteCode, trigger: 'blur' }
  ]
}

const toggleMode = () => {
  if (!allowRegistration.value) return
  isRegister.value = !isRegister.value
  if (formRef.value) formRef.value.resetFields()
}

onMounted(async () => {
  try {
    const config = await api.get('/system/config')
    allowRegistration.value = config.allowRegistration !== false
    requireInviteCode.value = config.requireInviteCode === true
  } catch (err) {
    allowRegistration.value = false
    requireInviteCode.value = false
  }
})

const handleSubmit = () => {
  if (!formRef.value) return
  formRef.value.validate(async (valid) => {
    if (valid) {
      loading.value = true
      try {
        if (isRegister.value) {
          if (!allowRegistration.value) {
            ElMessage.warning('本站未开放公开注册，请联系管理员')
            return
          }
          // Register flow
          await api.post('/auth/register', {
            username: form.username,
            password: form.password,
            inviteCode: form.inviteCode
          })
          ElMessage.success('注册成功，请登录！')
          isRegister.value = false
          form.confirmPassword = ''
          form.inviteCode = ''
        } else {
          // Login flow
          const data = await api.post('/auth/login', {
            username: form.username,
            password: form.password
          })
          localStorage.setItem('token', data.token)
          localStorage.setItem('user', JSON.stringify(data.user))
          ElMessage.success('登录成功')
          router.push({ name: 'Holdings' })
        }
      } catch (err) {
        ElMessage.error(err.message || '操作失败')
      } finally {
        loading.value = false
      }
    }
  })
}
</script>

<style scoped>
.login-wrapper {
  position: relative;
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background-color: #05070a;
  overflow: hidden;
  padding: 20px;
}

.glow-sphere {
  position: absolute;
  border-radius: 50%;
  filter: blur(100px);
  z-index: 1;
  opacity: 0.15;
}

.sphere-1 {
  width: 400px;
  height: 400px;
  background: var(--color-primary);
  top: 15%;
  left: 20%;
}

.sphere-2 {
  width: 350px;
  height: 350px;
  background: #ec4899; /* Pink */
  bottom: 15%;
  right: 20%;
}

.login-container {
  width: 100%;
  max-width: 420px;
  padding: 40px;
  z-index: 2;
  position: relative;
}

.login-header {
  text-align: center;
  margin-bottom: 30px;
}

.login-header h2 {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: 0.05em;
  background: linear-gradient(to right, #ffffff, var(--color-primary));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  margin-bottom: 6px;
}

.login-header p {
  color: var(--text-secondary);
  font-size: 14px;
}

.action-buttons {
  margin-top: 25px;
}

.submit-btn {
  width: 100%;
  height: 42px;
  font-size: 15px;
  letter-spacing: 0.2em;
}

.toggle-mode {
  text-align: center;
  margin-top: 20px;
  font-size: 13px;
  color: var(--text-secondary);
}

.toggle-mode a {
  color: var(--color-primary);
  text-decoration: none;
  font-weight: 500;
  margin-left: 6px;
}

.toggle-mode a:hover {
  text-decoration: underline;
}
</style>
