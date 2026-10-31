// Cockpit.tsx —— 业主驾驶舱 v2（T-0239 五块只读基座上重排——判态服务端派生，
// 前端纯渲染零判据）。v2（业主直批施工任务书·驾驶舱体验v2 2026-10-30）三件：
// ①美观对齐 TRX 基准——首档 trx-kpi-strip 大数字条+瓦片骨架+EmptyState 空态，
// 全套设计件同源（20-cards.css 既有类，零新 CSS 语言；唯一新增 .tile-scroll
// 定高滚动工具类）；②业主第一性四问重排——①钱安全②赚了多少置顶放大（人话
// 结论首行、数字第二层），③机器面次之，纯机器细节（无头面）最次；流连接
// 角标+asOf 快照戳常显（数据鲜度一眼可见）；③无头状态面本地切片分页
// （T-0262 工单板同款分页器 10 行/页——推送全量本地切，零后端改动）+定高
// 瓦片。降级语义原样保全（T-0248：sec/SecBoundary/missing 显缺不冒零）；
// T-0263 推送制照旧（Watch ev_headless 流帧+unary 快照 seq 水印，零定时器）；
// T-0268：无头面并入 getCockpit 首屏一往返（独立首调撤——hydrate 仅缺段兜底）。
import { useEffect, useState } from "react";
import { client } from "../api";
import { onLive } from "../stream";
import { applyHeadless, hydrate, useSlice } from "../store";
import { Component, type ReactNode } from "react";
import { Card, EmptyState, Skeleton, StateDot } from "../components/ui";
import { AGE } from "../components/panel";
import type { GetCockpitResponse, CockpitPlatformRow, CockpitSeat, CockpitTicket, CockpitArmed, HeadlessRow } from "../../gen/dashboardv1/core_pb";

const T_PAGE = 10; // 分页页长：③工单板=T-0262 后端 ticket_limit 同值；无头面=本地切片同值

// 无头五态显示字典（pump derive 串→人话——纯显示映射非判据，终态不猜在服务端）
const HL_STATE: Record<string, string> = {
	alive: "在飞", done: "完工", dead: "死亡", stalled: "停滞", overdue: "逾期",
};

// 行级呼吸灯（驾驶舱 v3.1 业主令：每票每席一盏）。ops_headless.state 语钉：
// 绿呼吸=alive；黄=stalled/等审计；红=dead/overdue（死=静止不装活）；灰 s3=done。
const HL_DOT: Record<string, [number, boolean]> = {
	alive: [0, true], stalled: [1, true], dead: [2, false], overdue: [2, false], done: [3, false],
};

const ts = (t?: { seconds: bigint }) =>
	t ? new Date(Number(t.seconds) * 1000).toLocaleString("zh-CN", { hour12: false }) : "—";

// LiveBadge 流连接角标（v2 常显——数据鲜度一眼可见；onLive 挂载即回放现态，
// 灯语同 Shell LiveDot：绿呼吸=流活，黄=断流重连中）。
function LiveBadge() {
	const [live, setLive] = useState<boolean | null>(null);
	useEffect(() => onLive((v?: unknown) => setLive(Boolean(v))), []);
	return (
		<span className="ink-3 fact">
			<StateDot
				state={live ? 0 : 1}
				breathe={Boolean(live)}
				title={live ? "Watch 流在线——事件驱动实时" : "Watch 断流重连中——数据可能陈"}
			/>
			{" "}流{live === null ? "连接中" : live ? "在线" : "断流重连中"}
		</span>
	);
}

// Pager 分页器（T-0262 工单板同款组件语——上一页/下一页/第 x/y 页（共 N 条）；
// pages≤1 不显。工单板=服务端下推翻页重取，无头面=本地切片，共用此件）。
function Pager(props: { page: number; pages: number; total: number; onPage: (p: number) => void }) {
	if (props.pages <= 1) return null;
	return (
		<div className="pager">
			<button type="button" disabled={props.page <= 1} onClick={() => props.onPage(props.page - 1)}>上一页</button>
			<span>第 {props.page} / {props.pages} 页（共 {props.total} 条）</span>
			<button type="button" disabled={props.page >= props.pages} onClick={() => props.onPage(props.page + 1)}>下一页</button>
		</div>
	);
}

