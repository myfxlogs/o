// Cockpit.tsx —— T-0239 业主驾驶舱（五块只读：存活/钱面/工单/待开/在做什么——
// 判态服务端派生，前端纯渲染零判据）；T-0261 全页瓦片化：四面板一一映射
// Card 瓦片入 .tile-grid（页名+快照戳入顶条同 System 页）；T-0263 无头状态面
// =推送制瓦片（Watch ev_headless 流帧+unary 快照 seq 水印，零定时器零轮询）；
// T-0268：无头面并入 getCockpit 首屏一往返（独立首调撤——hydrate 仅缺段兜底）。
import { useEffect, useState } from "react";
import { client } from "../api";
import { onLive } from "../stream";
import { applyHeadless, hydrate, useSlice } from "../store";
import { Component, type ReactNode } from "react";
import { Card } from "../components/ui";
import type { GetCockpitResponse, CockpitPlatformRow, CockpitSeat, CockpitTicket, CockpitArmed } from "../../gen/dashboardv1/core_pb";

const T_PAGE = 10; // ③工单板页长（T-0262——后端 ticket_limit 同值）

const ts = (t?: { seconds: bigint }) =>
	t ? new Date(Number(t.seconds) * 1000).toLocaleString("zh-CN", { hour12: false }) : "—";

// HeadlessTile 无头状态面（T-0263——推送制瓦片：Watch ev_headless 流帧+seq 水印；
// T-0268——挂载首取并入 getCockpit.headless（零独立首调），hydrate 仅缺段径
// unary 兜底（data 在=零请求）；onLive 即时回放非重连不补，断流复连才 unary
// 回补+接流 seq>水位帧补洞（补洞逻辑在 store.applyHeadless 水印判）。
// 推送源=既有 Watch 单泵——本件零 interval 零定时器；瓦片契约=Card 直子级入格）。
function HeadlessTile() {
	const hl = useSlice("headless");
	useEffect(() => {
		const refill = () => client.getHeadless({}).then(applyHeadless).catch(() => {});
		hydrate("headless", () => client.getHeadless({}));
		let replayed = false; // onLive 首调=现态即时回放非断流复连——跳过，不重演首取
		return onLive((v?: unknown) => {
			if (!replayed) { replayed = true; return; }
			if (v === true) refill();
		});
	}, []);
	if (!hl.data) return <Card title="无头状态面"><p className="dim">{hl.err ? "该段数据缺（读败）" : "载中…"}</p></Card>;
	const rows = hl.data.rows;
	return <Card title="无头状态面（推送制）">
		{rows.length === 0 ? <p className="dim">无在册无头单</p> :
			<table><thead><tr><th>票</th><th>席</th><th>底模</th><th>相位</th><th>五态</th><th>最后活动</th></tr></thead>
				<tbody>{rows.map((r) => <tr key={r.ticket}><td>{r.ticket}</td><td>{r.seat}</td>
					<td>{r.model || "—"}</td><td>{r.phase || "—"}</td><td>{r.state}</td><td>{ts(r.lastActivity)}</td></tr>)}</tbody></table>}
	</Card>;
}

