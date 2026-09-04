import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import AppLayout from './components/AppLayout'
import LoadingScreen from './components/LoadingScreen'
import ResourcePage from './components/ResourcePage'
import { ToastProvider } from './components/Toast'
import { resources } from './features/resources'
import { AuthProvider, useAuth } from './lib/auth'
import CommandsPage from './pages/CommandsPage'
import DashboardPage from './pages/DashboardPage'
import LoginPage from './pages/LoginPage'
import ProfilePage from './pages/ProfilePage'
import RegisterPage from './pages/RegisterPage'
import SettingsPage from './pages/SettingsPage'
import RecordingsPage from './pages/RecordingsPage'

function ProtectedLayout() {
  const { user, loading } = useAuth()
  const location = useLocation()
  if (loading) return <LoadingScreen />
  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  return <AppLayout />
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const { isAdmin } = useAuth()
  return isAdmin ? children : <Navigate to="/" replace />
}

function ApplicationRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route element={<ProtectedLayout />}>
        <Route index element={<DashboardPage />} />
        <Route path="profile" element={<ProfilePage />} />
        <Route path="my/devices" element={<ResourcePage config={resources.myDevices} />} />
        <Route path="my/collections" element={<ResourcePage config={resources.myCollections} />} />
        <Route path="my/address-books" element={<ResourcePage config={resources.myAddressBooks} />} />
        <Route path="my/tags" element={<ResourcePage config={resources.myTags} />} />
        <Route path="my/collection-rules" element={<ResourcePage config={resources.myCollectionRules} />} />
        <Route path="my/shares" element={<ResourcePage config={resources.myShares} />} />
        <Route path="my/login-logs" element={<ResourcePage config={resources.myLoginLogs} />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="admin/devices" element={<AdminRoute><ResourcePage config={resources.devices} /></AdminRoute>} />
        <Route path="admin/users" element={<AdminRoute><ResourcePage config={resources.users} /></AdminRoute>} />
        <Route path="admin/groups" element={<AdminRoute><ResourcePage config={resources.groups} /></AdminRoute>} />
        <Route path="admin/device-groups" element={<AdminRoute><ResourcePage config={resources.deviceGroups} /></AdminRoute>} />
        <Route path="admin/address-books" element={<AdminRoute><ResourcePage config={resources.addressBooks} /></AdminRoute>} />
        <Route path="admin/collections" element={<AdminRoute><ResourcePage config={resources.collections} /></AdminRoute>} />
        <Route path="admin/collection-rules" element={<AdminRoute><ResourcePage config={resources.collectionRules} /></AdminRoute>} />
        <Route path="admin/tags" element={<AdminRoute><ResourcePage config={resources.tags} /></AdminRoute>} />
        <Route path="admin/login-logs" element={<AdminRoute><ResourcePage config={resources.loginLogs} /></AdminRoute>} />
        <Route path="admin/connection-audit" element={<AdminRoute><ResourcePage config={resources.connectionAudit} /></AdminRoute>} />
        <Route path="admin/access-rules" element={<AdminRoute><ResourcePage config={resources.accessRules} /></AdminRoute>} />
        <Route path="admin/file-audit" element={<AdminRoute><ResourcePage config={resources.fileAudit} /></AdminRoute>} />
        <Route path="admin/recordings" element={<AdminRoute><RecordingsPage /></AdminRoute>} />
        <Route path="admin/tokens" element={<AdminRoute><ResourcePage config={resources.tokens} /></AdminRoute>} />
        <Route path="admin/shares" element={<AdminRoute><ResourcePage config={resources.shares} /></AdminRoute>} />
        <Route path="admin/commands" element={<AdminRoute><CommandsPage /></AdminRoute>} />
        <Route path="admin/oauth" element={<AdminRoute><ResourcePage config={resources.oauth} /></AdminRoute>} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default function App() {
  return <ToastProvider><AuthProvider><ApplicationRoutes /></AuthProvider></ToastProvider>
}
