import { useState, type FormEvent } from 'react'
import { ArrowLeft, UserPlus } from 'lucide-react'
import { Navigate, useNavigate } from 'react-router-dom'
import { useToast } from '../components/Toast'
import { errorMessage } from '../lib/api'
import { useAuth } from '../lib/auth'

export default function RegisterPage() {
  const { user, register } = useAuth()
  const { show } = useToast()
  const navigate = useNavigate()
  const [form, setForm] = useState({ username: '', email: '', password: '', confirm_password: '' })
  const [saving, setSaving] = useState(false)
  if (user) return <Navigate to="/" replace />

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (form.password !== form.confirm_password) { show('两次输入的密码不一致', 'error'); return }
    setSaving(true)
    try { await register(form); show('账号创建成功', 'success'); navigate('/', { replace: true }) }
    catch (error) { show(errorMessage(error), 'error') }
    finally { setSaving(false) }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-[#111719] px-5 py-10">
      <div className="w-full max-w-md">
        <button className="btn btn-ghost btn-sm mb-5 text-white/55 hover:bg-white/5 hover:text-white" onClick={() => navigate('/login')}><ArrowLeft size={16} />返回登录</button>
        <div className="rounded-lg border border-white/8 bg-white/[0.035] p-7 shadow-2xl">
          <div className="mb-7 flex items-center gap-3"><img src="./logo.png" alt="DeskLink" className="h-10 w-10 object-contain" /><div><h1 className="text-xl font-semibold text-white">创建社区账号</h1><p className="mt-1 text-xs text-white/35">DeskLink Community Server</p></div></div>
          <form className="space-y-4" onSubmit={(event) => void submit(event)}>
            {[
              ['username', '用户名', 'text'], ['email', '邮箱（可选）', 'email'], ['password', '密码', 'password'], ['confirm_password', '确认密码', 'password'],
            ].map(([key, label, type]) => <label key={key} className="desklink-field"><span className="label text-xs text-white/55">{label}</span><input type={type} className="input h-10 w-full border-white/12 bg-white/5 text-sm text-white placeholder:text-white/20 focus:border-emerald-500" value={form[key as keyof typeof form]} onChange={(event) => setForm((current) => ({ ...current, [key]: event.target.value }))} required={key !== 'email'} minLength={key === 'username' ? 2 : type === 'password' ? 4 : undefined} maxLength={type === 'password' ? 32 : undefined} /></label>)}
            <button className="btn h-10 w-full border-0 bg-emerald-500 text-white hover:bg-emerald-600" disabled={saving}>{saving ? <span className="loading loading-spinner loading-sm" /> : <UserPlus size={17} />}创建账号</button>
          </form>
        </div>
      </div>
    </div>
  )
}
