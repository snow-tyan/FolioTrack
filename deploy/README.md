# FolioTrack 部署说明

`deploy/` 目录提供两种 Docker 部署方式：本机试用部署和服务器公网部署。请根据使用场景选择对应文档。

## 本机部署

适用于本机试用、功能验证或开发调试，不需要公网 nginx，也不需要 HTTPS。

- Compose 文件：`docker-compose.local.yml`
- 详细文档：`README.local.md`
- 默认访问地址：`http://localhost:8088`

快速启动：

```bash
cd deploy
docker compose -f docker-compose.local.yml up -d --build
```

默认管理员账号：

```text
用户名：admin
密码：admin123
```

## 服务器部署

适用于已有公网 nginx 容器的服务器部署场景。FolioTrack 服务不直接暴露公网端口，而是通过 Docker 网络交给公网 nginx 反代，并使用 HTTPS 访问。

- Compose 文件：`docker-compose.yml`
- 详细文档：`README.server.md`
- nginx 模板目录：`nginx/`
- 域名替换脚本：`scripts/set-domain.sh`

服务器部署前，请先复制 `.env.example` 为 `.env` 并设置强密码和密钥。

## 常用文件

- `.env.example`：生产环境变量模板。
- `mysql/init.sql`：MySQL 初始化 SQL。
- `nginx/track.example.com.http.conf`：首次申请证书前使用的 HTTP 配置。
- `nginx/track.example.com.conf`：证书申请后使用的 HTTPS 配置。
