// moduledeck.tsx —— T-0411 液态玻璃座舱皮肤·模块 deck（交互①④载体）：
//   · 等大模块网格（.module-grid）——交互④钦定形：不等大格「最近格位」
//     无唯一定义，deck 只收等大格模块；
//   · 交互①：模块点击→FLIP 原位放大为详情视图（覆盖宿主，transform
//     first/last/invert/play，320ms 临界阻尼贝塞尔轻快不超调）；其余模块
//     缩为底部玻璃缩略条（.deck-strip，上限 8 枚，溢出横向滚动；缩略图=
//     模块名+状态点——状态点=卡头状态点 DOM 镜像（MutationObserver 只读
//     零判据），不渲实时数）；点击缩略图切换主视图；关闭（✕/Esc/收起）
//     沿原路收回（详情矩形→原格位逆向 FLIP）；
//   · 交互④：模块拖拽重排——控制面在 usedeckdrag.ts（分阈/随针/碰撞
//     让位/吸附），本件只携网格与让位 FLIP 回放；
//   · 模块顺序持久化=localStorage（票面钉：跨端同步不属本票）——存全
//     排列，验败（长度/成员/重复不符）静默弃用默认序（fail-safe 不假持久）。
// 数据语义/交互逻辑零改动：deck 只包壳不触卡内；网格展开期格位保挂
// （visibility 隐非卸载——卡状态/流订阅活，收回复原零重取），代价=展开
// 期卡件双挂载（QuoteBoard 每帧双取 RPC×2）——收口注记在案。
import {
	useEffect,
	useLayoutEffect,
	useRef,
	useState,
	type ReactNode,
} from "react";
import {
	FLIP_EASE, FLIP_MS, YIELD_MS, useDeckDrag,
} from "./usedeckdrag";

export type DeckModule = { id: string; title: string; el: ReactNode };

// 顺序持久化（localStorage 全排列；验败弃用默认序——不假持久）。
function loadOrder(key: string, ids: string[]): string[] {
	try {
		const raw = typeof localStorage === "undefined" ? null : localStorage.getItem(key);
		if (!raw) return ids;
		const arr: unknown = JSON.parse(raw);
		if (!Array.isArray(arr)) return ids;
		const kept = arr.filter((x): x is string => typeof x === "string" && ids.includes(x));
		if (kept.length !== ids.length || new Set(kept).size !== ids.length) return ids;
		return kept;
	} catch {
		return ids;
	}
}

