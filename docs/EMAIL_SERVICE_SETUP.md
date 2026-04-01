# 邮件服务商 API Key 配置（低成本）

目标：用于“邮箱验证码”发送。Go 本身不能直接发邮件，必须使用外部邮件服务（SMTP 或 API）。

以下给出一个**低成本/尽量免费**的落地方案与通用配置方式。具体价格会变化，请以服务商官网为准。

## 方案选择（建议）
1) **邮件服务商 API（推荐）**
- 通常有免费额度（适合开发/小规模）
- 发送稳定，易于监控

2) **SMTP（备用）**
- 如果学校或个人邮箱提供 SMTP
- 配置简单，但稳定性和配额不一定好

## 通用配置方式（API Key）
不管你选哪个服务商，做法通常一致：

1. 注册服务商账号并创建**发信域名**或**发信邮箱**
2. 通过 DNS 做域名验证（TXT/CNAME 记录）
3. 在控制台生成 **API Key**
4. 把 API Key 写入后端 `.env`

### 推荐的环境变量命名（示例）
在 `back/.env` 添加：
```
EMAIL_PROVIDER=ses           # 可选：ses/sendgrid/resend/mailgun/other
EMAIL_API_KEY=xxxxxxxxxxxx
EMAIL_FROM=noreply@yourdomain.com
EMAIL_REGION=ap-southeast-1   # 某些服务需要区域
```

后端代码读取这些变量，用对应的 SDK / HTTP API 发送邮件。

## 低成本建议
- **开发期**：优先用免费额度或沙箱模式
- **上线前**：完成域名验证，开启正式发信
- **控制成本**：
  - 每个邮箱每分钟最多 1 次验证码
  - 验证码有效期 5~10 分钟
  - 错误次数限制（例如 5 次）
  - 发送频率限制（IP + 邮箱维度）

## 如果暂时不想接第三方服务（仅开发）
可以在开发环境中：
- 把验证码写入日志
- 或返回给前端（仅 dev）
上线必须切换到真实邮件服务。

## 中国常见 SMTP 服务商与教程链接
下面是国内较常用、文档较完善的 SMTP 服务商（供你选择）。注意：实际价格/配额会变化，请以官方页面为准。

1) 腾讯云邮件推送（SMTP 发送指南）  
```
https://cloud.tencent.com/document/product/1288/65749
```

2) 阿里云邮件推送（SMTP 快速流程 / 使用 SMTP 发送邮件）  
```
https://help.aliyun.com/zh/direct-mail/getting-started/simplified-procedure-of-sending-by-api-and-smtp
https://help.aliyun.com/zh/direct-mail/user-guide/send-emails-using-smtp
```

3) SendCloud（SMTP 接入说明 / 文档中心）  
```
https://www.sendcloud.net/doc/guide/base/
https://www.sendcloud.net/doc/
```

> 备注：腾讯云近期对个人认证账号的 SMTP 发信有门槛限制，若你是个人账号可能需要升级企业认证或改用 API 发信。  
> 选择服务商时建议：优先看“单封成本 + 免费额度 + 审核/备案要求”。  

## 下一步（我可以帮你做）
如果你决定使用某个服务商（比如 SES / SendGrid / Resend / Mailgun），告诉我：
- 你选择的服务商
- 发信域名

我可以直接把后端发信逻辑接进去，并把 `.env` 模板写好。
