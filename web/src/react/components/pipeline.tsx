// pipeline.tsx —— 座舱管道卡件族（T-0411 自 Cockpit.tsx 拆出——行数棘轮
// D-588：卡件族自持，Cockpit 页只留页装配。PipelineTile=工作流管道主瓦片
// （v4 三张无头监控卡合一：1 票 1 行管道流，走过显/未到隐，在干席 spinner、
// 已移交席呼吸灯、行首行级灯、驳回环标注；判态全服务端派生
// pipeline_face.go，前端纯渲染零判据）；OrgPipelineTile=工程票组织流卡
// （T-0309 返修——ops_event 组织流挪系统页消费）；LiveBadge=流连接角标。
// 拆分为纯搬移零逻辑改动——消费面 System.tsx import 同步改道。）
import { useEffect, useState } from "react";
import { onLive } from "../stream";
import { useSlice } from "../store";
import { Card, EmptyState, Skeleton, StateDot } from "./ui";
import { FlowPulseStrip, WorkItemsBlock } from "./flowpulse";
import type { CockpitPipelineRow, CockpitPipelineStep } from "../../gen/dashboardv1/core_pb";

// 步/行结果人话字典（纯显示映射非判据——终态派生在服务端 pipeline_face.go）。
const STEP_RESULT: Record<string, string> = {
	working: "在干", closed: "完成移交", passed: "通过",
	rejected: "驳回", blocked: "挂起", error: "异常",
};

// 耗时格式（ms/min → 人话：45s / 32m / 2h6m）。
const durMs = (ms: bigint) => {
	const m = Number(ms) / 60000;
	if (m < 1) return `${Math.max(1, Math.round(Number(ms) / 1000))}s`;
	if (m < 60) return `${Math.round(m)}m`;
	return `${Math.floor(m / 60)}h${Math.round(m % 60)}m`;
};
const durMin = (min: bigint) => {
	const m = Number(min);
	return m < 60 ? `${m}m` : `${Math.floor(m / 60)}h${m % 60}m`;
};

// LiveBadge 流连接角标（v2 常显——数据鲜度一眼可见；onLive 挂载即回放现态，
// 灯语同 Shell LiveDot：绿呼吸=流活，黄=断流重连中）。
export function LiveBadge() {
	const [live, setLive] = useState<boolean | null>(null);
	useEffect(() => onLive((v?: unknown) => setLive(Boolean(v))), []);
	return (
		<span className="ink-3 fact">
			<StateDot
				state={live ? 0 : 1}
				breathe={Boolean(live)}
				title={live ? "Watch 流在线——事件驱动实时" : "Watch 断流重连中——数据可能陈"}
			/>
			{" "}流{live === null ? "连接中" : live ? "在线" : "断流重连中"}
		</span>
	);
}

// SeatBadge 步链徽（设计稿钉组件形 <SeatBadge seat result active>：
// active=true 渲染 spinner（⟳ 在干），否则呼吸灯（色=该席最终结果——
// 服务端派生 state/breathe，前端零判据）；title=席位级四要素悬停）。
function SeatBadge(props: { s: CockpitPipelineStep }) {
	const s = props.s;
	const hover = `${s.seat} | 进:${s.enter || "—"} | 出:${s.exit || "在干"} | ${STEP_RESULT[s.result] ?? s.result} | 耗时 ${durMs(s.ms)}`;
	if (s.active && s.spin) {
		return <span className="pipe-seat" title={hover}><span className="dot spin" />{s.seat}</span>;
	}
	return <span className="pipe-seat" title={hover}><StateDot state={s.state} breathe={s.breathe} title={hover} />{s.seat}</span>;
}

// PipelineTile 工作流管道（v4 主瓦片——pipeline 分片：首帧 getCockpit.pipeline
// 一往返，后 ev_pipeline 流推 seq 序号去重；断流复连 unary 回补。行=1 票：
// 行级灯+票号+标题+步链+已历时；活跃>20 截断「+N more」（服务端 total_active）。
// 完结（部署上线）票不入 rows——行自动退场即流推帧）。
export function PipelineTile() {
	const pl = useSlice("pipeline");
	const extra = pl.data ? <StateDot state={0} breathe title={`${pl.data.rows.length}/${pl.data.totalActive} 在飞 · 序号 ${pl.data.seq}`} /> : undefined;
	if (!pl.data) return <Card title="③ 工作流管道" extra={extra}>{pl.err ? <p className="dim">该段数据缺（读败）</p> : <Skeleton rows={4} />}</Card>;
	return (
		<Card title="③ 工作流管道" extra={extra}>
			<FlowPulseStrip />
			<WorkItemsBlock />
		</Card>
	);
}

// OrgPipelineTile 工程票组织流卡（T-0309 返修①②——ops_event 组织流挪系统页
// 消费：票号/标题/步链/灯整体挪位，GetPipeline 端点不删；钱径舱位让 WorkItems）。
export function OrgPipelineTile() {
	const pl = useSlice("pipeline");
	const extra = pl.data ? <StateDot state={0} breathe title={`${pl.data.rows.length}/${pl.data.totalActive} 在飞 · 序号 ${pl.data.seq}`} /> : undefined;
	if (!pl.data) return <Card title="工程票管道" extra={extra}>{pl.err ? <p className="dim">该段数据缺（读败）</p> : <Skeleton rows={4} />}</Card>;
	const rows = pl.data.rows;
	const more = Number(pl.data.totalActive) - rows.length;
	return (
		<Card title="工程票管道（组织流）" extra={extra}>
			{rows.length === 0 ? <EmptyState title="无在飞票" hint="活跃票（未部署上线）为空——管道空闲" /> :
				<div className="tile-scroll">{rows.map((r: CockpitPipelineRow) => {
					const rowHover = `${r.ticket} | ${r.title || "无题"} | 已历时 ${durMin(r.elapsedMin)} | 当前卡在 ${r.currentSeat}（${STEP_RESULT[r.currentState] ?? r.currentState}）`;
					return (
						<div className="pipe-row" key={r.ticket} title={rowHover}>
							<StateDot state={r.state} breathe={r.breathe} title={`行级灯（=当前活动席状态色） · ${rowHover}`} />
							<span className="pipe-ticket">{r.ticket}</span>
							{r.title && <span className="pipe-title">{r.title}</span>}
							<span className="pipe-chain">
								{r.steps.map((s, i) => (
									<span key={i} className="pipe-seat">
										{i > 0 && <span className="pipe-arrow">→</span>}
										<SeatBadge s={s} />
									</span>
								))}
							</span>
							<span className="pipe-elapsed">{durMin(r.elapsedMin)}</span>
						</div>
					);
				})}
				{more > 0 && <p className="dim">+{more} more…</p>}
				</div>}
		</Card>
	);
}
