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
    // staff：API 返才现卡（禁静态假名册——缺面藏卡如实）。
    const sc = document.getElementById('c-staff')
    const sb = document.getElementById('staff-body')
    if (sc && sb) {
      if (d.staff && d.staff.length) {
        sc.style.display = ''
        const dots = ['var(--up)', 'var(--warn)', 'var(--dn)']
        sb.innerHTML = `<h3>员 工 · ${d.staff.length} 在册</h3>` +
          d.staff.map((s: { name: string; duty: string; state: number }) =>
            `<div class="staffline"><span class="dot" style="background:${dots[s.state] || dots[0]}"></span>${h(s.name)} · ${h(s.duty)}</div>`).join('')
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