export function CockpitPage() {
	const [d, setD] = useState<GetCockpitResponse | null>(null);
	const [page, setPage] = useState(1);
	useEffect(() => {
		client.getCockpit({ ticketLimit: T_PAGE, ticketOffset: (page - 1) * T_PAGE })
			.then((d) => { if (d.headless) applyHeadless(d.headless); setD(d); })
			.catch(() => setD(null));
	}, [page]);
	if (!d) return <div className="page">载中…</div>;
	const miss = new Set(d.missing);
	const sec = (name: string, el: ReactNode) =>
		miss.has(name) ? <p className="dim">该段数据缺（查询败——不冒零）</p> :
			<SecBoundary>{el}</SecBoundary>;
	const m = d.money; // money 块缺段判据真源=服务端 missing 字段——?. 后链式禁裸调
	const tPages = Math.max(1, Math.ceil(d.ticketsTotal / T_PAGE));
	const safePage = Math.min(page, tPages);
	return (
		<div className="page">
			<div className="health-strip">
				<span>业主驾驶舱</span>
				<span className="ink-3">快照 {ts(d.asOf)}</span>
			</div>
			<div className="tile-grid">
				<Card title="①⑤ 存活与在做什么">
					{sec("seats", <>
						<table><thead><tr><th>席</th><th>票</th><th>阶段</th><th>最近自报</th><th>备注</th></tr></thead>
						<tbody>{d.seats.map((s: CockpitSeat) => <tr key={s.seat}><td>{s.seat}</td><td>{s.ticket}</td>
							<td>{s.stage}</td><td>{ts(s.ts)}</td><td>{s.note}</td></tr>)}</tbody></table></>)}
				</Card>
				<Card title="② 钱面">{sec("money", !m ? <p className="dim">该段数据缺</p> : <>
					<p>月净利 {m.netMonth.toFixed(2)}｜本金水位 {m.capital.toFixed(2)}</p>
					<p>敞口 {m.exposure.toFixed(0)} / 上限 {m.exposureLimit.toFixed(0)}（10% 硬线）：
						{{0:"合规绿",1:"顶格黄",2:"越限红"}[m.exposureState ?? 0]}｜
						停泊态：活期/Earn 停泊（闲资默认 Earn——见执行面件）</p>
					<p>近30d 日净：{m.daily.map(v => v.toFixed(1)).join(" · ")}</p>
					<table><thead><tr><th>平台</th><th>承诺</th><th>已入</th><th>停泊</th><th>敞口</th></tr></thead>
					<tbody>{m.platforms.map((p: CockpitPlatformRow) => <tr key={p.platform}><td>{p.platform}</td><td>{p.committed < 0 ? "未设" : p.committed.toFixed(0)}</td>
						<td>{p.platform === "MT5" && miss.has("mt5") ? "数据缺" : p.deposited.toFixed(0)}</td><td>{p.parked.toFixed(0)}</td><td>{p.exposure.toFixed(0)}</td></tr>)}</tbody></table>
					{miss.has("mt5") && <p className="dim">MT5 平台已入段数据缺（mt5Live 读败——合计行为不含 MT5 部分口径，非真 0）</p>}</>)}
				</Card>
				<Card title="③ 工单板">{sec("tickets", <>
					<table><thead><tr><th>票</th><th>状态</th><th>席</th><th>时间</th></tr></thead>
					<tbody>{d.tickets.map((t: CockpitTicket) => <tr key={t.ticket}><td>{t.ticket}</td><td>{t.state}</td>
						<td>{t.seat}</td><td>{ts(t.ts)}</td></tr>)}</tbody></table>
					{tPages > 1 && (
						<div className="pager">
							<button type="button" disabled={safePage <= 1} onClick={() => setPage(safePage - 1)}>上一页</button>
							<span>第 {safePage} / {tPages} 页（共 {d.ticketsTotal} 条）</span>
							<button type="button" disabled={safePage >= tPages} onClick={() => setPage(safePage + 1)}>下一页</button>
						</div>
					)}</>)}
				</Card>
				<Card title="④ 待开仓（armed）">{sec("armed", <>
					<table><thead><tr><th>#</th><th>标的</th><th>方向</th><th>建时</th><th>TTL余(h)</th></tr></thead>
					<tbody>{d.armed.map((a: CockpitArmed) => <tr key={a.id}><td>{a.id}</td><td>{a.instId}</td>
						<td>{a.direction}</td><td>{ts(a.created)}</td><td>{a.ttlHoursLeft.toFixed(1)}</td></tr>)}</tbody></table></>)}
				</Card>
				{/* T-0263 状态面瓦片插槽——新瓦片以 <Card> 件作 .tile-grid 直接子级
				    追加于此（armed 瓦片之后），禁裸 section/div 入格 */}
				<HeadlessTile />
			</div>
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
