# FolioTrack 服务器部署说明

本文档适用于服务器上已经有一个公网 `nginx` 容器，并且该容器已经绑定了宿主机的 `80:80` 和 `443:443` 端口的场景。

FolioTrack 的 MySQL、Redis、后端和前端不会直接暴露到公网；公网流量统一进入已有的 `nginx` 容器，再通过 Docker 外部网络 `web` 转发到 FolioTrack 前端容器。

以下示例域名为：

```text
track.example.com
```

如果你使用其他域名，请把文档和 nginx 配置文件中的 `track.example.com` 整体替换为你的真实域名。

也可以使用脚本一键替换：

```bash
sh deploy/scripts/set-domain.sh track.your-domain.com
```

## 1. 配置 DNS 解析

先在域名服务商处添加 A 记录：

```text
track.example.com -> 你的服务器公网 IP
```

可以用下面命令检查解析是否生效：

```bash
ping track.example.com
```

## 2. 创建 Docker 共享网络

在服务器上执行一次即可：

```bash
docker network create web
```

如果提示网络已经存在，可以忽略。

## 3. 修改现有公网 nginx 的 compose 配置

编辑服务器上的 `/home/ubuntu/docker-compose.yml`，给已有的 `nginx` 服务添加 `web` 网络。

示例：

```yaml
  nginx:
    image: nginx:stable
    container_name: nginx
    restart: unless-stopped
    depends_on:
      - cli-proxy-api
      # - 3x-ui
      - subconverter
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ~/nginx/conf.d:/etc/nginx/conf.d
      - ~/certbot/www:/var/www/certbot
      - ~/certbot/conf:/etc/letsencrypt
    networks:
      - default
      - web
```

然后在同一个 compose 文件底部添加：

```yaml
networks:
  web:
    external: true
    name: web
```

重启公网 nginx：

```bash
docker compose up -d --no-deps nginx
```

使用 `--no-deps` 可以避免顺手启动已经停用的 `3x-ui` 服务。

## 4. 配置 FolioTrack 环境变量

进入项目目录，例如：

```bash
cd /home/ubuntu/FolioTrack/deploy
cp .env.example .env
nano .env
```

根据实际情况修改：

```env
MYSQL_DATABASE=foliotrack
MYSQL_ROOT_PASSWORD=change-this-mysql-root-password
JWT_SECRET=change-this-to-a-long-random-secret
ADMIN_PASSWORD=change-this-admin-password
ALLOW_REGISTRATION=true
REGISTRATION_INVITE_CODE=change-this-invite-code
```

建议：

- `MYSQL_ROOT_PASSWORD` 使用字母和数字组合，避免 `@`、`/`、`:` 这类可能影响数据库连接串的字符。
- `JWT_SECRET` 使用足够长的随机字符串。
- `ADMIN_PASSWORD` 是首次初始化时创建的 `admin` 用户密码。
- `ALLOW_REGISTRATION=true` 表示允许注册入口。
- `REGISTRATION_INVITE_CODE` 是公网注册邀请码，请务必改成只有你知道的随机字符串。

## 5. 启动 FolioTrack

```bash
cd /home/ubuntu/FolioTrack/deploy
docker compose up -d --build
```

查看服务状态：

```bash
docker compose ps
```

正常应看到以下容器处于运行状态：

```text
foliotrack-db
foliotrack-redis
foliotrack-backend
foliotrack-frontend
```

## 6. 添加首次使用的 HTTP nginx 配置

证书还不存在时，先复制 HTTP-only 配置，确保 nginx 可以正常启动并完成 ACME 验证：

```bash
cd /home/ubuntu/FolioTrack
cp deploy/nginx/track.example.com.http.conf ~/nginx/conf.d/track.example.com.conf
docker exec nginx nginx -t
docker exec nginx nginx -s reload
```

此时可以先访问：

```text
http://track.example.com
```

## 7. 申请 HTTPS 证书

如果你使用当前服务器已有的 Certbot webroot 挂载方式，可以执行：

```bash
docker run --rm \
  -v ~/certbot/www:/var/www/certbot \
  -v ~/certbot/conf:/etc/letsencrypt \
  certbot/certbot certonly --webroot \
  -w /var/www/certbot \
  -d track.example.com
```

证书生成路径应为：

```text
~/certbot/conf/live/track.example.com/fullchain.pem
~/certbot/conf/live/track.example.com/privkey.pem
```

在 nginx 容器内对应为：

```text
/etc/letsencrypt/live/track.example.com/fullchain.pem
/etc/letsencrypt/live/track.example.com/privkey.pem
```

## 8. 切换到 HTTPS nginx 配置

证书申请成功后，替换为 HTTPS 配置：

```bash
cd /home/ubuntu/FolioTrack
cp deploy/nginx/track.example.com.conf ~/nginx/conf.d/track.example.com.conf
docker exec nginx nginx -t
docker exec nginx nginx -s reload
```

然后访问：

```text
https://track.example.com
```

## 9. 登录系统

默认管理员用户：

```text
用户名：admin
密码：.env 中的 ADMIN_PASSWORD
```

如果数据库里已经存在 `admin` 用户，修改 `.env` 中的 `ADMIN_PASSWORD` 不会自动重置已有密码。

## 常用运维命令

查看 FolioTrack 服务状态：

```bash
cd /home/ubuntu/FolioTrack/deploy
docker compose ps
```

查看后端日志：

```bash
cd /home/ubuntu/FolioTrack/deploy
docker compose logs -f backend
```

查看前端 nginx 日志：

```bash
cd /home/ubuntu/FolioTrack/deploy
docker compose logs -f frontend
```

重启 FolioTrack：

```bash
cd /home/ubuntu/FolioTrack/deploy
docker compose restart
```

停止 FolioTrack：

```bash
cd /home/ubuntu/FolioTrack/deploy
docker compose down
```

清空数据库并重新初始化：

```bash
cd /home/ubuntu/FolioTrack/deploy
docker compose down -v
```

注意：`docker compose down -v` 会删除 MySQL 数据卷，资产数据会被清空。
