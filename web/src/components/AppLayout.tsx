import { useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import {
  Activity,
  BookUser,
  ChevronDown,
  CircleUserRound,
  Command,
  ContactRound,
  Cpu,
  FileClock,
  FolderKanban,
  KeyRound,
  LayoutDashboard,
  LogOut,
  Menu,
  MonitorDot,
  Video,
  Network,
  UserCog,
  Settings,
  Share2,
  ShieldCheck,
  Tags,
  Users,
  UsersRound,
  X,
} from 'lucide-react'
import { useAuth } from '../lib/auth'

interface NavItem {
  label: string
  path: string
  icon: typeof LayoutDashboard
  admin?: boolean
  permission?: string
}

interface NavGroup {
  label: string
  items: NavItem[]
}

const groups: NavGroup[] = [
  {
    label: '工作台',
    items: [
      { label: '概览', path: '/', icon: LayoutDashboard },
      { label: '个人资料', path: '/profile', icon: CircleUserRound },
      { label: '我的设备', path: '/my/devices', icon: MonitorDot },
      { label: '地址簿集合', path: '/my/collections', icon: FolderKanban },
      { label: '我的地址簿', path: '/my/address-books', icon: BookUser },
      { label: '我的标签', path: '/my/tags', icon: Tags },
      { label: '共享规则', path: '/my/collection-rules', icon: Share2 },
      { label: '我的分享', path: '/my/shares', icon: ShieldCheck },
      { label: '我的登录记录', path: '/my/login-logs', icon: FileClock },
    ],
  },
  {
    label: '资源管理',
    items: [
      { label: '设备', path: '/admin/devices', icon: Cpu, admin: true, permission: 'devices' },
      { label: '用户', path: '/admin/users', icon: Users, admin: true, permission: 'users' },
      { label: '用户组', path: '/admin/groups', icon: UsersRound, admin: true, permission: 'groups' },
      { label: '设备组', path: '/admin/device-groups', icon: Network, admin: true, permission: 'device-groups' },
      { label: '地址簿', path: '/admin/address-books', icon: ContactRound, admin: true, permission: 'address-books' },
      { label: '地址簿集合', path: '/admin/collections', icon: FolderKanban, admin: true, permission: 'collections' },
      { label: '共享规则', path: '/admin/collection-rules', icon: Share2, admin: true, permission: 'collection-rules' },
      { label: '标签', path: '/admin/tags', icon: Tags, admin: true, permission: 'tags' },
    ],
  },
  {
    label: '安全审计',
    items: [
      { label: '登录日志', path: '/admin/login-logs', icon: FileClock, admin: true, permission: 'login-logs' },
      { label: '连接审计', path: '/admin/connection-audit', icon: Activity, admin: true, permission: 'connection-audit' },
      { label: '授权管理', path: '/admin/access-rules', icon: ShieldCheck, admin: true, permission: 'access-rules' },
      { label: '文件审计', path: '/admin/file-audit', icon: ShieldCheck, admin: true, permission: 'file-audit' },
      { label: '会话录像', path: '/admin/recordings', icon: Video, admin: true, permission: 'recordings' },
      { label: '访问令牌', path: '/admin/tokens', icon: KeyRound, admin: true, permission: 'tokens' },
      { label: '分享记录', path: '/admin/shares', icon: Share2, admin: true, permission: 'shares' },
    ],
  },
  {
    label: '服务配置',
    items: [
      { label: '服务指令', path: '/admin/commands', icon: Command, admin: true, permission: 'commands' },
      { label: 'OAuth / OIDC', path: '/admin/oauth', icon: UserCog, admin: true, permission: 'oauth' },
      { label: '角色权限', path: '/admin/roles', icon: ShieldCheck, admin: true, permission: 'roles' },
      { label: '系统信息', path: '/settings', icon: Settings, admin: true, permission: 'settings' },
    ],
  },
]

function Navigation({ close }: { close?: () => void }) {
  const { isAdmin, hasPermission } = useAuth()
  return (
    <nav className="flex-1 overflow-y-auto px-3 py-4">
      {groups.map((group) => {
        const items = group.items.filter((item) => !item.admin || (item.permission === 'roles' ? isAdmin : Boolean(item.permission && hasPermission(item.permission))))
        if (!items.length) return null
        return (
          <div key={group.label} className="mb-5">
            <div className="mb-1 px-3 text-[11px] font-semibold uppercase text-white/35">{group.label}</div>
            <ul className="menu w-full gap-0.5 p-0 text-[13px]">
              {items.map((item) => {
                const Icon = item.icon
                return (
                  <li key={item.path}>
                    <NavLink to={item.path} end={item.path === '/'} onClick={close} className={({ isActive }) => isActive ? 'active h-9 rounded-md' : 'h-9 rounded-md'}>
                      <Icon size={16} strokeWidth={1.8} />
                      {item.label}
                    </NavLink>
                  </li>
                )
              })}
            </ul>
          </div>
        )
      })}
    </nav>
  )
}

export default function AppLayout() {
  const { user, logout, isAdmin } = useAuth()
  const [mobileOpen, setMobileOpen] = useState(false)
  const location = useLocation()
  const current = groups.flatMap((group) => group.items).find((item) => item.path === location.pathname)

  return (
    <div className="desklink-shell">
      <aside className="desklink-sidebar desklink-desktop-nav fixed inset-y-0 left-0 z-40 flex w-[224px] flex-col border-r border-white/5">
        <div className="flex h-16 items-center gap-3 border-b border-white/8 px-5">
          <img src="./logo.png" alt="DeskLink" className="h-8 w-8 object-contain" />
          <div>
            <div className="text-[15px] font-semibold text-white">DeskLink</div>
            <div className="text-[10px] text-white/35">社区服务管理中心</div>
          </div>
        </div>
        <Navigation />
        <div className="border-t border-white/8 p-3">
          <div className="flex items-center gap-3 rounded-md px-2 py-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-emerald-500/15 text-emerald-400"><CircleUserRound size={17} /></div>
            <div className="min-w-0 flex-1">
              <div className="truncate text-xs font-medium text-white">{user?.nickname || user?.username}</div>
              <div className="truncate text-[10px] text-white/35">{user?.role_name || (isAdmin ? '管理员' : '普通用户')}</div>
            </div>
            <button className="btn btn-ghost btn-xs text-white/50 hover:text-white" onClick={() => void logout()} title="退出登录"><LogOut size={15} /></button>
          </div>
        </div>
      </aside>

      {mobileOpen && (
        <div className="fixed inset-0 z-50 flex lg:hidden">
          <button className="absolute inset-0 bg-black/40" onClick={() => setMobileOpen(false)} aria-label="关闭导航" />
          <aside className="desklink-sidebar relative flex w-[250px] flex-col">
            <div className="flex h-16 items-center gap-3 border-b border-white/8 px-5">
              <img src="./logo.png" alt="DeskLink" className="h-8 w-8 object-contain" />
              <span className="font-semibold text-white">DeskLink</span>
              <button className="btn btn-ghost btn-sm ml-auto text-white" onClick={() => setMobileOpen(false)}><X size={18} /></button>
            </div>
            <Navigation close={() => setMobileOpen(false)} />
          </aside>
        </div>
      )}

      <main className="desklink-main min-h-screen ml-[224px]">
        <header className="sticky top-0 z-30 flex h-16 items-center border-b border-base-300 bg-white/95 px-5 backdrop-blur lg:px-7">
          <button className="btn btn-ghost btn-sm mr-2 lg:hidden" onClick={() => setMobileOpen(true)} aria-label="打开导航"><Menu size={19} /></button>
          <div>
            <div className="text-sm font-semibold text-base-content">{current?.label || 'DeskLink'}</div>
            <div className="text-[11px] text-base-content/45">社区服务端 · 适配客户端 1.4.9</div>
          </div>
          <div className="ml-auto dropdown dropdown-end">
            <button tabIndex={0} className="btn btn-ghost btn-sm gap-2 text-xs font-normal">
              {user?.nickname || user?.username}
              <ChevronDown size={14} />
            </button>
            <ul tabIndex={0} className="menu dropdown-content z-[50] mt-2 w-40 rounded-md border border-base-300 bg-base-100 p-1 shadow-lg">
              <li><NavLink to="/settings"><Settings size={15} />系统信息</NavLink></li>
              <li><button onClick={() => void logout()}><LogOut size={15} />退出登录</button></li>
            </ul>
          </div>
        </header>
        <div className="p-4 lg:p-7"><Outlet /></div>
      </main>
    </div>
  )
}
