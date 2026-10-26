// O 总览活推（T-0067 脊+T-0075 卡绑定——API 四卡真值覆写空态；
// 断源=灰化露相禁演新鲜）。T-0156：REST /api/overview+SSE /api/watch+30s
// 轮询全撤——数据面唯一入口=CoreService Connect，活推=Watch 流帧（ev_* 脊，
// 事件驱动律；断流 3s 重连=传输维护非数据轮询）。
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { timestampDate } from '@bufbuild/protobuf/wkt'
import type { Timestamp } from '@bufbuild/protobuf/wkt'
import { CoreService } from './gen/dashboardv1/core_pb'
// 令牌单源（T-0159）：总览页调色板并入 arbcn 00-base.css——与 TRX 页同源
// 同名令牌（--page/--surface/--ink/--line/--good/--critical 双主题），本页
// 零独立调色板；index.html 内联表只消费令牌不定义色值。
import './react/styles/00-base.css'

const client = createClient(CoreService,
  createConnectTransport({ baseUrl: window.location.origin }))

const h = (s: string): string => {
  const d = document.createElement('div'); d.textContent = s; return d.innerHTML
}
async function ov(): Promise<void> {
  try {
    const d = await client.overview({})
    document.querySelectorAll('.card').forEach(c => (c as HTMLElement).style.opacity = '1')
    const b = document.querySelectorAll<HTMLElement>('.metric b')
    if (b.length >= 3) {
      b[0].textContent = '$' + (d.navTotal || 0).toLocaleString('en-US', { minimumFractionDigits: 2 })
      b[1].textContent = (d.pnlToday >= 0 ? '+' : '') + (d.pnlToday || 0).toFixed(2)
      b[2].textContent = (d.pnlCum >= 0 ? '+' : '') + (d.pnlCum || 0).toFixed(2)
      // T-0114 乙⑤：盈亏着色——负红正绿（--delta-down/--good-ink 双主题令牌，T-0159 arbcn 收敛）。
      b[1].style.color = 'var(' + ((d.pnlToday || 0) < 0 ? '--delta-down' : '--good-ink') + ')'
      b[2].style.color = 'var(' + ((d.pnlCum || 0) < 0 ? '--delta-down' : '--good-ink') + ')'
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
    // T-0096：钱在哪 venue→分区→行三级卡（okx→mt5→注册序；venue 块头携
    // logo——T-0088 件复用）。T-0095 v2：零遗留概念——卡面只出 API 给定行
    //（基金的钱=USDT+在持腿币，托管尘后端已过滤不返）；读件缺=显式行非假零。
    const mb = document.getElementById('money-body')
    if (mb && d.money && d.money.length) {
      const rows = d.money
      const order = ['okx', 'mt5']
      const venues = [...new Set(rows.map((m) => m.venue))]
        .sort((a, b) => (order.indexOf(a) + 1 || 99) - (order.indexOf(b) + 1 || 99))
      mb.innerHTML = venues.map((vn) => {
        const vrows = rows.filter((m) => m.venue === vn)
        const secs = [...new Set(vrows.map((m) => m.section))]
        // T-0114 乙②：场头右挂 venue 合计（OKX=可用 USDT 总额/MT5=汇总美元额）。
        const vsum = vrows.reduce((a, m) => a + m.usd, 0)
        return `<div class="venue-t">${VSVG[vn] || ''}${h(vn.toUpperCase())}` +
          `<i>$${vsum.toFixed(1)}</i></div>` +
          secs.map((sec) => {
            const srows = vrows.filter((m) => m.section === sec)
            const sum = srows.reduce((a, m) => a + m.usd, 0)
            // T-0114 乙④返修：户节头整行删净（其值已并入场头——零行零占位）。
            const secSum = sec === '户' ? '' : (srows.some((m) => m.note) ? '' : '$' + sum.toFixed(1))
            const secHead = sec === '户' ? '' : `<div class="sec-t">${h(sec)}<i>${secSum}</i></div>`
            return secHead +
              srows.map((m) =>
                m.note
                  ? `<div class="row lg-t"><span>${h(m.label)}</span><b>${h(m.note)}</b></div>`
                  // T-0114 乙③：结构性持仓 label 截短显产品号（hover 全长可溯）。
                  : sec === '结构性持仓'
                    ? `<div class="row"><span title="${h(m.label)}">${h(m.label.split(' ')[0])}` +
                      `${m.ccy === 'USDT' || m.ccy === 'USD' ? '' : ` <i>${m.amount} ${m.ccy}</i>`}</span>` +
                      `<b>$${m.usd.toFixed(1)}</b></div>`
                    : `<div class="row"><span>${h(m.label)}${m.ccy === 'USDT' || m.ccy === 'USD' ? '' : ` <i>${m.amount} ${m.ccy}</i>`}</span>` +
                    `<b>$${m.usd.toFixed(1)}</b></div>`).join('')
          }).join('')
      }).join('')
    }
    const rb = document.getElementById('recent-body')
    if (rb && d.recent && d.recent.length) {
      rb.innerHTML = d.recent.map((x: string) => `<div class="ev">${h(x)}</div>`).join('')
    }
    // 交易所在线树状卡（T-0088：venue 组→channel 行，三态真判，断行红显不藏）。
    const vb = document.getElementById('venues-body')
    if (vb && d.venues && d.venues.length) {
      vb.innerHTML = venuesHTML(d.venues as VGroup[])
    }
    // TRX 自营盘（T-0108：业绩外显形——proto 读面零计 nav_total；T-0156 活口）。
    renderTrx()
    // staff 组织架构图（T-0079 v2：直显不收起——宪章 §6 钱程拓扑；API 缺面藏卡如实）。
    const sc = document.getElementById('c-staff')
    if (sc) {
      if (d.staff && d.staff.length) {
        sc.style.display = ''
        const bad = d.staff.filter((x: { state: number }) => x.state === 2).length
        const warn = d.staff.filter((x: { state: number }) => x.state === 1).length
        const ss = document.getElementById('staff-sum')
        if (ss) ss.textContent = `员工 · ${d.staff.length} 在册 · ${bad ? bad + ' 离线' : warn ? warn + ' 迟滞' : '全员心跳正常'}`
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
async function watchPump(): Promise<void> {
  for await (const _ev of client.watch({})) ov()
}
function watchRun(): void {
  void watchPump()
    .catch(() => {}) // 断流吞错（下一帧自然回填）
    .finally(() => setTimeout(watchRun, 3000))
}
watchRun()
void ov()

// venuesHTML 交易所在线树（T-0088 v2——venue 行=logo+名+呼吸总灯（辖下
// 最差态聚合）；channel 行=├/└ 树线+呼吸三色灯+末拍龄 Ns/Nm/h；note 覆盖龄显）。
const VDOTS = ['s0', 's1', 's2']
const VSVG: Record<string, string> = {
  okx: `<svg viewBox="0 0 24 24" width="18" height="18" fill="var(--ink)">
    <rect x="2" y="2" width="6" height="6"/><rect x="9" y="2" width="6" height="6"/><rect x="16" y="2" width="6" height="6"/>
    <rect x="2" y="9" width="6" height="6"/><rect x="16" y="9" width="6" height="6"/>
    <rect x="2" y="16" width="6" height="6"/><rect x="9" y="16" width="6" height="6"/><rect x="16" y="16" width="6" height="6"/></svg>`,
  mt5: `<svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="var(--ink)" stroke-width="1.8">
    <path d="M3 20V6l5 8 4-6 4 6 5-8v14"/><path d="M3 20h18"/></svg>`,
}
interface VChan { name: string; channel: string; lastTs: string; state: number; note: string }
interface VGroup { name: string; channels: VChan[] }
function vAge(ts: string): string {
  const d = (Date.now() - new Date(ts).getTime()) / 1000
  if (d < 60) return `${Math.max(0, Math.floor(d))}s 前`
  if (d < 3600) return `${Math.floor(d / 60)}m 前`
  return `${Math.floor(d / 3600)}h 前`
}
function venuesHTML(gs: VGroup[]): string {
  return gs.map(g => {
    const worst = g.channels.length ? Math.max(...g.channels.map(c => c.state ?? 0)) : 2
    const chans = g.channels.map((c, i) => {
      const tw = i === g.channels.length - 1 ? '└' : '├'
      const age = c.note || (c.lastTs
        ? (c.state === 1 ? '迟滞 ' : c.state === 2 ? '离线 ' : '') + vAge(c.lastTs)
        : '读件缺')
      return `<div class="vline chan"><span class="tree">${tw}</span>${h(c.name)}` +
        `<span class="vage">${h(age)}</span><span class="vdot ${VDOTS[c.state ?? 0]}"></span></div>`
    }).join('')
    return `<div class="vline venue"><span class="vlogo">${VSVG[g.name] || ''}</span>` +
      `${h(g.name.toUpperCase())}<span class="vage"></span>` +
      `<span class="vdot ${VDOTS[worst]}"></span></div>${chans}`
  }).join('')
}

// orgSVG v2 组织架构图（T-0079 返修④钦定——宪章 §6「一笔钱的旅程」拓扑：
// 顶层=o-core(左)+业主·投资委(右金框)；钱程主链实线携产出物；
// 支撑回环+数据供数群虚线；状态灯=节点左上三色静点；素色无动画无辉光）。
function orgSVG(staff: { name: string; duty: string; state: number }[]): string {
  const byName = new Map(staff.map(s => [s.name, s]))
  const dots = ['var(--good)', 'var(--warn)', 'var(--critical)']
  const NW = 150, NH = 40
  const node = (nm: string, x: number, y: number, gold = false) => {
    if (!byName.has(nm)) return ''
    const st = byName.get(nm)!
    // T-0159 令牌收敛：节点面=--surface/线=--line，业主金框=--warn 描边（素底归中性）。
    const edge = gold ? 'stroke="var(--warn)"' : 'stroke="var(--line)"'
    return `<rect x="${x}" y="${y}" rx="8" width="${NW}" height="${NH}" fill="var(--surface)" ${edge}/>
      <circle cx="${x + 12}" cy="${y + 11}" r="4" fill="${dots[st.state] || dots[0]}"/>
      <text x="${x + 22}" y="${y + 17}" fill="var(--ink)" font-size="12">${h(nm)}</text>
      <text x="${x + 22}" y="${y + 31}" fill="var(--ink-3)" font-size="9">${h(st.duty)}</text>`
  }
  const seg = (pts: string, dash: string, arr: string) =>
    `<polyline points="${pts}" fill="none" stroke="var(--ink-3)" stroke-width="1.2" stroke-dasharray="${dash}" marker-end="url(#${arr})"/>`
  const lbl = (x: number, y: number, t: string) =>
    `<text x="${x}" y="${y}" fill="var(--ink-3)" font-size="8.5">${h(t)}</text>`

  const P: Record<string, [number, number]> = {
    'o-core': [40, 14], '业主·投资委': [760, 14],
    '探员': [40, 130], '验证官': [240, 130], '命官': [440, 130], '门官': [640, 130],
    '执事': [840, 130], '司账': [1040, 130], '核账': [1040, 240],
    '信使': [1040, 350],
    '评史官': [240, 240], '配管': [440, 240], '仓巡': [840, 240],
    '差扫': [40, 350],
    '集使': [40, 460], '行情巡': [240, 460], '私流守': [440, 460], '汇市巡': [640, 460],
  }
  const defs = `<defs>
    <marker id="ar" markerWidth="9" markerHeight="9" refX="7" refY="3.5" orient="auto"><path d="M0,0 L7,3.5 L0,7" fill="none" stroke="var(--ink-3)" stroke-width="1.2"/></marker>
  </defs>`
  // 业主金框（非 registry 静态位——金=--warn 描边，底同节点素面）。
  let g = defs + `<rect x="${P['业主·投资委'][0]}" y="14" rx="8" width="180" height="40" fill="var(--surface)" stroke="var(--warn)"/>
    <text x="${P['业主·投资委'][0] + 14}" y="30" fill="var(--ink)" font-size="12">业主 · 投资委</text>
    <text x="${P['业主·投资委'][0] + 14}" y="44" fill="var(--ink-3)" font-size="8.5">只收：经营报表 + 条件请求</text>`
  // 心跳回执（顶层虚线）。
  g += seg(`190,34 760,34`, '5 4', 'ar') + lbl(430, 27, '心跳活性回执')
  // 钱程主链（实线+产出物——G0→G11）。
  const chain: [string, string, string][] = [
    ['探员', '验证官', '提案'], ['验证官', '命官', '评估卡'], ['命官', '门官', '晋级令'],
    ['门官', '执事', '放行票'], ['执事', '司账', '成交回报'],
  ]
  for (const [a, b, t] of chain) {
    g += seg(`${P[a][0] + NW},${P[a][1] + 20} ${P[b][0]},${P[b][1] + 20}`, '0', 'ar')
    g += lbl((P[a][0] + NW + P[b][0]) / 2 - 16, P[a][1] + 13, t)
  }
  // 司账→核账→信使（右列下折）。
  g += seg(`${1040 + 75},170 ${1040 + 75},240`, '0', 'ar') + lbl(1120, 208, '账行')
  g += seg(`${1040 + 75},280 ${1040 + 75},350`, '0', 'ar') + lbl(1120, 318, '对账结论')
  // 信使→业主（右侧上行到顶层）。
  g += seg(`${1190},${370} ${1230},${370} ${1230},${34} ${940},${34}`, '0', 'ar')
  g += lbl(1120, 115, '经营报表/告警 ↑业主')
  // 支撑回环（虚线折线避让标签）。
  g += seg(`315,240 315,205 480,205 480,170`, '4 3', 'ar') + lbl(320, 200, '退役回写→命官')
  g += seg(`270,240 140,240 140,170`, '4 3', 'ar') + lbl(150, 233, '回炉重枚举→探员')
  g += seg(`${840 + 75},240 ${840 + 75},170`, '4 3', 'ar') + lbl(930, 208, '平仓触发')
  g += seg(`${440 + 75},240 ${440 + 75},170`, '4 3', 'ar') + lbl(530, 208, '停泊/抽调')
  // o-core 心跳环→全员（顶部虚线落主链各岗）。
  g += seg(`115,54 115,100 115,130`, '3 3', 'ar')
  // 数据供数群（底排虚线上供）。
  g += lbl(40, 445, '数据域 · 供数群（可观测世界）')
  g += seg(`110,460 110,420 60,420 60,170`, '3 3', 'ar') + lbl(64, 400, '供数')
  g += seg(`310,460 310,420 265,420 265,170`, '3 3', 'ar') + lbl(272, 400, '供数')
  g += seg(`510,460 510,420 480,420 480,170`, '3 3', 'ar')
  g += seg(`710,460 710,420 915,420 915,170`, '3 3', 'ar') + lbl(770, 400, '费率/账户事实')
  // 图例。
  g += `<line x1="40" y1="540" x2="75" y2="540" stroke="var(--ink-3)" stroke-width="1.2"/>
    <text x="82" y="544" fill="var(--ink-3)" font-size="10">实线=钱/仓流（产出物移交）</text>
    <line x1="240" y1="540" x2="275" y2="540" stroke="var(--ink-3)" stroke-width="1.2" stroke-dasharray="4 3"/>
    <text x="282" y="544" fill="var(--ink-3)" font-size="10">虚线=控制/数据/回环</text>
    <circle cx="430" cy="540" r="4" fill="var(--good)"/><text x="440" y="544" fill="var(--ink-3)" font-size="10">心跳正常</text>
    <circle cx="510" cy="540" r="4" fill="var(--warn)"/><text x="520" y="544" fill="var(--ink-3)" font-size="10">迟滞</text>
    <circle cx="570" cy="540" r="4" fill="var(--critical)"/><text x="580" y="544" fill="var(--ink-3)" font-size="10">离线</text>`
  for (const nm of ['o-core', '探员', '验证官', '命官', '门官', '执事', '司账', '核账', '信使',
    '评史官', '配管', '仓巡', '差扫', '集使', '行情巡', '私流守', '汇市巡']) {
    if (P[nm]) g += node(nm, P[nm][0], P[nm][1])
  }
  return `<svg viewBox="0 0 1260 570" style="width:100%;height:auto">${g}</svg>`
}

// renderTrx 自营盘四件（T-0108：账户资源/收益雷达/对冲结构/近期 txs——
// T-0156 proto 读面活口（REST 僵尸 face radar/vote/cmc/cmc_positions 已随
// mountTrxAPI 撤）；读件缺显形不造零；单面败不连坐其余区）。
async function renderTrx() {
  const tb = document.getElementById('trx-body')
  if (!tb) return
  const [acct, radar, hedge, txs] = await Promise.all([
    client.getTrxAccount({}).catch(() => null),
    client.getTrxYieldRadar({}).catch(() => null),
    client.getHedgeStructure({}).catch(() => null),
    client.listTrxTxs({ limit: 8 }).catch(() => null),
  ])
  const row = (l: string, v: string) =>
    `<div class="row"><span>${h(l)}</span><b>${h(v)}</b></div>`
  const pc = (v: number) => (v ? v.toFixed(2) + '%' : '—')
  const stamp = (t?: Timestamp): string =>
    t ? timestampDate(t).toISOString().slice(0, 16).replace('T', ' ') : '—'
  let h1 = '<div class="sec-t">账户资源<i>链上质押</i></div>'
  if (acct?.found) {
    h1 += row('余额', acct.trxBalance.toFixed(0) + ' TRX') +
      row('投票权', acct.tronPower.toFixed(0) + ' TRX → ' + (acct.srAddress || '未投票')) +
      row('能量', `${acct.energyUsed}/${acct.energyLimit}`) +
      row('带宽', `${acct.netUsed + acct.freeNetUsed}/${acct.netLimit + acct.freeNetLimit}`) +
      row('待领收益', acct.trxAllowance.toFixed(0) + ' TRX')
  } else h1 += row('账户资源', '读件缺')
  let h2 = '<div class="sec-t">收益雷达<i>SR APR 年化</i></div>'
  if (radar) {
    h2 += row('投票当前', pc(radar.voteAprCurrent)) +
      row('市场最优' + (radar.voteAprMaxSr ? ` ${radar.voteAprMaxSr}` : ''), pc(radar.voteAprMax)) +
      row('能量 7d / 30d', `${pc(radar.energyApr7d)} ｜ ${pc(radar.energyApr30d)}`)
  } else h2 += row('收益雷达', '读件缺')
  let h3 = '<div class="sec-t">对冲结构<i>CMC CFD 腿</i></div>'
  const st = hedge?.structure
  if (st) {
    h3 += row('账户净值', '$' + (st.accts?.cmcEquity ?? 0).toFixed(0)) +
      row('强平距离', (st.legShort?.distPct ?? 0).toFixed(1) + '%') +
      row('净敞口', `${st.netExposureTrx.toFixed(0)} TRX`)
  } else h3 += row('对冲结构', '读件缺')
  let h4 = '<div class="sec-t">近期 txs<i>链上动账</i></div>'
  if (txs && txs.txs.length) {
    h4 += txs.txs.map((t) =>
      row(`${t.kind} ${stamp(t.ts)}`, t.amountTrx.toFixed(0) + ' TRX')).join('')
  } else h4 += row('近期动账', '读件缺')
  tb.innerHTML = h1 + h2 + h3 + h4
}