export function ModuleDeck(props: {
	storageKey: string;
	modules: DeckModule[];
	initialExpanded?: string; // 展开初态注入（DOM 桩测面——交互开合仍走 useState）
}) {
	const ids = props.modules.map((m) => m.id);
	const byId = new Map(props.modules.map((m) => [m.id, m]));
	const [order, setOrder] = useState<string[]>(() => loadOrder(props.storageKey, ids));
	const [expanded, setExpanded] = useState<string | null>(props.initialExpanded ?? null);
	const [dragId, setDragId] = useState<string | null>(null);
	const [settlingId, setSettlingId] = useState<string | null>(null);
	// 缩略图状态点镜像（id → 卡头 .dot className——DOM 只读镜像零判据）。
	const [dots, setDots] = useState<Record<string, string>>({});

	const gridRef = useRef<HTMLDivElement | null>(null);
	const detailRef = useRef<HTMLDivElement | null>(null);
	const cellRefs = useRef(new Map<string, HTMLDivElement>());
	const pendingFirst = useRef<DOMRect | null>(null); // 放大 FLIP first 矩形
	const closing = useRef<{ id: string; rect: DOMRect } | null>(null); // 收回逆向 FLIP
	const pendingFlip = useRef<{ first: Map<string, DOMRect>; draggedId: string } | null>(null);
	const [flipBump, bumpFlip] = useState(0);
	const reducedMotion = () =>
		typeof window !== "undefined" &&
		window.matchMedia("(prefers-reduced-motion: reduce)").matches;

	// —— 拖拽重排控制面（usedeckdrag.ts）——
	const drag = useDeckDrag({
		order, expanded, cells: cellRefs, pendingFlip, pendingFirst,
		setOrder, bumpFlip, setDragId, setSettlingId, onExpand: setExpanded,
	});

	// —— 模块集变更同步：新模块尾插，缺席模块剔除（顺序存续）——
	useEffect(() => {
		setOrder((cur) => {
			const cs = new Set(cur);
			const merged = [...cur.filter((x) => ids.includes(x)), ...ids.filter((x) => !cs.has(x))];
			return merged.length === cur.length && merged.every((x, i) => x === cur[i]) ? cur : merged;
		});
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [ids.join("\n")]);

	// —— 顺序持久化 ——
	useEffect(() => {
		try {
			localStorage.setItem(props.storageKey, JSON.stringify(order));
		} catch { /* 持久化不可用=静默降级（会话内顺序仍活） */ }
	}, [order, props.storageKey]);

	// —— 缩略图状态点镜像：卡头 .dot className 只读镜像（rAF 节流）——
	useEffect(() => {
		const grid = gridRef.current;
		if (!grid || typeof MutationObserver === "undefined") return;
		let raf = 0;
		const read = () => {
			raf = 0;
			const next: Record<string, string> = {};
			for (const [id, el] of cellRefs.current) {
				const d = el.querySelector(".card-h .dot");
				if (d) next[id] = d.className;
			}
			setDots(next);
		};
		const schedule = () => {
			if (!raf) raf = requestAnimationFrame(read);
		};
		read();
		const mo = new MutationObserver(schedule);
		mo.observe(grid, { subtree: true, attributes: true, attributeFilter: ["class"], childList: true });
		return () => {
			mo.disconnect();
			if (raf) cancelAnimationFrame(raf);
		};
	}, [flipBump, expanded, order]);

	// —— 展开态：Esc 收 + 背景滚动锁 ——
	const expandedMod = expanded ? byId.get(expanded) ?? null : null;
	const closeRef = useRef(() => {});
	closeRef.current = () => {
		const el = detailRef.current;
		if (el && expanded) closing.current = { id: expanded, rect: el.getBoundingClientRect() };
		setExpanded(null);
	};
	useEffect(() => {
		if (!expanded) return;
		const onKey = (e: KeyboardEvent) => {
			if (e.key === "Escape") closeRef.current();
		};
		window.addEventListener("keydown", onKey);
		document.body.style.overflow = "hidden";
		return () => {
			window.removeEventListener("keydown", onKey);
			document.body.style.overflow = "";
		};
	}, [expanded]);

	// —— 放大 FLIP（first=原格位 → last=详情位）：transform 逆向回放 ——
	useLayoutEffect(() => {
		if (!expanded || !pendingFirst.current) return;
		const el = detailRef.current;
		const first = pendingFirst.current;
		pendingFirst.current = null;
		if (!el || reducedMotion()) return;
		const last = el.getBoundingClientRect();
		el.style.transition = "none";
		el.style.transformOrigin = "top left";
		el.style.transform =
			`translate(${first.left - last.left}px, ${first.top - last.top}px) ` +
			`scale(${first.width / last.width}, ${first.height / last.height})`;
		requestAnimationFrame(() => requestAnimationFrame(() => {
			el.style.transition = `transform ${FLIP_MS}ms ${FLIP_EASE}`;
			el.style.transform = "";
		}));
	}, [expanded]);

	// —— 收回 FLIP（last=详情位 → first=原格位，沿原路逆向回放）——
	useLayoutEffect(() => {
		if (expanded || !closing.current) return;
		const { id, rect } = closing.current;
		closing.current = null;
		const cell = cellRefs.current.get(id);
		if (!cell || reducedMotion()) return;
		const first = cell.getBoundingClientRect();
		cell.style.transition = "none";
		cell.style.transformOrigin = "top left";
		cell.style.transform =
			`translate(${rect.left - first.left}px, ${rect.top - first.top}px) ` +
			`scale(${rect.width / first.width}, ${rect.height / first.height})`;
		requestAnimationFrame(() => requestAnimationFrame(() => {
			cell.style.transition = `transform ${FLIP_MS}ms ${FLIP_EASE}`;
			cell.style.transform = "";
		}));
	}, [expanded]);

	// —— 让位 FLIP（拖拽重排：其余模块 200ms ease 平滑让位）——
	useLayoutEffect(() => {
		if (!pendingFlip.current) return;
		const { first, draggedId } = pendingFlip.current;
		pendingFlip.current = null;
		if (reducedMotion()) return;
		for (const [id, el] of cellRefs.current) {
			if (id === draggedId) continue;
			const f = first.get(id);
			if (!f) continue;
			const l = el.getBoundingClientRect();
			if (f.left === l.left && f.top === l.top) continue;
			el.style.transition = "none";
			el.style.transform = `translate(${f.left - l.left}px, ${f.top - l.top}px)`;
		}
		requestAnimationFrame(() => requestAnimationFrame(() => {
			for (const [id, el] of cellRefs.current) {
				if (id === draggedId) continue;
				el.style.transition = `transform ${YIELD_MS}ms ease`;
				el.style.transform = "";
			}
		}));
	}, [flipBump]);

	return (
		<>
			<div
				ref={gridRef}
				className={"module-grid" + (dragId ? " is-dragging" : "") + (expanded ? " is-dimmed" : "")}
			>
				{order.map((id) => {
					const m = byId.get(id);
					if (!m) return null;
					return (
						<div
							key={id}
							data-module-id={id}
							ref={(el) => {
								if (el) cellRefs.current.set(id, el);
								else cellRefs.current.delete(id);
							}}
							className={
								"module-cell" +
								(dragId === id ? " is-dragging" : "") +
								(settlingId === id ? " is-settling" : "")
							}
							onPointerDown={(e) => drag.onCellPointerDown(id, e)}
							onPointerMove={(e) => drag.onCellPointerMove(id, e)}
							onPointerUp={(e) => drag.onCellPointerUp(id, e)}
							onPointerCancel={() => drag.onCellPointerCancel(id)}
							onDragStart={(e) => e.preventDefault()}
						>
							{m.el}
						</div>
					);
				})}
			</div>
			{expandedMod && (
				<div className="deck-host">
					<div className="deck-detail" key={expanded} ref={detailRef}>
						<button className="btn-ghost deck-close-x" onClick={() => closeRef.current()}
							aria-label="收起模块详情">✕</button>
						{expandedMod.el}
					</div>
					<div className="deck-strip" role="toolbar" aria-label="模块缩略条">
						{props.modules.map((m) => (
							<button key={m.id}
								className={"deck-thumb" + (m.id === expanded ? " on" : "")}
								title={m.title}
								onClick={() => setExpanded(m.id)}>
								<span className={dots[m.id] ?? "dot s3"} aria-hidden />
								<span className="deck-thumb-name">{m.title}</span>
							</button>
						))}
						<button className="btn-ghost deck-thumb" onClick={() => closeRef.current()}>收起</button>
					</div>
				</div>
			)}
		</>
	);
}
