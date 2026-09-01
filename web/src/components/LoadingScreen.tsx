export default function LoadingScreen() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-base-200">
      <div className="flex items-center gap-3 text-sm text-base-content/60">
        <span className="loading loading-spinner loading-md text-success" />
        正在加载 DeskLink 管理中心
      </div>
    </div>
  )
}
