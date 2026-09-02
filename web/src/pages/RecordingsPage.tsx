import { useCallback, useEffect, useMemo, useState } from 'react'
import { Download, Film, Play, RefreshCw, Save, Search, Trash2, X } from 'lucide-react'
import { errorMessage, get, normalizePage, post, type PageData } from '../lib/api'
import { useToast } from '../components/Toast'

type PolicyMode = 'off' | 'all' | 'selected'

interface RecordingPolicy {
  mode: PolicyMode
  retention_days: number
  peer_ids: string[]
}

interface Peer {
  row_id: number
  id: string
  hostname: string
  alias: string
  username: string
  os: string
}

interface Recording {
  id: number
  peer_id: string
  from_peer: string
  from_name: string
  session_id: string
  original_name: string
  container: string
  codec: string
  status: string
  size: number
  duration_ms: number
  started_at: number
  completed_at: number
}

interface AccessResponse { url: string; expires_at: number }

const modeOptions: { value: PolicyMode; label: string; description: string }[] = [
  { value: 'off', label: '全局关闭', description: '所有设备均不自动录制' },
  { value: 'all', label: '全局开启', description: '所有被控设备强制自动录制' },
  { value: 'selected', label: '部分开启', description: '仅录制指定的被控设备' },
]

function formatBytes(value: number) {
  if (!value) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  return `${(value / 1024 ** index).toFixed(index ? 1 : 0)} ${units[index]}`
}

function formatDuration(ms: number) {
  const total = Math.max(0, Math.round(ms / 1000))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const seconds = total % 60
  return hours ? `${hours}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}` : `${minutes}:${String(seconds).padStart(2, '0')}`
}

function formatTime(timestamp: number) {
  return timestamp ? new Date(timestamp * 1000).toLocaleString('zh-CN', { hour12: false }) : '-'
}

