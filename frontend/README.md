# FolioTrack 前端应用

基于 Vue 3 (Composition API)、Vite、Element Plus 与 Apache ECharts 构建的资产仪表盘。

---

## 🎨 视觉与体验设计

- **暗黑风与毛玻璃效果**：采用定制的深靛蓝色调，卡片配有 `backdrop-filter` 模糊与微光边缘，极具科技感。
- **自定义涨跌颜色习惯**：支持国内（红涨绿跌）与国际（绿涨红跌）两种偏好。切换后，大盘指数栏、盈亏指标卡、持仓明细以及 ECharts 云图上的渐变色均会实时同步调整。
- **动态树图**：方块大小映射标的市值比例，颜色深度映射持仓的收益率大小。

---

## 📂 项目结构

```
frontend/
├── src/
│   ├── assets/       # 全局样式文件 (main.css 定义暗色变量)
│   ├── utils/        # Axios 封装与请求拦截器 (api.js)
│   ├── router/       # Vue Router 导航守卫与路由表
│   ├── components/   # 公共组件 (TreeMap, IndexTicker, MetricCards)
│   └── views/        # 主页面 (Dashboard, Holdings, Combos, ImportExport, Login)
```

---

## 💻 本地开发调试

1. **安装依赖**：
   ```bash
   npm install
   ```

2. **启动本地开发服务器**：
   ```bash
   npm run dev
   ```

3. **接口代理配置**：
   本地 `vite.config.js` 配置了代理规则，会自动将本地前端的 `/api` 请求代理到 `http://localhost:8080/api`。请确保此时本地 Go 后端已经启动。
