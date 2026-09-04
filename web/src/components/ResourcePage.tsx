import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { ChevronLeft, ChevronRight, Pencil, Plus, RefreshCw, Search, Trash2, X } from 'lucide-react'
import { errorMessage, get, normalizePage, post, type PageData } from '../lib/api'
import { useToast } from './Toast'
import type { ResourceColumn, ResourceConfig, ResourceField } from '../features/resources'

type Row = Record<string, any>

const detailLabels: Record<string, string> = {
  peer: '设备信息', disks: '磁盘明细', id: '设备 ID', alias: '别名', hostname: '主机名', os: '操作系统', username: '系统用户', uuid: 'UUID', version: '客户端版本', last_online_ip: '当前/最近 IP', online: '在线状态', last_online_time: '最后在线', cpu: 'CPU 摘要', cpu_usage: 'CPU 使用率', memory: '内存摘要', memory_total: '内存总量', memory_used: '内存已用', memory_usage: '内存使用率', disk_read_bps: '磁盘读取速率', disk_write_bps: '磁盘写入速率', metrics_at: '指标更新时间', mount: '挂载点', total: '总容量', used: '已用容量', usage: '使用率',
}

function formatDetailValue(key: string, value: unknown) {
  if (key.endsWith('_usage') && typeof value === 'number') return `${value.toFixed(1)}%`
  if (key.endsWith('_bps') && typeof value === 'number') return `${(value / 1024 / 1024).toFixed(2)} MB/s`
  if (['memory_total', 'memory_used', 'total', 'used'].includes(key) && typeof value === 'number') return `${(value / 1024 / 1024 / 1024).toFixed(2)} GB`
  if (key.endsWith('_at') || key === 'last_online_time') return formatDate(value)
  if (typeof value === 'boolean') return value ? '在线' : '离线'
  return String(value ?? '—')
}

function formatDate(value: unknown) {
  if (value === null || value === undefined || value === '' || Number(value) === 0) return '—'
  const raw = Number(value)
  const date = Number.isFinite(raw) ? new Date(raw < 10_000_000_000 ? raw * 1000 : raw) : new Date(String(value))
  return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString('zh-CN', { hour12: false })
}

type MetricSample = {
  timestamp?: number
  cpu_usage?: number
  memory_usage?: number
  disk_usage?: string | Array<{ usage?: number }>
}

