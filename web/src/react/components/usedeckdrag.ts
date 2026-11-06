// usedeckdrag.ts —— T-0411 模块 deck 拖拽重排控制钩（自 moduledeck.tsx 拆出
// ——行数棘轮 D-588）。交互④语义（全在 moduledeck.tsx 头注）：
//   点按/拖拽分阈——按下位移<6px=点击，≥6px=拖拽；拖起旋转 -3°（内联先转
//   再测抓取偏移，防类 rotate 同帧未提交致 bbox 膨胀跳位）；拖起随针
//   （rect 含当前 transform——先扣得基点再算新偏移，gx/gy 恒定自洽）；
//   矩形碰撞让位——最近格≠当前格才重排（让位后格位随动）；松手吸附最近
//   空闲格+弹性落位（is-settling 200ms ease）。
import { useRef, type PointerEvent as ReactPointerEvent } from "react";

// 点按/拖拽分阈（稿内钉死 6px）。
export const DRAG_THRESHOLD_PX = 6;
// 让位/吸附过渡（稿内钉 200ms ease）。
export const YIELD_MS = 200;
// FLIP 放大/收回时长（轻快不超调——临界阻尼族贝塞尔，CSS 同参）。
export const FLIP_MS = 320;
export const FLIP_EASE = "cubic-bezier(0.22, 1, 0.36, 1)";
// 格内交互件豁免（按这些件不启动拖拽/放大——卡内钮/表单语义优先）。
const INTERACTIVE_SEL =
	"button, a, input, select, textarea, label, summary, details, [role='button']";

type DragState = {
	id: string; startX: number; startY: number; moved: boolean;
	tx: number; ty: number; gx: number; gy: number; px: number; py: number;
};

// 矩形碰撞最近格（点→格矩形钳位距离，含在格内=0）。
function nearestCell(
	cells: Map<string, HTMLDivElement>, px: number, py: number,
): { id: string; dist: number } | null {
	let best: { id: string; dist: number } | null = null;
	for (const [id, el] of cells) {
		const r = el.getBoundingClientRect();
		const cx = Math.min(Math.max(px, r.left), r.right);
		const cy = Math.min(Math.max(py, r.top), r.bottom);
		const d = Math.hypot(px - cx, py - cy);
		if (!best || d < best.dist) best = { id, dist: d };
	}
	return best;
}

