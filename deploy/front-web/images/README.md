# front-web 镜像交付目录

这个目录用于存放前端静态站点镜像导出文件，例如：

```bash
cd /home/zz/workspace/projects/zfeed/deploy
docker compose --env-file .env -f docker-compose.yml build front-web
docker save "${FRONT_WEB_IMAGE}" -o ./front-web/images/zfeed-front-web-dev.tar
```

在目标机器导入：

```bash
cd /home/zz/workspace/projects/zfeed/deploy
docker load -i ./front-web/images/zfeed-front-web-dev.tar
```

默认前端构建不写死 API 域名，浏览器会继续走同源 `/v1/*`，由 `deploy/nginx/default.conf` 转发到 `front-api`。