// HeadlessTile 无头状态面（T-0263 推送制照旧——取数泵上提页级单源，本件纯
// 渲染；v2 本地切片分页：推送全量→本地切 10 行/页，零后端改动；.tile-scroll
// 定高滚动——瓦片高度固定，页高不被行数撑；T-0268 读败显缺不永转骨架）。
// v3 呼吸灯：瓦片右上角状态点（绿呼吸=无异响；红=含死亡/停滞/逾期单）。
function HeadlessTile() {
	const hl = useSlice("headless");
	const [page, setPage] = useState(1);
	if (!hl.data) return <Card title="③ 无头状态面（推送制）">{hl.err ? <p className="dim">该段数据缺（读败）</p> : <Skeleton rows={4} />}</Card>;
	const rows = hl.data.rows;
	const pages = Math.max(1, Math.ceil(rows.length / T_PAGE));
	const safePage = Math.min(page, pages);
	const view = rows.slice((safePage - 1) * T_PAGE, safePage * T_PAGE);
	const bad = rows.filter((r) => r.state === "dead" || r.state === "stalled" || r.state === "overdue").length;
	const alive = rows.filter((r) => r.state === "alive").length;
	const done = rows.filter((r) => r.state === "done").length;
	return (
		<Card title="③ 无头状态面（推送制）"
			extra={<StateDot state={bad > 0 ? 2 : 0} breathe={bad === 0}
				title={`${alive} 在飞 · ${done} 完工${bad > 0 ? ` · ${bad} 异常` : ""}`} />}>
			{rows.length === 0 ? <EmptyState title="无在册无头单" /> : <>
				<div className="tile-scroll">
					<table className="tile-table"><thead><tr><th>灯</th><th>票</th><th>席</th><th>底模</th><th>相位</th><th>五态</th><th>最后活动</th></tr></thead>
						<tbody>{view.map((r) => { const [st, br] = HL_DOT[r.state] ?? [3, false]; return <tr key={r.ticket}>
							<td><StateDot state={st} breathe={br} title={`${HL_STATE[r.state] ?? r.state} · ${r.ticket} · 活动 ${ts(r.lastActivity)}`} /></td>
							<td>{r.ticket}</td><td>{r.seat}</td>
							<td>{r.model || "—"}</td><td>{r.phase || "—"}</td><td>{HL_STATE[r.state] ?? r.state}</td><td>{ts(r.lastActivity)}</td></tr>; })}</tbody></table>
				</div>
				<Pager page={safePage} pages={pages} total={rows.length}
					onPage={(p) => setPage(Math.max(1, Math.min(p, pages)))} />
			</>}
		</Card>
	);
}

