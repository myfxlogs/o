// flowpulse.tsx —— 活水段段链（T-0307：运行时机器流——票=组织流语义不混；
// 灯色服务端 StageLamp 派生；行数闸拆分件）。
import { client } from "../api";
import { useLive } from "../hooks";
import { StateDot } from "./ui";

// 活水段（T-0307：运行时机器流段链——ops_event 票=组织流语义不混，
// 段=机器流；灯色服务端 StageLamp 派生——阈按各臂节拍常量钉）。
export function FlowPulseStrip() {
	const fp = useLive(["ev_funding", "ev_eval"], () => client.getFlowPulse({}));
	if (!fp.data) return fp.err ? <p className="dim">活水段数据缺（读败）</p> : null;
	const fmtLag = (s: number) => (s >= 3600 ? `${(s / 3600).toFixed(1)}h` : s >= 60 ? `${(s / 60).toFixed(0)}m` : `${s}s`);
	return (
		<div className="pulse-strip">
			<div className="pipe-chain">
				{fp.data.stages.map((st) => (
					<span key={st.key} className="pipe-seat" title={`${st.label} · 预期节拍 ${fmtLag(Number(st.expectS))} · 24h ${st.count24h} 行`}>
						<StateDot state={st.state} breathe={st.state === 0}
							title={`${st.label}：${st.ok ? (st.lastTs ? `末行 ${fmtLag(Number(st.lagS))} 前` : "无行") : "查询败显缺"}`} />
						{st.label}
						{st.ok && st.lastTs ? <span className="ink-3">·{fmtLag(Number(st.lagS))}</span> : <span className="ink-3">·缺</span>}
					</span>
				))}
			</div>
			<div className="ink-3" style={{ marginTop: 2 }}>
				{fp.data.rosterCount} 岗在岗 · 最坏滞后 {fmtLag(Number(fp.data.rosterWorstLagS))}
			</div>
		</div>
	);
}

