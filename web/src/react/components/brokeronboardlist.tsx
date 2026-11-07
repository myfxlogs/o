// brokeronboardlist.tsx —— T-0407/T-0430 broker 接入卡列表面拆件（行数棘轮
// D-077/D-588——纯渲染：人话字典+phase 判态+并集行表；phase 判据全服务端
// 派生，前端零判据纯渲染）。
import type { BrokerOnboardRow } from "../../gen/dashboardv1/core_pb";

// 人话字典（纯显示映射非判据——phase/台账态键→中文直译；未知值原样透出不吞）。
const PHASE_CN: Record<string, string> = {
	registered: "已注册·候验连",
	connecting: "验连中",
	verified: "已验证",
	failed: "接入败",
};
const STATE_CN: Record<string, string> = {
	registered: "已注册·候探测",
	probed: "已探测",
	mapped: "映射就绪",
	paper: "paper 激活",
	candidate: "候选",
	measuring: "度量中",
	recommended: "建议注册",
	rejected: "已拒",
	kyc_blocked: "KYC受限",
	adopted: "已采纳",
	dismissed: "已驳回",
	parked: "已停放",
};
const phaseTxt = (r: BrokerOnboardRow): string => {
	if (r.phase) return PHASE_CN[r.phase] ?? r.phase;
	if (r.proto === "mt4" && !r.state) return "凭据在册·MT4 链未接";
	if (!r.state) return "凭据在册·候发现";
	return STATE_CN[r.state] ?? r.state;
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
