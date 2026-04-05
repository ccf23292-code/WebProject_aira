# 文件上传与静态访问说明

## 1. 现在后端是如何存储前端上传图片/文件的

后端已经把静态资源目录固定为：

```txt
storage/
```

Gin 在启动时会暴露下面这个静态访问前缀：

```txt
/static
```

也就是：

- 磁盘文件 `storage/avatars/a.png`
- 对外访问路径就是 `/static/avatars/a.png`

本次改动后，后端会自动创建 `storage` 目录，并按照用途把上传内容放到不同子目录：

- 头像：`storage/avatars/YYYYMMDD/文件名`
- 普通图片：`storage/uploads/images/YYYYMMDD/文件名`
- 普通文件：`storage/uploads/files/YYYYMMDD/文件名`

文件名由后端自动生成，避免重名覆盖。

## 2. 前端应该如何拿到正确的 URL

后端现在会同时返回两种地址：

- `public_path`：相对静态路径，适合存库，例如 `/static/uploads/images/20260405/xxx.png`
- `url`：完整可访问 URL，前端可以直接用于 `img src`、下载链接、预览等

完整 URL 的生成规则：

1. 如果配置了环境变量 `PUBLIC_BASE_URL`，优先使用它。
2. 如果没有配置，就使用当前请求的协议 + Host 自动拼接。

例如：

```json
{
  "code": 201,
  "message": "success",
  "data": {
    "type": "image",
    "original_name": "cover.png",
    "filename": "img_xxx.png",
    "size": 1024,
    "content_type": "image/png",
    "public_path": "/static/uploads/images/20260405/img_xxx.png",
    "url": "http://localhost:3001/static/uploads/images/20260405/img_xxx.png"
  }
}
```

如果前后端分离部署，建议在后端 `.env` 或运行环境中增加：

```env
PUBLIC_BASE_URL=http://localhost:3001
```

部署到线上时改成线上域名即可。

## 3. 后端如何完成这套交互

推荐交互流程：

1. 前端先发 `multipart/form-data` 请求上传文件。
2. 后端保存文件到 `storage/...`。
3. 后端返回 `public_path` 和 `url`。
4. 前端如果只是立即展示，直接使用 `url`。
5. 前端如果还要把图片地址保存到业务表，建议把 `public_path` 再提交给业务接口。
6. 后端查询业务数据时，再把相对路径转换成完整 URL 返回给前端。

这样做的好处：

- 数据库存相对路径，不依赖具体域名
- 迁移环境时不需要批量改库
- 前端拿到的数据仍然可以直接访问

## 4. 新增/修改的接口

### 4.1 通用文件上传

接口：

```txt
POST /api/files/upload
```

鉴权：

- 需要登录
- `Authorization: Bearer <token>`

请求类型：

```txt
multipart/form-data
```

表单字段：

- `file`：上传的文件本体
- `type`：`image` 或 `file`

说明：

- `type=image` 时，只允许常见图片格式
- `type=file` 时，允许常见文档/压缩包/图片格式
- 单文件大小限制为 `20MB`

### 4.2 头像上传

接口：

```txt
POST /api/profile/avatar
```

请求类型：

```txt
multipart/form-data
```

表单字段：

- `avatar`：头像文件

说明：

- 只允许图片格式
- 单文件大小限制为 `5MB`
- 数据库存储相对路径
- 返回给前端时会自动转成完整头像 URL

## 5. 前端调用示例

```js
const formData = new FormData();
formData.append("file", file);
formData.append("type", "image");

const res = await fetch("http://localhost:3001/api/files/upload", {
  method: "POST",
  headers: {
    Authorization: `Bearer ${token}`,
  },
  body: formData,
});

const result = await res.json();
const imageUrl = result.data.url;
const imagePath = result.data.public_path;
```

头像上传：

```js
const formData = new FormData();
formData.append("avatar", file);

const res = await fetch("http://localhost:3001/api/profile/avatar", {
  method: "POST",
  headers: {
    Authorization: `Bearer ${token}`,
  },
  body: formData,
});

const result = await res.json();
const avatarUrl = result.data.avatar_url;
```

## 6. 本次改动记录

### 新增文件

- `routers/upload_support.go`
  - 新增统一上传辅助逻辑
  - 负责校验扩展名、创建目录、保存文件、生成 `public_path`
  - 新增完整 URL 生成逻辑，支持 `PUBLIC_BASE_URL`

- `routers/file_controller.go`
  - 新增通用文件上传接口 `POST /api/files/upload`
  - 支持 `image` 和 `file` 两种上传类型

- `routers/file_controller_test.go`
  - 新增上传测试
  - 验证文件会被真正写入目录，并返回正确 URL

- `storage/.gitkeep`
  - 保留空的上传目录结构入口

### 修改文件

- `main.go`
  - 启动时自动创建 `storage` 目录
  - 设置 `MaxMultipartMemory`
  - 注册新的 `/api/files/upload` 路由
  - 保持 `/static -> ./storage` 静态映射

- `routers/profile_controller.go`
  - 头像上传改为复用统一上传逻辑
  - 头像在数据库中保存相对路径
  - 返回前端时自动补成完整头像 URL

- `.gitignore`
  - 忽略运行时上传目录 `storage/*`
  - 保留 `storage/.gitkeep`

## 7. 我建议前端与后端的约定

- 前端上传成功后，展示时用 `url`
- 前端提交业务数据时，保存 `public_path`
- 后端对外返回业务数据时，再统一把相对路径转成完整 URL

这样最稳，也最方便以后切换本地、测试、生产环境。
