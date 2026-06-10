# FolioTrack 本机 Docker 部署说明

本文档适用于只在本机试用或调试 FolioTrack 的场景，不需要公网 nginx，也不需要 HTTPS。

## 启动本机环境

```bash
cd deploy
docker compose -f docker-compose.local.yml up -d --build
```

访问地址：

```text
http://localhost:8088
```

默认管理员账号：

```text
用户名：admin
密码：admin123
```

## 本机端口

- 前端：`8088`
- 后端：`18080`
- MySQL：`13306`
- Redis：`16379`

## 查看服务状态

```bash
cd deploy
docker compose -f docker-compose.local.yml ps
```

## 查看日志

后端日志：

```bash
cd deploy
docker compose -f docker-compose.local.yml logs -f backend
```

前端日志：

```bash
cd deploy
docker compose -f docker-compose.local.yml logs -f frontend
```

## 停止本机环境

```bash
cd deploy
docker compose -f docker-compose.local.yml down
```

## 清空本机数据库

```bash
cd deploy
docker compose -f docker-compose.local.yml down -v
```

注意：`down -v` 会删除本机 MySQL 数据卷，所有本机测试数据都会被清空。
