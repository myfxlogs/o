// QuoteBoard.tsx —— T-0382 行情·价格提醒卡（业主令：「想看到 BTCUSDT/
// TRXUSDT 实时报价，并想预设一个价格，飞书通知我」）+ T-0391 自选报价条
// （业主令 2026-11-06：「订阅某1个symbol的实时报价，这个symbol需要我来
// 定义，要支持搜索；NOX归档总览页就有，我要的是这个效果，前端要适合现有
// 前端」——NOX QuoteStrip/WatchlistModal 语义移植，视觉/组件零拷贝全走
// O 现有 Card/Modal/样式件系）：
//   · 自选报价条段：tiles 显 watchlist 成员的 bid/ask+鲜度（T-0406 业主令
//     去 last；龄服务端派生——前端零判据）；成员序=后端 ord 保存序直渲
//     （排序全后端禁前端 sort）；零成员/成员零行情如实缺态不假零；
//   · 「自选✎」浮层：搜索框子串过滤候选（大小写不敏）→点选加入→↑↓排序
//     +✕删→保存=全量替换即生效；候选域=md_ticks 在采品种（有行情才可选）；
//   · 建提醒表单收口：symbol=可过滤输入（input+datalist——敲子串即筛；
//     业主 11-06 三令「缺少搜索功能」），候选=watchlist 成员优先（成员空
//     则退回全候选——不 fail-closed 卡死建单）；
//   · 报价面=提醒行 symbol 集一单两用——行=fin_price_alert 同 symbol 渲
//     染；预设表单/停用恢复语义同 T-0382（单向告警非条件单——文案不暗示
//     下单；裸态键禁入可见面）。
// 刷新=ev_ticks 信号帧（useLive 事件径零定时器）；写径裸调不入合并窗。
import { useState, type FormEvent } from "react";
import { client, errText } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, Skeleton, StateDot } from "./ui";
import { WatchEditHost } from "./QuoteWatch";
import { GlassSearchShell, HighlightMatch, filterCands } from "./glasssearch";
import type { PriceAlertRow, QuoteWatchRow } from "../../gen/dashboardv1/core_pb";

// 人话字典（纯显示映射非判据——态派生在服务端/迁移 CHECK）。
const DIR_TXT: Record<string, string> = { ">=": "涨到 ≥", "<=": "跌破 ≤" };
const STATE_TXT: Record<string, string> = {
	active: "盯梢中", fired: "已提醒", disabled: "已停用",
};

// 鲜度人话（秒龄→可读串——服务端派生龄直译非判据）。
const ageTxt = (s: bigint) => {
	const n = Number(s);
	if (n < 10) return "刚刚";
	if (n < 60) return `${n} 秒前`;
	if (n < 3600) return `${Math.floor(n / 60)} 分前`;
	return `${(n / 3600).toFixed(1)} 时前`;
};

// 价串（千分位可读、小数保精度——拒科学计数黑话）。
const px = (v: number) =>
	v.toLocaleString("en-US", { maximumFractionDigits: 8 });

export type AlertForm = {
	symbol: string;
	direction: string;
	target: number;
	note: string;
};