export function useDeckDrag(opts: {
	order: string[];
	expanded: string | null;
	cells: { current: Map<string, HTMLDivElement> };
	pendingFlip: { current: { first: Map<string, DOMRect>; draggedId: string } | null };
	pendingFirst: { current: DOMRect | null };
	setOrder: (next: string[]) => void;
	bumpFlip: (fn: (v: number) => number) => void;
	setDragId: (id: string | null) => void;
	setSettlingId: (fn: (cur: string | null) => string | null) => void;
	onExpand: (id: string) => void;
}) {
	const dragRef = useRef<DragState | null>(null);

	// —— 重排（让位 FLIP 装配：first 矩形→setOrder→layout effect 回放；
	//     纯 setState——first 测量/pendingFlip 装配在本函数体内、updater
	//     外，StrictMode 双调安全）——
	const reorder = (draggedId: string, targetIndex: number) => {
		const from = opts.order.indexOf(draggedId);
		if (from === targetIndex || from < 0) return;
		const first = new Map<string, DOMRect>();
		for (const [id, el] of opts.cells.current) {
			if (id !== draggedId && el) first.set(id, el.getBoundingClientRect());
		}
		opts.pendingFlip.current = { first, draggedId };
		const next = [...opts.order];
		next.splice(from, 1);
		next.splice(targetIndex, 0, draggedId);
		opts.setOrder(next);
		opts.bumpFlip((v) => v + 1);
	};

	const onCellPointerDown = (id: string, e: ReactPointerEvent<HTMLDivElement>) => {
		if (opts.expanded || e.button !== 0) return;
		if ((e.target as Element).closest(INTERACTIVE_SEL)) return;
		dragRef.current = {
			id, startX: e.clientX, startY: e.clientY, moved: false,
			tx: 0, ty: 0, gx: 0, gy: 0, px: e.clientX, py: e.clientY,
		};
		try { e.currentTarget.setPointerCapture(e.pointerId); } catch { /* 捕获失败降级窗内拖 */ }
	};

	const onCellPointerMove = (id: string, e: ReactPointerEvent<HTMLDivElement>) => {
		const dr = dragRef.current;
		if (!dr || dr.id !== id || opts.expanded) return;
		dr.px = e.clientX;
		dr.py = e.clientY;
		if (!dr.moved) {
			if (Math.hypot(e.clientX - dr.startX, e.clientY - dr.startY) < DRAG_THRESHOLD_PX) return;
			dr.moved = true;
			opts.setDragId(id);
			const cell = opts.cells.current.get(id);
			if (cell) {
				// 拖起 -3° 先内联落位再测（类 rotate 同帧未提交——bbox 膨胀
				// 会让 gx 取偏，旋转后卡在指针下微跳）
				cell.style.rotate = "-3deg";
				const r = cell.getBoundingClientRect();
				dr.gx = dr.px - r.left;
				dr.gy = dr.py - r.top;
			}
		}
		// 拖起随针：rect 含当前 transform——先扣得基点再算新偏移（gx/gy 恒定）
		const cell = opts.cells.current.get(id);
		if (cell) {
			const r = cell.getBoundingClientRect();
			const baseLeft = r.left - dr.tx;
			const baseTop = r.top - dr.ty;
			dr.tx = dr.px - dr.gx - baseLeft;
			dr.ty = dr.py - dr.gy - baseTop;
			cell.style.transform = `translate(${dr.tx}px, ${dr.ty}px)`;
		}
		// 矩形碰撞让位：最近格≠当前格才重排（让位后格位随动）
		const near = nearestCell(opts.cells.current, dr.px, dr.py);
		if (near && near.id !== id) {
			const cur = opts.cells.current.get(id)?.getBoundingClientRect();
			if (cur) {
				const cx = Math.min(Math.max(dr.px, cur.left), cur.right);
				const cy = Math.min(Math.max(dr.py, cur.top), cur.bottom);
				if (near.dist < Math.hypot(dr.px - cx, dr.py - cy)) reorder(id, opts.order.indexOf(near.id));
			}
		}
	};

	const endDrag = (id: string, doClick: boolean) => {
		const dr = dragRef.current;
		dragRef.current = null;
		if (!dr || dr.id !== id) return;
		if (!dr.moved) {
			if (doClick) {
				const cell = opts.cells.current.get(id);
				if (cell) opts.pendingFirst.current = cell.getBoundingClientRect();
				opts.onExpand(id);
			}
			return;
		}
		// 松手吸附：矩形碰撞最近空闲格——≠当前格则重排，随后弹性落位
		const near = nearestCell(opts.cells.current, dr.px, dr.py);
		const curIdx = opts.order.indexOf(id);
		if (near) {
			const nextIdx = opts.order.indexOf(near.id);
			if (nextIdx >= 0 && nextIdx !== curIdx) reorder(id, nextIdx);
		}
		opts.setDragId(null);
		opts.setSettlingId((cur) => (cur === null ? id : cur));
		const cell = opts.cells.current.get(id);
		if (cell) {
			cell.style.transform = "";
			cell.style.rotate = "";
		}
		window.setTimeout(() => opts.setSettlingId((s) => (s === id ? null : s)), YIELD_MS + 40);
	};

	const onCellPointerUp = (id: string, e: ReactPointerEvent<HTMLDivElement>) => {
		const dr = dragRef.current;
		if (!dr || dr.id !== id) return;
		dr.px = e.clientX;
		dr.py = e.clientY;
		endDrag(id, true);
	};
	const onCellPointerCancel = (id: string) => endDrag(id, false);

	return { onCellPointerDown, onCellPointerMove, onCellPointerUp, onCellPointerCancel };
}
