// Cockpit.tsx —— 业主驾驶舱 v4（设计稿·驾驶舱管道视图 2026-10-31 业主钦定：
// 三张无头监控卡合一——「③ 席位与在做什么」「③ 无头状态面」「③ 工单板」撤，
// 换「工作流管道」瓦片：1 票 1 行管道流，走过显/未到隐，在干席 spinner、
// 已移交席呼吸灯（绿=完成/通过 黄=驳回/挂起/等审计 红=死/异常）、行首行级灯、
// 驳回环 R2/R3 标注；完结=部署上线→行自动退场（ev_pipeline 流推 ≤10s）。
// 判态全服务端派生（pipeline_face.go），前端纯渲染零判据。保留不动：①② 钱面
// （KPI 条+平台明细）、④ 待开仓、System 页；三张旧卡的 ops 数据后端保留。
// 降级语义原样保全（T-0248：sec/SecBoundary/missing 显缺不冒零）；流连接
// 角标+asOf 快照戳常显；headless 分片保留=① 存活卡「无头面」注行数据源
// （分片仍在流推，只撤其独立卡面）。
import { useEffect, useState } from "react";
import { client } from "../api";
import { onLive } from "../stream";
import { applyHeadless, applyPipeline, hydrate, useSlice } from "../store";
import { Component, type ReactNode } from "react";
import { Card, EmptyState, Skeleton, StateDot, UtcClock } from "../components/ui";
import { LiveBadge, PipelineTile } from "../components/pipeline";
import { MetaSeats } from "../components/metaseats";
import { QuoteBoard } from "../components/QuoteBoard";
import { AGE } from "../components/panel";
import { DIRECTION_CN, HEADLESS_STATE_CN, PLATFORM_CN, cn } from "../components/labels";
import type {
  GetCockpitResponse,
  CockpitPlatformRow,
  CockpitArmed,
} from "../../gen/dashboardv1/core_pb";

const ts = (t?: { seconds: bigint }) =>
	t ? new Date(Number(t.seconds) * 1000).toLocaleString("zh-CN", { hour12: false }) : "—";

