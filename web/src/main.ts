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
        if (ss && !ss.dataset.wired) {
          ss.dataset.wired = '1'
          ss.onclick = () => {
            sc.style.gridColumn = 'span 12'
            const g = document.getElementById('staff-graph')
            if (g) g.style.display = 'block'
          }
        }
        if (ss) ss.innerHTML = `<h3>员 工 · ${d.staff.length} 在册</h3>
          <div class="staffline" style="color:var(--dim)">${bad ? `${bad} 心跳迟滞` : warn ? `${warn} 心跳偏慢` : '全员心跳正常'} · 展开架构图 ▸</div>`
        const sg = document.getElementById('staff-graph')
        if (sg) {
          sg.innerHTML = orgSVG(d.staff)
          if (!sg.dataset.wired) {
            sg.dataset.wired = '1'
            sg.onclick = (e) => {
              e.stopPropagation()
              sg.style.display = 'none'
              sc.style.gridColumn = 'span 6'
            }
          }
        }
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

// orgSVG 霓虹关系图（T-0079 钦定规格：深色底+feGaussianBlur 辉光连线；
// 实线青=钱/仓流主链，虚线紫=控制/数据/回环；节点=圆角卡名+职+呼吸灯；
// 右上徽标 ◉ N 在岗；拓扑按 16 岗语义写死非力导向；底部图例行）。
function orgSVG(staff: { name: string; duty: string; state: number }[]): string {
  // 钦定座次（16 岗语义固定——registry name 对位，缺席=跳过不虚构）。
  const grid: [string, number, number][] = [
    ['探员', 60, 40], ['验证官', 220, 40], ['命官', 380, 40], ['门官', 540, 40],
    ['执事', 700, 40], ['评史官', 60, 140], ['配管', 220, 140], ['仓巡', 380, 140],
    ['司账', 540, 140], ['核账', 620, 140], ['信使', 700, 140],
    ['行情巡', 60, 240], ['私流守', 220, 240], ['汇市巡', 380, 240],
    ['集使', 540, 240], ['phase-core', 700, 240],
  ]
  const byName = new Map(staff.map(s => [s.name, s]))
  const pos = new Map<string, [number, number]>()
  for (const [nm, x, y] of grid) if (byName.has(nm)) pos.set(nm, [x, y])
  const solid: [string, string][] = [
    ['探员', '验证官'], ['验证官', '命官'], ['命官', '门官'], ['门官', '执事'],
    ['执事', '司账'], ['司账', '核账'], ['核账', '信使'],
  ]
  const dash: [string, string][] = [
    ['评史官', '探员'], ['配管', '命官'], ['仓巡', '执事'],
    ['行情巡', '探员'], ['私流守', '执事'], ['汇市巡', '执事'], ['集使', '命官'],
    ['phase-core', '信使'],
  ]
  const link = (a: string, b: string, c: string, d: string) => {
    const p = pos.get(a), q = pos.get(b)
    if (!p || !q) return ''
    return `<line x1="${p[0] + 40}" y1="${p[1] + 14}" x2="${q[0] + 40}" y2="${q[1] + 14}"
      stroke="${c}" stroke-width="1.5" stroke-dasharray="${d}" />`
  }
  const dots = ['#34d399', '#fbbf24', '#f87171']
  let g = ''
  for (const [a, b] of solid) g += link(a, b, '#7dd3fc', '0')
  for (const [a, b] of dash) g += link(a, b, '#94a3b8', '4 3')
  for (const [nm, [x, y]] of pos) {
    const s = byName.get(nm)!
    g += `<rect x="${x}" y="${y}" rx="8" width="120" height="30" fill="rgba(255,255,255,.06)"
      stroke="rgba(255,255,255,.12)"/>
      <circle cx="${x + 8}" cy="${y + 8}" r="4" fill="${dots[s.state] || dots[0]}"
      />
      <text x="${x + 18}" y="${y + 13}" fill="var(--txt)" font-size="11">${h(nm)}</text>
      <text x="${x + 18}" y="${y + 25}" fill="var(--dim)" font-size="9">${h(s.duty)}</text>`
  }
  g += `<text x="620" y="20" fill="var(--dim)" font-size="12">◉ ${staff.length} 个员工在岗</text>
    <text x="20" y="290" fill="var(--dim)" font-size="11">实线=钱/仓流｜虚线=控制/数据/回环｜灯三色</text>`
  return `<svg viewBox="0 0 780 310" style="width:100%;height:auto">${g}</svg>`
}
