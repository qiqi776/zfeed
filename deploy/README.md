# deploy 目录说明

## 入口

- 业务入口网关：`http://127.0.0.1:18080`
- 直连 API：`http://127.0.0.1:5000`
- Prometheus：`http://127.0.0.1:19090`
- Grafana：`http://127.0.0.1:13000`
- Jaeger：`http://127.0.0.1:16686`

## 一键启动

仓库根目录执行：

```bash
bash ./script/start.sh
```

该脚本会通过 `deploy/docker-compose.yml` 拉起：

- 基础设施：`etcd`、`redis`、`mysql`、`kafka`、`canal`、`xxl-job-admin`
- 后端服务：`front-api`、`user-rpc`、`content-rpc`、`interaction-rpc`、`count-rpc`、`search-rpc`
- 交付入口：`front-web`、`nginx`
- 观测组件：`prometheus`
- 可选链路：`jaeger`、`otel-collector`、`logstash`、`filebeat`

停止：

```bash
bash ./script/stop.sh
```

## front-web 静态交付

`front-web` 镜像通过 [build/front-web.Dockerfile](../build/front-web.Dockerfile) 构建，运行后由容器内 Nginx 托管 Vite 产物，并通过 SPA fallback 处理前端路由。

默认构建不写死 `VITE_API_BASE_URL`，前端请求继续走同源 `/v1/*`，再由 [default.conf](./nginx/default.conf) 反向代理到 `front-api`。如果部署环境必须跨域直连 API，可以在构建前显式传入该变量。

构建并导出镜像：

```bash
cd /home/zz/workspace/projects/zfeed/deploy
docker compose --env-file .env -f docker-compose.yml build front-web
docker save "${FRONT_WEB_IMAGE}" -o ./front-web/images/zfeed-front-web-dev.tar
```

在目标环境导入：

```bash
cd /home/zz/workspace/projects/zfeed/deploy
docker load -i ./front-web/images/zfeed-front-web-dev.tar
docker compose --env-file .env -f docker-compose.yml up -d front-web nginx front-api
```

## 网关路由

- `/v1/*` -> `front-api:5000`
- `/` -> `front-web:80`
