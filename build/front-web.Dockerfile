ARG NODE_VERSION=22.16.0
ARG NGINX_VERSION=1.27-alpine

FROM node:${NODE_VERSION}-alpine AS build
WORKDIR /src/zfeed-web

COPY zfeed-web/package.json zfeed-web/package-lock.json ./
RUN npm ci

COPY zfeed-web/ ./

ARG VITE_API_BASE_URL=
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

RUN npm run build

FROM nginx:${NGINX_VERSION}
WORKDIR /usr/share/nginx/html

COPY deploy/front-web/default.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/zfeed-web/dist ./

EXPOSE 80
