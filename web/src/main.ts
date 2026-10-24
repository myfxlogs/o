// PHASE 总览活推（T-0067：/api/overview JSON+SSE /api/watch——所见即所得零缓存）。
async function ov(): Promise<void> {
  try {
    const r = await fetch('/api/overview')
    const d = await r.json()
    const b = document.querySelectorAll<HTMLElement>('.metric b')
    if (b.length >= 3) {
      b[0].textContent = '$' + (d.nav_total || 0).toLocaleString('en-US', { minimumFractionDigits: 2 })
      b[1].textContent = (d.pnl_today >= 0 ? '+' : '') + (d.pnl_today || 0).toFixed(2)
      b[2].textContent = (d.pnl_cum >= 0 ? '+' : '') + (d.pnl_cum || 0).toFixed(2)
    }
    const st = document.querySelector('.hero .stamp')
    if (st) st.textContent = '数据 ' + (() => { const d = new Date(); return String(d.getUTCHours()).padStart(2,'0') + ':' + String(d.getUTCMinutes()).padStart(2,'0') + ':' + String(d.getUTCSeconds()).padStart(2,'0') + ' UTC' })() + ' · 刚刚'
  } catch { /* 断源=读数灰化露相面（禁演新鲜） */ }
}
new EventSource('/api/watch').onmessage = () => ov()
ov()
setInterval(ov, 30000) // 兜底刷（SSE 断线期兜底非周期环主径）