function formatBytes(value: unknown) {
  const bytes = Number(value)
  if (!Number.isFinite(bytes)) return '—'
  return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`
}

function diskEntries(value: unknown): Array<{ mount?: string; total?: number; used?: number; usage?: number }> {
  if (Array.isArray(value)) return value as Array<{ mount?: string; total?: number; used?: number; usage?: number }>
  if (typeof value !== 'string' || !value) return []
  try {
    const parsed = JSON.parse(value)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function averageDiskUsage(value: unknown) {
  const entries = diskEntries(value).filter((entry) => typeof entry.usage === 'number')
  return entries.length ? entries.reduce((sum, entry) => sum + Number(entry.usage), 0) / entries.length : 0
}

function MetricChart({ samples }: { samples: MetricSample[] }) {
  const width = 720
  const height = 220
  const padding = { left: 38, right: 16, top: 18, bottom: 28 }
  const plotWidth = width - padding.left - padding.right
  const plotHeight = height - padding.top - padding.bottom
  const values = [
    { key: 'cpu_usage', label: 'CPU', color: '#f59e0b', points: samples.map((sample) => Number(sample.cpu_usage || 0)) },
    { key: 'memory_usage', label: '内存', color: '#2563eb', points: samples.map((sample) => Number(sample.memory_usage || 0)) },
    { key: 'disk_usage', label: '磁盘', color: '#16a34a', points: samples.map((sample) => averageDiskUsage(sample.disk_usage)) },
  ]
  const pointString = (points: number[]) => points.map((value, index) => `${padding.left + (samples.length <= 1 ? plotWidth / 2 : index * plotWidth / (samples.length - 1))},${padding.top + plotHeight - Math.max(0, Math.min(100, value)) / 100 * plotHeight}`).join(' ')
  return (
    <div className="overflow-hidden rounded-md border border-base-200 bg-white p-3">
      <div className="mb-2 flex flex-wrap items-center gap-4 text-xs text-base-content/65">{values.map((value) => <span key={value.key} className="inline-flex items-center gap-1.5"><span className="h-2 w-2 rounded-full" style={{ backgroundColor: value.color }} />{value.label}</span>)}</div>
      {samples.length ? <svg viewBox={`0 0 ${width} ${height}`} className="h-56 w-full" role="img" aria-label="资源使用率趋势图">
        {[0, 25, 50, 75, 100].map((tick) => { const y = padding.top + plotHeight - tick / 100 * plotHeight; return <g key={tick}><line x1={padding.left} x2={width - padding.right} y1={y} y2={y} stroke="#e5e7eb" strokeWidth="1" /><text x={padding.left - 8} y={y + 4} textAnchor="end" fontSize="11" fill="#6b7280">{tick}%</text></g> })}
        {values.map((value) => <polyline key={value.key} points={pointString(value.points)} fill="none" stroke={value.color} strokeWidth="2.5" strokeLinejoin="round" strokeLinecap="round" />)}
        <text x={padding.left} y={height - 7} fontSize="11" fill="#6b7280">{formatDate(samples[0]?.timestamp)}</text>
        <text x={width - padding.right} y={height - 7} textAnchor="end" fontSize="11" fill="#6b7280">{formatDate(samples[samples.length - 1]?.timestamp)}</text>
      </svg> : <div className="flex h-56 items-center justify-center text-sm text-base-content/40">暂无历史指标</div>}
    </div>
  )
}

function DeviceDetail({ detail, samples, range, onRangeChange }: { detail: Row; samples: MetricSample[]; range: string; onRangeChange: (range: string) => void }) {
  const peer = (detail.peer || {}) as Row
  const disks = diskEntries(detail.disks || peer.disk_usage)
  const ranges = [{ value: '1h', label: '1 小时' }, { value: '24h', label: '24 小时' }, { value: '7d', label: '7 天' }, { value: '30d', label: '30 天' }]
  return <div className="space-y-4">
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
      <div className="rounded-md border border-base-200 bg-base-200/30 p-3"><div className="text-xs text-base-content/50">在线状态</div><div className={`mt-1 text-lg font-semibold ${peer.online ? 'text-emerald-600' : 'text-base-content/55'}`}>{peer.online ? '在线' : '离线'}</div><div className="mt-1 text-xs text-base-content/45">最后心跳 {formatDate(peer.last_online_time)}</div></div>
      <div className="rounded-md border border-base-200 bg-base-200/30 p-3"><div className="text-xs text-base-content/50">当前/最近 IP</div><div className="mt-1 font-mono text-sm">{peer.last_online_ip || '—'}</div></div>
      <div className="rounded-md border border-base-200 bg-base-200/30 p-3"><div className="text-xs text-base-content/50">CPU 使用率</div><div className="mt-1 text-lg font-semibold">{formatDetailValue('cpu_usage', peer.cpu_usage)}</div></div>
      <div className="rounded-md border border-base-200 bg-base-200/30 p-3"><div className="text-xs text-base-content/50">内存使用率</div><div className="mt-1 text-lg font-semibold">{formatDetailValue('memory_usage', peer.memory_usage)}</div></div>
      <div className="rounded-md border border-base-200 bg-base-200/30 p-3"><div className="text-xs text-base-content/50">内存容量</div><div className="mt-1 text-sm font-semibold">{formatBytes(peer.memory_used)} / {formatBytes(peer.memory_total)}</div></div>
      <div className="rounded-md border border-base-200 bg-base-200/30 p-3"><div className="text-xs text-base-content/50">磁盘读写</div><div className="mt-1 text-sm font-semibold">{formatDetailValue('disk_read_bps', peer.disk_read_bps)} / {formatDetailValue('disk_write_bps', peer.disk_write_bps)}</div></div>
    </div>
    <div className="grid gap-3 sm:grid-cols-2">
      {[['设备 ID', peer.id], ['主机名', peer.hostname], ['操作系统', peer.os], ['系统用户', peer.username], ['客户端版本', peer.version], ['UUID', peer.uuid]].map(([label, value]) => <div key={label} className="flex min-w-0 justify-between gap-4 rounded-md border border-base-200 p-3 text-sm"><span className="text-base-content/55">{label}</span><span className="max-w-[65%] truncate text-right" title={String(value || '')}>{String(value || '—')}</span></div>)}
    </div>
    <div>
      <div className="mb-2 flex items-center justify-between"><h4 className="font-semibold">磁盘使用情况</h4><span className="text-xs text-base-content/45">容量单位：GB</span></div>
      <div className="grid gap-3 md:grid-cols-2">{disks.length ? disks.map((disk, index) => <div key={`${disk.mount || 'disk'}-${index}`} className="rounded-md border border-base-200 p-3"><div className="mb-2 flex items-center justify-between"><span className="font-medium">{disk.mount || '未命名磁盘'}</span><span className="text-sm font-semibold">{Number(disk.usage || 0).toFixed(1)}%</span></div><div className="h-2 overflow-hidden rounded-full bg-base-200"><div className="h-full rounded-full bg-blue-600" style={{ width: `${Math.max(0, Math.min(100, Number(disk.usage || 0)))}%` }} /></div><div className="mt-2 text-xs text-base-content/55">{formatBytes(disk.used)} / {formatBytes(disk.total)} 已用</div></div>) : <div className="rounded-md border border-base-200 p-4 text-sm text-base-content/40">暂无磁盘数据</div>}</div>
    </div>
    <div>
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2"><h4 className="font-semibold">资源使用趋势</h4><div className="join">{ranges.map((item) => <button key={item.value} className={`btn btn-xs join-item ${range === item.value ? 'btn-neutral' : 'btn-ghost border border-base-300'}`} onClick={() => onRangeChange(item.value)}>{item.label}</button>)}</div></div>
      <MetricChart samples={samples} />
    </div>
  </div>
}

function Cell({ column, value }: { column: ResourceColumn; value: unknown }) {
  if (column.format === 'status') {
    const enabled = Number(value) === 1
    return <span className={`badge badge-sm ${enabled ? 'badge-success badge-soft' : 'badge-ghost'}`}>{enabled ? '启用' : '禁用'}</span>
  }
  if (column.format === 'boolean') return <span className={`badge badge-sm ${value ? 'badge-info badge-soft' : 'badge-ghost'}`}>{value ? '是' : '否'}</span>
  if (column.format === 'online') return <span className="inline-flex items-center gap-1.5"><span className={`h-1.5 w-1.5 rounded-full ${value ? 'bg-emerald-500' : 'bg-base-content/25'}`} />{value ? '在线' : '离线'}</span>
  if (column.format === 'datetime') return <span className="whitespace-nowrap text-base-content/65">{formatDate(value)}</span>
  if (column.format === 'tags') {
    const tags = Array.isArray(value) ? value : typeof value === 'string' ? (() => { try { return JSON.parse(value) } catch { return value.split(',').filter(Boolean) } })() : []
    return <div className="flex flex-wrap gap-1">{tags.length ? tags.map((tag: string) => <span key={tag} className="badge badge-ghost badge-sm">{tag}</span>) : '—'}</div>
  }
  if (column.format === 'secret') {
    const text = String(value || '')
    return <span className="font-mono text-xs">{text ? `${text.slice(0, 7)}••••${text.slice(-4)}` : '—'}</span>
  }
  if (column.format === 'platform') return <span className="capitalize">{String(value || '—')}</span>
  if (typeof value === 'boolean') return value ? '是' : '否'
  return <span className="max-w-[260px] truncate" title={String(value ?? '')}>{value === null || value === undefined || value === '' ? '—' : String(value)}</span>
}

function FormField({ field, value, onChange, required }: { field: ResourceField; value: unknown; onChange: (value: unknown) => void; required: boolean }) {
  if (field.kind === 'boolean') {
    return (
      <label className="desklink-field">
        <span className="label text-xs text-base-content/60">{field.label}</span>
        <span className="flex h-9 items-center justify-between rounded-md border border-base-300 px-3">
          <span className="text-xs text-base-content/50">{value ? '已启用' : '未启用'}</span>
          <input type="checkbox" className="toggle toggle-success toggle-sm" checked={Boolean(value)} onChange={(event) => onChange(event.target.checked)} />
        </span>
      </label>
    )
  }
  if (field.kind === 'select') {
    return (
      <label className="desklink-field">
        <span className="label text-xs text-base-content/60">{field.label}</span>
        <select className="select select-bordered select-sm w-full" value={String(value ?? '')} onChange={(event) => {
          const option = field.options?.find((item) => String(item.value) === event.target.value)
          onChange(option?.value ?? event.target.value)
        }} required={required}>
          {field.options?.map((option) => <option key={String(option.value)} value={String(option.value)}>{option.label}</option>)}
        </select>
      </label>
    )
  }
  const display = field.kind === 'tags' && Array.isArray(value) ? value.join(', ') : String(value ?? '')
  return (
    <label className="desklink-field">
      <span className="label text-xs text-base-content/60">{field.label}{required && <span className="ml-1 text-error">*</span>}</span>
      <input type={field.kind === 'number' ? 'number' : field.kind === 'password' ? 'password' : 'text'} className="input input-bordered input-sm w-full" value={display} placeholder={field.placeholder} required={required} onChange={(event) => {
        if (field.kind === 'number') onChange(event.target.value === '' ? 0 : Number(event.target.value))
        else if (field.kind === 'tags') onChange(event.target.value.split(',').map((item) => item.trim()).filter(Boolean))
        else onChange(event.target.value)
      }} />
    </label>
  )
}

export default function ResourcePage({ config }: { config: ResourceConfig }) {
  const { show } = useToast()
  const [data, setData] = useState<PageData<Row>>(normalizePage())
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [query, setQuery] = useState<Record<string, string>>({})
  const [appliedQuery, setAppliedQuery] = useState<Record<string, string>>({})
  const [editing, setEditing] = useState<Row | null | undefined>(undefined)
  const [form, setForm] = useState<Row>({})
  const [users, setUsers] = useState<Record<string, string>>({})
  const [detail, setDetail] = useState<Row | null>(null)
  const [detailSamples, setDetailSamples] = useState<MetricSample[]>([])
  const [detailRange, setDetailRange] = useState('24h')
  const pageSize = config.pageSize || 20

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const response = await get<PageData<Row>>(config.listPath, { page, page_size: pageSize, ...appliedQuery })
      setData(normalizePage(response))
    } catch (error) {
      show(errorMessage(error), 'error')
      setData(normalizePage())
    } finally {
      setLoading(false)
    }
  }, [config.listPath, page, pageSize, appliedQuery, show])

  useEffect(() => { void load() }, [load])

  useEffect(() => {
    if (!config.columns.some((column) => column.format === 'user')) return
    get<PageData<Row>>('/user/list', { page: 1, page_size: 1000 })
      .then((response) => setUsers(Object.fromEntries(normalizePage(response).list.map((user) => [String(user.id), String(user.username || user.nickname || user.id)]))))
      .catch(() => setUsers({}))
  }, [config.columns])

  const totalPages = Math.max(1, Math.ceil(data.total / pageSize))
  const startCreate = () => {
    const initial: Row = { ...(config.defaults || {}) }
    config.fields?.forEach((field) => { if (field.defaultValue !== undefined) initial[field.key] = field.defaultValue })
    setForm(initial)
    setEditing(null)
  }
  const startEdit = (row: Row) => { setForm({ ...row }); setEditing(row) }

  const openDetail = async (row: Row) => {
    if (!config.detailPath) return
    try {
      const value = await get<Row>(`${config.detailPath}/${row[config.idKey || 'id']}`)
      setDetail(value)
      setDetailRange('24h')
      if (config.metricsPath) {
        const to = Math.floor(Date.now() / 1000)
        const from = to - 24 * 60 * 60
        const metrics = await get<{ samples?: MetricSample[] }>(`${config.metricsPath}/${row[config.idKey || 'id']}`, { from, to })
        setDetailSamples(metrics.samples || [])
      } else {
        setDetailSamples([])
      }
    } catch (error) { show(errorMessage(error), 'error') }
  }

  const loadDetailRange = async (range: string) => {
    if (!detail || !config.metricsPath) return
    const peer = (detail.peer || detail) as Row
    const id = config.idKey === 'row_id' ? peer.row_id : peer.id
    const seconds = range === '1h' ? 3600 : range === '7d' ? 7 * 86400 : range === '30d' ? 30 * 86400 : 86400
    const to = Math.floor(Date.now() / 1000)
    try {
      const metrics = await get<{ samples?: MetricSample[] }>(`${config.metricsPath}/${id}`, { from: to - seconds, to })
      setDetailSamples(metrics.samples || [])
      setDetailRange(range)
    } catch (error) { show(errorMessage(error), 'error') }
  }

  const save = async (event: FormEvent) => {
    event.preventDefault()
    const path = editing ? config.updatePath : config.createPath
    if (!path) return
    try {
      await post(path, form)
      show(editing ? '已保存修改' : '已创建', 'success')
      setEditing(undefined)
      await load()
    } catch (error) { show(errorMessage(error), 'error') }
  }

  const remove = async (row: Row) => {
    if (!config.deletePath || !window.confirm('确定删除这条记录吗？此操作无法撤销。')) return
    const idKey = config.idKey || 'id'
    const bodyKey = config.deleteBodyKey || 'id'
    try {
      await post(config.deletePath, { [bodyKey]: row[idKey] })
      show('记录已删除', 'success')
      await load()
    } catch (error) { show(errorMessage(error), 'error') }
  }

  const visibleFields = useMemo(() => config.fields?.filter((field) => {
    if (editing ? field.createOnly : field.editOnly) return false
    if (!field.visibleWhen) return true
    return field.visibleWhen.values.some((value) => String(value) === String(form[field.visibleWhen!.key]))
  }) || [], [config.fields, editing, form])
  const modalResourceName = config.title.endsWith('管理') ? config.title.slice(0, -2) : config.title

  return (
    <section>
      <div className="mb-5 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-base-content">{config.title}</h1>
          <p className="mt-1 text-sm text-base-content/50">{config.description}</p>
        </div>
        <div className="flex gap-2">
          <button className="btn btn-sm btn-ghost desklink-icon-action border border-base-300 bg-white" onClick={() => void load()} title="刷新"><RefreshCw size={15} className={loading ? 'animate-spin' : ''} /></button>
          {!config.readOnly && config.createPath && <button className="btn btn-sm desklink-action border-0 bg-emerald-600 px-4 text-white hover:bg-emerald-700" onClick={startCreate}><Plus size={16} />新增</button>}
        </div>
      </div>

      {config.search?.length ? (
        <form className="desklink-card mb-4 flex flex-wrap items-end gap-3 p-4" onSubmit={(event) => { event.preventDefault(); setPage(1); setAppliedQuery(query) }}>
          {config.search.map((field) => (
            <label key={field.key} className="desklink-field w-full sm:w-52">
              <span className="label text-[11px] text-base-content/55">{field.label}</span>
              {field.options ? (
                <select className="select select-bordered select-sm w-full bg-white" value={query[field.key] || ''} onChange={(event) => setQuery((current) => ({ ...current, [field.key]: event.target.value }))}>
                  {field.options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                </select>
              ) : (
                <input className="input input-bordered input-sm w-full bg-white" value={query[field.key] || ''} placeholder={field.placeholder || `输入${field.label}`} onChange={(event) => setQuery((current) => ({ ...current, [field.key]: event.target.value }))} />
              )}
            </label>
          ))}
          <button className="btn btn-sm desklink-action btn-neutral px-4"><Search size={15} />筛选</button>
          <button type="button" className="btn btn-sm desklink-action btn-ghost px-4" onClick={() => { setQuery({}); setAppliedQuery({}); setPage(1) }}>重置</button>
        </form>
      ) : null}

      <div className="desklink-card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="table desklink-table table-sm w-full">
            <thead className="bg-[#f8f9fa]"><tr>{config.columns.map((column) => <th key={column.key} className="h-10 whitespace-nowrap">{column.label}</th>)}{(!config.readOnly && (config.updatePath || config.deletePath)) && <th className="w-24 text-right">操作</th>}</tr></thead>
            <tbody>
              {loading ? (
                <tr><td colSpan={config.columns.length + 1} className="h-52 text-center"><span className="loading loading-spinner loading-md text-success" /></td></tr>
              ) : data.list.length === 0 ? (
                <tr><td colSpan={config.columns.length + 1} className="h-52 text-center text-sm text-base-content/40">暂无数据</td></tr>
              ) : data.list.map((row, index) => (
                <tr key={String(row[config.idKey || 'id'] ?? index)} className={`hover:bg-base-200/40 ${config.detailPath ? 'cursor-pointer' : ''}`} onClick={() => void openDetail(row)}>
                  {config.columns.map((column) => <td key={column.key}>{column.format === 'user' ? <span>{users[String(row[column.key])] ? `${users[String(row[column.key])]} (${row[column.key]})` : row[column.key] || '—'}</span> : <Cell column={column} value={row[column.key]} />}</td>)}
                  {!config.readOnly && (config.updatePath || config.deletePath) && (
                    <td><div className="flex justify-end gap-1">{config.detailPath && <button className="btn btn-ghost btn-xs" onClick={(event) => { event.stopPropagation(); void openDetail(row) }} title="详情">详情</button>}{config.updatePath && <button className="btn btn-ghost btn-xs" onClick={(event) => { event.stopPropagation(); startEdit(row) }} title="编辑"><Pencil size={14} /></button>}{config.deletePath && <button className="btn btn-ghost btn-xs text-error" onClick={(event) => { event.stopPropagation(); void remove(row) }} title="删除"><Trash2 size={14} /></button>}</div></td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="flex h-12 items-center justify-between border-t border-base-300 px-4 text-xs text-base-content/50">
          <span>共 {data.total} 条</span>
          <div className="flex items-center gap-2">
            <button className="btn btn-ghost btn-xs" disabled={page <= 1} onClick={() => setPage((value) => value - 1)}><ChevronLeft size={15} /></button>
            <span>{page} / {totalPages}</span>
            <button className="btn btn-ghost btn-xs" disabled={page >= totalPages} onClick={() => setPage((value) => value + 1)}><ChevronRight size={15} /></button>
          </div>
        </div>
      </div>

      {editing !== undefined && (
        <div className="modal modal-open">
          <div className="modal-box desklink-modal max-w-2xl rounded-lg p-0">
            <div className="flex h-14 items-center justify-between border-b border-base-300 px-5">
              <h3 className="font-semibold">{editing ? `编辑${modalResourceName}` : `新增${modalResourceName}`}</h3>
              <button className="btn btn-ghost btn-sm desklink-icon-action" onClick={() => setEditing(undefined)}><X size={18} /></button>
            </div>
            <form onSubmit={(event) => void save(event)}>
              <div className="grid max-h-[65vh] grid-cols-1 gap-4 overflow-y-auto p-5 sm:grid-cols-2">
                {visibleFields.map((field) => <FormField key={field.key} field={field} value={form[field.key]} required={Boolean(field.required || (!editing && field.requiredOnCreate))} onChange={(value) => setForm((current) => ({ ...current, [field.key]: value }))} />)}
              </div>
              <div className="flex justify-end gap-2 border-t border-base-300 bg-base-200/40 px-5 py-4">
                <button type="button" className="btn btn-sm desklink-action btn-ghost px-4" onClick={() => setEditing(undefined)}>取消</button>
                <button className="btn btn-sm desklink-action border-0 bg-emerald-600 px-4 text-white hover:bg-emerald-700">保存</button>
              </div>
            </form>
          </div>
          <button className="modal-backdrop" onClick={() => setEditing(undefined)} aria-label="关闭" />
        </div>
      )}

      {detail && (
        <div className="modal modal-open"><div className="modal-box desklink-modal max-w-5xl rounded-lg p-0"><div className="flex h-14 items-center justify-between border-b border-base-300 px-5"><h3 className="font-semibold">设备详情</h3><button className="btn btn-ghost btn-sm" onClick={() => setDetail(null)}><X size={18} /></button></div><div className="max-h-[78vh] overflow-y-auto p-5">{config.metricsPath ? <DeviceDetail detail={detail} samples={detailSamples} range={detailRange} onRangeChange={(range) => void loadDetailRange(range)} /> : Object.entries(detail).map(([key, value]) => <div key={key} className="mb-4 last:mb-0"><div className="mb-2 text-sm font-semibold text-base-content/70">{detailLabels[key] || key}</div>{Array.isArray(value) ? <div className="grid gap-3 sm:grid-cols-2">{value.map((item, index) => <div key={index} className="rounded-md border border-base-200 p-3">{Object.entries(item || {}).map(([itemKey, itemValue]) => <div key={itemKey} className="flex justify-between gap-3 border-b border-base-200 py-1.5 last:border-0"><span className="text-xs text-base-content/50">{detailLabels[itemKey] || itemKey}</span><span className="text-right text-sm">{formatDetailValue(itemKey, itemValue)}</span></div>)}</div>)}</div> : typeof value === 'object' && value !== null ? <div className="grid gap-3 sm:grid-cols-2">{Object.entries(value).map(([itemKey, itemValue]) => <div key={itemKey} className="flex justify-between gap-3 rounded-md border border-base-200 p-3"><span className="text-xs text-base-content/50">{detailLabels[itemKey] || itemKey}</span><span className="text-right text-sm">{formatDetailValue(itemKey, itemValue)}</span></div>)}</div> : <div className="rounded-md border border-base-200 p-3 text-sm">{formatDetailValue(key, value)}</div>}</div>)}</div></div><button className="modal-backdrop" onClick={() => setDetail(null)} aria-label="关闭" /></div>
      )}
    </section>
  )
}
