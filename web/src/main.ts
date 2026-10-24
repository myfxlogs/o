// PHASE 总览活推（T-0067 脊+T-0075 卡绑定——API 四卡真值覆写空态；
// 断源=灰化露相禁演新鲜）。
const h = (s: string): string => {
  const d = document.createElement('div'); d.textContent = s; return d.innerHTML
}
async function ov(): Promise<void> {
  try {
    const r = await fetch('/api/overview')
    const d = await r.json()
    document.querySelectorAll('.card').forEach(c => (c as HTMLElement).style.opacity = '1')
    const b = document.querySelectorAll<HTMLElement>('.metric b')
    if (b.length >= 3) {
      b[0].textContent = '$' + (d.nav_total || 0).toLocaleString('en-US', { minimumFractionDigits: 2 })
      b[1].textContent = (d.pnl_today >= 0 ? '+' : '') + (d.pnl_today || 0).toFixed(2)
      b[2].textContent = (d.pnl_cum >= 0 ? '+' : '') + (d.pnl_cum || 0).toFixed(2)
    }
    const st = document.querySelector('.hero .stamp')
    if (st) st.textContent = '数据 ' + utcStamp() + ' · 刚刚'
    // doing：持仓行/静持空态（真空才出空态——有数据必渲染）。
    const db = document.getElementById('doing-body')
    if (db && d.doing && d.doing.length) {
      const live = d.doing.filter((x: { text: string }) => !x.text.startsWith('静持'))
      db.innerHTML = live.length
        ? live.map((x: { text: string; stamp: string }) =>
            `<div class="pos"><span class="dot"></span>${h(x.text)}</div>` +
            `<div class="ev">${h(x.stamp)} UTC</div>`).join('')
        : `<div class="empty"><p>静持中</p><i>「没有持仓，就是最深的持仓。」</i></div>`
    }
    const mb = document.getElementById('money-body')
    if (mb && d.money && d.money.length) {
      mb.innerHTML = d.money.map((m: { label: string; amount: number }) =>
        `<div class="row"><span>${h(m.label)}</span><b>$${m.amount.toFixed(1)}</b></div>`).join('')
    }
    const rb = document.getElementById('recent-body')
    if (rb && d.recent && d.recent.length) {
      rb.innerHTML = d.recent.map((x: string) => `<div class="ev">${h(x)}</div>`).join('')
    }
    // staff 关系图（T-0078：16 节点 SVG 拓扑+呼吸灯——折叠摘要「N 在岗·
    // 全员心跳正常/有迟滞」展开见图；API 缺面藏卡如实禁静态假名册）。
    const sc = document.getElementById('c-staff')
    if (sc) {
      if (d.staff && d.staff.length) {
        sc.style.display = ''
        const ss = document.getElementById('staff-sum')
        const bad = d.staff.filter((x: { state: number }) => x.state === 2).length
        const warn = d.staff.filter((x: { state: number }) => x.state === 1).length
        if (ss) ss.innerHTML = `<h3>员 工 · ${d.staff.length} 在册</h3>
          <div class="staffline" style="color:var(--dim)">${bad ? `${bad} 心跳迟滞` : warn ? `${warn} 心跳偏慢` : '全员心跳正常'} · 展开架构图 ▸</div>`
        const sg = document.getElementById('staff-graph')
        if (sg) sg.innerHTML = orgSVG(d.staff)
      } else {
        sc.style.display = 'none'
      }
    }
  } catch {
    document.querySelectorAll('.card').forEach(c => (c as HTMLElement).style.opacity = '.55') // 断源灰化露相
  }
}
function utcStamp(): string {
  const d = new Date()
  return String(d.getUTCHours()).padStart(2, '0') + ':' + String(d.getUTCMinutes()).padStart(2, '0') +
    ':' + String(d.getUTCSeconds()).padStart(2, '0') + ' UTC'
}
new EventSource('/api/watch').onmessage = () => ov()
ov()
setInterval(ov, 30000)

// orgSVG 16 节点关系图（T-0078：实线=数据/钱流，虚线=控制——呼吸灯态
// 绿<60s/琥珀<5m/红>5m 直映 last_beat；折叠态=摘要行点击展开）。
function orgSVG(staff: { name: string; duty: string; state: number }[]): string {
  const pos: Record<string, [number, number]> = {}
  const n = staff.length
  staff.forEach((s, i) => {
    const col = i % 4, row = Math.floor(i / 4)
    pos[s.name] = [70 + col * 180, 40 + row * 70]
  })
  const dots = ['#34d399', '#fbbf24', '#f87171']
  const anim = ['breath 2.4s ease-in-out infinite', 'breath .8s ease-in-out infinite', 'none']
  const dataFlow = staff.slice(1).map(s => s.name) // 简化拓扑：首节点=中枢汇聚
  let e = ''
  for (const t of dataFlow) {
    if (!pos[t] || !pos[staff[0].name]) continue
    const [x1, y1] = pos[staff[0].name]; const [x2, y2] = pos[t]
    e += `<line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="rgba(255,255,255,.15)" stroke-width="1"/>`
  }
  let g = e
  for (const s of staff) {
    const [x, y] = pos[s.name]
    g += `<circle cx="${x}" cy="${y}" r="6" fill="${dots[s.state] || dots[0]}"
      style="animation:${anim[s.state] || anim[0]}"/>
      <text x="${x + 12}" y="${y + 4}" fill="var(--txt)" font-size="12">${h(s.name)} ${h(s.duty)}</text>`
  }
  return `<svg viewBox="0 0 760 ${Math.ceil(n / 4) * 70 + 40}" style="width:100%;height:auto">${g}</svg>`
}
