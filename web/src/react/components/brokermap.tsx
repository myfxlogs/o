// brokermap.tsx —— T-0432 向导步④符号映射面（MT5 专有——映射=确认件非
// 推断件：官名清单→候选机造下拉（score 排序服务端出，只省翻页不做决定）
// →席位挑→硬验并落行（拒收即回零落行，挑定可改再提）；逐只挑节奏——落行
// 成即清选续挑下只。回调容器装配 RPC，本件纯选转零直连。
// O-35250 兜底面：官名下拉外自由文本（形闸 [a-zA-Z0-9_-]——未在册官名
// 手输同径）；venue_symbol 手输+目录浏览（候选机外/候选空死路——目录
// 原文如实给席位挑，禁替席决定；手输与下拉同走 SubmitSymbolMapping 硬验）。
import { useEffect, useId, useState, type ChangeEvent } from "react";
import { errText } from "../api";
import type { SymbolCandidate } from "../../gen/dashboardv1/core_pb";

export type MappingQuery = { venue: string; login: string; canonical: string };
export type MappingQueryResult = { rows: SymbolCandidate[]; scanned: number; symbols?: string[] };
export type MappingSubmit = { venue: string; login: string; canonical: string; venueSymbol: string; note: string };

// canonGate 官名自由文本形闸（O-35250③——同形闸，禁散放行）。
const canonGate = /^[a-zA-Z0-9_-]*$/;
// canonDebounce 手输官名防抖拍（逐键 RPC=会话拨号风暴——350ms 合流）。
const canonDebounce = 350;

