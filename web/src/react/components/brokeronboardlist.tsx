// brokeronboardlist.tsx —— T-0407/T-0430 broker 接入卡列表面拆件（行数棘轮
// D-077/D-588——纯渲染：人话字典+phase 判态+并集行表；phase 判据全服务端
// 派生，前端零判据纯渲染）。
// O-41377 刀1：接入态=phase 单源（roster 户=会话环自报 roster.state 活性面
// ——okxpriv/mt5 reportVenueState 同源；非名册行=beat 末拍 fallback）——
// beatFailed 不再并入态栏（败+愈同帧：ws-login 1006 败行冻结不再盖活会话），
// 败行降历史注「末败@t」落末拍列+stale 龄标注。
import type { BrokerOnboardRow } from "../../gen/dashboardv1/core_pb";
import { BROKER_PHASE_CN, BROKER_STATE_CN, cn } from "./labels";

// 人话字典归一 labels.ts（T-0441）——未知值人话兜底不裸出机读枚举。
const phaseTxt = (r: BrokerOnboardRow): string => {
	if (r.phase) return cn(BROKER_PHASE_CN, r.phase, "未识别态");
	if (r.proto === "mt4" && !r.state) return "凭据在册·MT4 链未接";
	if (!r.state) return "凭据在册·候发现";
	return cn(BROKER_STATE_CN, r.state, "未识别态");
};

// 末拍时刻短式（MM-DD HH:mm——「末败@t」注用；title 悬停仍全时刻）。
const beatTime = (r: BrokerOnboardRow): string => {
	const s = r.lastBeatAt ? Number(r.lastBeatAt.seconds) : 0;
	if (!s) return "";
	const d = new Date(s * 1000);
	const p = (n: number) => String(n).padStart(2, "0");
	return `${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
};

// stale 龄注（活性源缺席 fallback 末拍显——末拍越陈越该让人看见）。
const beatAge = (r: BrokerOnboardRow): string => {
	const s = r.lastBeatAt ? Number(r.lastBeatAt.seconds) : 0;
	if (!s) return "";
	const sec = Math.max(0, Date.now() / 1000 - s);
	if (sec < 3600) return `${Math.max(1, Math.floor(sec / 60))}分前`;
	if (sec < 86400) return `${Math.floor(sec / 3600)}时前`;
	return `${Math.floor(sec / 86400)}天前`;
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
							{r.phase === "failed"
								? <span className="chip chip-critical">接入败</span>
								: phaseTxt(r)}
						</td>
						<td className="num mono">{String(r.strategies)}</td>
						<td className="ink-3" title={r.lastBeatAt
							? new Date(Number(r.lastBeatAt.seconds) * 1000).toLocaleString("zh-CN", { hour12: false })
							: undefined}>
							{r.beatFailed ? `末败@${beatTime(r) || "?"} ` : ""}
							{r.beatNote || "—"}
							{beatAge(r) ? ` ·${beatAge(r)}` : ""}
						</td>
					</tr>
				))}
			</tbody>
		</table>
	);
}
