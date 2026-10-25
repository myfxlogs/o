// stream.ts —— O EventSource 主刷新面（T-0113：onCase("dataRefresh") 等价件——
// /api/watch 主事件+30s 保底轮询，等价位记档）。
type Fn = (v?: unknown) => void
const subs: Fn[] = []
let booted = false
function boot() {
  if (booted) return
  booted = true
  const es = new EventSource('/api/watch')
  es.onmessage = () => subs.forEach(f => f({}))
  setInterval(() => subs.forEach(f => f({})), 30000) // 保底轮询（无专属 topic=主刷新等价位）
}
export function onCase(name: string, fn: Fn): () => void {
  boot()
  if (name === 'dataRefresh') subs.push(fn)
  return () => {
    const i = subs.indexOf(fn)
    if (i >= 0) subs.splice(i, 1)
  }
}
