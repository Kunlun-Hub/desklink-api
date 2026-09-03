# DeskLink Docker + MySQL 部署

本方案使用两个独立镜像：

- `ohoimager/desklink-server:1.4.9-secure-tcp`：包含 `hbbs` 与 `hbbr`。
- `ohoimager/desklink-api:1.4.9-session-merge-v6`：包含 Go API、React 管理端与 Web Client。

数据库使用 `mysql:8.4`。生产编排文件是仓库根目录的
`docker-compose.mysql.yml`。

## 1. 服务器要求

- 64 位 Linux（当前发布镜像为 `linux/amd64`）
- Docker Engine 24 或更高版本
- Docker Compose v2
- 一个指向服务器公网 IP 的域名，例如 `desklink.example.com`
- 推荐使用 Nginx、Caddy 或 Traefik 为 API 提供 HTTPS

需要开放：

| 端口 | 协议 | 用途 |
| --- | --- | --- |
| 21115 | TCP | NAT 类型测试 |
| 21116 | TCP + UDP | ID 注册和连接协调 |
| 21117 | TCP | 中继服务 |
| 21118 | TCP | WebSocket ID 服务 |
| 21119 | TCP | WebSocket Relay 服务 |
| 21114 | TCP | API 和管理端；建议只通过 HTTPS 反向代理公开 |

MySQL 默认只映射到宿主机 `127.0.0.1:3306`，不会直接暴露公网。
hbbs 内部 grant 接口只监听 `127.0.0.1:21120`，不要通过防火墙或反向代理公开。

## 2. 准备部署目录

```bash
mkdir -p /opt/desklink
cd /opt/desklink
curl -O https://raw.githubusercontent.com/Kunlun-Hub/desklink-api/master/docker-compose.mysql.yml
curl -o .env https://raw.githubusercontent.com/Kunlun-Hub/desklink-api/master/.env.mysql.example
```

生成三个随机密码：

```bash
openssl rand -hex 32
openssl rand -hex 32
openssl rand -hex 32
```

编辑 `.env`，至少修改：

```dotenv
DESKLINK_PUBLIC_HOST=desklink.example.com
DESKLINK_API_PUBLIC_URL=https://desklink.example.com
DESKLINK_HBBS_INTERNAL_KEY=第一段随机值
MYSQL_ROOT_PASSWORD=第二段随机值
MYSQL_PASSWORD=第三段随机值
```

`.env` 含有数据库密码，不要提交到 Git。

## 3. 启动

```bash
docker compose --env-file .env -f docker-compose.mysql.yml pull
docker compose --env-file .env -f docker-compose.mysql.yml up -d
docker compose --env-file .env -f docker-compose.mysql.yml ps
```

首次启动时：

1. hbbs 在 `desklink_server_data` 卷中生成服务密钥；
2. API 从共享只读卷读取 `id_ed25519.pub`；
3. API 自动初始化 MySQL 表并创建管理员账号；
4. 初始管理员密码打印在 API 日志中。

查看初始密码：

```bash
docker compose --env-file .env -f docker-compose.mysql.yml logs api \
  | grep 'Admin Password Is'
```

管理端地址：

```text
https://desklink.example.com/_admin/
```

## 4. HTTPS 反向代理

只需将公网 HTTPS 域名代理到 `127.0.0.1:21114`。Nginx 示例：

