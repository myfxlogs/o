// metaseats.tsx —— T-0312 第四刀：①存活卡 META-* 无头席读面区块（Cockpit 行数
// 闸拆分件——票务管机件「dev 会话在飞」注行被本块取代）。
// T-0349 域界根修：META 注行改白名单制——只渲业务服务席（名册式显列：
// REGIME 分析席 / VENUEWATCH venue 哨兵；工程席 DM/VERDICT/GAUGES/GC/Dev-*
// 与后续新 META 席默认不渲——防漏域界再溃）；白名单内超 metaCadenceS 龄
// 剔行不渲（业主钦定过期不显示——死席缺席即信号，告警归 gauges 升件径）。
import { StateDot } from "./ui";
import type { HeadlessRow } from "../../gen/dashboardv1/core_pb";

// 业务服务席白名单（名册式——票面红线：勿 ticket 前缀匹配通吃，新席入册才显）
const metaSeatAllow = new Set(["META-REGIME", "META-VENUEWATCH"]);
// 节拍常量源=tick/timer 实钉：META-REGIME 30min（regime_tick）/META-VERDICT 4h/
// META-GAUGES 30min（o-gauges-tick）/META-VENUEWATCH 1h（o-venue-watch.timer
// OnUnitActiveSec=1h）——VERDICT/GAUGES 已剔出卡面，册项保留对钉防漂移
// （同名常量册若复用别面，口径即此）。
const metaCadenceS: Record<string, number> = { "META-REGIME": 1800, "META-VERDICT": 14400, "META-GAUGES": 1800, "META-VENUEWATCH": 3600 };

export function MetaSeats({ rows, hlL, fmtLagMin }: {
	rows: HeadlessRow[];
	hlL: (s: string) => string;
	fmtLagMin: (s: number) => string;
}) {
	const age = (r: HeadlessRow) => {
		const ts = r.lastActivity?.seconds;
		return ts ? Math.floor((Date.now() - Number(ts) * 1000) / 1000) : 0;
	};
	const metaRows = rows.filter((r) =>
		metaSeatAllow.has(r.ticket) && age(r) <= (metaCadenceS[r.ticket] ?? 3600));
	return (<>{metaRows.map((r) => {
		const st = hlL(r.state);
		const a = age(r);
		const color = st === "停滞" || st === "死亡" || st === "逾期" ? "u-bad"
			: st === "完工" ? "dim" : "u-good";
		return <div className="kpi-note" key={r.ticket} title={`${r.ticket} 态=${r.state}`}>
			<StateDot state={color === "u-bad" ? 2 : color === "dim" ? 3 : 0} breathe={st === "在飞"} /> {r.seat || r.ticket} · {st} · {fmtLagMin(a)}
		</div>;
	})}</>);
}