export default function RecordingsPage() {
  const { show } = useToast()
  const [policy, setPolicy] = useState<RecordingPolicy>({ mode: 'off', retention_days: 30, peer_ids: [] })
  const [peers, setPeers] = useState<Peer[]>([])
  const [records, setRecords] = useState<PageData<Recording>>(normalizePage())
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [page, setPage] = useState(1)
  const [peerFilter, setPeerFilter] = useState('')
  const [fromFilter, setFromFilter] = useState('')
  const [deviceSearch, setDeviceSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [startedAfter, setStartedAfter] = useState('')
  const [startedBefore, setStartedBefore] = useState('')
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  const [preview, setPreview] = useState<{ record: Recording; url: string } | null>(null)

  const loadPolicy = useCallback(async () => {
    const value = await get<RecordingPolicy>('/recordings/policy')
    setPolicy({ ...value, peer_ids: Array.isArray(value.peer_ids) ? value.peer_ids : [] })
  }, [])

  const loadPeers = useCallback(async () => {
    const value = await get<Partial<PageData<Peer>>>('/peer/list', { page: 1, page_size: 1000 })
    setPeers(normalizePage(value).list)
  }, [])

  const loadRecords = useCallback(async () => {
    const value = await get<Partial<PageData<Recording>>>('/recordings/list', {
      page, page_size: 20, peer_id: peerFilter || undefined, from_peer: fromFilter || undefined,
      status: statusFilter || undefined,
      started_after: startedAfter ? Math.floor(new Date(startedAfter).getTime() / 1000) : undefined,
      started_before: startedBefore ? Math.floor(new Date(startedBefore).getTime() / 1000) + 86399 : undefined,
    })
    setRecords(normalizePage(value))
    setSelectedIds([])
  }, [page, peerFilter, fromFilter, statusFilter, startedAfter, startedBefore])

  useEffect(() => {
    Promise.all([loadPolicy(), loadPeers(), loadRecords()])
      .catch((error) => show(errorMessage(error), 'error'))
      .finally(() => setLoading(false))
  }, [loadPolicy, loadPeers, loadRecords, show])

  const visiblePeers = useMemo(() => {
    const term = deviceSearch.trim().toLowerCase()
    if (!term) return peers
    return peers.filter((peer) => [peer.id, peer.hostname, peer.alias, peer.username].some((value) => value?.toLowerCase().includes(term)))
  }, [deviceSearch, peers])

  const togglePeer = (peerId: string) => {
    setPolicy((current) => ({
      ...current,
      peer_ids: current.peer_ids.includes(peerId) ? current.peer_ids.filter((id) => id !== peerId) : [...current.peer_ids, peerId],
    }))
  }

  const savePolicy = async () => {
    setSaving(true)
    try {
      const value = await post<RecordingPolicy>('/recordings/policy', policy)
      setPolicy(value)
      show('录制策略已保存', 'success')
    } catch (error) {
      show(errorMessage(error), 'error')
    } finally {
      setSaving(false)
    }
  }

  const requestAccess = async (record: Recording, download: boolean) => {
    try {
      const access = await get<AccessResponse>(`/recordings/${record.id}/access`, { download: download ? 1 : 0 })
      if (download) window.location.assign(access.url)
      else setPreview({ record, url: access.url })
    } catch (error) {
      show(errorMessage(error), 'error')
    }
  }

  const deleteRecording = async (record: Recording) => {
    if (!window.confirm(`确定删除录像 ${record.original_name} 吗？`)) return
    try {
      await post('/recordings/delete', { id: record.id })
      show('录像已删除', 'success')
      await loadRecords()
    } catch (error) {
      show(errorMessage(error), 'error')
    }
  }

  const batchDelete = async () => {
    if (!selectedIds.length || !window.confirm(`确定删除选中的 ${selectedIds.length} 条录像吗？`)) return
    try {
      await post('/recordings/batchDelete', { ids: selectedIds })
      show('选中录像已删除', 'success')
      await loadRecords()
    } catch (error) {
      show(errorMessage(error), 'error')
    }
  }

  if (loading) return <div className="flex min-h-64 items-center justify-center"><span className="loading loading-spinner text-emerald-600" /></div>

  return (
    <section className="space-y-5">
      <div className="flex items-end justify-between gap-4">
        <div><h1 className="text-xl font-semibold">会话录像</h1><p className="mt-1 text-sm text-base-content/50">管理被控设备的强制录制策略和审计录像。</p></div>
        <button className="btn btn-sm border-0 bg-emerald-600 text-white hover:bg-emerald-700" onClick={() => void savePolicy()} disabled={saving}>
          {saving ? <span className="loading loading-spinner loading-xs" /> : <Save size={15} />}保存策略
        </button>
      </div>

      <div className="desklink-card p-5">
        <div className="grid gap-3 lg:grid-cols-3">
          {modeOptions.map((option) => (
            <button key={option.value} className={`min-h-20 rounded-md border p-4 text-left transition ${policy.mode === option.value ? 'border-emerald-500 bg-emerald-50' : 'border-base-300 hover:border-base-content/25'}`} onClick={() => setPolicy((value) => ({ ...value, mode: option.value }))}>
              <span className="flex items-center gap-2 text-sm font-semibold"><span className={`h-2.5 w-2.5 rounded-full ${policy.mode === option.value ? 'bg-emerald-500' : 'bg-base-300'}`} />{option.label}</span>
              <span className="mt-1.5 block text-xs text-base-content/50">{option.description}</span>
            </button>
          ))}
        </div>
        <label className="mt-4 flex max-w-xs items-center gap-3 text-sm"><span className="whitespace-nowrap text-base-content/60">保留天数</span><input type="number" min={1} max={3650} className="input input-bordered input-sm w-28" value={policy.retention_days} onChange={(event) => setPolicy((value) => ({ ...value, retention_days: Number(event.target.value) }))} /></label>
      </div>

      {policy.mode === 'selected' && (
        <div className="desklink-card overflow-hidden">
          <div className="flex flex-wrap items-center gap-3 border-b border-base-300 px-5 py-4">
            <div><div className="text-sm font-semibold">录制设备</div><div className="text-xs text-base-content/45">已选择 {policy.peer_ids.length} 台设备</div></div>
            <label className="input input-bordered input-sm ml-auto flex w-64 items-center gap-2"><Search size={14} /><input className="grow" value={deviceSearch} onChange={(event) => setDeviceSearch(event.target.value)} placeholder="搜索设备 ID 或名称" /></label>
          </div>
          <div className="grid max-h-64 overflow-y-auto sm:grid-cols-2 xl:grid-cols-3">
            {visiblePeers.map((peer) => <label key={peer.id} className="flex min-h-14 cursor-pointer items-center gap-3 border-b border-r border-base-200 px-5 py-3 hover:bg-base-200/50"><input type="checkbox" className="checkbox checkbox-success checkbox-sm" checked={policy.peer_ids.includes(peer.id)} onChange={() => togglePeer(peer.id)} /><span className="min-w-0"><span className="block truncate text-sm font-medium">{peer.alias || peer.hostname || peer.id}</span><span className="block truncate text-xs text-base-content/45">{peer.id} · {peer.os || '未知系统'}</span></span></label>)}
            {!visiblePeers.length && <div className="col-span-full p-8 text-center text-sm text-base-content/40">没有匹配的设备</div>}
          </div>
        </div>
      )}

      <div className="desklink-card overflow-hidden">
        <div className="flex flex-wrap items-center gap-2 border-b border-base-300 px-5 py-4">
          <div className="mr-auto"><div className="text-sm font-semibold">录像记录</div><div className="text-xs text-base-content/45">共 {records.total} 条</div></div>
          <input className="input input-bordered input-sm w-40" placeholder="被控设备 ID" value={peerFilter} onChange={(event) => { setPage(1); setPeerFilter(event.target.value) }} />
          <input className="input input-bordered input-sm w-40" placeholder="控制方 ID/名称" value={fromFilter} onChange={(event) => { setPage(1); setFromFilter(event.target.value) }} />
          <select className="select select-bordered select-sm w-28" value={statusFilter} onChange={(event) => { setPage(1); setStatusFilter(event.target.value) }}><option value="">全部状态</option><option value="complete">已完成</option><option value="uploading">上传中</option><option value="transcoding">转码中</option><option value="failed">失败</option></select>
          <input type="date" className="input input-bordered input-sm w-36" title="开始日期" value={startedAfter} onChange={(event) => { setPage(1); setStartedAfter(event.target.value) }} />
          <input type="date" className="input input-bordered input-sm w-36" title="结束日期" value={startedBefore} onChange={(event) => { setPage(1); setStartedBefore(event.target.value) }} />
          {selectedIds.length > 0 && <button className="btn btn-error btn-outline btn-sm" onClick={() => void batchDelete()}><Trash2 size={14} />删除 {selectedIds.length} 项</button>}
          <button className="btn btn-ghost btn-sm" title="刷新" onClick={() => void loadRecords()}><RefreshCw size={15} /></button>
        </div>
        <div className="overflow-x-auto">
          <table className="table table-sm">
            <thead><tr><th><input type="checkbox" className="checkbox checkbox-sm" aria-label="选择当前页" checked={records.list.length > 0 && records.list.every((record) => selectedIds.includes(record.id))} onChange={(event) => setSelectedIds(event.target.checked ? records.list.map((record) => record.id) : [])} /></th><th>开始时间</th><th>被控设备</th><th>控制方</th><th>时长</th><th>格式</th><th>大小</th><th>状态</th><th className="text-right">操作</th></tr></thead>
            <tbody>
              {records.list.map((record) => <tr key={record.id}><td><input type="checkbox" className="checkbox checkbox-sm" aria-label={`选择录像 ${record.id}`} checked={selectedIds.includes(record.id)} onChange={() => setSelectedIds((ids) => ids.includes(record.id) ? ids.filter((id) => id !== record.id) : [...ids, record.id])} /></td><td className="whitespace-nowrap">{formatTime(record.started_at)}</td><td className="font-medium">{record.peer_id}</td><td><span className="block">{record.from_name || record.from_peer || '-'}</span>{record.from_name && record.from_peer && <span className="block text-xs text-base-content/40">{record.from_peer}</span>}</td><td>{formatDuration(record.duration_ms)}</td><td><span className="badge badge-ghost badge-sm uppercase">{record.codec || record.container}</span></td><td>{formatBytes(record.size)}</td><td><span className={`badge badge-sm ${record.status === 'complete' ? 'badge-success badge-soft' : record.status === 'failed' ? 'badge-error badge-soft' : 'badge-warning badge-soft'}`}>{record.status === 'complete' ? '已完成' : record.status === 'failed' ? '失败' : record.status === 'transcoding' ? '转码中' : '上传中'}</span></td><td><div className="flex justify-end gap-1"><button className="btn btn-ghost btn-xs" title="预览" disabled={record.status !== 'complete'} onClick={() => void requestAccess(record, false)}><Play size={14} /></button><button className="btn btn-ghost btn-xs" title="下载原始文件" disabled={record.status === 'uploading'} onClick={() => void requestAccess(record, true)}><Download size={14} /></button><button className="btn btn-ghost btn-xs text-error" title="删除" onClick={() => void deleteRecording(record)}><Trash2 size={14} /></button></div></td></tr>)}
              {!records.list.length && <tr><td colSpan={9}><div className="flex flex-col items-center gap-2 py-12 text-base-content/35"><Film size={30} strokeWidth={1.4} /><span className="text-sm">暂无会话录像</span></div></td></tr>}
            </tbody>
          </table>
        </div>
        {records.total > records.page_size && <div className="flex items-center justify-end gap-2 border-t border-base-300 px-5 py-3"><button className="btn btn-outline btn-xs" disabled={page <= 1} onClick={() => setPage((value) => value - 1)}>上一页</button><span className="text-xs text-base-content/50">第 {page} 页</span><button className="btn btn-outline btn-xs" disabled={page * records.page_size >= records.total} onClick={() => setPage((value) => value + 1)}>下一页</button></div>}
      </div>

      {preview && <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/70 p-5"><div className="w-full max-w-5xl overflow-hidden rounded-md bg-[#101617] shadow-2xl"><div className="flex h-12 items-center border-b border-white/10 px-4 text-white"><Film size={16} className="mr-2 text-emerald-400" /><span className="min-w-0 flex-1 truncate text-sm">{preview.record.original_name}</span><button className="btn btn-ghost btn-sm text-white/60 hover:text-white" onClick={() => setPreview(null)}><X size={18} /></button></div><video className="max-h-[75vh] w-full bg-black" src={preview.url} controls autoPlay /></div></div>}
    </section>
  )
}
