// Cockpit.tsx —— T-0239 业主驾驶舱（五块只读：存活/钱面/工单/待开/在做什么——
// 判态服务端派生，前端纯渲染零判据）。
import { useEffect, useState } from "react";
import { client } from "../api";
import { Component, type ReactNode } from "react";
import type { GetCockpitResponse, CockpitPlatformRow, CockpitSeat, CockpitTicket, CockpitArmed } from "../../gen/dashboardv1/core_pb";

export function CockpitPage() {
	const [d, setD] = useState<GetCockpitResponse | null>(null);
	useEffect(() => {
		client.getCockpit({}).then(setD).catch(() => setD(null));
	}, []);
	if (!d) return <div className="page">载中…</div>;
	const ts = (t?: { seconds: bigint }) =>
		t ? new Date(Number(t.seconds) * 1000).toLocaleString("zh-CN", { hour12: false }) : "—";
	const miss = new Set(d.missing);
	const sec = (name: string, el: ReactNode) =>
		miss.has(name) ? <p className="dim">该段数据缺（查询败——不冒零）</p> :
			<SecBoundary>{el}</SecBoundary>;
	const m = d.money; // money 块缺段判据真源=服务端 missing 字段——?. 后链式禁裸调
	return (
		<div className="page">
			<h2>业主驾驶舱（快照 {ts(d.asOf)}）</h2>
			<section><h3>①⑤ 存活与在做什么</h3>{sec("seats", <>
				<table><thead><tr><th>席</th><th>票</th><th>阶段</th><th>最近自报</th><th>备注</th></tr></thead>
				<tbody>{d.seats.map((s: CockpitSeat) => <tr key={s.seat}><td>{s.seat}</td><td>{s.ticket}</td>
					<td>{s.stage}</td><td>{ts(s.ts)}</td><td>{s.note}</td></tr>)}</tbody></table></>)}</section>
			<section><h3>② 钱面</h3>{sec("money", !m ? <p className="dim">该段数据缺</p> : <>
				<p>月净利 {m.netMonth.toFixed(2)}｜本金水位 {m.capital.toFixed(2)}</p>
				<p>敞口 {m.exposure.toFixed(0)} / 上限 {m.exposureLimit.toFixed(0)}（10% 硬线）：
					{{0:"合规绿",1:"顶格黄",2:"越限红"}[m.exposureState ?? 0]}｜
					停泊态：活期/Earn 停泊（闲资默认 Earn——见执行面件）</p>
				<p>近30d 日净：{m.daily.map(v => v.toFixed(1)).join(" · ")}</p>
				<table><thead><tr><th>平台</th><th>承诺</th><th>已入</th><th>停泊</th><th>敞口</th></tr></thead>
				<tbody>{m.platforms.map((p: CockpitPlatformRow) => <tr key={p.platform}><td>{p.platform}</td><td>{p.committed < 0 ? "未设" : p.committed.toFixed(0)}</td>
					<td>{p.deposited.toFixed(0)}</td><td>{p.parked.toFixed(0)}</td><td>{p.exposure.toFixed(0)}</td></tr>)}</tbody></table></>)}</section>
			<section><h3>③ 工单板</h3>{sec("tickets", <>
				<table><thead><tr><th>票</th><th>状态</th><th>席</th><th>时间</th></tr></thead>
				<tbody>{d.tickets.map((t: CockpitTicket) => <tr key={t.ticket}><td>{t.ticket}</td><td>{t.state}</td>
					<td>{t.seat}</td><td>{ts(t.ts)}</td></tr>)}</tbody></table></>)}</section>
			<section><h3>④ 待开仓（armed）</h3>{sec("armed", <>
				<table><thead><tr><th>#</th><th>标的</th><th>方向</th><th>建时</th><th>TTL余(h)</th></tr></thead>
				<tbody>{d.armed.map((a: CockpitArmed) => <tr key={a.id}><td>{a.id}</td><td>{a.instId}</td>
					<td>{a.direction}</td><td>{ts(a.created)}</td><td>{a.ttlHoursLeft.toFixed(1)}</td></tr>)}</tbody></table></>)}</section>
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
