# FolioTrack 一键部署服务

本目录包含了使用 Docker Compose 快速部署 FolioTrack 系统所需的所有配置。

---

## 📦 部署架构

- **前端服务** (`frontend`)：Nginx 服务，托管 Vue 3 静态编译文件，并在内部将以 `/api/` 开头的接口请求代理到 `backend` 容器。
- **后端服务** (`backend`)：运行 Go 编译后的二进制服务，监听 `8080` 端口。
- **数据库** (`db`)：MySQL 8.0 容器，自动初始化数据库。

---

## 🛠 部署步骤

1. **准备 Docker 环境**：
   确保您的服务器/主机已安装 Docker 和 Docker Compose。

2. **一键启动**：
   在当前目录下执行：
   ```bash
   docker-compose up -d --build
   ```
   *注意：该命令会就地构建前端与后端容器镜像，并下载 MySQL 镜像。*

3. **进入系统**：
   服务启动后，在浏览器直接访问：`http://localhost`

---

## 📋 常用运维命令

- **查看服务运行状态**：
  ```bash
  docker-compose ps
  ```

- **查看后端服务运行日志**：
  ```bash
  docker-compose logs -f backend
  ```

- **停止并移除所有容器**：
  ```bash
  docker-compose down
  ```

- **重置数据**：
  如果需要清空所有数据库数据并重新开始，可运行：
  ```bash
  docker-compose down -v
  ```
  *(警告：这会删除 Docker 卷 `db_data` 中的所有持久化资产数据！)*
