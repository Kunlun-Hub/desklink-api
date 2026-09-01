import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { ChevronLeft, ChevronRight, Pencil, Plus, RefreshCw, Search, Trash2, X } from 'lucide-react'
import { errorMessage, get, normalizePage, post, type PageData } from '../lib/api'
import { useToast } from './Toast'
import type { ResourceColumn, ResourceConfig, ResourceField } from '../features/resources'

type Row = Record<string, any>

function formatDate(value: unknown) {
  if (value === null || value === undefined || value === '') return '—'
  const raw = Number(value)
  const date = Number.isFinite(raw) ? new Date(raw < 10_000_000_000 ? raw * 1000 : raw) : new Date(String(value))
  return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString('zh-CN', { hour12: false })
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

function FormField({ field, value, onChange }: { field: ResourceField; value: unknown; onChange: (value: unknown) => void }) {
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
        }} required={field.required}>
          {field.options?.map((option) => <option key={String(option.value)} value={String(option.value)}>{option.label}</option>)}
        </select>
      </label>
    )
  }
  const display = field.kind === 'tags' && Array.isArray(value) ? value.join(', ') : String(value ?? '')
  return (
    <label className="desklink-field">
      <span className="label text-xs text-base-content/60">{field.label}{field.required && <span className="ml-1 text-error">*</span>}</span>
      <input type={field.kind === 'number' ? 'number' : field.kind === 'password' ? 'password' : 'text'} className="input input-bordered input-sm w-full" value={display} placeholder={field.placeholder} required={field.required} onChange={(event) => {
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

  const totalPages = Math.max(1, Math.ceil(data.total / pageSize))
  const startCreate = () => {
    const initial: Row = { ...(config.defaults || {}) }
    config.fields?.forEach((field) => { if (field.defaultValue !== undefined) initial[field.key] = field.defaultValue })
    setForm(initial)
    setEditing(null)
  }
  const startEdit = (row: Row) => { setForm({ ...row }); setEditing(row) }

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

  const visibleFields = useMemo(() => config.fields?.filter((field) => editing ? !field.createOnly : !field.editOnly) || [], [config.fields, editing])
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
              <input className="input input-bordered input-sm w-full bg-white" value={query[field.key] || ''} placeholder={field.placeholder || `输入${field.label}`} onChange={(event) => setQuery((current) => ({ ...current, [field.key]: event.target.value }))} />
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
                <tr key={String(row[config.idKey || 'id'] ?? index)} className="hover:bg-base-200/40">
                  {config.columns.map((column) => <td key={column.key}><Cell column={column} value={row[column.key]} /></td>)}
                  {!config.readOnly && (config.updatePath || config.deletePath) && (
                    <td><div className="flex justify-end gap-1">{config.updatePath && <button className="btn btn-ghost btn-xs" onClick={() => startEdit(row)} title="编辑"><Pencil size={14} /></button>}{config.deletePath && <button className="btn btn-ghost btn-xs text-error" onClick={() => void remove(row)} title="删除"><Trash2 size={14} /></button>}</div></td>
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
                {visibleFields.map((field) => <FormField key={field.key} field={field} value={form[field.key]} onChange={(value) => setForm((current) => ({ ...current, [field.key]: value }))} />)}
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
    </section>
  )
}
