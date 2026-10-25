// trx.ts —— TRX 自营盘独立页（T-0110 nox pages/trx.tsx+Trx* 组件 vanilla TS
// 重写——四区：收益雷达→对冲结构→对冲流水→监控提醒）。D-629：全页「业绩外·
// 自营盘」标显，零 NAV/业绩计入。开页刷新踢=trxAccountRefresh 双踢（体感
// 优化静默败）；刷新节拍=30s 轮询（O 无 trx 流帧 topic——nox onCase 等价位）。
const h = (s: string): string => {
  const d = document.createElement('div'); d.textContent = s; return d.innerHTML
}
const usd = (v: number, dp = 2): string =>
  '$' + (v || 0).toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
const num = (v: number, dp = 3): string =>
  (v || 0).toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
const signed = (v: number, dp = 2): string => (v >= 0 ? '+' : '') + usd(v, dp)
const pnlCls = (v: number): string => (v > 0.005 ? 'up' : v < -0.005 ? 'dn' : '')
const stamp = (): string => {
  const d = new Date()
  return '数据 ' + String(d.getUTCHours()).padStart(2, '0') + ':' +
    String(d.getUTCMinutes()).padStart(2, '0') + ':' +
    String(d.getUTCSeconds()).padStart(2, '0') + ' UTC'
}
async function get<T>(path: string): Promise<T | null> {
  try {
    const r = await fetch(path)
    if (!r.ok) return null
    return (await r.json()) as T
  } catch { return null }
}

// ①收益雷达（nox TrxYieldRadar 对拍——APR 分布+floor+过渡护栏显形）。
type Radar = { energy_apr_7d: number; energy_apr_30d: number; vote_apr_current: number; vote_apr_30d_avg: number; floor_pct: number; sample_days_7d: number; sample_days_30d: number; transition: boolean; vote_apr_max: number; vote_apr_max_sr: string; vote_apr_spread: number }
function radarHTML(d: Radar): string {
  const kpi = (label: string, v: number, sub = '') =>
    `<div class="kpi"><b>${v.toFixed(2)}%</b><span>${h(label)}${sub}</span></div>`
  const guard = d.transition ? '<div class="warn">· 扩容过渡期（7d 窗内在押变化>20%）——floor 判定无效</div>' : ''
  const below = !d.transition && d.energy_apr_30d > 0 && d.energy_apr_30d < d.floor_pct
    ? '<div class="dn">· 能量 30d 年化低于 floor ' + d.floor_pct.toFixed(1) + '%</div>' : ''
  return `<div class="grid">` +
    kpi('能量租金 7d 年化', d.energy_apr_7d, d.sample_days_7d < 7 ? ' · 样本不足' : '') +
    kpi('能量租金 30d 年化', d.energy_apr_30d, d.sample_days_30d < 30 ? ' · 样本不足' : '') +
    kpi('投票当前 APR', d.vote_apr_current) +
    kpi('投票 30d 均值', d.vote_apr_30d_avg) +
    kpi('市场最高 APR', d.vote_apr_max, d.vote_apr_max_sr ? ' · ' + h(d.vote_apr_max_sr) : '') +
    `</div><div class="muted" style="margin-top:10px">${guard}${below || (d.energy_apr_30d === 0 && d.energy_apr_7d === 0) ? '' : ''}</div>`
}

