// QuoteBoard.tsx —— T-0382 行情·价格提醒卡（业主令：「想看到 BTCUSDT/
// TRXUSDT 实时报价，并想预设一个价格，飞书通知我」）+ T-0391 自选报价条
// （NOX QuoteStrip/WatchlistModal 语义移植，视觉/组件零拷贝全走 O 现有
// Card/Modal/样式件系）+ T-0442 行情卡三件（业主令 2026-11-07）：
//   · 件① 提醒删除=钦定软删——表体见 QuoteAlertTable.tsx（disabled 折叠
//     留痕恢复可逆，fired 史行无删禁改）；
//   · 件② 一品种双阈同条——建单表单上限（涨到≥）/下限（跌破≤）各可空
//     （双空=禁建——前端+服务端双闸），主表按品种分组一卡看全双阈；
//   · 件③ 自选报价源选择：自选瓦片显源约束+报价来源 venue（选源后只显
//     所选源价——服务端过滤实证面）；「自选✎」浮层 venue chip 集（Nox
//     形制引进——候选源=实采∪实扫派生，空=全部源）；
//   · 自选报价条段：tiles 显 watchlist 成员的 bid/ask+鲜度（T-0406 业主令
//     去 last；龄服务端派生——前端零判据）；成员序=后端 ord 保存序直渲
//     （排序全后端禁前端 sort）；零成员/成员零行情如实缺态不假零；
//   · 建提醒表单收口：symbol=可过滤输入（input+datalist——敲子串即筛；
//     业主 11-06 三令「缺少搜索功能」），候选=watchlist 成员优先（成员空
//     则退回全候选——不 fail-closed 卡死建单）；
//   · 报价面=提醒行 symbol 集一单两用；单向告警非条件单——文案不暗示
//     下单；裸态键禁入可见面。
// 刷新=ev_ticks 信号帧（useLive 事件径零定时器）；写径裸调不入合并窗。
import { useState, type FormEvent } from "react";
import { client, errText } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, Skeleton, StateDot } from "./ui";
import { QuoteAlertTable, ageTxt, px } from "./QuoteAlertTable";
import { WatchEditHost, type WatchItem } from "./QuoteWatch";
import type {
	PriceAlertRow, QuoteSrcSet, QuoteWatchRow,
} from "../../gen/dashboardv1/core_pb";

// AlertForm 建单载荷（T-0442 件②——上限/下限各可空双空禁建；undefined=未填）。
export type AlertForm = {
	symbol: string;
	upper?: number;
	lower?: number;
	note: string;
};

