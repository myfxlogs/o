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

// orgSVG v2 组织架构图（T-0079 返修④钦定——宪章 §6「一笔钱的旅程」拓扑：
// 顶层=phase-core(左)+业主·投资委(右金框)；钱程主链实线携产出物；
// 支撑回环+数据供数群虚线；状态灯=节点左上三色静点；素色无动画无辉光）。
function orgSVG(staff: { name: string; duty: string; state: number }[]): string {
  const byName = new Map(staff.map(s => [s.name, s]))
  const dots = ['#34d399', '#fbbf24', '#f87171']
  const node = (nm: string, x: number, y: number, gold = false) => {
    if (!byName.has(nm)) return ''
    const st = byName.get(nm)!
    const edge = gold ? 'stroke="#fbbf24"' : 'stroke="rgba(255,255,255,.12)"'
    return `<rect x="${x}" y="${y}" rx="8" width="140" height="36" fill="rgba(255,255,255,.06)" ${edge}/>
      <circle cx="${x + 10}" cy="${y + 9}" r="4" fill="${dots[st.state] || dots[0]}"/>
      <text x="${x + 18}" y="${y + 16}" fill="var(--txt)" font-size="12">${h(nm)}</text>
      <text x="${x + 18}" y="${y + 29}" fill="var(--dim)" font-size="9">${h(st.duty)}</text>`
  }
  const line = (x1: number, y1: number, x2: number, y2: number, dash = '') =>
    `<line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="#94a3b8" stroke-width="1.2" stroke-dasharray="${dash}"/>`
  const arrow = (x1: number, y1: number, x2: number, y2: number, label: string) =>
    line(x1, y1, x2, y2) +
    `<text x="${(x1 + x2) / 2 - 18}" y="${y1 - 6}" fill="var(--dim)" font-size="8">${h(label)}</text>`

  // 钦定座次（三层：顶层 20 / 主链+回环 110 / 供数群 250）。
  const P: Record<string, [number, number]> = {
    'phase-core': [40, 10], '业主·投资委': [640, 10],
    '探员': [40, 100], '验证官': [220, 100], '命官': [400, 100], '门官': [580, 100],
    '执事': [760, 100], '司账': [940, 100], '核账': [1120, 100], '信使': [1120, 220],
    '评史官': [220, 200], '配管': [400, 200], '仓巡': [760, 200],
    '集使': [40, 340], '行情巡': [220, 340], '私流守': [400, 340], '汇市巡': [580, 340],
  }
  // 业主节点=静态位（非 registry——金框只收报表+条件请求钦定）。
  let g = `<rect x="640" y="10" rx="8" width="170" height="36" fill="rgba(255,187,36,.08)" stroke="#fbbf24"/>
    <text x="652" y="26" fill="var(--txt)" font-size="12">业主·投资委</text>
    <text x="652" y="40" fill="var(--dim)" font-size="8">只收：经营报表+条件请求</text>`
  g += node('phase-core', P['phase-core'][0], P['phase-core'][1])
  // 心跳虚线：phase-core→业主（顶层回执）。
  g += line(210, 28, 640, 28, '5 4')
  g += `<text x="360" y="22" fill="var(--dim)" font-size="9">心跳活性回执</text>`
  // 钱程主链（实线+产出物标注——G0→G11 钦定序）。
  const chain: [string, string, string][] = [
    ['探员', '验证官', '提案'], ['验证官', '命官', '评估卡'], ['命官', '门官', '晋级令'],
    ['门官', '执事', '放行票'], ['执事', '司账', '成交回报'], ['司账', '核账', '账行'],
    ['核账', '信使', '对账结论'],
  ]
  for (const [a, b, lbl] of chain) {
    if (!P[a] || !P[b]) continue
    g += arrow(P[a][0] + 140, P[a][1] + 18, P[b][0], P[b][1] + 18, lbl)
  }
  // 信使→业主（报表上行实线+产出物）。
  g += line(P['信使'][0] + 70, P['信使'][1], P['信使'][0] + 70, P['业主·投资委'][1] + 60) +
       line(P['信使'][0] + 70, P['业主·投资委'][1] + 60, P['业主·投资委'][0] + 60, P['业主·投资委'][1] + 46)
  g += `<text x="1150" y="300" fill="var(--dim)" font-size="9">经营报表/告警→业主</text>`
  // 支撑回环（虚线）：评史官→命官/探员、仓巡→执事、配管→命官。
  const loop: [string, string][] = [['评史官', '命官'], ['评史官', '探员'], ['仓巡', '执事'], ['配管', '命官']]
  const loopLbl: Record<string, string> = {
    '评史官|命官': '退役/进化回写→命官', '评史官|探员': '回炉重枚举→探员',
    '仓巡|执事': '平仓触发', '配管|命官': '停泊/抽调',
  }
  for (const [a, b] of loop) {
    if (!P[a] || !P[b]) continue
    g += line(P[a][0] + 70, P[a][1], P[b][0] + 70, P[b][1] + 36, '4 3')
    g += `<text x="${(P[a][0] + P[b][0]) / 2 + 60}" y="${(P[a][1] + P[b][1]) / 2}" fill="var(--dim)" font-size="8">${h(loopLbl[a + '|' + b] || '')}</text>`
  }
  // 数据供数群虚线：行情巡/私流守/汇市巡→主链供数（集使调度→全员）。
  for (const t of ['行情巡', '私流守', '汇市巡']) {
    if (!P[t]) continue
    g += line(P[t][0] + 70, P[t][1], P[t][0] + 70, 180, '3 3')
  }
  if (P['集使']) g += line(P['集使'][0] + 140, P['集使'][1] + 18, 940, P['集使'][1] + 18, '3 3')
  g += `<text x="40" y="330" fill="var(--dim)" font-size="9">数据域·供数群（可观测世界）</text>`
  // 标题+图例（钦定两行——一眼看出谁在工作）。
  g += `<text x="40" y="80" fill="var(--dim)" font-size="11">钱程主链（一笔钱的旅程 G0→G11）</text>`
  g += `<line x1="40" y1="420" x2="70" y2="420" stroke="#94a3b8" stroke-width="1.2"/>
    <text x="76" y="424" fill="var(--dim)" font-size="10">实线=钱/仓流（产出物移交）</text>
    <line x1="220" y1="420" x2="250" y2="420" stroke="#94a3b8" stroke-width="1.2" stroke-dasharray="4 3"/>
    <text x="256" y="424" fill="var(--dim)" font-size="10">虚线=控制/数据/回环</text>
    <circle cx="400" cy="420" r="4" fill="#34d399"/><text x="408" y="424" fill="var(--dim)" font-size="10">心跳正常</text>
    <circle cx="460" cy="420" r="4" fill="#fbbf24"/><text x="468" y="424" fill="var(--dim)" font-size="10">迟滞</text>
    <circle cx="510" cy="420" r="4" fill="#f87171"/><text x="518" y="424" fill="var(--dim)" font-size="10">离线</text>
    <text x="580" y="424" fill="var(--dim)" font-size="10">灯在节点左上角——一眼看出谁在拖</text>
    <text x="40" y="445" fill="var(--dim)" font-size="9">组织架构：业主→投资委→phase-core 运行时中枢→六域员工；谁在上=域为部门，链为工序；产出物标在箭头旁=谁干活给谁。</text>`
  for (const nm of ['探员', '验证官', '命官', '门官', '执事', '司账', '核账', '信使',
    '评史官', '配管', '仓巡', '集使', '行情巡', '私流守', '汇市巡']) {
    if (P[nm]) g += node(nm, P[nm][0], P[nm][1])
  }
  return `<svg viewBox="0 0 1300 460" style="width:100%;height:auto;background:#0b0e14;border-radius:12px">${g}</svg>`
}