export function SymbolMappingStep(props: {
	venue: string;
	login: string;
	loadUniverse?: () => Promise<string[]>;
	onMappingQuery?: (f: MappingQuery) => Promise<MappingQueryResult>;
	onSubmitMapping?: (f: MappingSubmit) => Promise<void>;
	// 测试注面（同 brokerboard initialStep/initialDraft 惯例——domtest 静态渲）。
	initialCanonical?: string;
	initialRows?: SymbolCandidate[];
	initialSymbols?: string[];
	initialScanned?: number;
	initialBrowse?: boolean;
}) {
	const [universe, setUniverse] = useState<string[]>([]);
	const [canonDraft, setCanonDraft] = useState(props.initialCanonical ?? ""); // 官名输入框值（下拉建议+自由文本同件）
	const [canonical, setCanonical] = useState(props.initialCanonical ?? "");   // 已发查的官名（提交判据——未查成不入）
	const [rows, setRows] = useState<SymbolCandidate[]>(props.initialRows ?? []);
	const [symbols, setSymbols] = useState<string[]>(props.initialSymbols ?? []); // 目录原文（rankCandidates 前——浏览面数据源）
	const [scanned, setScanned] = useState(props.initialScanned ?? 0);
	const [picked, setPicked] = useState("");           // venue_symbol 单一挑定源（下拉/手输/浏览同写）
	const [browse, setBrowse] = useState(props.initialBrowse ?? false);
	const [browseQ, setBrowseQ] = useState("");
	const [note, setNote] = useState("");
	const [busy, setBusy] = useState(false);
	const [err, setErr] = useState<string | null>(null);
	const [ok, setOk] = useState<string | null>(null);
	const canonListId = useId();
	// 官名清单一次载（缺=空集如实——映射操作前置 okx.symbols 在册）。
	useEffect(() => {
		if (!props.loadUniverse) return;
		props.loadUniverse().then(setUniverse).catch(() => setUniverse([]));
	}, [props.loadUniverse]);
	// 候选机造查询（官名定稿即查——返回者皆 venue 实认存在面）。
	const query = (c: string) => {
		setCanonical(c);
		setPicked("");
		setRows([]);
		setSymbols([]);
		setErr(null);
		setOk(null);
		if (!c || !props.onMappingQuery) return;
		setBusy(true);
		props.onMappingQuery({ venue: props.venue, login: props.login, canonical: c })
			.then((r) => { setRows(r.rows); setScanned(r.scanned); setSymbols(r.symbols ?? []); })
			.catch((er) => setErr(errText(er)))
			.finally(() => setBusy(false));
	};
	// 官名手输防抖查（datalist 建议选中/自由文本同径——形闸外不发 RPC）。
	useEffect(() => {
		const v = canonDraft.trim();
		if (v === canonical) return;
		if (v === "") {
			setCanonical(""); setRows([]); setSymbols([]); setScanned(0);
			setPicked(""); setErr(null); setOk(null);
			return;
		}
		if (!canonGate.test(v)) {
			setErr("官名形闸 [a-zA-Z0-9_-]——非法字符不查不提");
			return;
		}
		const t = setTimeout(() => query(v), canonDebounce);
		return () => clearTimeout(t);
	}, [canonDraft]); // eslint-disable-line react-hooks/exhaustive-deps——query 闭包只吃 props/setState
	const pick = (e: ChangeEvent<HTMLInputElement>) => {
		setPicked(e.target.value);
		setOk(null);
	};
	const pickSymbol = (s: string) => {
		setPicked(s);
		setBrowseQ("");
		setOk(null);
	};
	const submit = async () => {
		if (!props.onSubmitMapping || busy || !canonical || !picked) return;
		setBusy(true);
		setErr(null);
		setOk(null);
		try {
			await props.onSubmitMapping({ venue: props.venue, login: props.login, canonical, venueSymbol: picked, note });
			setOk(`映射落行：${canonical} → ${picked}（硬验过——下只官名续挑）`);
			setCanonDraft("");
			setCanonical("");
			setPicked("");
			setRows([]);
			setSymbols([]);
			setScanned(0);
		} catch (er) {
			setErr(errText(er)); // 硬验拒收即回——零落行，挑定可改再提
		} finally {
			setBusy(false);
		}
	};
	const browseHits = symbols.filter((s) =>
		browseQ === "" || s.toLowerCase().includes(browseQ.toLowerCase()));
	return (
		<div>
			<div style={{ display: "flex", gap: 6, flexWrap: "wrap", alignItems: "center" }}>
				<input className="modal-input" list={canonListId} value={canonDraft}
					onChange={(e) => { setCanonDraft(e.target.value); setPicked(""); setOk(null); }}
					title="官名——内部策略/仓位/台账唯一用名（下拉建议在册册；不在册=手输同形闸）"
					placeholder="官名（如 BTC-USDT-SWAP）" style={{ width: 220 }} />
				<datalist id={canonListId}>
					{universe.map((s) => <option key={s} value={s} />)}
				</datalist>
				<span className="ink-3">映射=确认件非推断件——机器只列候选+硬验，真身席位挑（逐只挑）</span>
			</div>
			{rows.length > 0 && (
				<table className="rows" style={{ maxWidth: 620, marginTop: 6 }}>
					<thead><tr><th></th><th>候选方言名</th><th>合约面值</th><th>最小数量</th><th>最小步进</th><th>价格位数</th><th>买卖价差</th><th>匹配分</th></tr></thead>
					<tbody>
						{rows.map((c) => (
							<tr key={c.venueSymbol}>
								<td><input type="radio" name="venueSymbol" value={c.venueSymbol}
									checked={picked === c.venueSymbol} onChange={pick} /></td>
								<td className="mono">{c.venueSymbol}</td>
								<td className="num mono">{String(c.contractSize)}</td>
								<td className="num mono">{String(c.minSz)}</td>
								<td className="num mono">{String(c.lotStep)}</td>
								<td className="num mono">{String(c.digits)}</td>
								<td className="num mono">{String(c.spread)}</td>
								<td className="num mono">{String(c.score)}</td>
								</tr>
						))}
					</tbody>
				</table>
			)}
			{canonical && !busy && rows.length === 0 && (
				<p className="ink-3">候选空（目录 {scanned} 符枚举——base 命中零行，如实；手输或目录浏览挑 venue 符号）</p>
			)}
			{canonical && (
				<div style={{ display: "flex", gap: 6, marginTop: 6, alignItems: "center", flexWrap: "wrap" }}>
					<input className="modal-input" value={picked} onChange={pick}
						autoComplete="off" placeholder="venue 符号手输（候选机外亦可——硬验同闸）" style={{ width: 220 }} />
					<button type="button" className="btn-ghost" onClick={() => setBrowse((b) => !b)}>
						{browse ? "收目录" : "目录浏览"}
					</button>
					<span className="ink-3">目录 {scanned} 符原文——机造清单如实给席位挑</span>
				</div>
			)}
			{canonical && browse && (
				<div style={{ marginTop: 4, maxWidth: 620 }}>
					<input className="modal-input" value={browseQ} autoComplete="off"
						onChange={(e) => setBrowseQ(e.target.value)}
						placeholder="搜目录符（子串——如 BTC/GOLD/#）" style={{ width: 260 }} />
					<div className="ink-3" style={{ margin: "2px 0" }}>
						命中 {browseHits.length}/{symbols.length} 符{browseHits.length > 100 ? "——显示前 100" : ""}
					</div>
					<div style={{ maxHeight: 180, overflowY: "auto", border: "1px solid var(--line, #333)", borderRadius: 4 }}>
						{browseHits.slice(0, 100).map((s) => (
							<button key={s} type="button" className="btn-ghost"
								style={{ display: "block", width: "100%", textAlign: "left", fontFamily: "monospace" }}
								onClick={() => pickSymbol(s)}>{s}</button>
						))}
					</div>
				</div>
			)}
			<div style={{ display: "flex", gap: 6, marginTop: 6, alignItems: "center" }}>
				<input className="modal-input" value={note} onChange={(e) => setNote(e.target.value)}
					autoComplete="off" placeholder="映射备注（可空）" style={{ width: 200 }} />
				<button type="button" className="btn-primary" disabled={busy || !canonical || !picked}
					onClick={submit}>硬验并落行</button>
				{scanned > 0 && <span className="ink-3">目录枚举 {scanned} 符</span>}
			</div>
			{ok && <p className="ink-3">{ok}</p>}
			{err && <p className="dim">操作未成：{err}</p>}
		</div>
	);
}
