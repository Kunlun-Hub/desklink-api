import { useEffect, useState } from 'react'
import { Activity, BookUser, Cpu, Server, ShieldCheck, Users } from 'lucide-react'
import { errorMessage, get, normalizePage, type PageData } from '../lib/api'
import { useAuth } from '../lib/auth'
import { useToast } from '../components/Toast'

interface ServerConfig { id_server: string; relay_server: string; api_server: string; key: string }
interface AdminConfig { title: string; hello?: string }

export default function DashboardPage() {
  const { user, isAdmin } = useAuth()
  const { show } = useToast()
  const [counts, setCounts] = useState({ users: 0, devices: 0, onlineDevices: 0, addressBooks: 0, logs: 0 })
  const [server, setServer] = useState<ServerConfig | null>(null)
  const [adminConfig, setAdminConfig] = useState<AdminConfig | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const load = async () => {
      try {
        const common = [get<AdminConfig>('/config/admin'), get<ServerConfig>('/config/server')]
        const metricRequests = isAdmin
          ? [get<PageData<unknown>>('/user/list', { page: 1, page_size: 1 }), get<PageData<unknown>>('/peer/list', { page: 1, page_size: 1 }), get<PageData<unknown>>('/peer/list', { page: 1, page_size: 1, online: 'true' }), get<PageData<unknown>>('/address_book/list', { page: 1, page_size: 1 }), get<PageData<unknown>>('/audit_conn/list', { page: 1, page_size: 1 })]
          : [Promise.resolve(undefined), get<PageData<unknown>>('/my/peer/list', { page: 1, page_size: 1 }), get<PageData<unknown>>('/my/peer/list', { page: 1, page_size: 1, online: 'true' }), get<PageData<unknown>>('/my/address_book/list', { page: 1, page_size: 1 }), get<PageData<unknown>>('/my/login_log/list', { page: 1, page_size: 1 })]
        const [adminResult, serverResult, users, devices, onlineDevices, addressBooks, logs] = await Promise.all([...common, ...metricRequests])
        setAdminConfig(adminResult as AdminConfig)
        setServer(serverResult as ServerConfig)
        setCounts({ users: normalizePage(users as PageData<unknown> | undefined).total, devices: normalizePage(devices as PageData<unknown>).total, onlineDevices: normalizePage(onlineDevices as PageData<unknown>).total, addressBooks: normalizePage(addressBooks as PageData<unknown>).total, logs: normalizePage(logs as PageData<unknown>).total })
      } catch (error) { show(errorMessage(error), 'error') }
      finally { setLoading(false) }
    }
    void load()
  }, [isAdmin, show])

  const stats = [
    ...(isAdmin ? [{ label: '用户总数', value: counts.users, icon: Users, color: 'text-sky-600 bg-sky-50' }] : []),
    { label: '设备总数', value: counts.devices, icon: Cpu, color: 'text-emerald-600 bg-emerald-50' },
    { label: '在线设备', value: counts.onlineDevices, icon: Server, color: 'text-cyan-600 bg-cyan-50' },
    { label: '地址簿条目', value: counts.addressBooks, icon: BookUser, color: 'text-violet-600 bg-violet-50' },
    { label: isAdmin ? '连接记录' : '登录记录', value: counts.logs, icon: Activity, color: 'text-amber-600 bg-amber-50' },
  ]

  return (
    <section>
      <div className="mb-6">
        <h1 className="text-xl font-semibold">你好，{user?.nickname || user?.username}</h1>
        <p className="mt-1 text-sm text-base-content/50">{adminConfig?.hello ? adminConfig.hello.replace(/<[^>]*>/g, '') : '查看 DeskLink 社区服务的关键状态。'}</p>
      </div>
      <div className={`grid gap-4 sm:grid-cols-2 ${isAdmin ? 'xl:grid-cols-4' : 'xl:grid-cols-3'}`}>
        {stats.map((item) => {
          const Icon = item.icon
          return (
            <div key={item.label} className="desklink-card flex items-center gap-4 p-5">
              <div className={`flex h-10 w-10 items-center justify-center rounded-md ${item.color}`}><Icon size={20} /></div>
              <div><div className="text-xs text-base-content/50">{item.label}</div><div className="mt-1 text-2xl font-semibold">{loading ? <span className="loading loading-dots loading-sm" /> : item.value}</div></div>
            </div>
          )
        })}
      </div>

      <div className="mt-5 grid gap-5 xl:grid-cols-[1.35fr_0.65fr]">
        <div className="desklink-card p-5">
          <div className="mb-5 flex items-center gap-2"><Server size={18} className="text-emerald-600" /><h2 className="text-sm font-semibold">服务连接</h2><span className="badge badge-success badge-soft badge-sm ml-auto"><span className="mr-1 h-1.5 w-1.5 rounded-full bg-success" />API 运行中</span></div>
          <dl className="grid gap-0 text-sm">
            {[
              ['API 服务', server?.api_server], ['ID 服务', server?.id_server], ['Relay 服务', server?.relay_server], ['客户端兼容', 'DeskLink 1.4.9'],
            ].map(([label, value]) => <div key={label} className="grid grid-cols-[130px_1fr] border-t border-base-200 py-3 first:border-0"><dt className="text-base-content/45">{label}</dt><dd className="truncate font-mono text-xs text-base-content/75">{loading ? '加载中…' : value || '未配置'}</dd></div>)}
          </dl>
        </div>
        <div className="desklink-card p-5">
          <div className="mb-5 flex items-center gap-2"><ShieldCheck size={18} className="text-sky-600" /><h2 className="text-sm font-semibold">运行状态</h2></div>
          <div className="space-y-4 text-sm">
            <div className="flex items-center justify-between"><span className="text-base-content/55">API 接口</span><span className="inline-flex items-center gap-2 text-emerald-600"><span className="h-2 w-2 rounded-full bg-emerald-500" />正常</span></div>
            <div className="flex items-center justify-between"><span className="text-base-content/55">SQLite 数据库</span><span className="inline-flex items-center gap-2 text-emerald-600"><span className="h-2 w-2 rounded-full bg-emerald-500" />已连接</span></div>
            <div className="flex items-center justify-between"><span className="text-base-content/55">管理端</span><span className="inline-flex items-center gap-2 text-emerald-600"><span className="h-2 w-2 rounded-full bg-emerald-500" />React</span></div>
          </div>
        </div>
      </div>
    </section>
  )
}
