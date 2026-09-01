import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Pencil, Play, Plus, RefreshCw, ServerCog, Terminal, Trash2, X } from 'lucide-react'
import { errorMessage, get, normalizePage, post, type PageData } from '../lib/api'
import { useToast } from '../components/Toast'

interface CommandItem { id: number; cmd: string; alias: string; option: string; explain: string; target: string }

export default function CommandsPage() {
  const { show } = useToast()
  const [commands, setCommands] = useState<CommandItem[]>([])
  const [selected, setSelected] = useState<CommandItem | null>(null)
  const [option, setOption] = useState('')
  const [result, setResult] = useState('')
  const [sending, setSending] = useState(false)
  const [status, setStatus] = useState({ id: false, relay: false, checking: true })
  const [editing, setEditing] = useState<CommandItem | null | undefined>(undefined)
  const [form, setForm] = useState<Partial<CommandItem>>({ target: '21115' })

  const load = useCallback(async () => {
    try {
      const data = await get<PageData<CommandItem>>('/rustdesk/cmdList', { page: 1, page_size: 9999 })
      setCommands(normalizePage(data).list)
    } catch (error) { show(errorMessage(error), 'error') }
  }, [show])
  const check = useCallback(async () => {
    setStatus((value) => ({ ...value, checking: true }))
    const [id, relay] = await Promise.allSettled([post<string>('/rustdesk/sendCmd', { cmd: 'h', option: '', target: '21115' }), post<string>('/rustdesk/sendCmd', { cmd: 'h', option: '', target: '21117' })])
    setStatus({ id: id.status === 'fulfilled', relay: relay.status === 'fulfilled', checking: false })
  }, [])
  useEffect(() => { void load(); void check() }, [load, check])

  const send = async (event: FormEvent) => {
    event.preventDefault()
    if (!selected) return
    setSending(true)
    try {
      const response = await post<string>('/rustdesk/sendCmd', { cmd: selected.cmd, option, target: selected.target })
      setResult(response || '命令执行完成，无返回内容。')
      show('命令已发送', 'success')
    } catch (error) { setResult(errorMessage(error)); show(errorMessage(error), 'error') }
    finally { setSending(false) }
  }

  const openEditor = (command?: CommandItem) => {
    setEditing(command || null)
    setForm(command ? { ...command } : { cmd: '', alias: '', option: '', explain: '', target: '21115' })
  }
  const saveCommand = async (event: FormEvent) => {
    event.preventDefault()
    try {
      await post(editing ? '/rustdesk/cmdUpdate' : '/rustdesk/cmdCreate', form)
      show(editing ? '自定义命令已更新' : '自定义命令已创建', 'success')
      setEditing(undefined)
      await load()
    } catch (error) { show(errorMessage(error), 'error') }
  }
  const deleteCommand = async (command: CommandItem) => {
    if (!window.confirm(`确定删除命令 ${command.cmd} 吗？`)) return
    try { await post('/rustdesk/cmdDelete', { id: command.id }); show('自定义命令已删除', 'success'); await load() }
    catch (error) { show(errorMessage(error), 'error') }
  }

  return (
    <section>
      <div className="mb-5 flex flex-wrap items-end justify-between gap-3"><div><h1 className="text-xl font-semibold">服务指令</h1><p className="mt-1 text-sm text-base-content/50">向 ID Server 或 Relay Server 发送管理命令。</p></div><div className="flex gap-2"><button className="btn btn-sm desklink-action btn-ghost border border-base-300 bg-white px-4" onClick={() => void check()}><RefreshCw size={15} className={status.checking ? 'animate-spin' : ''} />检测服务</button><button className="btn btn-sm desklink-action border-0 bg-emerald-600 px-4 text-white hover:bg-emerald-700" onClick={() => openEditor()}><Plus size={15} />自定义命令</button></div></div>
      <div className="mb-5 grid gap-4 sm:grid-cols-2">
        {[['ID Server', status.id, '21115'], ['Relay Server', status.relay, '21117']].map(([name, available, port]) => <div key={String(name)} className="desklink-card flex items-center gap-4 p-4"><div className="flex h-9 w-9 items-center justify-center rounded-md bg-base-200"><ServerCog size={18} /></div><div><div className="text-sm font-medium">{name}</div><div className="text-xs text-base-content/40">控制端口 {port}</div></div><span className={`badge badge-sm ml-auto ${available ? 'badge-success badge-soft' : 'badge-error badge-soft'}`}>{status.checking ? '检测中' : available ? '可用' : '不可用'}</span></div>)}
      </div>
      <div className="grid gap-5 xl:grid-cols-[1fr_0.8fr]">
        <div className="desklink-card overflow-hidden"><div className="desklink-panel-header flex items-center border-b border-base-300 px-4 text-sm font-semibold">可用命令</div><div className="max-h-[580px] overflow-auto"><table className="table table-sm desklink-table"><thead><tr><th>命令</th><th>目标</th><th>说明</th><th /></tr></thead><tbody>{commands.map((command, index) => <tr key={`${command.target}-${command.cmd}-${index}`}><td><code className="text-xs">{command.cmd}</code>{command.alias && <span className="ml-2 text-xs text-base-content/35">{command.alias}</span>}</td><td><span className="badge badge-ghost badge-sm">{command.target === '21115' ? 'ID' : 'Relay'}</span></td><td className="max-w-xs truncate text-xs text-base-content/55">{command.explain}</td><td><div className="flex justify-end"><button className="btn btn-ghost btn-xs" onClick={() => { setSelected(command); setOption(''); setResult('') }} title="发送"><Play size={14} /></button>{command.id > 0 && <><button className="btn btn-ghost btn-xs" onClick={() => openEditor(command)} title="编辑"><Pencil size={14} /></button><button className="btn btn-ghost btn-xs text-error" onClick={() => void deleteCommand(command)} title="删除"><Trash2 size={14} /></button></>}</div></td></tr>)}</tbody></table></div></div>
        <div className="desklink-card self-start overflow-hidden"><div className="desklink-panel-header flex items-center gap-2 border-b border-base-300 px-4 text-sm font-semibold"><Terminal size={16} />命令控制台</div>{selected ? <form onSubmit={(event) => void send(event)} className="p-5"><div className="mb-4 rounded-md bg-[#111719] p-4 font-mono text-xs text-emerald-300"><span className="text-white/35">$ </span>{selected.cmd} {option}</div><label className="desklink-field"><span className="label text-xs text-base-content/55">参数 <span className="text-base-content/35">{selected.option}</span></span><input className="input input-bordered input-sm" value={option} onChange={(event) => setOption(event.target.value)} placeholder={selected.option} /></label><button className="btn btn-sm desklink-action mt-4 border-0 bg-emerald-600 px-4 text-white hover:bg-emerald-700" disabled={sending}>{sending && <span className="loading loading-spinner loading-xs" />}发送命令</button><label className="desklink-field mt-5"><span className="label text-xs text-base-content/55">返回结果</span><textarea className="textarea textarea-bordered h-44 bg-base-200 font-mono text-xs" value={result} readOnly placeholder="执行结果将显示在这里" /></label></form> : <div className="flex h-64 items-center justify-center text-sm text-base-content/40">从左侧选择一条命令</div>}</div>
      </div>
      {editing !== undefined && <div className="modal modal-open"><div className="modal-box max-w-lg rounded-lg p-0"><div className="flex h-14 items-center justify-between border-b border-base-300 px-5"><h3 className="font-semibold">{editing ? '编辑自定义命令' : '新增自定义命令'}</h3><button className="btn btn-ghost btn-sm desklink-icon-action" onClick={() => setEditing(undefined)}><X size={18} /></button></div><form onSubmit={(event) => void saveCommand(event)}><div className="grid gap-4 p-5 sm:grid-cols-2"><label className="desklink-field"><span className="label text-xs">命令</span><input className="input input-bordered input-sm" value={form.cmd || ''} onChange={(event) => setForm((value) => ({ ...value, cmd: event.target.value }))} required /></label><label className="desklink-field"><span className="label text-xs">别名</span><input className="input input-bordered input-sm" value={form.alias || ''} onChange={(event) => setForm((value) => ({ ...value, alias: event.target.value }))} /></label><label className="desklink-field"><span className="label text-xs">参数提示</span><input className="input input-bordered input-sm" value={form.option || ''} onChange={(event) => setForm((value) => ({ ...value, option: event.target.value }))} /></label><label className="desklink-field"><span className="label text-xs">目标服务</span><select className="select select-bordered select-sm" value={form.target || '21115'} onChange={(event) => setForm((value) => ({ ...value, target: event.target.value }))}><option value="21115">ID Server</option><option value="21117">Relay Server</option></select></label><label className="desklink-field sm:col-span-2"><span className="label text-xs">说明</span><input className="input input-bordered input-sm" value={form.explain || ''} onChange={(event) => setForm((value) => ({ ...value, explain: event.target.value }))} /></label></div><div className="flex justify-end gap-2 border-t border-base-300 px-5 py-4"><button type="button" className="btn btn-ghost btn-sm desklink-action px-4" onClick={() => setEditing(undefined)}>取消</button><button className="btn btn-sm desklink-action border-0 bg-emerald-600 px-4 text-white">保存</button></div></form></div><button className="modal-backdrop" onClick={() => setEditing(undefined)} aria-label="关闭" /></div>}
    </section>
  )
}
