// pager.tsx —— 前端分页件（业主令 11-11：榜单类不撑高页面——哑弹榜/探针
// 心跳/处置链同形消费）。数据已全取回时纯前端切片；页脚条显「第x/y页·共N」。
import { useState } from "react";

export function usePager<T>(rows: T[], pageSize: number) {
	const [page, setPage] = useState(0);
	const pages = Math.max(1, Math.ceil(rows.length / pageSize));
	const cur = Math.min(page, pages - 1);
	return {
		page: cur,
		pages,
		setPage,
		slice: rows.slice(cur * pageSize, cur * pageSize + pageSize),
		total: rows.length,
	};
}

export function PagerBar(props: {
	page: number;
	pages: number;
	total: number;
	onPage: (p: number) => void;
}) {
	if (props.pages <= 1) return <span className="ink-3">{props.total} 行</span>;
	return (
		<span className="pager">
			<button
				className="pager-btn"
				disabled={props.page === 0}
				onClick={() => props.onPage(props.page - 1)}
				aria-label="上一页"
			>
				‹
			</button>
			<span className="ink-3">
				{props.page + 1}/{props.pages} · 共{props.total}
			</span>
			<button
				className="pager-btn"
				disabled={props.page >= props.pages - 1}
				onClick={() => props.onPage(props.page + 1)}
				aria-label="下一页"
			>
				›
			</button>
		</span>
	);
}
