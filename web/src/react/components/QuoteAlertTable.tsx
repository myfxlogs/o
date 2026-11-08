// QuoteAlertTable.tsx —— 提醒主表+已停用折叠区（自 QuoteBoard.tsx 拆分：
// 表体独立成件防单件超标；T-0382 表基 + T-0442 三件落点）：
//   · 件② 一品种双阈同条：按品种分组 rowSpan 合并品种/报价格——一卡看全
//     双阈（条目减半=分组呈现零改行层）；
//   · 件① 删除=钦定软删：active 行「删除」钮→翻 disabled 留痕（行不物理
//     删，fired 史行无删禁改——审计面）；disabled 行折叠「已停用」区默认
//     收起不占主列表，区行「恢复」钮可逆；
//   · 晋升中/暂无行情显缺不假零（T-0412/T-0399 语义直译）。
import type { PriceAlertRow } from "../../gen/dashboardv1/core_pb";
import { ALERT_DIR_CN, ALERT_STATE_CN } from "./labels";

// 人话字典（纯显示映射非判据——态派生在服务端/迁移 CHECK）。
// T-0441：direction/state 词表归一 labels.ts（未知值人话兜底不裸出）。
const dirTxt = (v: string) => ALERT_DIR_CN[v] ?? "未识别方向";
const stateTxt = (v: string) => ALERT_STATE_CN[v] ?? "未识别态";

// 鲜度人话（秒龄→可读串——服务端派生龄直译非判据；瓦片段同源共用）。
export const ageTxt = (s: bigint) => {
	const n = Number(s);
	if (n < 10) return "刚刚";
	if (n < 60) return `${n} 秒前`;
	if (n < 3600) return `${Math.floor(n / 60)} 分前`;
	return `${(n / 3600).toFixed(1)} 时前`;
};

// 价串（千分位可读、小数保精度——拒科学计数黑话）。
export const px = (v: number) =>
	v.toLocaleString("en-US", { maximumFractionDigits: 8 });

// alertGroups 按品种分组（保首现序=id 降序；组内 >= 上限在前 <= 下限在后
// ——一品种一卡看全双阈，条目减半=分组呈现零改行层）。
type AlertGroup = { symbol: string; items: PriceAlertRow[] };
const alertGroups = (rows: PriceAlertRow[]): AlertGroup[] => {
	const gs: AlertGroup[] = [];
	const idx = new Map<string, AlertGroup>();
	for (const r of rows) {
		let g = idx.get(r.symbol);
		if (!g) {
			g = { symbol: r.symbol, items: [] };
			idx.set(r.symbol, g);
			gs.push(g);
		}
		g.items.push(r);
	}
	for (const g of gs) {
		g.items.sort((a, b) => (a.direction === b.direction ? 0 : a.direction === ">=" ? -1 : 1));
	}
	return gs;
};

// QuoteAlertTable 提醒主表+已停用折叠区纯渲件（DOM 桩测面——行/回调全注入）。
export function QuoteAlertTable(props: {
	rows: PriceAlertRow[];
	pendingSyms?: string[]; // T-0412 未订阅候选（目录符非活——选建单即提名晋升）
	busy?: boolean;
	onToggle?: (id: bigint, state: "active" | "disabled") => void;
}) {
	const pending = new Set(props.pendingSyms ?? []);
	const condTxt = (r: PriceAlertRow) =>
		`${dirTxt(r.direction)} ${px(r.target)}`;
	// 件② 分组：主表=active+fired（fired 史行只读）；disabled→已停用折叠区
	const groups = alertGroups(props.rows.filter((r) => r.state !== "disabled"));
	const disabledRows = props.rows.filter((r) => r.state === "disabled");
	if (groups.length === 0 && disabledRows.length === 0) {
		return null;
	}
	return (<>
		{groups.length > 0 && (
			<div className="tile-scroll"><table className="tile-table"><thead>
				<tr><th>品种</th><th>最新</th><th>买一</th><th>卖一</th><th>报价鲜度</th><th>提醒</th><th>状态</th><th /></tr>
			</thead><tbody>
				{/* 件② 一品种一卡：品种/报价格 rowSpan 合并，条件逐行（fired 只读） */}
				{groups.map((g) => {
					const q = g.items.find((r) => r.hasQuote);
					const anyPend = !q && pending.has(g.symbol);
					return g.items.map((r, i) => (
						<tr key={String(r.id)}>
							{i === 0 && <td rowSpan={g.items.length}>{g.symbol}</td>}
							{i === 0 && (q ? (<>
								<td rowSpan={g.items.length}>{px(q.last)}</td>
								<td rowSpan={g.items.length} className="u-good">{px(q.bid)}</td>
								<td rowSpan={g.items.length} className="u-bad">{px(q.ask)}</td>
								<td rowSpan={g.items.length} title="距最新行情写入的时长">{ageTxt(q.quoteAgeS)}</td>
							</>) : anyPend ? (
								<td colSpan={4} rowSpan={g.items.length} className="dim">晋升中——订阅已提名，报价接通后可评估</td>
							) : (
								<td colSpan={4} rowSpan={g.items.length} className="dim">暂无行情——该品种暂未采集到报价</td>
							))}
							<td title={r.note ? `备注 ${r.note}` : undefined}>{condTxt(r)}</td>
							<td title={r.firedAt
								? `已于 ${new Date(Number(r.firedAt.seconds) * 1000).toLocaleString("zh-CN", { hour12: false })} 推送`
								: undefined}>{stateTxt(r.state)}</td>
							<td>
								{/* 件① 删除=软删 disabled 留痕（fired 史行无删——禁删禁改） */}
								{r.state === "active" && props.onToggle &&
									<button className="btn-ghost" disabled={props.busy}
										title="停用留痕——行入「已停用」折叠区，可恢复"
										onClick={() => props.onToggle!(r.id, "disabled")}>删除</button>}
							</td>
						</tr>
					));
				})}
			</tbody></table></div>
		)}
		{/* 件① 已停用折叠区——软删留痕行默认收起不占主列表，恢复可逆 */}
		{disabledRows.length > 0 && (
			<details style={{ marginTop: 8 }}>
				<summary className="dim">已停用 {disabledRows.length} 条（软删留痕——点开可恢复）</summary>
				<div className="tile-scroll"><table className="tile-table"><tbody>
					{disabledRows.map((r) => (
						<tr key={String(r.id)}>
							<td>{r.symbol}</td>
							<td>{condTxt(r)}</td>
							<td>{stateTxt(r.state)}</td>
							<td>{props.onToggle &&
								<button className="btn-ghost" disabled={props.busy}
									onClick={() => props.onToggle!(r.id, "active")}>恢复</button>}</td>
						</tr>
					))}
				</tbody></table></div>
			</details>
		)}
	</>);
}