// QuoteBoardView 纯渲件（DOM 桩测面——容器喂行+回调；浮层开合=纯 UI 态；
// 浮层件=QuoteWatch.tsx WatchEditHost/WatchEditView；表体件=QuoteAlertTable）。
export function QuoteBoardView(props: {
	rows: PriceAlertRow[];
	symbols: string[];
	pendingSyms?: string[]; // T-0412 未订阅候选（目录符非活——选建单即提名晋升）
	watchRows: QuoteWatchRow[];
	watchCands: string[];
	srcSets?: QuoteSrcSet[]; // T-0442 各符可选源集（浮层 chip 候选=实采派生）
	err?: string | null;
	busy?: boolean;
	editOpen?: boolean; // 浮层初态注入（DOM 桩测面——交互开合仍走 useState）
	onCreate?: (f: AlertForm) => void;
	onToggle?: (id: bigint, state: "active" | "disabled") => void;
	onSaveWatch?: (items: WatchItem[]) => Promise<unknown>;
}) {
	const [edit, setEdit] = useState(props.editOpen ?? false);
	const [formErr, setFormErr] = useState<string | null>(null);
	const pending = new Set(props.pendingSyms ?? []);
	const submit = (e: FormEvent<HTMLFormElement>) => {
		e.preventDefault();
		setFormErr(null);
		if (!props.onCreate) return;
		const f = new FormData(e.currentTarget);
		const sym = String(f.get("symbol") ?? "").trim();
		const uv = String(f.get("upper") ?? "").trim();
		const lv = String(f.get("lower") ?? "").trim();
		// 件② 双空=禁建（前端闸——服务端 400 同闸不放松）
		if (uv === "" && lv === "") {
			setFormErr("上限/下限至少填一——双空禁建");
			return;
		}
		props.onCreate({
			symbol: sym,
			upper: uv === "" ? undefined : Number(uv),
			lower: lv === "" ? undefined : Number(lv),
			note: String(f.get("note") ?? ""),
		});
		e.currentTarget.reset();
	};
	// 建提醒候选=watchlist 成员优先（成员空退回全候选——不 fail-closed 卡死建单）。
	const formSyms = props.watchRows.length > 0
		? props.watchRows.map((r) => r.symbol)
		: props.symbols;
	return (<>
		<Card title="行情·价格提醒"
			extra={<StateDot state={0} breathe title="报价行=提醒单品种集——行情帧到即刷新" />}>
			{/* —— T-0391 自选报价条段（成员序后端 ord 直渲；零成员如实缺态；
			    T-0442 件③：瓦片携源约束标+报价来源 venue 实证） —— */}
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
							{r.venues.length > 0 &&
								<span className="dim" title="源约束——只取所选源报价">{r.venues.join("/")}</span>}
							{r.hasQuote ? (<>
								<span className="num u-good" title="买一">{px(r.bid)}</span>
								<span className="num dim">/</span>
								<span className="num u-bad" title="卖一">{px(r.ask)}</span>
								{r.quoteVenue &&
									<span className="dim" title="报价来源 venue（空约束=最新跨源）">@{r.quoteVenue}</span>}
								<span className="dim" title="距最新行情写入的时长">{ageTxt(r.quoteAgeS)}</span>
							</>) : r.pending || pending.has(r.symbol) ? (
								<span className="dim" title="订阅晋升已提名——报价接通中">晋升中…</span>
							) : (
								<span className="dim">暂无行情</span>
							)}
						</span>
					))
				)}
				{props.onSaveWatch &&
					<button className="btn-ghost" style={{ marginLeft: "auto" }}
						title="自选报价条编辑——可限报价来源（broker/cex）" onClick={() => setEdit(true)}>自选✎</button>}
			</div>
			{props.rows.length === 0 ? (
				<EmptyState title="暂无价格提醒"
					hint="下方预设品种+上限/下限——现价命中即飞书通知" />
			) : (
				<QuoteAlertTable rows={props.rows} pendingSyms={props.pendingSyms}
					busy={props.busy} onToggle={props.onToggle} />
			)}
			{props.onCreate && (
				<form onSubmit={submit} style={{ display: "flex", gap: 6, flexWrap: "wrap", marginTop: 10 }}>
					<input name="symbol" className="modal-input" list="alert-symbols"
						required autoComplete="off" style={{ width: 190 }}
						placeholder="品种——敲子串筛选"
						title="提醒品种——候选=自选成员（未设自选退回全部在采品种）" />
					<datalist id="alert-symbols">
						{formSyms.map((s) => <option key={s} value={s}
							label={pending.has(s) ? `${s} · 晋升中（选建即提名订阅）` : undefined} />)}
					</datalist>
					{/* 件② 双阈形制——上限/下限各可空（双空禁建前端闸+服务端闸） */}
					<input name="upper" className="modal-input" type="number" step="any"
						min="0" placeholder="上限（涨到≥）" style={{ width: 130 }}
						title="涨到该价提醒——可空" />
					<input name="lower" className="modal-input" type="number" step="any"
						min="0" placeholder="下限（跌破≤）" style={{ width: 130 }}
						title="跌破该价提醒——可空" />
					<input name="note" className="modal-input" placeholder="备注（可空）"
						style={{ width: 130 }} />
					<button type="submit" className="btn-primary" disabled={props.busy}>建提醒</button>
				</form>
			)}
			{formErr && <p className="dim">建单未成：{formErr}</p>}
			{props.err && <p className="dim">操作未成：{props.err}</p>}
		</Card>
		{/* T-0406 弹窗出卡：.card backdrop-filter 使卡成 fixed 包含块+叠层阱——
		    overlay 在卡内渲即被卡界截尾、后续平铺卡盖上（保存钮不可点病灶）；
		    浮层作卡外兄弟节点渲，position:fixed 归视口 z:200 置顶。 */}
		{edit && props.onSaveWatch && (
			<WatchEditHost
				members={props.watchRows.map((r) => ({ symbol: r.symbol, venues: [...r.venues] }))}
				candidates={props.watchCands}
				srcSets={Object.fromEntries((props.srcSets ?? []).map((s) => [s.symbol, s.venues]))}
				pending={pending}
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
		return <Card title="行情·价格提醒"><Skeleton rows={3} /></Card>;
	}
	if (!live.data) {
		return (
			<Card title="行情·价格提醒">
				<p className="dim">该段数据缺（读败{live.err ? `：${live.err}` : ""}）</p>
			</Card>
		);
	}
	return (
		<QuoteBoardView
			rows={live.data.alerts.rows}
			symbols={live.data.alerts.symbols}
			pendingSyms={[...live.data.alerts.pendingSymbols, ...live.data.watch.pendingSymbols]}
			watchRows={live.data.watch.rows}
			watchCands={live.data.watch.symbols}
			srcSets={live.data.watch.srcSets}
			err={err}
			busy={busy}
			onCreate={(f) => void run(() => client.createPriceAlert({
				symbol: f.symbol,
				hasUpper: f.upper !== undefined,
				upperTarget: f.upper ?? 0,
				hasLower: f.lower !== undefined,
				lowerTarget: f.lower ?? 0,
				note: f.note,
			}))}
			onToggle={(id, state) => void run(() => client.setPriceAlertState({ id, state }))}
			onSaveWatch={(items) => run(() => client.setQuoteWatchlist({ items }))}
		/>
	);
}
