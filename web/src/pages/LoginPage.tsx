import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Eye, EyeOff, KeyRound, LockKeyhole, UserRound } from 'lucide-react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { ApiError, errorMessage, get, post } from '../lib/api'
import { useAuth } from '../lib/auth'
import { useToast } from '../components/Toast'

interface LoginOptions {
  ops: string[]
  register: boolean
  need_captcha: boolean
  disable_pwd: boolean
  auto_oidc: boolean
}

interface CaptchaResponse { captcha: { id: string; b64: string } }

const providerLabels: Record<string, string> = {
  dingtalk: '钉钉',
  wecom: '企业微信',
  github: 'GitHub',
  google: 'Google',
  linuxdo: 'Linux.do',
  oidc: 'OIDC',
}

function platformName() {
  const value = navigator.platform.toLowerCase()
  if (value.includes('win')) return 'windows'
  if (value.includes('mac')) return 'mac'
  if (value.includes('linux')) return 'linux'
  return value || 'web'
}

export default function LoginPage() {
  const { user, login, completeOidc } = useAuth()
  const { show } = useToast()
  const navigate = useNavigate()
  const location = useLocation()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [options, setOptions] = useState<LoginOptions | null>(null)
  const [captcha, setCaptcha] = useState<CaptchaResponse['captcha'] | null>(null)
  const [captchaValue, setCaptchaValue] = useState('')

  const loadCaptcha = useCallback(async () => {
    try {
      const result = await get<CaptchaResponse>('/captcha')
      setCaptcha(result.captcha)
    } catch (error) {
      show(errorMessage(error), 'error')
    }
  }, [show])

  useEffect(() => {
    const oidcCode = localStorage.getItem('desklink_oidc_code')
    if (oidcCode) {
      completeOidc(oidcCode).then(() => {
        show('第三方登录成功', 'success')
        navigate('/', { replace: true })
      }).catch((error) => {
        localStorage.removeItem('desklink_oidc_code')
        show(errorMessage(error), 'error')
      })
    }
    get<LoginOptions>('/login-options').then((result) => {
      setOptions(result)
      if (result.need_captcha) void loadCaptcha()
    }).catch((error) => show(errorMessage(error), 'error'))
  }, [completeOidc, loadCaptcha, navigate, show])

  if (user) return <Navigate to="/" replace />

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSubmitting(true)
    try {
      await login({
        username,
        password,
        platform: platformName(),
        captcha: captchaValue || undefined,
        captcha_id: captcha?.id,
      })
      show('登录成功', 'success')
      const target = (location.state as { from?: string } | null)?.from || '/'
      navigate(target, { replace: true })
    } catch (error) {
      show(errorMessage(error), 'error')
      if (error instanceof ApiError && error.code === 110) void loadCaptcha()
    } finally {
      setSubmitting(false)
    }
  }

  const oidcLogin = async (provider: string) => {
    try {
      const result = await post<{ code: string; url: string }>('/oidc/auth', {
        deviceInfo: { name: navigator.userAgent, os: platformName(), type: 'webadmin' },
        id: `webadmin-${platformName()}`,
        op: provider,
        uuid: '',
      })
      localStorage.setItem('desklink_oidc_code', result.code)
      window.location.href = result.url
    } catch (error) { show(errorMessage(error), 'error') }
  }

  return (
    <div className="grid min-h-screen bg-[#111719] lg:grid-cols-[minmax(380px,0.82fr)_1.18fr]">
      <div className="flex items-center justify-center px-6 py-10 lg:px-12">
        <div className="w-full max-w-[390px]">
          <div className="mb-10 flex items-center gap-3">
            <img src="./logo.png" alt="DeskLink" className="h-11 w-11 object-contain" />
            <div>
              <div className="text-xl font-semibold text-white">DeskLink</div>
              <div className="text-xs text-white/40">社区服务管理中心</div>
            </div>
          </div>
          <div className="mb-7">
            <h1 className="text-2xl font-semibold text-white">登录管理中心</h1>
            <p className="mt-2 text-sm text-white/45">管理用户、设备、地址簿和服务审计记录</p>
          </div>

          {!options?.disable_pwd && (
            <form onSubmit={submit} className="space-y-4">
              <label className="desklink-field w-full">
                <span className="label pb-1 text-xs text-white/55">用户名</span>
                <div className="flex h-11 items-center rounded-md border border-white/12 bg-white/5 px-3 focus-within:border-emerald-500/70">
                  <UserRound size={17} className="mr-2 text-white/35" />
                  <input className="min-w-0 flex-1 bg-transparent text-sm text-white outline-none placeholder:text-white/20" value={username} onChange={(event) => setUsername(event.target.value)} autoComplete="username" placeholder="请输入用户名" required />
                </div>
              </label>
              <label className="desklink-field w-full">
                <span className="label pb-1 text-xs text-white/55">密码</span>
                <div className="flex h-11 items-center rounded-md border border-white/12 bg-white/5 px-3 focus-within:border-emerald-500/70">
                  <LockKeyhole size={17} className="mr-2 text-white/35" />
                  <input type={showPassword ? 'text' : 'password'} className="min-w-0 flex-1 bg-transparent text-sm text-white outline-none placeholder:text-white/20" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="current-password" placeholder="请输入密码" required />
                  <button type="button" className="text-white/35 hover:text-white/70" onClick={() => setShowPassword((value) => !value)} aria-label="显示密码">{showPassword ? <EyeOff size={17} /> : <Eye size={17} />}</button>
                </div>
              </label>
              {captcha && (
                <label className="desklink-field w-full">
                  <span className="label pb-1 text-xs text-white/55">验证码</span>
                  <div className="flex h-11 overflow-hidden rounded-md border border-white/12 bg-white/5 focus-within:border-emerald-500/70">
                    <input className="min-w-0 flex-1 bg-transparent px-3 text-sm text-white outline-none" value={captchaValue} onChange={(event) => setCaptchaValue(event.target.value)} required />
                    <button type="button" onClick={() => void loadCaptcha()} title="刷新验证码"><img src={captcha.b64} alt="验证码" className="h-11 w-32 object-cover" /></button>
                  </div>
                </label>
              )}
              <button className="btn h-11 w-full border-0 bg-emerald-500 text-sm text-white hover:bg-emerald-600" disabled={submitting}>
                {submitting ? <span className="loading loading-spinner loading-sm" /> : <KeyRound size={17} />}
                登录
              </button>
              {options?.register && <button type="button" className="btn btn-ghost h-10 w-full text-white/65 hover:bg-white/5 hover:text-white" onClick={() => navigate('/register')}>创建社区账号</button>}
            </form>
          )}

          {options?.ops?.length ? (
            <div className="mt-6">
              <div className="divider text-xs text-white/30">其他登录方式</div>
              <div className="grid gap-2">
                {options.ops.map((option) => <button key={option} onClick={() => void oidcLogin(option)} className="btn btn-outline border-white/15 text-white/70 hover:border-white/30 hover:bg-white/5">使用 {providerLabels[option.toLowerCase()] || option} 登录</button>)}
              </div>
            </div>
          ) : null}
          <div className="mt-8 text-xs text-white/25">DeskLink Community Server · Client 1.4.9</div>
        </div>
      </div>

      <div className="relative hidden overflow-hidden border-l border-white/5 bg-[#182024] lg:flex lg:items-end">
        <div className="absolute inset-0 opacity-50" style={{ backgroundImage: 'linear-gradient(rgba(255,255,255,.025) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,.025) 1px, transparent 1px)', backgroundSize: '38px 38px' }} />
        <div className="relative max-w-2xl p-14 xl:p-20">
          <div className="mb-5 inline-flex items-center gap-2 rounded-full border border-emerald-400/20 bg-emerald-400/8 px-3 py-1 text-xs text-emerald-300">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
            社区服务已就绪
          </div>
          <h2 className="max-w-xl text-4xl font-semibold leading-tight text-white">自己的设备，自己的数据，自己的连接方式。</h2>
          <p className="mt-5 max-w-lg text-sm leading-7 text-white/45">统一管理 DeskLink 用户、设备、权限与审计，让社区服务保持透明、可控。</p>
        </div>
      </div>
    </div>
  )
}
