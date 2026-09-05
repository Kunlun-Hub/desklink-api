import { useEffect, useState, type FormEvent } from 'react'
import { KeyRound, Link2, Mail, ShieldCheck, Unlink, UserRound } from 'lucide-react'
import { errorMessage, post } from '../lib/api'
import { useAuth } from '../lib/auth'
import { useToast } from '../components/Toast'

interface OauthBinding { op: string; status: number }

const providerLabels: Record<string, string> = { dingtalk: '钉钉', wecom: '企业微信' }

export default function ProfilePage() {
  const { user } = useAuth()
  const { show } = useToast()
  const [bindings, setBindings] = useState<OauthBinding[]>([])
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [saving, setSaving] = useState(false)

  const loadBindings = async () => {
    try {
      const result = await post<OauthBinding[] | null>('/user/myOauth')
      setBindings(Array.isArray(result) ? result : [])
    }
    catch { setBindings([]) }
  }
  useEffect(() => { void loadBindings() }, [])

  const changePassword = async (event: FormEvent) => {
    event.preventDefault()
    if (newPassword !== confirmPassword) { show('两次输入的新密码不一致', 'error'); return }
    setSaving(true)
    try {
      await post('/user/changeCurPwd', { old_password: oldPassword, new_password: newPassword })
      setOldPassword(''); setNewPassword(''); setConfirmPassword('')
      show('密码已更新', 'success')
    } catch (error) { show(errorMessage(error), 'error') }
    finally { setSaving(false) }
  }

  const bind = async (item: OauthBinding) => {
    try {
      const result = await post<{ url: string }>('/oauth/bind', { op: item.op })
      window.open(result.url, '_blank', 'noopener,noreferrer')
    } catch (error) { show(errorMessage(error), 'error') }
  }
  const unbind = async (item: OauthBinding) => {
    if (!window.confirm(`确定解除 ${item.op} 绑定吗？`)) return
    try { await post('/oauth/unbind', { op: item.op }); show('已解除绑定', 'success'); await loadBindings() }
    catch (error) { show(errorMessage(error), 'error') }
  }

  return (
    <section className="max-w-4xl">
      <div className="mb-5"><h1 className="text-xl font-semibold">个人资料</h1><p className="mt-1 text-sm text-base-content/50">管理账号安全和第三方登录绑定。</p></div>
      <div className="grid items-stretch gap-5 lg:grid-cols-2">
        <div className="desklink-card h-full p-5">
          <div className="mb-5 flex items-center gap-2"><UserRound size={18} className="text-emerald-600" /><h2 className="text-sm font-semibold">账号信息</h2></div>
          <dl className="space-y-4 text-sm">
            <div className="flex items-center justify-between border-b border-base-200 pb-3"><dt className="flex items-center gap-2 text-base-content/50"><UserRound size={15} />用户名</dt><dd className="font-medium">{user?.username}</dd></div>
            <div className="flex items-center justify-between border-b border-base-200 pb-3"><dt className="flex items-center gap-2 text-base-content/50"><Mail size={15} />邮箱</dt><dd>{user?.email || '未设置'}</dd></div>
            <div className="flex items-center justify-between"><dt className="flex items-center gap-2 text-base-content/50"><ShieldCheck size={15} />角色</dt><dd><span className="badge badge-success badge-soft badge-sm">{user?.role_name || (user?.route_names?.includes('*') ? '管理员' : '普通用户')}</span></dd></div>
          </dl>
        </div>
        <form data-testid="password-form" className="desklink-card flex h-full flex-col p-5" onSubmit={(event) => void changePassword(event)}>
          <div className="mb-5 flex items-center gap-2"><KeyRound size={18} className="text-amber-600" /><h2 className="text-sm font-semibold">修改密码</h2></div>
          <div className="space-y-3">
            <label className="desklink-field"><span className="label text-xs text-base-content/55">当前密码</span><input type="password" className="input input-bordered input-sm" value={oldPassword} onChange={(event) => setOldPassword(event.target.value)} required minLength={4} /></label>
            <label className="desklink-field"><span className="label text-xs text-base-content/55">新密码</span><input type="password" className="input input-bordered input-sm" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} required minLength={4} maxLength={32} /></label>
            <label className="desklink-field"><span className="label text-xs text-base-content/55">确认新密码</span><input type="password" className="input input-bordered input-sm" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} required /></label>
          </div>
          <button className="btn btn-sm desklink-action mt-4 self-start border-0 bg-emerald-600 px-4 text-white hover:bg-emerald-700" disabled={saving}>{saving && <span className="loading loading-spinner loading-xs" />}更新密码</button>
        </form>
      </div>

      <div className="desklink-card mt-5 overflow-hidden">
        <div className="border-b border-base-300 px-5 py-4"><h2 className="text-sm font-semibold">第三方账号</h2><p className="mt-1 text-xs text-base-content/45">绑定后可使用对应身份提供商登录。</p></div>
        {bindings.length ? <div className="divide-y divide-base-200">{bindings.map((item) => <div key={item.op} className="flex items-center gap-3 px-5 py-4"><div className="flex h-9 w-9 items-center justify-center rounded-md bg-base-200"><Link2 size={17} /></div><div className="flex-1"><div className="text-sm font-medium capitalize">{providerLabels[item.op.toLowerCase()] || item.op}</div><div className="text-xs text-base-content/40">{item.status === 1 ? '已绑定' : '未绑定'}</div></div>{item.status === 1 ? <button className="btn btn-ghost btn-sm text-error" onClick={() => void unbind(item)}><Unlink size={15} />解除</button> : <button className="btn btn-outline btn-sm" onClick={() => void bind(item)}><Link2 size={15} />绑定</button>}</div>)}</div> : <div className="p-8 text-center text-sm text-base-content/40">未配置第三方身份提供商</div>}
      </div>
    </section>
  )
}