export function CockpitPage() {
	const [d, setD] = useState<GetCockpitResponse | null>(null);
	const hl = useSlice("headless");
	// 取数泵页级单源（v2 上提；v4 加 pipeline 分片同族径）：T-0268 挂载首取
	// 并入 getCockpit（headless+pipeline 两段）——hydrate 仅缺段径 unary 兜底；
	// onLive 即时回放非断流复连不补，复连才 unary 回补+接流 seq>水位帧补洞
	// （水印判在 store.applyHeadless/applyPipeline）。零 interval 零定时器。
	useEffect(() => {
		const refill = () => {
			client.getHeadless({}).then(applyHeadless).catch(() => {});
			client.getPipeline({}).then(applyPipeline).catch(() => {});
		};
		hydrate("headless", () => client.getHeadless({}));
		hydrate("pipeline", () => client.getPipeline({}));
		let replayed = false; // onLive 首调=现态即时回放非断流复连——跳过，不重演首取
		return onLive((v?: unknown) => {
			if (!replayed) { replayed = true; return; }
			if (v === true) refill();
		});
	}, []);
	useEffect(() => {
		client.getCockpit({})
			.then((d) => {
				if (d.headless) applyHeadless(d.headless);
				if (d.pipeline) applyPipeline(d.pipeline);
				setD(d);
			})
			.catch(() => setD(null));
	}, []);
	if (!d) return (
		<div className="page">
			<div className="health-strip">
				<span>业主驾驶舱</span>
				<LiveBadge />
				<span className="ink-3">更新 …</span>
			</div>
			{/* 载中=瓦片骨架（诚实空态律——骨架=卡面独立，不整页灰） */}
			<div className="trx-kpi-strip">
				{["① 钱安全吗", "② 赚了多少（MTD）", "② 近 30 天", "① 存活"].map((k) => (
					<Card key={k} className="kpi" title={k}><Skeleton rows={2} /></Card>
				))}
			</div>
			<Card title="③ 工作流管道"><Skeleton rows={4} /></Card>
			<div className="duo">
				<Card title="④ 待开仓"><Skeleton rows={3} /></Card>
				<Card title="①② 钱面平台明细"><Skeleton rows={4} /></Card>
			</div>
		</div>
	);
	const miss = new Set(d.missing);
	const sec = (name: string, el: ReactNode) =>
		miss.has(name) ? <p className="dim">该段数据缺（查询败——不冒零）</p> :
			<SecBoundary>{el}</SecBoundary>;
	const m = d.money; // money 块缺段判据真源=服务端 missing 字段——?. 后链式禁裸调
	// 展示层聚合（渲染算术，非判据——判态全部服务端派生）
	const expSt = m?.exposureState ?? 0;
	const parked = m ? m.platforms.reduce((a, p) => a + p.parked, 0) : 0;
	const d30 = m ? m.daily.reduce((a, v) => a + v, 0) : 0;
	const posDays = m ? m.daily.filter((v) => v > 0).length : 0;
	const hlRows = hl.data?.rows ?? [];
	// 计数双语归一（服务端判态中文标 or pump 原键——HEADLESS_STATE_CN 正向
	// 映射后按显示标判，两径通吃；词表归一 labels.ts T-0441）。
	const hlL = (s: string) => cn(HEADLESS_STATE_CN, s, s);

	const rosterStale = !miss.has("roster") && Number(d.rosterWorstLagS) > 600; // T-0308：active 岗心跳停滞>10min 钉常量转红
	const rosterWorstSeat = d.rosterWorstSeat || "";
	const fmtLagMin = (sec: number) => (sec >= 3600 ? `${(sec / 3600).toFixed(1)}h` : sec >= 60 ? `${(sec / 60).toFixed(0)}m` : `${sec}s`);
	// v3 呼吸灯：瓦片右上角状态点统一灯语（绿呼吸=段数据在；红=段缺/异常），
	// title 携鲜度（快照 asOf / 最近自报——AGE 同 panel 件）。纯显示映射零判据。
	const asOfD = d.asOf ? new Date(Number(d.asOf.seconds) * 1000) : null;
	const asOfAge = asOfD ? AGE(asOfD) : "—";
	return (
		<div className="page">
			<div className="health-strip">
				<span>业主驾驶舱</span>
				<LiveBadge />
				<span className="ink-3">更新 {ts(d.asOf)}</span>
			</div>
			{/* —— 首档·答①②置顶放大（业主第一性四问重排——人话结论大字居首，
			    数字第二层；trx-kpi-strip=TRX 首档同源件，零新 CSS）—— */}
			<div className="trx-kpi-strip">
				<Card className="kpi" title="① 钱安全吗"
					extra={m ? <StateDot state={expSt} breathe={expSt === 0} title={`敞口硬线判态（服务端派生）· 更新 ${asOfAge}`} /> : undefined}>
					{sec("money", !m ? <p className="dim">该段数据缺</p> : <>
						<div className={"kpi-v " + (expSt === 0 ? "u-good" : expSt === 2 ? "u-bad" : "")}>
							{{ 0: "合规绿", 1: "顶格黄", 2: "越限红" }[expSt] ?? "—"}
						</div>
						<div className="kpi-note">敞口 ${m.exposure.toFixed(0)} / 上限 ${m.exposureLimit.toFixed(0)}（10% 硬线）</div>
						<div className="kpi-note">停泊 ${parked.toFixed(0)}（闲资默认活期/Earn——见执行面件）</div>
					</>)}
				</Card>
				<Card className="kpi" title="② 赚了多少（MTD）"
					extra={<StateDot state={miss.has("money") ? 2 : 0} breathe={!miss.has("money")} title={`更新 ${asOfAge}`} />}>
					{sec("money", !m ? <p className="dim">该段数据缺</p> : <>
						<div className={"kpi-v " + (m.netMonth >= 0 ? "u-good" : "u-bad")}>
							{m.netMonth >= 0 ? "+" : "-"}${Math.abs(m.netMonth).toFixed(2)}
						</div>
						<div className="kpi-note">本金水位 ${m.capital.toFixed(2)}</div>
					</>)}
				</Card>
				<Card className="kpi" title="② 近 30 天"
					extra={<StateDot state={miss.has("money") ? 2 : 0} breathe={!miss.has("money")} title={`更新 ${asOfAge}`} />}>
					{sec("money", !m ? <p className="dim">该段数据缺</p> : <>
						<div className={"kpi-v " + (d30 >= 0 ? "u-good" : "u-bad")}>
							{d30 >= 0 ? "+" : "-"}${Math.abs(d30).toFixed(2)}
						</div>
						<div className="kpi-note">日净 {posDays} 正 / {m.daily.length - posDays} 负</div>
					</>)}
				</Card>
				<Card className="kpi" title="① 存活"
					extra={!miss.has("roster") ? <StateDot state={rosterStale ? 2 : 0} breathe={!rosterStale}
						title={rosterStale ? `岗心跳停滞 ${rosterWorstSeat} ${fmtLagMin(Number(d.rosterWorstLagS))}` : "生产岗心跳正常"} /> : undefined}>
					{sec("roster", <>
						<div className="kpi-v">{d.rosterActive}<span className="kpi-unit">生产岗在岗</span></div>
						<div className="kpi-note">最坏心跳滞后 {fmtLagMin(Number(d.rosterWorstLagS))}{rosterStale ? `（${d.rosterWorstSeat} 停滞超 10min）` : ""}</div>
					</>)}
					<MetaSeats rows={hlRows} hlL={hlL} fmtLagMin={fmtLagMin} />
				</Card>
			</div>
			{/* —— 次档·答③机器面（v4：三张无头监控卡合一——工作流管道主瓦片，
			    1 票 1 行管道流，ev_pipeline 流推秒级到屏）—— */}
			<PipelineTile />
			{/* —— T-0382 行情·价格提醒（观测面——单向飞书告警非条件单）—— */}
			<QuoteBoard />
			{/* —— 底档·④要我动的（armed）+①②平台明细（v3.2 同档保留卡不动）—— */}
			<div className="duo">
				<Card title="④ 待开仓"
					extra={<span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
						<UtcClock />
						<StateDot state={miss.has("armed") ? 2 : 0} breathe={!miss.has("armed")} title={`${d.armed.length} 单 · 更新 ${asOfAge}`} />
					</span>}>{sec("armed",
						d.armed.length === 0 ? <EmptyState title="无待开仓" hint="待开仓队列为空——当前没有待处理事项" /> :
						<div className="tile-scroll"><table className="tile-table"><thead><tr><th>#</th><th>标的</th><th>方向</th><th>建时</th><th>TTL余(h)</th></tr></thead>
							<tbody>{d.armed.map((a: CockpitArmed) => <tr key={a.id}><td>{a.id}</td><td>{a.instId}</td>
								<td title={a.direction}>{cn(DIRECTION_CN, a.direction, "对冲方向未识别")}</td><td>{ts(a.created)}</td><td>{a.ttlHoursLeft.toFixed(1)}</td></tr>)}</tbody></table></div>)}
				</Card>
				<Card title="①② 钱面平台明细"
					extra={<StateDot state={miss.has("money") ? 2 : 0} breathe={!miss.has("money")} title={`${m?.platforms.length ?? 0} 平台 · 更新 ${asOfAge}`} />}>{sec("money", !m ? <p className="dim">该段数据缺</p> : <>
					<table className="tile-table"><thead><tr><th>平台</th><th>承诺</th><th>已入</th><th>停泊</th><th>敞口</th></tr></thead>
						<tbody>{m.platforms.map((p: CockpitPlatformRow) => <tr key={p.platform}><td>{cn(PLATFORM_CN, p.platform, "未识别平台")}</td><td>{p.committed < 0 ? "未设" : p.committed.toFixed(0)}</td>
							<td>{p.platform === "MT5" && miss.has("mt5") ? "数据缺" : p.deposited.toFixed(0)}</td><td>{p.parked.toFixed(0)}</td><td>{p.exposure.toFixed(0)}</td></tr>)}</tbody></table>
					{miss.has("mt5") && <p className="dim">MT5 平台已入段数据缺（mt5Live 读败——合计行为不含 MT5 部分口径，非真 0）</p>}
					<p className="dim fact">近30d 日净：{m.daily.map((v) => v.toFixed(1)).join(" · ")}</p>
				</>)}
				</Card>
			</div>
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
