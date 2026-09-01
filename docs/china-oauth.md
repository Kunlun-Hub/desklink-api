# 钉钉与企业微信登录配置

管理端入口：`/_admin/#/admin/oauth`

两种登录方式共用以下回调地址：

```text
https://你的-API-域名/api/oidc/callback
```

生产环境必须使用外部可访问的 HTTPS API 地址，并确保 `rustdesk.api-server` 配置与上述域名一致。

## 钉钉

1. 在钉钉开放平台创建企业内部应用。
2. 将回调地址加入登录授权回调域名。
3. 为应用开通个人信息读取权限。当前流程使用新版个人授权接口：
   - `Contact.User.Read`
   - `Contact.Member.Read`
4. 在 DeskLink 管理端新建 OAuth 配置：
   - 类型：`钉钉`
   - Client ID / AppKey：钉钉应用的 AppKey
   - Client Secret / AppSecret：钉钉应用的 AppSecret
   - Scopes：默认 `openid`；限制企业时会自动附加 `corpid`
   - CorpID：可选；填写后会拒绝其他企业的账号
   - 允许自动注册：首次登录时是否自动创建 DeskLink 用户

## 企业微信

1. 在企业微信管理后台创建自建应用。
2. 配置应用的网页授权及企业微信授权登录回调域。
3. 确保应用可以读取登录成员的基础资料；如需邮箱和头像，还需相应通讯录权限。
4. 在 DeskLink 管理端新建 OAuth 配置：
   - 类型：`企业微信`
   - Client ID / AppKey / CorpID：企业的 CorpID
   - Client Secret / AppSecret：自建应用 Secret
   - AgentID：自建应用 AgentID
   - 允许自动注册：首次登录时是否自动创建 DeskLink 用户

企业微信登录仅接受企业内部成员。外部联系人只返回 OpenID、没有成员 UserID 时会被拒绝，避免创建无法关联企业身份的账号。

## 账号绑定

已登录用户可以在“个人资料 -> 第三方账号”中绑定或解除绑定。若关闭“允许自动注册”，未绑定的第三方账号会进入现有的账号绑定流程，不会直接创建新用户。
