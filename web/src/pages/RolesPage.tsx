import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Pencil, Plus, RefreshCw, ShieldCheck, Trash2, X } from 'lucide-react'
import { errorMessage, get, normalizePage, post, type PageData } from '../lib/api'
import { useToast } from '../components/Toast'

interface Permission { key: string; label: string }
interface Role { id: number; name: string; code: string; built_in: boolean; status: number; permissions: string[]; remark?: string }

const emptyRole = (): Role => ({ id: 0, name: '', code: '', built_in: false, status: 1, permissions: [], remark: '' })

export default function RolesPage() {
  const { show } = useToast()
  const [roles, setRoles] = useState<Role[]>([])
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [editing, setEditing] = useState<Role | null | undefined>(undefined)
  const [form, setForm] = useState<Role>(emptyRole())
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [rolePage, permissionList] = await Promise.all([get<PageData<Role>>('/role/list', { page: 1, page_size: 1000 }), get<Permission[]>('/role/permissions')])
      setRoles(normalizePage(rolePage).list)
      setPermissions(Array.isArray(permissionList) ? permissionList : [])
    } catch (error) { show(errorMessage(error), 'error') }
    finally { setLoading(false) }
  }, [show])
  useEffect(() => { void load() }, [load])

  const openEditor = (role?: Role) => { setForm(role ? { ...role, permissions: [...(role.permissions || [])] } : emptyRole()); setEditing(role || null) }
  const save = async (event: FormEvent) => {
    event.preventDefault()
    try {
      await post(form.id ? '/role/update' : '/role/create', form)
      show(form.id ? '角色已更新' : '角色已创建', 'success')
      setEditing(undefined)
      await load()
    } catch (error) { show(errorMessage(error), 'error') }
  }
  const remove = async (role: Role) => {
    if (role.built_in || !window.confirm(`确定删除角色“${role.name}”吗？`)) return
    try { await post('/role/delete', { id: role.id }); show('角色已删除', 'success'); await load() }
    catch (error) { show(errorMessage(error), 'error') }
  }
  const togglePermission = (key: string) => setForm((value) => ({ ...value, permissions: value.permissions.includes(key) ? value.permissions.filter((item) => item !== key) : [...value.permissions, key] }))

  return <section>
    <div className="mb-5 flex flex-wrap items-end justify-between gap-3"><div><h1 className="text-xl font-semibold">角色权限</h1><p className="mt-1 text-sm text-base-content/50">为控制台账号分配菜单权限。管理员、审计员和操作员为内置角色，自定义角色可按菜单范围配置。</p></div><div className="flex gap-2"><button className="btn btn-sm btn-ghost desklink-icon-action border border-base-300 bg-white" onClick={() => void load()} title="刷新"><RefreshCw size={15} className={loading ? 'animate-spin' : ''} /></button><button className="btn btn-sm desklink-action border-0 bg-emerald-600 px-4 text-white hover:bg-emerald-700" onClick={() => openEditor()}><Plus size={15} />新增角色</button></div></div>
    <div className="desklink-card overflow-hidden"><div className="overflow-x-auto"><table className="table desklink-table table-sm w-full"><thead className="bg-[#f8f9fa]"><tr><th>角色名称</th><th>角色代码</th><th>类型</th><th>菜单权限</th><th className="w-28 text-right">操作</th></tr></thead><tbody>{loading ? <tr><td colSpan={5} className="h-52 text-center"><span className="loading loading-spinner loading-md text-success" /></td></tr> : roles.length === 0 ? <tr><td colSpan={5} className="h-52 text-center text-sm text-base-content/40">暂无角色</td></tr> : roles.map((role) => <tr key={role.id}><td><div className="flex items-center gap-2 font-medium"><ShieldCheck size={15} className="text-emerald-600" />{role.name}</div></td><td><code className="text-xs text-base-content/55">{role.code}</code></td><td><span className={`badge badge-sm ${role.built_in ? 'badge-info badge-soft' : 'badge-ghost'}`}>{role.built_in ? '内置' : '自定义'}</span></td><td><div className="flex max-w-xl flex-wrap gap-1">{role.permissions.includes('*') ? <span className="badge badge-success badge-soft badge-sm">全部菜单</span> : role.permissions.map((key) => <span key={key} className="badge badge-ghost badge-sm">{permissions.find((item) => item.key === key)?.label || key}</span>)}</div></td><td><div className="flex justify-end gap-1"><button className="btn btn-ghost btn-xs" onClick={() => openEditor(role)} title="编辑"><Pencil size={14} /></button>{!role.built_in && <button className="btn btn-ghost btn-xs text-error" onClick={() => void remove(role)} title="删除"><Trash2 size={14} /></button>}</div></td></tr>)}</tbody></table></div></div>
    {editing !== undefined && <div className="modal modal-open"><div className="modal-box desklink-modal max-w-2xl rounded-lg p-0"><div className="flex h-14 items-center justify-between border-b border-base-300 px-5"><h3 className="font-semibold">{editing ? `编辑${form.name}` : '新增角色'}</h3><button className="btn btn-ghost btn-sm desklink-icon-action" onClick={() => setEditing(undefined)}><X size={18} /></button></div><form onSubmit={(event) => void save(event)}><div className="space-y-4 p-5"><div className="grid gap-4 sm:grid-cols-2"><label className="desklink-field"><span className="label text-xs text-base-content/60">角色名称</span><input className="input input-bordered input-sm" value={form.name} disabled={form.built_in} onChange={(event) => setForm((value) => ({ ...value, name: event.target.value }))} required /></label><label className="desklink-field"><span className="label text-xs text-base-content/60">角色代码</span><input className="input input-bordered input-sm" value={form.code} disabled={form.built_in} onChange={(event) => setForm((value) => ({ ...value, code: event.target.value }))} required /></label></div><label className="desklink-field"><span className="label text-xs text-base-content/60">备注</span><input className="input input-bordered input-sm" value={form.remark || ''} disabled={form.built_in} onChange={(event) => setForm((value) => ({ ...value, remark: event.target.value }))} /></label><div><div className="mb-2 text-xs text-base-content/60">菜单权限</div><div className="grid gap-2 rounded-md border border-base-200 p-3 sm:grid-cols-2">{permissions.map((permission) => <label key={permission.key} className="flex items-center gap-2 rounded px-2 py-1.5 text-sm hover:bg-base-200"><input type="checkbox" className="checkbox checkbox-sm checkbox-success" disabled={form.built_in} checked={form.permissions.includes('*') || form.permissions.includes(permission.key)} onChange={() => togglePermission(permission.key)} /><span>{permission.label}</span></label>)}</div></div></div><div className="flex justify-end gap-2 border-t border-base-300 bg-base-200/40 px-5 py-4"><button type="button" className="btn btn-sm desklink-action btn-ghost px-4" onClick={() => setEditing(undefined)}>关闭</button>{!form.built_in && <button className="btn btn-sm desklink-action border-0 bg-emerald-600 px-4 text-white hover:bg-emerald-700">保存</button>}</div></form></div><button className="modal-backdrop" onClick={() => setEditing(undefined)} aria-label="关闭" /></div>}
  </section>
}