// QuoteBoardView 纯渲件（DOM 桩测面——容器喂行+回调；浮层开合=纯 UI 态；
// 浮层件=QuoteWatch.tsx WatchEditHost/WatchEditView+glasssearch 共享壳）。
export function QuoteBoardView(props: {
	rows: PriceAlertRow[];
	symbols: string[];
	watchRows: QuoteWatchRow[];
	watchCands: string[];
	err?: string | null;
	busy?: boolean;
	editOpen?: boolean; // 浮层初态注入（DOM 桩测面——交互开合仍走 useState）
	pickOpen?: boolean; // 建提醒 symbol 候选浮层初态注入（同上）
	onCreate?: (f: AlertForm) => void;
	onToggle?: (id: bigint, state: "active" | "disabled") => void;
	onSaveWatch?: (symbols: string[]) => Promise<unknown>;
}) {
	const [edit, setEdit] = useState(props.editOpen ?? false);
	// 建提醒 symbol 候选浮层（T-0411 交互②消费面：datalist 升玻璃搜索浮层
	// ——输入实时过滤+高亮匹配段；表单 FormData 语义零改动：input 仍
	// name="symbol" 受控可敲，浮层点选回填）。
	const [pick, setPick] = useState(props.pickOpen ?? false);
	const [symbol, setSymbol] = useState("");
	const submit = (e: FormEvent<HTMLFormElement>) => {
		e.preventDefault();
		if (!props.onCreate) return;
		const f = new FormData(e.currentTarget);
		props.onCreate({
			symbol: String(f.get("symbol") ?? ""),
			direction: String(f.get("direction") ?? ">="),
			target: Number(f.get("target") ?? 0),
			note: String(f.get("note") ?? ""),
		});
		e.currentTarget.reset();
		setSymbol("");
	};
	// 建提醒候选=watchlist 成员优先（成员空退回全候选——不 fail-closed 卡死建单）。
	const formSyms = props.watchRows.length > 0
		? props.watchRows.map((r) => r.symbol)
		: props.symbols;
	const pickCands = filterCands(formSyms, symbol);
	return (<>
		<Card className="card-solid" title="行情·价格提醒"
			extra={<StateDot state={0} breathe title="报价行=提醒单品种集——行情帧到即刷新" />}>
			{/* —— T-0391 自选报价条段（成员序后端 ord 直渲；零成员如实缺态） —— */}
			<div style={{ display: "flex", gap: 8, flexWrap: "wrap", alignItems: "center", marginBottom: 10 }}>
				<span className="dim">自选报价</span>
				{props.watchRows.length === 0 ? (
					<span className="dim">未设自选品种——点「自选✎」搜索加入</span>
				) : (
					props.watchRows.map((r) => (
						<span key={r.symbol} className="quote-tile" style={{
							border: "1px solid var(--ring)", borderRadius: 6,
							padding: "3px 10px", display: "inline-flex", gap: 8,
							alignItems: "baseline",
						}}>
							<b>{r.symbol}</b>
							{r.hasQuote ? (<>
								<span className="num u-good" title="买一">{px(r.bid)}</span>
								<span className="num dim">/</span>
								<span className="num u-bad" title="卖一">{px(r.ask)}</span>
								<span className="dim" title="距最新行情写入的时长">{ageTxt(r.quoteAgeS)}</span>
							</>) : (
								<span className="dim">暂无行情</span>
							)}
						</span>
					))
				)}
				{props.onSaveWatch &&
					<button className="btn-ghost" style={{ marginLeft: "auto" }}
						title="自选报价条编辑" onClick={() => setEdit(true)}>自选✎</button>}
			</div>
			{props.rows.length === 0 ? (
				<EmptyState title="暂无价格提醒"
					hint="下方预设品种+目标价——现价命中即飞书通知" />
			) : (
				<div className="tile-scroll"><table className="tile-table"><thead>
					<tr><th>品种</th><th>最新</th><th>买一</th><th>卖一</th><th>报价鲜度</th><th>提醒</th><th>状态</th><th /></tr>
				</thead><tbody>
					{props.rows.map((r) => (
						<tr key={String(r.id)}>
							<td>{r.symbol}</td>
							{r.hasQuote ? (<>
								<td>{px(r.last)}</td>
								<td className="u-good">{px(r.bid)}</td>
								<td className="u-bad">{px(r.ask)}</td>
								<td title="距最新行情写入的时长">{ageTxt(r.quoteAgeS)}</td>
							</>) : (
								<td colSpan={4} className="dim">暂无行情——该品种暂未采集到报价</td>
							)}
							<td title={`目标价 ${px(r.target)}${r.note ? " · " + r.note : ""}`}>
								{DIR_TXT[r.direction] ?? r.direction} {px(r.target)}
							</td>
							<td title={r.firedAt
								? `已于 ${new Date(Number(r.firedAt.seconds) * 1000).toLocaleString("zh-CN", { hour12: false })} 推送`
								: undefined}>{STATE_TXT[r.state] ?? r.state}</td>
							<td>
								{r.state === "active" && props.onToggle &&
									<button className="btn-ghost" disabled={props.busy}
										onClick={() => props.onToggle!(r.id, "disabled")}>停用</button>}
								{r.state === "disabled" && props.onToggle &&
									<button className="btn-ghost" disabled={props.busy}
										onClick={() => props.onToggle!(r.id, "active")}>恢复</button>}
							</td>
						</tr>
					))}
				</tbody></table></div>
			)}
			{props.onCreate && (
				<form onSubmit={submit} style={{ display: "flex", gap: 6, flexWrap: "wrap", marginTop: 10 }}>
					<input name="symbol" className="modal-input"
						required autoComplete="off" style={{ width: 200 }}
						value={symbol}
						onChange={(e) => setSymbol(e.target.value)}
						onFocus={() => setPick(true)}
						placeholder="品种——敲子串筛选"
						title="提醒品种——候选=自选成员（未设自选退回全部在采品种）；聚焦弹玻璃搜索浮层" />
					<select name="direction" className="modal-input" title="命中方向">
						<option value=">=">涨到 ≥</option>
						<option value="<=">跌破 ≤</option>
					</select>
					<input name="target" className="modal-input" type="number" step="any"
						min="0" required placeholder="目标价" style={{ width: 120 }} />
					<input name="note" className="modal-input" placeholder="备注（可空）"
						style={{ width: 140 }} />
					<button type="submit" className="btn-primary" disabled={props.busy}>建提醒</button>
				</form>
			)}
			{props.err && <p className="dim">操作未成：{props.err}</p>}
		</Card>
		{/* T-0411 交互②：建提醒 symbol 候选=玻璃搜索浮层（共享壳 GlassSearchShell
		    ——压暗+blur(8px)+实时过滤+高亮匹配；卡外兄弟节点渲=避开 .card
		    backdrop-filter fixed 包含块阱，同 T-0406 径）。 */}
		{pick && props.onCreate && (
			<GlassSearchShell ariaLabel="提醒品种候选" onClose={() => setPick(false)}>
				<h2>选品种</h2>
				<input className="modal-input" placeholder="搜索品种——敲子串筛选"
					value={symbol} onChange={(e) => setSymbol(e.target.value)}
					autoFocus aria-label="品种过滤" />
				<div className="watchlist-candidates">
					<b>候选（{pickCands.length}）</b>
					<div className="watchlist-candidate-scroll">
						{pickCands.map((s) => (
							<button key={s} className="btn-ghost watchlist-candidate-item"
								onClick={() => { setSymbol(s); setPick(false); }}>
								<HighlightMatch text={s} query={symbol} />＋
							</button>
						))}
						{pickCands.length === 0 &&
							<span className="dim">无匹配候选——候选域=在采行情品种</span>}
					</div>
				</div>
			</GlassSearchShell>
		)}
		{/* T-0406 弹窗出卡：.card backdrop-filter 使卡成 fixed 包含块+叠层阱——
		    overlay 在卡内渲即被卡界截尾、后续平铺卡盖上（保存钮不可点病灶）；
		    浮层作卡外兄弟节点渲，position:fixed 归视口 z:200 置顶。 */}
		{edit && props.onSaveWatch && (
			<WatchEditHost
				members={props.watchRows.map((r) => r.symbol)}
				candidates={props.watchCands}
				onSave={props.onSaveWatch}
				onClose={() => setEdit(false)}
			/>
		)}
	</>);
}

