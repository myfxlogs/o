// metaseats.tsx —— T-0312 第四刀：①存活卡 META-* 无头席读面区块（Cockpit 行数
// 闸拆分件——票务管机件「dev 会话在飞」注行被本块取代；节拍常量源=tick.sh
// 实钉：META-REGIME 30min/META-VERDICT 4h——龄超=标红显警不阻）。
import { StateDot } from "./ui";
import type { HeadlessRow } from "../../gen/dashboardv1/core_pb";

const metaCadenceS: Record<string, number> = { "META-REGIME": 1800, "META-VERDICT": 14400 };

export function MetaSeats({ rows, hlL, fmtLagMin }: {
	rows: HeadlessRow[];
	hlL: (s: string) => string;
	fmtLagMin: (s: number) => string;
}) {
	const metaRows = rows.filter((r) => r.ticket.startsWith("META-"));
	const age = (r: HeadlessRow) => {
		const ts = r.lastActivity?.seconds;
		return ts ? Math.floor((Date.now() - Number(ts) * 1000) / 1000) : 0;
	};
	return (<>{metaRows.map((r) => {
		const st = hlL(r.state);
		const a = age(r);
		const over = a > (metaCadenceS[r.ticket] ?? 3600);
		const color = over || st === "停滞" || st === "死亡" || st === "逾期" ? "u-bad"
			: st === "完工" ? "dim" : "u-good";
		return <div className="kpi-note" key={r.ticket} title={`${r.ticket} 态=${r.state}`}>
			<StateDot state={color === "u-bad" ? 2 : color === "dim" ? 3 : 0} breathe={st === "在飞" && !over} /> {r.seat || r.ticket} · {st} · {fmtLagMin(a)}
		</div>;
	})}</>);
}
