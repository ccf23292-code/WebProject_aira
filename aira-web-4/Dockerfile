# ============================================
# AIRAWeb 前端 Docker 镜像
#
# 构建：docker compose up --build
# 访问：http://localhost:3000
# ============================================

FROM node:20-alpine

WORKDIR /app

# 复制 package.json（monorepo 根 + 各 workspace）
COPY package.json ./
COPY apps/web/package.json ./apps/web/
COPY packages/shared/package.json ./packages/shared/

# 安装依赖
RUN npm install

# 复制全部源码
COPY . .

# 后端 API 地址（构建时注入，默认 localhost:3001）
# 如需修改，在 docker-compose.yml 的 build.args 中覆盖
ARG NEXT_PUBLIC_API_URL=http://localhost:3001/api
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

# 构建 Next.js
RUN npm run build:web

EXPOSE 3000

# 启动生产服务器
CMD ["npm", "run", "start:web"]
