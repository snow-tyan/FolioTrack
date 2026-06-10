import axios from 'axios'
import router from '../router'

const api = axios.create({
  baseURL: '/api',
  timeout: 15000
})

// Request Interceptor: append JWT Token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response Interceptor: handle token expiration or auth errors
api.interceptors.response.use(
  (response) => {
    // Our backend returns standard { code, message, data }
    const res = response.data
    if (res.code !== 0) {
      return Promise.reject(new Error(res.message || 'Error'))
    }
    return res.data
  },
  (error) => {
    if (error.response && error.response.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      router.push({ name: 'Login' })
    }

    const message = error.response?.data?.message || error.message || '请求失败'
    const normalizedError = new Error(message)
    normalizedError.status = error.response?.status
    normalizedError.code = error.response?.data?.code
    normalizedError.response = error.response
    return Promise.reject(normalizedError)
  }
)

export default api
