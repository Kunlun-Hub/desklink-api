export type FieldKind = 'text' | 'number' | 'boolean' | 'select' | 'password' | 'tags'

export interface ResourceField {
  key: string
  label: string
  kind?: FieldKind
  required?: boolean
  createOnly?: boolean
  editOnly?: boolean
  options?: Array<{ label: string; value: string | number | boolean }>
  placeholder?: string
  defaultValue?: unknown
}

export interface ResourceColumn {
  key: string
  label: string
  format?: 'status' | 'boolean' | 'online' | 'datetime' | 'tags' | 'platform' | 'secret'
}

export interface ResourceConfig {
  key: string
  title: string
  description: string
  listPath: string
  createPath?: string
  updatePath?: string
  deletePath?: string
  idKey?: string
  deleteBodyKey?: string
  columns: ResourceColumn[]
  fields?: ResourceField[]
  search?: Array<{ key: string; label: string; placeholder?: string }>
  defaults?: Record<string, unknown>
  readOnly?: boolean
  pageSize?: number
}

const enabledOptions = [
  { label: '启用', value: 1 },
  { label: '禁用', value: 2 },
]

export const resources: Record<string, ResourceConfig> = {
  users: {
    key: 'users', title: '用户管理', description: '管理社区账号、角色与账号状态', listPath: '/user/list', createPath: '/user/create', updatePath: '/user/update', deletePath: '/user/delete',
    columns: [
      { key: 'username', label: '用户名' }, { key: 'nickname', label: '显示名称' }, { key: 'email', label: '邮箱' },
      { key: 'group_id', label: '用户组' }, { key: 'is_admin', label: '管理员', format: 'boolean' }, { key: 'status', label: '状态', format: 'status' }, { key: 'created_at', label: '创建时间', format: 'datetime' },
    ],
    fields: [
      { key: 'username', label: '用户名', required: true }, { key: 'nickname', label: '显示名称' }, { key: 'email', label: '邮箱' },
      { key: 'group_id', label: '用户组 ID', kind: 'number', required: true, defaultValue: 1 },
      { key: 'is_admin', label: '管理员权限', kind: 'boolean', defaultValue: false },
      { key: 'status', label: '账号状态', kind: 'select', options: enabledOptions, defaultValue: 1 }, { key: 'remark', label: '备注' },
    ], search: [{ key: 'username', label: '用户名', placeholder: '搜索用户名' }], defaults: { group_id: 1, status: 1, is_admin: false },
  },
  devices: {
    key: 'devices', title: '设备管理', description: '查看客户端上报的设备状态和版本信息', listPath: '/peer/list', createPath: '/peer/create', updatePath: '/peer/update', deletePath: '/peer/delete', idKey: 'row_id', deleteBodyKey: 'row_id',
    columns: [
      { key: 'id', label: '设备 ID' }, { key: 'alias', label: '别名' }, { key: 'hostname', label: '主机名' }, { key: 'os', label: '平台', format: 'platform' },
      { key: 'username', label: '系统用户' }, { key: 'version', label: '客户端版本' }, { key: 'last_online_time', label: '最后在线', format: 'datetime' },
    ],
    fields: [
      { key: 'id', label: '设备 ID', required: true }, { key: 'alias', label: '别名' }, { key: 'hostname', label: '主机名' },
      { key: 'os', label: '操作系统' }, { key: 'username', label: '系统用户' }, { key: 'uuid', label: 'UUID' },
      { key: 'version', label: '客户端版本' }, { key: 'group_id', label: '设备组 ID', kind: 'number', defaultValue: 0 },
    ], search: [{ key: 'id', label: '设备 ID' }, { key: 'hostname', label: '主机名' }, { key: 'username', label: '系统用户' }],
  },
  groups: {
    key: 'groups', title: '用户组', description: '组织用户并控制共享范围', listPath: '/group/list', createPath: '/group/create', updatePath: '/group/update', deletePath: '/group/delete',
    columns: [{ key: 'id', label: 'ID' }, { key: 'name', label: '名称' }, { key: 'type', label: '类型' }, { key: 'created_at', label: '创建时间', format: 'datetime' }],
    fields: [{ key: 'name', label: '名称', required: true }, { key: 'type', label: '类型', kind: 'select', options: [{ label: '普通组', value: 1 }, { label: '共享组', value: 2 }], defaultValue: 1 }], defaults: { type: 1 },
  },
  deviceGroups: {
    key: 'device-groups', title: '设备组', description: '按部门或用途组织可访问设备', listPath: '/device_group/list', createPath: '/device_group/create', updatePath: '/device_group/update', deletePath: '/device_group/delete',
    columns: [{ key: 'id', label: 'ID' }, { key: 'name', label: '名称' }, { key: 'created_at', label: '创建时间', format: 'datetime' }],
    fields: [{ key: 'name', label: '名称', required: true }],
  },
  addressBooks: {
    key: 'address-books', title: '地址簿', description: '管理用户的远程设备条目和连接信息', listPath: '/address_book/list', createPath: '/address_book/create', updatePath: '/address_book/update', deletePath: '/address_book/delete', idKey: 'row_id', deleteBodyKey: 'row_id',
    columns: [
      { key: 'id', label: '设备 ID' }, { key: 'alias', label: '别名' }, { key: 'hostname', label: '主机名' }, { key: 'platform', label: '平台', format: 'platform' },
      { key: 'tags', label: '标签', format: 'tags' }, { key: 'user_id', label: '所属用户' }, { key: 'online', label: '在线', format: 'online' },
    ],
    fields: [
      { key: 'id', label: '设备 ID', required: true }, { key: 'alias', label: '别名' }, { key: 'hostname', label: '主机名' }, { key: 'username', label: '系统用户' },
      { key: 'platform', label: '平台' }, { key: 'tags', label: '标签', kind: 'tags', placeholder: '多个标签用逗号分隔' },
      { key: 'user_id', label: '所属用户 ID', kind: 'number', required: true }, { key: 'collection_id', label: '地址簿 ID', kind: 'number', defaultValue: 0 },
      { key: 'forceAlwaysRelay', label: '强制中继', kind: 'boolean' }, { key: 'rdpPort', label: 'RDP 端口' }, { key: 'rdpUsername', label: 'RDP 用户名' },
    ], search: [{ key: 'id', label: '设备 ID' }, { key: 'hostname', label: '主机名' }, { key: 'username', label: '用户' }],
  },
  tags: {
    key: 'tags', title: '标签管理', description: '维护地址簿标签和颜色', listPath: '/tag/list', createPath: '/tag/create', updatePath: '/tag/update', deletePath: '/tag/delete',
    columns: [{ key: 'name', label: '名称' }, { key: 'color', label: '颜色值' }, { key: 'user_id', label: '用户 ID' }, { key: 'collection_id', label: '地址簿 ID' }],
    fields: [{ key: 'name', label: '名称', required: true }, { key: 'color', label: 'Flutter 颜色值', kind: 'number', required: true, defaultValue: 4278255360 }, { key: 'user_id', label: '用户 ID', kind: 'number' }, { key: 'collection_id', label: '地址簿 ID', kind: 'number' }],
  },
  collections: {
    key: 'collections', title: '地址簿集合', description: '为用户创建和组织多个独立地址簿', listPath: '/address_book_collection/list', createPath: '/address_book_collection/create', updatePath: '/address_book_collection/update', deletePath: '/address_book_collection/delete',
    columns: [{ key: 'id', label: 'ID' }, { key: 'name', label: '名称' }, { key: 'user_id', label: '所属用户' }, { key: 'created_at', label: '创建时间', format: 'datetime' }],
    fields: [{ key: 'name', label: '名称', required: true }, { key: 'user_id', label: '所属用户 ID', kind: 'number', required: true }],
    search: [{ key: 'user_id', label: '用户 ID' }],
  },
  collectionRules: {
    key: 'collection-rules', title: '共享规则', description: '将地址簿按只读、读写或完全控制权限共享给用户或用户组', listPath: '/address_book_collection_rule/list', createPath: '/address_book_collection_rule/create', updatePath: '/address_book_collection_rule/update', deletePath: '/address_book_collection_rule/delete',
    columns: [{ key: 'id', label: 'ID' }, { key: 'collection_id', label: '地址簿 ID' }, { key: 'user_id', label: '所有者' }, { key: 'type', label: '共享对象' }, { key: 'to_id', label: '目标 ID' }, { key: 'rule', label: '权限' }],
    fields: [
      { key: 'collection_id', label: '地址簿 ID', kind: 'number', required: true }, { key: 'user_id', label: '所有者用户 ID', kind: 'number', required: true },
      { key: 'type', label: '共享对象', kind: 'select', options: [{ label: '用户', value: 1 }, { label: '用户组', value: 2 }], defaultValue: 1 },
      { key: 'to_id', label: '目标用户/组 ID', kind: 'number', required: true },
      { key: 'rule', label: '权限', kind: 'select', options: [{ label: '只读', value: 1 }, { label: '读写', value: 2 }, { label: '完全控制', value: 3 }], defaultValue: 1 },
    ], defaults: { type: 1, rule: 1 }, search: [{ key: 'collection_id', label: '地址簿 ID' }, { key: 'user_id', label: '所有者 ID' }],
  },
  oauth: {
    key: 'oauth', title: 'OAuth / OIDC', description: '配置第三方身份提供商和自动注册策略', listPath: '/oauth/list', createPath: '/oauth/create', updatePath: '/oauth/update', deletePath: '/oauth/delete',
    columns: [{ key: 'op', label: '标识' }, { key: 'oauth_type', label: '类型' }, { key: 'client_id', label: 'Client ID', format: 'secret' }, { key: 'issuer', label: 'Issuer' }, { key: 'auto_register', label: '自动注册', format: 'boolean' }, { key: 'pkce_enable', label: 'PKCE', format: 'boolean' }],
    fields: [
      { key: 'oauth_type', label: '类型', kind: 'select', required: true, options: [{ label: 'OIDC', value: 'oidc' }, { label: 'GitHub', value: 'github' }, { label: 'Google', value: 'google' }, { label: 'Linux.do', value: 'linuxdo' }], defaultValue: 'oidc' },
      { key: 'op', label: '提供商标识', placeholder: '例如 company-oidc' }, { key: 'issuer', label: 'Issuer URL' }, { key: 'scopes', label: 'Scopes', placeholder: 'openid,profile,email' },
      { key: 'client_id', label: 'Client ID', required: true }, { key: 'client_secret', label: 'Client Secret', kind: 'password', required: true },
      { key: 'auto_register', label: '允许自动注册', kind: 'boolean', defaultValue: false }, { key: 'pkce_enable', label: '启用 PKCE', kind: 'boolean', defaultValue: false },
      { key: 'pkce_method', label: 'PKCE 方法', kind: 'select', options: [{ label: 'S256', value: 'S256' }, { label: 'plain', value: 'plain' }], defaultValue: 'S256' },
    ], defaults: { oauth_type: 'oidc', scopes: 'openid,profile,email', auto_register: false, pkce_enable: false, pkce_method: 'S256' },
  },
  myDevices: {
    key: 'my-devices', title: '我的设备', description: '当前账号绑定和最近上报的设备', listPath: '/my/peer/list', readOnly: true,
    columns: [{ key: 'id', label: '设备 ID' }, { key: 'alias', label: '别名' }, { key: 'hostname', label: '主机名' }, { key: 'os', label: '平台', format: 'platform' }, { key: 'version', label: '版本' }, { key: 'last_online_time', label: '最后在线', format: 'datetime' }],
  },
  myAddressBooks: {
    key: 'my-address-books', title: '我的地址簿', description: '维护当前账号可访问的远程设备', listPath: '/my/address_book/list', createPath: '/my/address_book/create', updatePath: '/my/address_book/update', deletePath: '/my/address_book/delete', idKey: 'row_id', deleteBodyKey: 'row_id',
    columns: [{ key: 'id', label: '设备 ID' }, { key: 'alias', label: '别名' }, { key: 'hostname', label: '主机名' }, { key: 'platform', label: '平台', format: 'platform' }, { key: 'tags', label: '标签', format: 'tags' }, { key: 'online', label: '在线', format: 'online' }],
    fields: [
      { key: 'id', label: '设备 ID', required: true }, { key: 'alias', label: '别名' }, { key: 'hostname', label: '主机名' }, { key: 'username', label: '系统用户' },
      { key: 'platform', label: '平台' }, { key: 'tags', label: '标签', kind: 'tags' }, { key: 'collection_id', label: '地址簿 ID', kind: 'number', defaultValue: 0 },
      { key: 'forceAlwaysRelay', label: '强制中继', kind: 'boolean' },
    ],
  },
  myCollections: {
    key: 'my-collections', title: '我的地址簿集合', description: '创建和整理自己的多个地址簿', listPath: '/my/address_book_collection/list', createPath: '/my/address_book_collection/create', updatePath: '/my/address_book_collection/update', deletePath: '/my/address_book_collection/delete',
    columns: [{ key: 'id', label: 'ID' }, { key: 'name', label: '名称' }, { key: 'created_at', label: '创建时间', format: 'datetime' }], fields: [{ key: 'name', label: '名称', required: true }],
  },
  myCollectionRules: {
    key: 'my-collection-rules', title: '我的共享规则', description: '控制其他用户或用户组对地址簿的访问权限', listPath: '/my/address_book_collection_rule/list', createPath: '/my/address_book_collection_rule/create', updatePath: '/my/address_book_collection_rule/update', deletePath: '/my/address_book_collection_rule/delete',
    columns: [{ key: 'collection_id', label: '地址簿 ID' }, { key: 'type', label: '共享对象' }, { key: 'to_id', label: '目标 ID' }, { key: 'rule', label: '权限' }, { key: 'created_at', label: '创建时间', format: 'datetime' }],
    fields: [
      { key: 'collection_id', label: '地址簿 ID', kind: 'number', required: true }, { key: 'type', label: '共享对象', kind: 'select', options: [{ label: '用户', value: 1 }, { label: '用户组', value: 2 }], defaultValue: 1 },
      { key: 'to_id', label: '目标用户/组 ID', kind: 'number', required: true }, { key: 'rule', label: '权限', kind: 'select', options: [{ label: '只读', value: 1 }, { label: '读写', value: 2 }, { label: '完全控制', value: 3 }], defaultValue: 1 },
    ], defaults: { type: 1, rule: 1 }, search: [{ key: 'collection_id', label: '地址簿 ID' }],
  },
  myTags: {
    key: 'my-tags', title: '我的标签', description: '维护个人地址簿标签', listPath: '/my/tag/list', createPath: '/my/tag/create', updatePath: '/my/tag/update', deletePath: '/my/tag/delete',
    columns: [{ key: 'name', label: '名称' }, { key: 'color', label: '颜色值' }, { key: 'collection_id', label: '地址簿 ID' }],
    fields: [{ key: 'name', label: '名称', required: true }, { key: 'color', label: 'Flutter 颜色值', kind: 'number', required: true, defaultValue: 4278255360 }, { key: 'collection_id', label: '地址簿 ID', kind: 'number' }],
  },
  myLoginLogs: {
    key: 'my-login-logs', title: '我的登录记录', description: '查看当前账号的登录设备和来源', listPath: '/my/login_log/list', deletePath: '/my/login_log/delete',
    columns: [{ key: 'client', label: '客户端' }, { key: 'device_id', label: '设备 ID' }, { key: 'ip', label: 'IP 地址' }, { key: 'type', label: '类型' }, { key: 'platform', label: '平台', format: 'platform' }, { key: 'created_at', label: '登录时间', format: 'datetime' }],
  },
  myShares: {
    key: 'my-shares', title: '我的分享记录', description: '管理当前账号创建的临时访问链接', listPath: '/my/share_record/list', deletePath: '/my/share_record/delete',
    columns: [{ key: 'peer_id', label: '设备 ID' }, { key: 'password_type', label: '密码类型' }, { key: 'share_token', label: '分享令牌', format: 'secret' }, { key: 'expire', label: '过期时间', format: 'datetime' }, { key: 'created_at', label: '创建时间', format: 'datetime' }],
  },
  loginLogs: {
    key: 'login-logs', title: '登录日志', description: '检查账号登录来源和客户端类型', listPath: '/login_log/list', deletePath: '/login_log/delete',
    columns: [{ key: 'user_id', label: '用户 ID' }, { key: 'client', label: '客户端' }, { key: 'device_id', label: '设备 ID' }, { key: 'ip', label: 'IP 地址' }, { key: 'type', label: '类型' }, { key: 'platform', label: '平台', format: 'platform' }, { key: 'created_at', label: '登录时间', format: 'datetime' }],
  },
  connectionAudit: {
    key: 'connection-audit', title: '连接审计', description: '记录远程会话的建立与关闭', listPath: '/audit_conn/list', deletePath: '/audit_conn/delete',
    columns: [{ key: 'action', label: '动作' }, { key: 'peer_id', label: '目标设备' }, { key: 'from_peer', label: '来源设备' }, { key: 'from_name', label: '来源名称' }, { key: 'ip', label: 'IP 地址' }, { key: 'session_id', label: '会话 ID' }, { key: 'created_at', label: '时间', format: 'datetime' }],
    search: [{ key: 'peer_id', label: '目标设备' }, { key: 'from_peer', label: '来源设备' }],
  },
  fileAudit: {
    key: 'file-audit', title: '文件审计', description: '记录远程文件传输活动', listPath: '/audit_file/list', deletePath: '/audit_file/delete',
    columns: [{ key: 'peer_id', label: '目标设备' }, { key: 'from_peer', label: '来源设备' }, { key: 'from_name', label: '来源名称' }, { key: 'path', label: '路径' }, { key: 'info', label: '信息' }, { key: 'ip', label: 'IP 地址' }, { key: 'created_at', label: '时间', format: 'datetime' }],
    search: [{ key: 'peer_id', label: '目标设备' }, { key: 'from_peer', label: '来源设备' }],
  },
  tokens: {
    key: 'tokens', title: '访问令牌', description: '查看和撤销已登录客户端令牌', listPath: '/user_token/list', deletePath: '/user_token/delete',
    columns: [{ key: 'user_id', label: '用户 ID' }, { key: 'device_id', label: '设备 ID' }, { key: 'device_uuid', label: '设备 UUID' }, { key: 'token', label: '令牌', format: 'secret' }, { key: 'expired_at', label: '过期时间', format: 'datetime' }, { key: 'created_at', label: '创建时间', format: 'datetime' }],
  },
  shares: {
    key: 'shares', title: '分享记录', description: '管理 Web 客户端临时分享记录', listPath: '/share_record/list', deletePath: '/share_record/delete',
    columns: [{ key: 'user_id', label: '用户 ID' }, { key: 'peer_id', label: '设备 ID' }, { key: 'password_type', label: '密码类型' }, { key: 'share_token', label: '分享令牌', format: 'secret' }, { key: 'expire', label: '过期时间', format: 'datetime' }, { key: 'created_at', label: '创建时间', format: 'datetime' }],
  },
}
