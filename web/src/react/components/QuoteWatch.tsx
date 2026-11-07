// QuoteWatch.tsx —— T-0391 自选报价条编辑浮层（NOX WatchlistModal 语义移植，
// 视觉/组件零拷贝全走 O 现有 .modal-*+watchlist-* 既有 CSS 件系）：
//   搜索框子串过滤候选（大小写不敏——搜索=展示面过滤非业务判据）→点选加入
//   →↑↓排序+✕删→保存=SetQuoteWatchlist 全量替换即生效（顺序即 ord——排序
//   全后端禁前端 sort）；候选域=md_ticks 在采品种（有行情才可选，不开任意
//   symbol 输入）；已选剔出候选；保存败留层显错不吞，成则关层。
//   T-0442 件③ 源约束（Nox WatchlistModal chip 形制引进）：成员行携 venue
//   chip 集（srcSets=实采∪实扫派生零硬编码词表）——chip 亮=含该源/暗=不含，
//   空集=「全部源」（最新跨源现语义零破坏）；候选行携源集文本预览。
import { useState } from "react";
import { errText } from "../api";

// 成员形制（源约束随行——保存全量替换携 venues）
export type WatchItem = { symbol: string; venues: string[] };

// WatchEditView 自选浮层纯渲件（DOM 桩测面——items/candidates/query 全受控，
// 过滤在渲染内完成：候选=候选域−已选，子串过滤大小写不敏）。
export function WatchEditView(props: {
	items: WatchItem[];
	candidates: string[];
	srcSets?: Record<string, string[]>; // T-0442 各符可选源集（实采∪实扫派生）
	pending?: Set<string>; // T-0412 未订阅候选（目录符非活——选入即提名晋升）
	query: string;
	busy?: boolean;
	err?: string | null;
	onQuery?: (q: string) => void;
	onAdd?: (symbol: string) => void;
	onMove?: (i: number, d: -1 | 1) => void;
	onRemove?: (i: number) => void;
	onToggleVenue?: (i: number, venue: string) => void; // T-0442 chip 切换含/不含
	onSave?: () => void;
	onCancel?: () => void;
}) {
	const picked = new Set(props.items.map((it) => it.symbol));
	const q = props.query.trim().toLowerCase();
	const cands = props.candidates
		.filter((s) => !picked.has(s))
		.filter((s) => !q || s.toLowerCase().includes(q));
	return (
		<div className="modal-overlay" onClick={props.onCancel}>
			<div className="modal" role="dialog" aria-label="自选报价条编辑"
				onClick={(e) => e.stopPropagation()}>
				<h2>自选报价条编辑</h2>
				{props.err && <p className="dim">操作未成：{props.err}</p>}
				<input className="modal-input" placeholder="搜索品种——敲子串筛选"
					value={props.query} onChange={(e) => props.onQuery?.(e.target.value)} />
				<div className="watchlist-current">
					<b>已选（{props.items.length}）</b>
					<ul className="watchlist-order watchlist-candidate-scroll">
						{props.items.map((it, i) => (
							<li key={it.symbol}>
								<span className="watchlist-sym">{it.symbol}</span>
								<span className="watchlist-chips">
									{(props.srcSets?.[it.symbol] ?? []).map((v) => (
										<span key={v}
											className={"chip" + (it.venues.includes(v) ? " chip-active" : "")}
											title={it.venues.includes(v)
												? `限定源 ${v}——点选取消（空集=全部源）`
												: `点选限定源 ${v}`}
											onClick={() => props.onToggleVenue?.(i, v)}>{v}</span>
									))}
									{(props.srcSets?.[it.symbol]?.length ?? 0) === 0 &&
										<span className="dim">源集待实采</span>}
								</span>
								<span className="dim" style={{ flex: "none" }}>
									{it.venues.length === 0 ? "全部源" : it.venues.join("/")}
								</span>
								<span style={{ flex: 1 }} />
								<button className="btn-ghost" onClick={() => props.onMove?.(i, -1)}>↑</button>
								<button className="btn-ghost" onClick={() => props.onMove?.(i, 1)}>↓</button>
								<button className="btn-ghost" onClick={() => props.onRemove?.(i)}>✕</button>
							</li>
						))}
						{props.items.length === 0 &&
							<li className="dim">未选品种——下方候选点选加入</li>}
					</ul>
				</div>
				<div className="watchlist-candidates">
					<b>候选</b>
					<div className="watchlist-candidate-scroll">
						{cands.map((s) => (
							<button key={s} className="btn-ghost watchlist-candidate-item"
								title={props.pending?.has(s) ? "未订阅目录符——选入即提名订阅晋升，报价稍后接通" : undefined}
								onClick={() => props.onAdd?.(s)}>
								{s}{props.pending?.has(s) &&
									<span className="dim">·晋升中</span>}
								{(props.srcSets?.[s]?.length ?? 0) > 0 &&
									<span className="dim"> · 源 {props.srcSets![s].join(" ")}</span>}＋
							</button>
						))}
						{cands.length === 0 &&
							<span className="dim">无匹配候选——候选域=在采行情∪目录品种</span>}
					</div>
				</div>
				<div className="modal-actions">
					<button className="btn-primary" disabled={props.busy}
						onClick={props.onSave}>保存（全量替换生效）</button>
					<button className="btn-ghost" onClick={props.onCancel}>取消</button>
				</div>
			</div>
		</div>
	);
}

// WatchEditHost 浮层容器（items/query 编辑态自持——初始=后端成员序快照携
// 源约束；chip 切换=含/不含翻转；保存败留层显错不吞，成则关层）。
export function WatchEditHost(props: {
	members: WatchItem[];
	candidates: string[];
	srcSets?: Record<string, string[]>;
	pending?: Set<string>;
	onSave: (items: WatchItem[]) => Promise<unknown>;
	onClose: () => void;
}) {
	const [items, setItems] = useState<WatchItem[]>(props.members);
	const [query, setQuery] = useState("");
	const [err, setErr] = useState<string | null>(null);
	const [busy, setBusy] = useState(false);
	const move = (i: number, d: -1 | 1) =>
		setItems((l) => {
			const n = [...l];
			const j = i + d;
			if (j < 0 || j >= n.length) return n;
			[n[i], n[j]] = [n[j], n[i]];
			return n;
		});
	const save = () => {
		setBusy(true);
		setErr(null);
		props.onSave(items)
			.then(props.onClose)
			.catch((e: unknown) => setErr(errText(e)))
			.finally(() => setBusy(false));
	};
	return (
		<WatchEditView
			items={items} candidates={props.candidates} srcSets={props.srcSets}
			pending={props.pending} query={query}
			busy={busy} err={err}
			onQuery={setQuery}
			onAdd={(s) => setItems((l) => (l.some((it) => it.symbol === s) ? l : [...l, { symbol: s, venues: [] }]))}
			onMove={move}
			onRemove={(i) => setItems((l) => l.filter((_, j) => j !== i))}
			onToggleVenue={(i, v) =>
				setItems((l) => {
					const n = [...l];
					const cur = n[i].venues.includes(v)
						? n[i].venues.filter((x) => x !== v)
						: [...n[i].venues, v];
					n[i] = { ...n[i], venues: cur };
					return n;
				})}
			onSave={save}
			onCancel={props.onClose}
		/>
	);
}
