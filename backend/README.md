# FolioTrack 后端服务

基于 Go (Gin + GORM + MySQL) 开发的高性能 API 引擎，负责用户认证、持仓计算、组合管理以及证券/基金的实时行情解析。

---

## 🛠 技术选型

- **Web 框架**：[Gin Engine](https://github.com/gin-gonic/gin)
- **ORM 框架**：[GORM](https://gorm.io)
- **数据库**：MySQL 8.0
- **数据源**：新浪财经及东方财富公开 API
- **安全**：JWT (JSON Web Token) 鉴权，`bcrypt` 密码单向哈希加密

---

## ⚙️ 环境变量配置

系统支持通过环境变量进行配置：

| 变量名 | 说明 | 默认值 |
| :--- | :--- | :--- |
| `PORT` | 服务启动端口 | `8080` |
| `DB_DSN` | MySQL 连接 DSN | `root:rootpass@tcp(127.0.0.1:3306)/foliotrack?charset=utf8mb4&parseTime=True&loc=Local` |
| `JWT_SECRET` | 签名 Token 的密钥 | `foliotrack-super-secret-key-change-in-prod` |

---

## 💻 本地开发运行

如果在本地不通过 Docker 开发运行后端：

1. **准备 MySQL 数据库**：
   确保本地有一个名为 `foliotrack` 的数据库在运行，并配置对应的账号密码。

2. **下载依赖**：
   ```bash
   go mod download
   ```

3. **运行服务**：
   ```bash
   # Linux / macOS
   DB_DSN="user:password@tcp(localhost:3306)/foliotrack?charset=utf8mb4&parseTime=True&loc=Local" go run main.go
   ```

后端服务将在 `http://localhost:8080` 启动，并自动在数据库中建立所需的表结构。

---

## 📊 行情接口适配细则

系统采用的统一代码映射逻辑如下：
- **A 股**：自动根据代码前缀分包。`6` 开头补 `sh`；`0`/`3` 开头补 `sz`；`8`/`4` 开头补 `bj`（支持北交所）。
- **港股**：自动填充为 5 位数字，附加 `rt_hk` 前缀。
- **美股**：自动转为小写，附加 `gb_` 前缀。
- **基金**：附加 `f_` 前缀。

同时内置了 1 分钟缓存策略。如果某证券的缓存未过期，将直接返回本地 DB 价格，最大限度降低外部接口请求频率。
