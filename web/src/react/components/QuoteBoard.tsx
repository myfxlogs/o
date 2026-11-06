// QuoteBoard.tsx —— T-0382 行情·价格提醒卡（业主令：「想看到 BTCUSDT/
// TRXUSDT 实时报价，并想预设一个价格，飞书通知我」）：
//   · 报价面=提醒行 symbol 集一单两用——行=fin_price_alert 同 symbol 渲
//     染；最新/买一/卖一+报价鲜度（龄由服务端派生——前端零判据）；无行情
//     行=「暂无行情」显缺不假零；
//   · 预设表单：品种（在采枚举下拉）+条件（涨到 ≥/跌破 ≤）+目标价+备注；
//   · 停用/恢复=active↔disabled 翻位；已提醒=终态标无操作；
//   · 单向告警非条件单——文案不暗示下单；人话面照 T-0371 规（裸态键
//     active/fired/disabled 禁入可见面）。
// 刷新=ev_ticks 信号帧（useLive 事件径零定时器）；写径裸调不入合并窗。
import { useState, type FormEvent } from "react";
import { client, errText } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, Skeleton, StateDot } from "./ui";
import type { PriceAlertRow } from "../../gen/dashboardv1/core_pb";

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

// QuoteBoardView 纯渲件（DOM 桩测面——零取数零副作用；容器喂行+回调）。
export function QuoteBoardView(props: {
	rows: PriceAlertRow[];
	symbols: string[];
	err?: string | null;
	busy?: boolean;
	onCreate?: (f: AlertForm) => void;
	onToggle?: (id: bigint, state: "active" | "disabled") => void;
}) {
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
	};
	return (
		<Card title="行情·价格提醒"
			extra={<StateDot state={0} breathe title="报价行=提醒单品种集——行情帧到即刷新" />}>
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
								<td>{px(r.bid)}</td>
								<td>{px(r.ask)}</td>
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
					<select name="symbol" className="modal-input" required
						title="提醒品种——当前在采集的行情品种">
						{props.symbols.map((s) => <option key={s} value={s}>{s}</option>)}
					</select>
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
	);
}

// QuoteBoard 容器件（取数+写径装配——ev_ticks 帧重取；写后真值重读由服务端
// 返行+reload 双径对齐）。
export function QuoteBoard() {
	const live = useLive(["ev_ticks"], () => client.listPriceAlerts({}));
	const [busy, setBusy] = useState(false);
	const [err, setErr] = useState<string | null>(null);
	const run = (fn: () => Promise<unknown>) => {
		setBusy(true);
		setErr(null);
		fn()
			.then(() => live.reload())
			.catch((e: unknown) => setErr(errText(e)))
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
			rows={live.data.rows}
			symbols={live.data.symbols}
			err={err}
			busy={busy}
			onCreate={(f) => run(() => client.createPriceAlert(f))}
			onToggle={(id, state) => run(() => client.setPriceAlertState({ id, state }))}
		/>
	);
}
