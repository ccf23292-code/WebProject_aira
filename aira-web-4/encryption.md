# 登录、注册、验证码加密与传输规范

本文档用于前后端、网关与运维对接登录、注册、验证码相关的安全格式。

## 1. 总体原则

1. 所有认证相关接口必须只通过 `HTTPS` 提供服务。
2. 不再依赖“前端自己加密密码再上传”来替代传输层安全，账号密码、邮箱、验证码都通过 `HTTPS + JSON` 传输。
3. 服务端只保存密码哈希，不保存密码明文。
4. 服务端不保存验证码明文，改为保存验证码哈希。
5. 服务端不保存 access token / refresh token 明文，改为保存 token 哈希。
6. 日志、埋点、错误上报中禁止记录密码、验证码、token 明文。

## 2. 传输层格式

### 2.1 协议要求

- 协议：`HTTPS`
- TLS 版本：`TLS 1.2` 及以上
- 明文 `HTTP`：
  - 开发环境可临时使用
  - 测试、预发、生产环境禁止使用

### 2.2 请求体格式

- Content-Type：`application/json`
- 字符编码：`UTF-8`
- 请求体不再额外做 RSA/AES 字段加密

原因：

- 如果已经使用 `HTTPS`，再做一层前端字段加密，收益有限，但会显著提高联调和密钥管理复杂度。
- 当前项目更需要先补齐“传输层 TLS + 服务端哈希存储”。

## 3. 接口字段传输格式

### 3.1 注册接口

- 路径：`POST /api/auth/register`
- 请求体：

```json
{
  "username": "alice",
  "email": "alice@zju.edu.cn",
  "password": "Abcd1234",
  "confirmPassword": "Abcd1234",
  "verificationCode": "123456",
  "agreeToPolicy": true
}
```

字段说明：

- `password`：前端输入的原始密码，通过 `HTTPS` 传输
- `confirmPassword`：前端输入的原始确认密码，通过 `HTTPS` 传输
- `verificationCode`：6 位验证码，通过 `HTTPS` 传输

### 3.2 登录接口

- 路径：`POST /api/auth/login`
- 请求体：

```json
{
  "username": "alice",
  "password": "Abcd1234",
  "otp": "123456",
  "rememberMe": true
}
```

字段说明：

- `password`：前端输入的原始密码，通过 `HTTPS` 传输
- `otp`：
  - 当前代码里字段已预留
  - 若未启用登录二次验证，可传空字符串
  - 若后续启用 2FA，则传一次性验证码明文，通过 `HTTPS` 传输

### 3.3 发送验证码接口

- 路径：`POST /api/auth/verification-code`
- 请求体：

```json
{
  "email": "alice@zju.edu.cn"
}
```

说明：

- 邮箱地址通过 `HTTPS` 传输
- 响应中禁止返回验证码明文
- 仅开发联调环境可通过显式开关返回验证码

## 4. 服务端存储格式

### 4.1 密码

- 算法：`bcrypt`
- 存储内容：`bcrypt hash`
- 不保存内容：
  - 密码明文
  - 可逆加密后的密码

格式示例：

```txt
$2a$10$w7QX...
```

说明：

- 每个密码使用独立 salt
- cost 使用 `bcrypt.DefaultCost`，后续如需统一升级可单独调整

### 4.2 邮箱验证码

- 原始格式：6 位数字字符串，例如 `123456`
- 存储算法：`SHA-256`
- 存储内容：验证码哈希的十六进制小写字符串

计算规则：

```txt
code_hash = SHA256(email_lower + ":" + code + ":" + scene + ":" + server_secret)
```

字段说明：

- `email_lower`：转小写后的邮箱
- `code`：原始 6 位验证码
- `scene`：业务场景，如 `register`
- `server_secret`：服务端环境变量密钥，不下发前端

存储示例：

```txt
8f14e45fceea167a5a36dedd4bea2543...
```

校验规则：

1. 服务端收到验证码明文
2. 按相同规则重新计算 `SHA-256`
3. 与数据库中的 `code_hash` 比较
4. 验证成功后立即删除该验证码记录

说明：

- 不保存验证码明文
- 即使数据库泄漏，也不能直接看到验证码原值

### 4.3 Access Token / Refresh Token

- 原始格式：高强度随机字符串，推荐 `32 bytes` 以上随机源
- 对外返回格式：`base64url` 或 `uuid+random` 风格字符串
- 数据库存储算法：`SHA-256`
- 数据库存储内容：token 哈希，不保存 token 明文

推荐计算规则：

```txt
token_hash = SHA256(raw_token + ":" + server_secret)
```

请求传输格式：

- `accessToken`：登录/注册响应 JSON 返回
- 后续请求头：

```txt
Authorization: Bearer <access_token>
```

- `refreshToken`：登录/注册响应 JSON 返回；刷新或登出时放在请求体中

说明：

- 客户端持有 token 明文
- 服务端只保存哈希值
- 校验时，对客户端传入 token 再做同样的 `SHA-256` 计算后查库

## 5. 环境变量与密钥要求

新增统一服务端密钥：

```env
AUTH_SECRET=replace-with-long-random-secret
```

要求：

- 长度至少 `32` 字节
- 使用高强度随机值
- 只保存在服务端环境变量或密钥管理系统中
- 不写入前端代码
- 不提交到 Git 仓库

用途：

- 验证码哈希计算
- token 哈希计算
- 后续如改为 JWT 签名，也可复用或拆分专用密钥

## 6. 当前项目状态与目标状态

### 6.1 当前状态

- 登录、注册、验证码接口使用 `JSON` 明文字段
- 项目代码中未看到应用层 `HTTPS` 监听配置
- 密码已使用 `bcrypt` 哈希存储
- 验证码当前仍为明文存库
- access token / refresh token 当前仍为明文存库

### 6.2 目标状态

- 所有认证接口仅允许 `HTTPS`
- 密码继续使用 `bcrypt`
- 验证码改为 `SHA-256 + secret` 哈希存储
- access token / refresh token 改为 `SHA-256 + secret` 哈希存储
- 生产环境关闭验证码回显
- 清理日志中的敏感字段

## 7. 前端对接约定

前端需要遵循以下规则：

1. 只调用 `HTTPS` 地址。
2. 按 JSON 明文字段提交 `password`、`verificationCode`、`otp`，不要自行做二次加密。
3. 不在 localStorage、sessionStorage、日志系统中打印密码和验证码。
4. token 如需持久化，优先使用更安全的存储策略；若放浏览器存储，需要配合 XSS 防护。
5. 验证码输入框只保留 6 位数字。

## 8. 后端实现要求

后端改造时按以下顺序落地：

1. 增加 `AUTH_SECRET` 环境变量。
2. 将 `EmailVerification.Code` 改为 `CodeHash`。
3. 将 `AuthSession.AccessToken`、`AuthSession.RefreshToken` 改为对应哈希字段。
4. 登录态校验改为“传入 token -> 哈希 -> 查库”。
5. 登出改为按 `refresh token hash` 删除。
6. 生产环境强制关闭验证码回显。
7. 反向代理或网关层强制启用 `HTTPS`。

## 9. 一句话结论

对接时请按以下口径理解：

- 传输：`HTTPS + JSON`
- 密码存储：`bcrypt`
- 验证码存储：`SHA-256(email + code + scene + secret)`
- token 存储：`SHA-256(token + secret)`

