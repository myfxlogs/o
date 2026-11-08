// brokeronboardlist.tsx —— T-0407/T-0430 broker 接入卡列表面拆件（行数棘轮
// D-077/D-588——纯渲染：人话字典+phase 判态+并集行表；phase 判据全服务端
// 派生，前端零判据纯渲染）。
import type { BrokerOnboardRow } from "../../gen/dashboardv1/core_pb";
import { BROKER_PHASE_CN, BROKER_STATE_CN, cn } from "./labels";

// 人话字典归一 labels.ts（T-0441）——未知值人话兜底不裸出机读枚举。
const phaseTxt = (r: BrokerOnboardRow): string => {
	if (r.phase) return cn(BROKER_PHASE_CN, r.phase, "未识别态");
	if (r.proto === "mt4" && !r.state) return "凭据在册·MT4 链未接";
	if (!r.state) return "凭据在册·候发现";
	return cn(BROKER_STATE_CN, r.state, "未识别态");
};


// BrokerOnboardList 在册 broker 行表（空行态由调用方 Card 呈）。
export function BrokerOnboardList(props: { rows: BrokerOnboardRow[] }) {
	return props.rows.length === 0 ? null : (
		<table className="rows">
			<thead>
				<tr><th>broker</th><th>协议</th><th>login</th><th>接入态</th><th>在跑策略</th><th>末拍/败因</th></tr>
			</thead>
			<tbody>
				{props.rows.map((r) => (
					<tr key={r.venue}>
						<td>{r.venue}</td>
						<td className="ink-3">{(r.proto || "—").toUpperCase()}</td>
						<td className="mono">{r.login || "—"}</td>
						<td>
							{r.phase === "failed" || r.beatFailed
								? <span className="chip chip-critical">接入败</span>
								: phaseTxt(r)}
						</td>
						<td className="num mono">{String(r.strategies)}</td>
						<td className="ink-3" title={r.lastBeatAt
							? new Date(Number(r.lastBeatAt.seconds) * 1000).toLocaleString("zh-CN", { hour12: false })
							: undefined}>
							{r.beatNote || "—"}
						</td>
					</tr>
				))}
			</tbody>
		</table>
	);
}
