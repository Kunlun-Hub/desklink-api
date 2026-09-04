import { useEffect, useState } from 'react'
import { CheckCircle2, Copy, Database, KeyRound, Network, Server } from 'lucide-react'
import { errorMessage, get, post } from '../lib/api'
import { useToast } from '../components/Toast'
import { useAuth } from '../lib/auth'

interface Config { id_server: string; relay_server: string; api_server: string; key: string }
interface AgentMetricsConfig { interval_seconds: number }

export default function SettingsPage() {
  const { show } = useToast()
  const { isAdmin } = useAuth()
  const [config, setConfig] = useState<Config | null>(null)
  const [metrics, setMetrics] = useState<AgentMetricsConfig | null>(null)
  const [interval, setIntervalValue] = useState('5')
  const [saving, setSaving] = useState(false)
  useEffect(() => {
    get<Config>('/config/server').then(setConfig).catch((error) => show(errorMessage(error), 'error'))
    get<AgentMetricsConfig>('/config/agent-metrics').then((value) => { setMetrics(value); setIntervalValue(String(value.interval_seconds)) }).catch((error) => show(errorMessage(error), 'error'))
  }, [show])
  const copy = async (value: string) => { await navigator.clipboard.writeText(value); show('已复制', 'success') }
  const saveMetrics = async () => {
    const value = Number(interval)
    if (!Number.isInteger(value) || value < 5 || value > 3600) { show('采集频率必须在 5 到 3600 秒之间', 'error'); return }
    setSaving(true)
    try { const next = await post<AgentMetricsConfig>('/config/agent-metrics', { interval_seconds: value }); setMetrics(next); setIntervalValue(String(next.interval_seconds)); show('采集设置已保存', 'success') } catch (error) { show(errorMessage(error), 'error') } finally { setSaving(false) }
  }

  const rows = [
    { label: 'API 服务地址', value: config?.api_server || '', icon: Server },
    { label: 'ID 服务地址', value: config?.id_server || '', icon: Network },
    { label: 'Relay 服务地址', value: config?.relay_server || '', icon: Database },
    { label: '服务公钥', value: config?.key || '', icon: KeyRound },
  ]
  return (
    <section className="max-w-4xl">
      <div className="mb-5"><h1 className="text-xl font-semibold">系统信息</h1><p className="mt-1 text-sm text-base-content/50">查看服务端地址、公钥和 Agent 采集设置。</p></div>
      <div className="desklink-card overflow-hidden">
        <div className="flex items-center gap-3 border-b border-base-300 p-5"><div className="flex h-9 w-9 items-center justify-center rounded-md bg-emerald-50 text-emerald-600"><CheckCircle2 size={19} /></div><div><div className="text-sm font-semibold">DeskLink Community API</div><div className="text-xs text-base-content/45">运行中 · 客户端兼容版本 1.4.9</div></div></div>
        <div className="divide-y divide-base-200">
          {rows.map((row) => {
            const Icon = row.icon
            return <div key={row.label} className="grid min-h-16 gap-3 px-5 py-3 sm:grid-cols-[180px_minmax(0,1fr)_32px] sm:items-center"><div className="flex items-center gap-2 text-sm text-base-content/55"><Icon size={16} />{row.label}</div><code className="flex h-9 min-w-0 items-center truncate rounded bg-base-200 px-3 text-xs">{config ? row.value || '未配置' : '加载中…'}</code><div className="flex h-8 w-8 items-center justify-center">{row.value && <button className="btn btn-ghost btn-xs h-8 min-h-8 w-8 p-0" onClick={() => void copy(row.value)} title="复制"><Copy size={14} /></button>}</div></div>
          })}
        </div>
      </div>
      {isAdmin && <div className="desklink-card mt-5 overflow-hidden">
        <div className="border-b border-base-300 p-5"><h2 className="text-sm font-semibold">Agent 采集设置</h2><p className="mt-1 text-xs text-base-content/50">控制客户端 CPU、内存、磁盘指标的上报频率，范围 5 到 3600 秒。</p></div>
        <div className="flex flex-wrap items-end gap-3 p-5"><label className="desklink-field w-56"><span className="label text-xs text-base-content/60">采集间隔（秒）</span><input className="input input-bordered input-sm" type="number" min="5" max="3600" value={interval} onChange={(event) => setIntervalValue(event.target.value)} /></label><button className="btn btn-sm desklink-action border-0 bg-emerald-600 px-4 text-white hover:bg-emerald-700" disabled={saving || !metrics} onClick={() => void saveMetrics()}>{saving ? '保存中…' : '保存设置'}</button></div>
      </div>}
    </section>
  )
}