// ②对冲结构（nox TrxHedgeCard 对拍——镜像双腿+净敞口+双账户块+口径注记）。
type Hedge = {
  baseline_price: number; mid: number; ask: number; ws_connected: boolean
  leg_spot: { qty_staked: number; qty_liquid: number; pnl_vs_base: number; vote_apr_current: number }
  leg_short: { qty: number; pnl_vs_base: number; lev: number; liq: number; dist_pct: number }
  net_exposure_trx: number; net_exposure_usd: number
  carry: { trx_rental_7d_trx: number; trx_vote_7d_trx: number; cmc_overnight_7d_usd: number; cum_usd: number; annualized_pct: number; hedged_cum_usd: number; hedged_annualized_pct: number; mtd_net_usd: number; mtd_sample_days: number }
  accts: Record<string, number | string>
  platform_alert_state: string; closeout_level_usd: number; warning_level_usd: number
}
function hedgeHTML(d: Hedge): string {
  const ready = d.mid > 0
  if (!ready) return '<div class="empty">数据未就绪（采集拍回填）</div>'
  const a = d.accts
  const bw = String(a.bandwidth_status ?? 'normal')
  const dot = d.ws_connected ? '<span class="dot ok"></span><span class="muted">WS 在线</span>'
    : '<span class="dot off"></span><span class="muted">REST 兜底</span>'
  const alert = d.platform_alert_state !== 'normal'
    ? `<span class="dot bad"></span><span class="warn">${d.platform_alert_state === 'breach' ? '平台强平触碰' : '平台预警'}</span>` : ''
  const lev = (d.leg_short.lev ?? 0) > 0 ? d.leg_short.lev + '×' : '—'
  const liq = (d.leg_short.liq ?? 0) > 0 ? usd(d.leg_short.liq) : '—'
  const dist = (d.leg_short.dist_pct ?? 0) > 0 ? d.leg_short.dist_pct.toFixed(2) + '%' : '—'
  return `<table><thead><tr><th>腿</th><th>数量</th><th>开仓价</th><th>现价</th><th>浮动盈亏</th><th>杠杆</th><th>强平价</th><th>距强平%</th></tr></thead><tbody>` +
    `<tr><td>质押现货</td><td>质押 ${num(d.leg_spot.qty_staked)} TRX</td><td>${usd(d.baseline_price, 4)}</td><td>${usd(d.ask || d.mid, 4)}</td>` +
    `<td class="${pnlCls(d.leg_spot.pnl_vs_base)}">${signed(d.leg_spot.pnl_vs_base)}</td><td>—</td><td>—</td><td>—</td></tr>` +
    `<tr><td>CMC 空单</td><td>${num(d.leg_short.qty)} TRX</td><td>${usd(d.baseline_price, 4)}</td><td>${usd(d.ask || d.mid, 4)}</td>` +
    `<td class="${pnlCls(d.leg_short.pnl_vs_base)}">${signed(d.leg_short.pnl_vs_base)}</td><td>${lev}</td><td>${liq}</td><td class="${(d.leg_short.dist_pct ?? 0) < 0 ? 'dn' : ''}">${dist}</td></tr>` +
    `</tbody></table>` +
    `<div class="muted" style="margin:10px 0">净敞口 <b>${num(d.net_exposure_trx)} TRX</b>（≈ ${usd(d.net_exposure_usd)} USD）· 近似中性 · 口径：强平=保证金 50%，预警=80%（平台标定）</div>` +
    `<div class="grid">` +
    `<div class="kpi"><b>${num(Number(a.trx_balance))}</b><span>TRX 余额 · 待领取 ${num(Number(a.trx_allowance))} · 能量委托 ${num(Number(a.energy_delegated))} · 票权 ${num(Number(a.votes), 0)}</span>` +
    `<div class="muted">剩余带宽 ${num((Number(a.free_net_limit) - Number(a.free_net_used)) + (Number(a.net_limit) - Number(a.net_used)), 0)} BP${bw === 'red' ? ' <span class="dn">· 红</span>' : bw === 'yellow' ? ' <span class="warn">· 黄</span>' : ''} · 剩余能量 ${num(Number(a.energy_limit) - Number(a.energy_used), 0)}</div></div>` +
    `<div class="kpi"><b>${usd(Number(a.cmc_cash))}</b><span>CMC 现金 · 净值 ${usd(Number(a.cmc_equity))} USD</span>` +
    `<div class="muted">强平净值 ${usd(d.closeout_level_usd)} / 预警净值 ${usd(d.warning_level_usd)}</div></div>` +
    `<div class="kpi"><b>${usd(d.carry.cum_usd)}</b><span>carry 累计 · 年化 ${d.carry.annualized_pct.toFixed(2)}%</span>` +
    `<div class="muted">对冲后 ${usd(d.carry.hedged_cum_usd)}（${d.carry.hedged_annualized_pct.toFixed(2)}%）· 租金 7d ${num(d.carry.trx_rental_7d_trx)} + 票 7d ${num(d.carry.trx_vote_7d_trx)} TRX · MTD ${usd(d.carry.mtd_net_usd)}${d.carry.mtd_sample_days === 0 ? '（样本—）' : ''}</div></div>` +
    `</div><div style="margin-top:8px">${dot}${alert}</div>`
}

