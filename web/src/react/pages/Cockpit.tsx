// Cockpit.tsx —— T-0239 业主驾驶舱（五块只读：存活/钱面/工单/待开/在做什么——
// 判态服务端派生，前端纯渲染零判据）。
import { useEffect, useState } from "react";
import { client } from "../api";
import type { GetCockpitResponse } from "../../gen/dashboardv1/core_pb";

export function CockpitPage() {
	const [d, setD] = useState<GetCockpitResponse | null>(null);
	useEffect(() => {
		client.getCockpit({}).then(setD).catch(() => setD(null));
	}, []);
	if (!d) return <div className="page">载中…</div>;
	const ts = (t?: { seconds: bigint }) =>
		t ? new Date(Number(t.seconds) * 1000).toLocaleString("zh-CN", { hour12: false }) : "—";
	return (
		<div className="page">
			<h2>业主驾驶舱</h2>
			<section><h3>①⑤ 存活与在做什么</h3>
				<table><thead><tr><th>席</th><th>票</th><th>阶段</th><th>最近自报</th><th>备注</th></tr></thead>
				<tbody>{d.seats.map(s => <tr key={s.seat}><td>{s.seat}</td><td>{s.ticket}</td>
					<td>{s.stage}</td><td>{ts(s.ts)}</td><td>{s.note}</td></tr>)}</tbody></table></section>
			<section><h3>② 钱面</h3>
				<p>月净利 {d.money?.netMonth.toFixed(2)}｜本金水位 {d.money?.capital.toFixed(2)}</p>
				<p>近30d 日净：{d.money?.daily.map(v => v.toFixed(1)).join(" · ")}</p></section>
			<section><h3>③ 工单板</h3>
				<table><thead><tr><th>票</th><th>状态</th><th>席</th><th>时间</th></tr></thead>
				<tbody>{d.tickets.map(t => <tr key={t.ticket}><td>{t.ticket}</td><td>{t.state}</td>
					<td>{t.seat}</td><td>{ts(t.ts)}</td></tr>)}</tbody></table></section>
			<section><h3>④ 待开仓（armed）</h3>
				<table><thead><tr><th>#</th><th>标的</th><th>方向</th><th>建时</th><th>TTL余(h)</th></tr></thead>
				<tbody>{d.armed.map(a => <tr key={a.id}><td>{a.id}</td><td>{a.instId}</td>
					<td>{a.direction}</td><td>{ts(a.created)}</td><td>{a.ttlHoursLeft.toFixed(1)}</td></tr>)}</tbody></table></section>
		</div>
	);
}
