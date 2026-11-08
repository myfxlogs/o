// brokermap.tsx —— T-0432 向导步④符号映射面（MT5 专有——映射=确认件非
// 推断件：官名清单→候选机造下拉（score 排序服务端出，只省翻页不做决定）
// →席位挑→硬验并落行（拒收即回零落行，挑定可改再提）；逐只挑节奏——落行
// 成即清选续挑下只。回调容器装配 RPC，本件纯选转零直连。
import { useEffect, useState, type ChangeEvent } from "react";
import { errText } from "../api";
import type { SymbolCandidate } from "../../gen/dashboardv1/core_pb";

export type MappingQuery = { venue: string; login: string; canonical: string };
export type MappingSubmit = { venue: string; login: string; canonical: string; venueSymbol: string; note: string };

export function SymbolMappingStep(props: {
	venue: string;
	login: string;
	loadUniverse?: () => Promise<string[]>;
	onMappingQuery?: (f: MappingQuery) => Promise<{ rows: SymbolCandidate[]; scanned: number }>;
	onSubmitMapping?: (f: MappingSubmit) => Promise<void>;
}) {
	const [universe, setUniverse] = useState<string[]>([]);
	const [canonical, setCanonical] = useState("");
	const [rows, setRows] = useState<SymbolCandidate[]>([]);
	const [scanned, setScanned] = useState(0);
	const [picked, setPicked] = useState("");
	const [note, setNote] = useState("");
	const [busy, setBusy] = useState(false);
	const [err, setErr] = useState<string | null>(null);
	const [ok, setOk] = useState<string | null>(null);
	// 官名清单一次载（缺=空集如实——映射操作前置 okx.symbols 在册）。
	useEffect(() => {
		if (!props.loadUniverse) return;
		props.loadUniverse().then(setUniverse).catch(() => setUniverse([]));
	}, [props.loadUniverse]);
	// 候选机造查询（官名选变即查——返回者皆 venue 实认存在面）。
	const query = (c: string) => {
		setCanonical(c);
		setPicked("");
		setRows([]);
		setErr(null);
		setOk(null);
		if (!c || !props.onMappingQuery) return;
		setBusy(true);
		props.onMappingQuery({ venue: props.venue, login: props.login, canonical: c })
			.then((r) => { setRows(r.rows); setScanned(r.scanned); })
			.catch((er) => setErr(errText(er)))
			.finally(() => setBusy(false));
	};
	const pick = (e: ChangeEvent<HTMLInputElement>) => {
		setPicked(e.target.value);
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
			setCanonical("");
			setPicked("");
			setRows([]);
		} catch (er) {
			setErr(errText(er)); // 硬验拒收即回——零落行，挑定可改再提
		} finally {
			setBusy(false);
		}
	};
	return (
		<div>
			<div style={{ display: "flex", gap: 6, flexWrap: "wrap", alignItems: "center" }}>
				<select className="modal-input" value={canonical} title="官名——内部策略/仓位/台账唯一用名"
					onChange={(e) => query(e.target.value)}>
					<option value="">—选官名—</option>
					{universe.map((s) => <option key={s} value={s}>{s}</option>)}
				</select>
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
				<p className="ink-3">候选空（目录 {scanned} 符枚举——base 命中零行，如实；换官名或核 broker 目录）</p>
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
