// flowpulse.tsx —— 活水段段链（T-0307：运行时机器流——票=组织流语义不混；
// 灯色服务端 StageLamp 派生；行数闸拆分件）。
import { client } from "../api";
import { useLive } from "../hooks";
import { EmptyState, StateDot } from "./ui";

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
						{/* T-0350 需求驱动段态标非龄——服务端 status 非空即渲态（空=周期段/exec 活态，龄渲沿用） */}
						{st.ok && st.status ? <span className="ink-3">·{st.status}</span>
							: st.ok && st.lastTs ? <span className="ink-3">·{fmtLag(Number(st.lagS))}</span> : <span className="ink-3">·缺</span>}
					</span>
				))}
			</div>
			<div className="ink-3" style={{ marginTop: 2 }}>
				{fp.data.rosterCount} 岗在岗 · 最坏滞后 {fmtLag(Number(fp.data.rosterWorstLagS))}
			</div>
		</div>
	);
}


// WorkItemsBlock 钱径工件行（T-0309——③卡下半段：工件从「被看见」走向
// 「变成利润」的在飞集合；行=ref+stage 徽标+在飞时长；服务端排序 proposal→
// position→venue cap10；空态「无在飞工件」）。
export function WorkItemsBlock() {
	const fp = useLive(["ev_funding", "ev_eval", "ev_proposal", "ev_position"], () => client.getFlowPulse({}));
	if (!fp.data) return fp.err ? <p className="dim">工件行数据缺（读败）</p> : null;
	const items = fp.data.workItems ?? [];
	if (items.length === 0) return <EmptyState title="无在飞工件" hint="钱径工件（提案/持仓/venue 度量）为零" />;
	const ago = (t?: { seconds: bigint }) => {
		if (!t) return "—";
		const s = Math.floor((Date.now() - Number(t.seconds) * 1000) / 1000);
		return s >= 3600 ? `${(s / 3600).toFixed(1)}h` : s >= 60 ? `${(s / 60).toFixed(0)}m` : `${s}s`;
	};
	const KIND: Record<string, string> = { proposal: "提案", position: "持仓", venue: "侦察" };
	return (
		<div className="tile-scroll">
			{items.map((w) => (
				<div className="pipe-row" key={w.kind + ":" + w.ref}>
					<span className="pipe-ticket">{KIND[w.kind] ?? w.kind}</span>
					<span className="pipe-title">{w.ref}</span>
					<span className="pipe-seat">{w.stage}</span>
					{w.progress && <span className="pipe-seat">{w.progress}</span>}
					<span className="pipe-elapsed">{ago(w.sinceTs)}</span>
				</div>
			))}
		</div>
	);
}
