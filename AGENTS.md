# Repository Guidelines

## 项目结构与模块组织

FolioTrack 由 Go 后端和 Vue 前端组成。后端代码位于 `backend/`：`routes/` 负责 HTTP 路由，`controllers/` 负责请求处理，`models/` 负责数据模型与存储连接，`services/` 放业务逻辑，`config/` 放配置加载。前端代码位于 `frontend/src/`：`views/` 放页面，`components/` 放复用组件，`router/` 放路由，`utils/` 放 API 工具，`assets/` 放全局样式。部署相关文件位于 `deploy/`，包括 Docker Compose、MySQL 初始化 SQL、nginx 模板和辅助脚本。

## 构建、测试与开发命令

- `cd backend && go run .`：本地启动后端 API，需要按需设置 `DB_DSN`、`REDIS_ADDR`、`JWT_SECRET` 和 `ADMIN_PASSWORD`。
- `cd backend && go test ./...`：运行全部 Go 测试。
- `cd frontend && npm install`：安装前端依赖。
- `cd frontend && npm run dev`：启动 Vite 开发服务。
- `cd frontend && npm run build`：构建生产前端资源。
- `cd deploy && docker compose up -d --build`：构建并启动完整部署栈。

## 代码风格与命名规范

Go 代码必须使用 `gofmt` 格式化，并保持包职责清晰。命名应清楚表达含义，例如 `PriceFetcher`、`ImportHoldings`；除短循环外避免单字母变量。Vue 组件文件使用 PascalCase，例如 `MetricCards.vue`；页面组件应放在 `views/` 并与路由语义保持一致。JavaScript 使用 ES Modules，保持两空格缩进，工具模块命名应简洁明确，例如 `api.js`。

## 测试规范

后端测试使用 Go 标准库 `testing`。测试文件应与被测代码放在相邻目录，并命名为 `*_test.go`，例如 `backend/services/price_fetcher_test.go`。新增服务逻辑、解析逻辑或边界行为时，应补充聚焦的单元测试。提交后端改动前运行 `go test ./...`。前端目前未配置测试框架，除非任务明确需要，不要引入新的测试框架。

## 提交与 Pull Request 规范

当前提交历史采用类似 Conventional Commits 的格式，常见形式为 `feat: ...`。建议使用 `feat:`、`fix:`、`docs:`、`refactor:`、`test:` 等前缀。Pull Request 应包含简短变更说明、已运行的验证命令、配置或迁移注意事项；涉及 UI 变化时应附截图。有关联 issue 时请在 PR 中链接。

## 安全与配置建议

不要提交 `.env`、生成的证书、数据库卷、本地运行数据或真实生产域名。公开示例统一使用 `track.example.com`；部署到服务器副本时，再运行 `sh deploy/scripts/set-domain.sh your.domain.com` 替换为真实域名。生产环境必须使用强 `JWT_SECRET`、`ADMIN_PASSWORD` 和数据库密码。