```nginx
server {
    listen 443 ssl http2;
    server_name desklink.example.com;

    ssl_certificate /etc/letsencrypt/live/desklink.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/desklink.example.com/privkey.pem;

    client_max_body_size 100m;

    location / {
        proxy_pass http://127.0.0.1:21114;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

完成 HTTPS 后确认 `.env` 中 `DESKLINK_API_PUBLIC_URL` 与实际外部地址完全一致。
钉钉和企业微信统一回调地址为：

```text
https://desklink.example.com/api/oidc/callback
```

## 5. 常用参数

### 镜像

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `DESKLINK_SERVER_IMAGE` | `ohoimager/desklink-server:1.4.9-secure-tcp` | ID/Relay 镜像 |
| `DESKLINK_API_IMAGE` | `ohoimager/desklink-api:1.4.9-session-merge-v6` | API/管理端镜像 |
| `DESKLINK_JWT_KEY` | 无，必填 | 录像预览、下载和 API JWT 的签名密钥，生产环境使用随机长字符串 |

### 外部服务

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `DESKLINK_PUBLIC_HOST` | 必填 | 客户端可访问的域名或公网 IP，不带端口 |
| `DESKLINK_API_PUBLIC_URL` | 必填 | API 完整外部 URL，推荐 HTTPS |
| `DESKLINK_MUST_LOGIN` | `Y` | 是否必须登录才能发起远控 |
| `DESKLINK_HBBS_INTERNAL_KEY` | 必填 | API 与 hbbs 内部调用共享密钥 |
| `RUST_LOG` | `info` | hbbs/hbbr 日志等级 |
| `TEST_HBBS` | `no` | 是否执行 hbbs 启动自检 |

### MySQL

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `MYSQL_DATABASE` | `desklink` | 数据库名 |
| `MYSQL_USER` | `desklink` | 业务账号 |
| `MYSQL_PASSWORD` | 必填 | 业务账号密码 |
| `MYSQL_ROOT_PASSWORD` | 必填 | root 密码 |
| `MYSQL_BIND_PORT` | `3306` | 宿主机回环映射端口 |
| `MYSQL_TLS` | `false` | MySQL TLS 模式 |
| `MYSQL_MAX_IDLE_CONNS` | `10` | 最大空闲连接数 |
| `MYSQL_MAX_OPEN_CONNS` | `100` | 最大连接数 |

### API 功能

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `DESKLINK_LANG` | `zh-CN` | 默认语言 |
| `DESKLINK_ALLOW_REGISTER` | `false` | 是否允许自主注册 |
| `DESKLINK_SHOW_SWAGGER` | `0` | 是否显示 Swagger |
| `DESKLINK_WEB_CLIENT` | `1` | 是否启用 Web Client |
| `DESKLINK_PERSONAL_ADDRESS_BOOK` | `1` | 是否启用个人/共享地址簿 API |
| `DESKLINK_RECORDING_MAX_CHUNK_SIZE` | `8388608` | 会话录像上传分片上限，单位字节，建议 4-8 MiB |
| `DESKLINK_RECORDING_MOUNT_PATH` | `/opt/desklink/recordings-external` | 宿主机已挂载的 NFS/SMB 目录，映射到容器 `/recordings-external` |

### 会话录像

管理端“安全审计 -> 会话录像”支持全局关闭、全局开启和指定设备三种策略。
录像由被控端直接复用远控链路中的编码帧并封装，不重复采集屏幕：

- VP9/AV1 保存为 WebM，H.264/H.265 保存为 MP4；
- VP9、AV1 和 H.264 文件直接在线预览；
- H.265 原文件保持不变，API 异步生成 H.264 预览副本；
- 上传块使用设备 ID + 安装 UUID 校验，后续请求使用随机上传令牌；
- 播放与下载地址由管理员登录态换取，有效期 5 分钟；
- 录像保留天数在管理端策略中设置，后台每 6 小时执行一次清理；
- 超过 24 小时未完成的上传会标记为失败。
- 同一远程会话的分段使用 `peer_id + from_peer + session_id` 归并；切换显示器或显示参数不会产生最终重复录像。服务端在最后一个分段完成后短暂等待，再使用 FFmpeg 合并并清理分段记录。

管理端可将完成的录像归档到本地目录、FTP、S3 兼容对象存储，或宿主机已挂载的
NFS/SMB 目录。分片上传和 H.265 转码始终先在
`desklink_api_runtime:/app/runtime/recordings` 暂存，完成校验后再归档。FTP/S3
凭据使用 `DESKLINK_JWT_KEY` 派生的 AES-GCM 密钥加密后写入数据库。

NFS/SMB 不由 API 容器执行系统挂载。先在宿主机挂载共享目录，并设置
`DESKLINK_RECORDING_MOUNT_PATH`；管理端配置中的路径使用容器内的
`/recordings-external`。每条录像绑定创建时的存储配置版本，后续切换 Bucket、目录
或协议只影响新录像。暂存卷和外部存储都应纳入备份与容量监控。

其他可用参数与 `conf/config.yaml` 一一对应，环境变量格式为：

```text
RUSTDESK_API_ + 配置路径大写并用下划线连接
```

例如 `app.token-expire` 对应 `RUSTDESK_API_APP_TOKEN_EXPIRE`。

## 6. 查看状态与日志

```bash
docker compose --env-file .env -f docker-compose.mysql.yml ps
docker compose --env-file .env -f docker-compose.mysql.yml logs -f api
docker compose --env-file .env -f docker-compose.mysql.yml logs -f hbbs hbbr
```

健康检查：

```bash
curl http://127.0.0.1:21114/api/version
```

## 7. 备份与恢复

备份 MySQL：

```bash
docker exec desklink-mysql sh -c \
  'exec mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --single-transaction "$MYSQL_DATABASE"' \
  > desklink-$(date +%F).sql
```

恢复 MySQL：

```bash
docker exec -i desklink-mysql sh -c \
  'exec mysql -uroot -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE"' \
  < desklink-2026-09-01.sql
```

同时备份：

- `desklink_server_data`：hbbs 私钥、公钥和设备注册数据库；
- `desklink_api_runtime`：录像暂存文件、本地录像和 API 运行数据；外部录像存储需单独备份。

丢失 `desklink_server_data` 会导致服务公钥改变，客户端配置需要同步更新。

## 8. 升级

```bash
docker compose --env-file .env -f docker-compose.mysql.yml pull
docker compose --env-file .env -f docker-compose.mysql.yml up -d
docker image prune
```

固定使用版本标签可避免 `latest` 自动带来不兼容更新。

## 9. 停止

```bash
docker compose --env-file .env -f docker-compose.mysql.yml down
```

不要使用 `down -v`，除非明确要删除 MySQL、服务密钥和设备注册数据。