// QuoteBoard 容器件（取数+写径装配——ev_ticks 帧重取；写后真值重读由服务端
// 返行+reload 双径对齐）。
export function QuoteBoard() {
	const live = useLive(["ev_ticks"], () =>
		Promise.all([client.listPriceAlerts({}), client.getQuoteWatchlist({})])
			.then(([alerts, watch]) => ({ alerts, watch })));
	const [busy, setBusy] = useState(false);
	const [err, setErr] = useState<string | null>(null);
	const run = (fn: () => Promise<unknown>) => {
		setBusy(true);
		setErr(null);
		return fn()
			.then(() => live.reload())
			.catch((e: unknown) => {
				setErr(errText(e));
				throw e;
			})
			.finally(() => setBusy(false));
	};
	if (live.loading && !live.data) {
		return <Card className="card-solid" title="行情·价格提醒"><Skeleton rows={3} /></Card>;
	}
	if (!live.data) {
		return (
			<Card className="card-solid" title="行情·价格提醒">
				<p className="dim">该段数据缺（读败{live.err ? `：${live.err}` : ""}）</p>
			</Card>
		);
	}
	return (
		<QuoteBoardView
			rows={live.data.alerts.rows}
			symbols={live.data.alerts.symbols}
			watchRows={live.data.watch.rows}
			watchCands={live.data.watch.symbols}
			err={err}
			busy={busy}
			onCreate={(f) => void run(() => client.createPriceAlert(f))}
			onToggle={(id, state) => void run(() => client.setPriceAlertState({ id, state }))}
			onSaveWatch={(symbols) => run(() => client.setQuoteWatchlist({ symbols }))}
		/>
	);
}
