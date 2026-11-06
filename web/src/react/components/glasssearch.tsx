// glasssearch.tsx —— T-0411 交互②「玻璃搜索浮层」共享件（业主定域=候选
// 列表过滤非全局搜索）：浮层弹出+背景压暗+blur(8px)（整页模糊全站唯此
// 一处低频用）+输入实时过滤+高亮匹配段。消费面：自选✎编辑（QuoteWatch
// ——子串过滤逻辑现成换皮即得）/建提醒 symbol 候选（QuoteBoard——datalist
// 升浮层）/将来一切候选选择面——过滤逻辑不重写，新活只在本壳。
// 复用既有 .modal-overlay/.modal 几何契约（fixed 置顶/90vh 封顶——T-0406
// domtest 锚在案），玻璃质感挂 .glass-overlay/.glass-modal 修饰类。
import { type ReactNode } from "react";

// GlassSearchShell 浮层壳（压暗+blur8 玻璃面板+点暗处关）。
export function GlassSearchShell(props: {
	ariaLabel: string;
	onClose: () => void;
	children: ReactNode;
}) {
	return (
		<div className="modal-overlay glass-overlay" onClick={props.onClose}>
			<div className="modal glass-modal" role="dialog" aria-label={props.ariaLabel}
				onClick={(e) => e.stopPropagation()}>
				{props.children}
			</div>
		</div>
	);
}

// filterCands 候选过滤（共享单源：子串大小写不敏——搜索=展示面过滤非
// 业务判据；exclude=已选剔出）。
export function filterCands(cands: string[], query: string, exclude?: Set<string>): string[] {
	const q = query.trim().toLowerCase();
	return cands
		.filter((s) => !exclude?.has(s))
		.filter((s) => !q || s.toLowerCase().includes(q));
}

// HighlightMatch 高亮匹配段（大小写不敏定位，<mark class="glass-hl"> 包裹；
// 空 query=原文直出）。
export function HighlightMatch(props: { text: string; query: string }) {
	const q = props.query.trim().toLowerCase();
	if (!q) return <>{props.text}</>;
	const lower = props.text.toLowerCase();
	const out: ReactNode[] = [];
	let i = 0;
	for (;;) {
		const hit = lower.indexOf(q, i);
		if (hit < 0) {
			out.push(props.text.slice(i));
			break;
		}
		if (hit > i) out.push(props.text.slice(i, hit));
		out.push(<mark key={hit} className="glass-hl">{props.text.slice(hit, hit + q.length)}</mark>);
		i = hit + q.length;
	}
	return <>{out}</>;
}