// ③对冲流水（nox HedgeFlow 对拍——trx/cmc 双源行+source 徽章）。
type FlowItem = { source: string; ts: string; kind: string; amount_value: number; amount_unit: string; counterparty: string; tx_id: string }
function flowHTML(txs: FlowItem[]): string {
  if (!txs.length) return '<div class="empty">读件缺（采集拍回填）</div>'
  return `<table><thead><tr><th>源</th><th>时点</th><th>类型</th><th>金额</th><th>对手方</th></tr></thead><tbody>` +
    txs.map((t) => `<tr><td>${t.source.toUpperCase()}</td><td class="muted">${h(t.ts.slice(0, 16).replace('T', ' '))}</td>` +
      `<td>${h(t.kind)}</td><td class="${t.amount_value >= 0 ? 'up' : 'dn'}">${t.amount_value >= 0 ? '+' : ''}${num(t.amount_value)} ${h(t.amount_unit)}</td>` +
      `<td class="muted">${h(t.counterparty || '—')}</td></tr>`).join('') + `</tbody></table>`
}

// ④监控提醒（系统信号面——O 读面核销形：watch_rule CRUD 未在 T-0110 读面清单，
// 本区=带宽/平台预警/过渡护栏/引擎旗态如实显形）。
type Acct = { bandwidth_status: string; trx_balance: number }
function watchHTML(a: Acct | null, hedge: Hedge | null, voteOn: boolean): string {
  const items: string[] = []
  if (a) {
    const bw = a.bandwidth_status
    items.push(`<div><span class="dot ${bw === 'red' ? 'bad' : bw === 'yellow' ? 'off' : 'ok'}"></span>` +
      `带宽状态 ${bw}${bw !== 'normal' ? '——领奖/改投将跳过（预检线 250/500 BP）' : ''}</div>`)
  }
  if (hedge && hedge.platform_alert_state !== 'normal') {
    items.push(`<div><span class="dot bad"></span>平台预警 ${h(hedge.platform_alert_state)}——强平净值 ${usd(hedge.closeout_level_usd)}</div>`)
  }
  items.push(`<div><span class="dot ${voteOn ? 'ok' : 'off'}"></span>投票引擎${voteOn ? '在驶' : '停驶待令'}（trx.vote_enabled=${voteOn ? 'true' : 'false'}——翻旗=业主显式授权点）</div>`)
  return items.join('')
}

async function tick(): Promise<void> {
  const st = document.getElementById('stamp')
  if (st) st.textContent = stamp()
  const radar = await get<Radar>('/api/trx/yield_radar')
  const rb = document.getElementById('radar')
  if (rb) rb.innerHTML = radar ? radarHTML(radar) : '<div class="empty">读件缺</div>'
  const hedge = await get<Hedge>('/api/trx/hedge_structure')
  const hb = document.getElementById('hedge')
  if (hb) hb.innerHTML = hedge ? hedgeHTML(hedge) : '<div class="empty">读件缺</div>'
  const flow = await get<{ total: number; txs: FlowItem[] }>('/api/trx/hedge_flow?limit=12')
  const fb = document.getElementById('flow')
  if (fb) fb.innerHTML = flow ? flowHTML(flow.txs) : '<div class="empty">读件缺</div>'
  const acct = await get<Acct>('/api/trx/account')
  const wb = document.getElementById('watch')
  if (wb) wb.innerHTML = watchHTML(acct, hedge, false)
}
async function kick(): Promise<void> {
  await get('/api/trx/account_refresh') // D-254 开页刷新踢（读侧拉新——静默败）
}
void (async () => {
  await kick()
  await tick()
})()
setInterval(tick, 30000)
const btn = document.getElementById('refresh')
btn?.addEventListener('click', () => { void kick().then(tick) })