export function CockpitPage() {
	const [d, setD] = useState<GetCockpitResponse | null>(null);
	const [page, setPage] = useState(1);
	const hl = useSlice("headless");
	// 无头取数泵页级单源（v2 上提——①存活卡与③状态面同读 headless 分片，
	// 一页一取数点）；T-0268 挂载首取并入 getCockpit.headless——本效零独立
	// 首调，hydrate 仅缺段径 unary 兜底（data 在=零请求）；onLive 即时回放
	// 非断流复连不补，复连才 unary 回补+接流 seq>水位帧补洞（水印判在
	// store.applyHeadless）。零 interval 零定时器。
	useEffect(() => {
		const refill = () => client.getHeadless({}).then(applyHeadless).catch(() => {});
		hydrate("headless", () => client.getHeadless({}));
		let replayed = false; // onLive 首调=现态即时回放非断流复连——跳过，不重演首取
		return onLive((v?: unknown) => {
			if (!replayed) { replayed = true; return; }
			if (v === true) refill();
		});
	}, []);
	useEffect(() => {
		client.getCockpit({ ticketLimit: T_PAGE, ticketOffset: (page - 1) * T_PAGE })
			.then((d) => { if (d.headless) applyHeadless(d.headless); setD(d); })
			.catch(() => setD(null));
	}, [page]);
	if (!d) return (
		<div className="page">
			<div className="health-strip">
				<span>业主驾驶舱</span>
				<LiveBadge />
				<span className="ink-3">快照 …</span>
			</div>
			{/* 载中=瓦片骨架（诚实空态律——骨架=卡面独立，不整页灰） */}
			<div className="trx-kpi-strip">
				{["① 钱安全吗", "② 赚了多少（MTD）", "② 近 30 天", "① 存活"].map((k) => (
					<Card key={k} className="kpi" title={k}><Skeleton rows={2} /></Card>
				))}
			</div>
			<div className="duo">
				<Card title="④ 待开仓（armed）"><Skeleton rows={3} /></Card>
				<Card title="①② 钱面平台明细"><Skeleton rows={4} /></Card>
			</div>
		</div>
	);
	const miss = new Set(d.missing);
	const sec = (name: string, el: ReactNode) =>
		miss.has(name) ? <p className="dim">该段数据缺（查询败——不冒零）</p> :
			<SecBoundary>{el}</SecBoundary>;
	const m = d.money; // money 块缺段判据真源=服务端 missing 字段——?. 后链式禁裸调
	const tPages = Math.max(1, Math.ceil(d.ticketsTotal / T_PAGE));
	const safePage = Math.min(page, tPages);
	// 展示层聚合（渲染算术，非判据——判态全部服务端派生）
	const expSt = m?.exposureState ?? 0;
	const parked = m ? m.platforms.reduce((a, p) => a + p.parked, 0) : 0;
	const d30 = m ? m.daily.reduce((a, v) => a + v, 0) : 0;
	const posDays = m ? m.daily.filter((v) => v > 0).length : 0;
	let lastSeatTs: { seconds: bigint } | undefined;
	for (const s of d.seats) if (s.ts && (!lastSeatTs || s.ts.seconds > lastSeatTs.seconds)) lastSeatTs = s.ts;
	const hlRows = hl.data?.rows ?? [];
	// v3.1 行灯联查面：工单×无头状态按票号/席位双键联（ops_headless 有票字段）。
	const hlByTicket = new Map<string, HeadlessRow>(hlRows.map((r) => [r.ticket, r]));
	const hlBySeat = new Map<string, HeadlessRow>(hlRows.map((r) => [r.seat, r]));
	// 席行灯：ops_headless.state 语钉（绿呼吸=alive/黄=stalled·等审计/红=dead·overdue/灰=done或无在册单）。
	const seatDot = (s: CockpitSeat): { state: number; breathe: boolean; title: string } => {
		const r = (s.ticket ? hlByTicket.get(s.ticket) : undefined) ?? hlBySeat.get(s.seat);
		if (!r) return { state: 3, breathe: false, title: `${s.seat} · 无在册无头单 · 自报 ${ts(s.ts)}` };
		const [st, br] = HL_DOT[r.state] ?? [3, false];
		return { state: st, breathe: br, title: `${HL_STATE[r.state] ?? r.state} · ${r.ticket} · 活动 ${ts(r.lastActivity)}` };
	};
	// 票行灯：灰=待派/已批（无领）；绿呼吸=施工中/在飞且联上活席；黄=收口待审·等审计/
	// 席位失联（票面在飞但 ops_headless 无活席=断链嫌疑）；实绿=完成/闭环；红=驳回/挂起。
	const ticketDot = (t: CockpitTicket): { state: number; breathe: boolean; title: string } => {
		const st = t.state ?? "";
		const ttl = (x: string) => `${st} · ${t.seat || x} · ${ts(t.ts)}`;
		if (/待派|已批|pending|approved/i.test(st)) return { state: 3, breathe: false, title: ttl("无人领") };
		if (/收口待审|审计|review/i.test(st)) return { state: 1, breathe: true, title: ttl("等审计") };
		if (/完成|闭环|done|closed/i.test(st)) return { state: 0, breathe: false, title: ttl("闭环") };
		if (/驳回|挂起|作废|reject|block|void/i.test(st)) return { state: 2, breathe: false, title: ttl("异常态") };
		const r = hlByTicket.get(t.ticket);
		if (r && r.state === "alive") return { state: 0, breathe: true, title: `${st} · ${r.seat || t.seat || "—"} · 活动 ${ts(r.lastActivity)}` };
		return { state: 1, breathe: true, title: `${st} · ${t.seat || "—"} · 席位失联（票面在飞无活席） · ${ts(t.ts)}` };
	};
	// 计数双语归一（服务端判态中文标 or pump 原键——v2 裸键滤恒 0 病灶修：
	// HL_STATE 正向映射后按显示标判，两径通吃）。
	const hlL = (s: string) => HL_STATE[s] ?? s;
	const hlBad = hlRows.filter((r) => ["死亡", "停滞", "逾期"].includes(hlL(r.state))).length;
	const hlAlive = hlRows.filter((r) => hlL(r.state) === "在飞").length;
	const hlDone = hlRows.filter((r) => hlL(r.state) === "完工").length;
	// v3 呼吸灯：瓦片右上角状态点统一灯语（绿呼吸=段数据在；红=段缺/异常），
	// title 携鲜度（快照 asOf / 最近自报——AGE 同 panel 件）。纯显示映射零判据。
	const asOfD = d.asOf ? new Date(Number(d.asOf.seconds) * 1000) : null;
	const asOfAge = asOfD ? AGE(asOfD) : "—";
	const lastSeatAge = lastSeatTs ? AGE(new Date(Number(lastSeatTs.seconds) * 1000)) : "—";
	return (
		<div className="page">
			<div className="health-strip">
				<span>业主驾驶舱</span>
				<LiveBadge />
				<span className="ink-3">快照 {ts(d.asOf)}</span>
			</div>
			{/* —— 首档·答①②置顶放大（业主第一性四问重排——人话结论大字居首，
			    数字第二层；trx-kpi-strip=TRX 首档同源件，零新 CSS）—— */}
			<div className="trx-kpi-strip">
				<Card className="kpi" title="① 钱安全吗"
					extra={m ? <StateDot state={expSt} breathe={expSt === 0} title={`敞口硬线判态（服务端派生）· 快照 ${asOfAge}`} /> : undefined}>
					{sec("money", !m ? <p className="dim">该段数据缺</p> : <>
						<div className={"kpi-v " + (expSt === 0 ? "u-good" : expSt === 2 ? "u-bad" : "")}>
							{{ 0: "合规绿", 1: "顶格黄", 2: "越限红" }[expSt] ?? "—"}
						</div>
						<div className="kpi-note">敞口 ${m.exposure.toFixed(0)} / 上限 ${m.exposureLimit.toFixed(0)}（10% 硬线）</div>
						<div className="kpi-note">停泊 ${parked.toFixed(0)}（闲资默认活期/Earn——见执行面件）</div>
					</>)}
				</Card>
				<Card className="kpi" title="② 赚了多少（MTD）"
					extra={<StateDot state={miss.has("money") ? 2 : 0} breathe={!miss.has("money")} title={`快照 ${asOfAge}`} />}>
					{sec("money", !m ? <p className="dim">该段数据缺</p> : <>
						<div className={"kpi-v " + (m.netMonth >= 0 ? "u-good" : "u-bad")}>
							{m.netMonth >= 0 ? "+" : "-"}${Math.abs(m.netMonth).toFixed(2)}
						</div>
						<div className="kpi-note">本金水位 ${m.capital.toFixed(2)}</div>
					</>)}
				</Card>
				<Card className="kpi" title="② 近 30 天"
					extra={<StateDot state={miss.has("money") ? 2 : 0} breathe={!miss.has("money")} title={`快照 ${asOfAge}`} />}>
					{sec("money", !m ? <p className="dim">该段数据缺</p> : <>
						<div className={"kpi-v " + (d30 >= 0 ? "u-good" : "u-bad")}>
							{d30 >= 0 ? "+" : "-"}${Math.abs(d30).toFixed(2)}
						</div>
						<div className="kpi-note">日净 {posDays} 正 / {m.daily.length - posDays} 负</div>
					</>)}
				</Card>
				<Card className="kpi" title="① 存活"
					extra={hl.data ? <StateDot state={hlBad > 0 ? 2 : 0} breathe={hlBad === 0}
						title={hlBad > 0 ? `无头面 ${hlBad} 单异常（死亡/停滞/逾期）` : "无头面无异响"} /> : undefined}>
					{sec("seats", <>
						<div className="kpi-v">{d.seats.length}<span className="kpi-unit">席在册</span></div>
						<div className="kpi-note">最近自报 {ts(lastSeatTs)}</div>
					</>)}
					{hl.data && hlRows.length > 0 && (
						<div className="kpi-note">无头面 {hlAlive} 在飞 · {hlDone} 完工{hlBad > 0 ? ` · ${hlBad} 异常` : ""}</div>
					)}
				</Card>
			</div>
			{/* —— 次档·答④要我动的（armed）+①②数字第二层（平台明细；
			    近30d 日净原序列折此可查，首档只留人话结论）—— */}
			<div className="duo">
				<Card title="④ 待开仓（armed）"
					extra={<StateDot state={miss.has("armed") ? 2 : 0} breathe={!miss.has("armed")} title={`${d.armed.length} 单 · 快照 ${asOfAge}`} />}>{sec("armed",
					d.armed.length === 0 ? <EmptyState title="无待开仓" hint="armed 队列为空——无待动事项" /> :
						<div className="tile-scroll"><table className="tile-table"><thead><tr><th>#</th><th>标的</th><th>方向</th><th>建时</th><th>TTL余(h)</th></tr></thead>
							<tbody>{d.armed.map((a: CockpitArmed) => <tr key={a.id}><td>{a.id}</td><td>{a.instId}</td>
								<td>{a.direction}</td><td>{ts(a.created)}</td><td>{a.ttlHoursLeft.toFixed(1)}</td></tr>)}</tbody></table></div>)}
				</Card>
				<Card title="①② 钱面平台明细"
					extra={<StateDot state={miss.has("money") ? 2 : 0} breathe={!miss.has("money")} title={`${m?.platforms.length ?? 0} 平台 · 快照 ${asOfAge}`} />}>{sec("money", !m ? <p className="dim">该段数据缺</p> : <>
					<table className="tile-table"><thead><tr><th>平台</th><th>承诺</th><th>已入</th><th>停泊</th><th>敞口</th></tr></thead>
						<tbody>{m.platforms.map((p: CockpitPlatformRow) => <tr key={p.platform}><td>{p.platform}</td><td>{p.committed < 0 ? "未设" : p.committed.toFixed(0)}</td>
							<td>{p.platform === "MT5" && miss.has("mt5") ? "数据缺" : p.deposited.toFixed(0)}</td><td>{p.parked.toFixed(0)}</td><td>{p.exposure.toFixed(0)}</td></tr>)}</tbody></table>
					{miss.has("mt5") && <p className="dim">MT5 平台已入段数据缺（mt5Live 读败——合计行为不含 MT5 部分口径，非真 0）</p>}
					<p className="dim fact">近30d 日净：{m.daily.map((v) => v.toFixed(1)).join(" · ")}</p>
				</>)}
				</Card>
			</div>
			{/* —— 三档·答③机器面（席位/工单概要）—— */}
			<div className="duo">
				<Card title="③ 席位与在做什么"
					extra={<StateDot state={miss.has("seats") ? 2 : 0} breathe={!miss.has("seats")} title={`最近自报 ${lastSeatAge}`} />}>{sec("seats",
					d.seats.length === 0 ? <EmptyState title="无在册席位" /> :
						<table className="tile-table"><thead><tr><th>灯</th><th>席</th><th>票</th><th>阶段</th><th>最近自报</th><th>备注</th></tr></thead>
							<tbody>{d.seats.map((s: CockpitSeat) => { const sd = seatDot(s); return <tr key={s.seat}>
								<td><StateDot state={sd.state} breathe={sd.breathe} title={sd.title} /></td>
								<td>{s.seat}</td><td>{s.ticket}</td>
								<td>{s.stage}</td><td>{ts(s.ts)}</td><td>{s.note}</td></tr>; })}</tbody></table>)}
				</Card>
				<Card title="③ 工单板"
					extra={<StateDot state={miss.has("tickets") ? 2 : 0} breathe={!miss.has("tickets")} title={`${d.ticketsTotal} 单 · 快照 ${asOfAge}`} />}>{sec("tickets",
					d.tickets.length === 0 ? <EmptyState title="无工单" /> :
						<table className="tile-table"><thead><tr><th>灯</th><th>票</th><th>状态</th><th>席</th><th>用量(tok)</th><th>时间</th></tr></thead>
							<tbody>{d.tickets.map((t: CockpitTicket) => { const td = ticketDot(t); return <tr key={t.ticket}>
								<td><StateDot state={td.state} breathe={td.breathe} title={td.title} /></td>
								<td>{t.ticket}</td><td>{t.state}</td>
								<td>{t.seat}</td><td>{Number(t.usageRows) > 0 ? Number(t.totalTokens).toLocaleString() : "—"}</td><td>{ts(t.ts)}</td></tr>; })}</tbody></table>)}
					<Pager page={safePage} pages={tPages} total={d.ticketsTotal} onPage={setPage} />
				</Card>
			</div>
			{/* —— 底档·纯机器细节（③无头状态面——推送制+本地分页+定高）—— */}
			<HeadlessTile />
		</div>
	);
}

// SecBoundary 块级 error boundary（T-0248——一块抛错不掀全页，显缺注占位）。
class SecBoundary extends Component<{ children: ReactNode }, { err: boolean }> {
	state = { err: false };
	static getDerivedStateFromError() { return { err: true }; }
	render() {
		return this.state.err ? <p className="dim">该段数据缺（渲染败）</p> : this.props.children;
	}
}
