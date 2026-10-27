// flows.tsx —— 复盘页面件库（T-0173 棘轮拆分：历史订单卡/流水筛常量）。
import { useState } from "react";
import { client } from "../api";
import { useSnap } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton } from "./ui";
import { pbDate } from "./fmt";

export function ClosedPositionsCard() {
  const [page, setPage] = useState(0);
  const c = useSnap(["ev_position"], () =>
    client.listClosedPositions({ limit: 10, offset: page * 10 })
  );
  const total = c.data?.total ?? 0;
  const pages = Math.max(1, Math.ceil(total / 10));
  const dur = (a: Date, b: Date) => {
    const h = (b.getTime() - a.getTime()) / 3600000;
    if (h < 48) return `${h.toFixed(0)} 时`;
    return `${(h / 24).toFixed(1)} 天`;
  };
  return (
    <Card title="历史订单（已平仓）">
      {c.loading && !c.data ? (
        <Skeleton rows={5} />
      ) : c.err && !c.data ? (
        <ErrorState msg={c.err} onRetry={c.reload} />
      ) : (c.data?.positions ?? []).length === 0 ? (
        <EmptyState title="暂无已平仓订单" hint="首仓平仓后此处逐行立账" />
      ) : (
        <>
          <table className="rows">
            <thead>
              <tr>
                <th>合约</th>
                <th>开仓</th>
                <th>平仓</th>
                <th>持仓</th>
                <th className="num">结算</th>
              </tr>
            </thead>
            <tbody>
              {(c.data?.positions ?? []).map((p) => (
                <tr key={p.id}>
                  <td>{p.instId}</td>
                  <td className="mono">{(pbDate(p.openedAt)?.toISOString().slice(5, 16).replace("T", " ") ?? "—")}</td>
                  <td className="mono">{(pbDate(p.closedAt)?.toISOString().slice(5, 16).replace("T", " ") ?? "—")}</td>
                  <td className="ink-3">
                    {pbDate(p.openedAt) && pbDate(p.closedAt)
                      ? dur(pbDate(p.openedAt)!, pbDate(p.closedAt)!)
                      : "—"}
                  </td>
                  <td className="num mono">
                    {p.settledAmount !== 0
                      ? `${p.settledAmount.toLocaleString("zh-CN", { maximumFractionDigits: 4 })} ${p.settledCcy}`
                      : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="pager">
            <button className="btn" disabled={page === 0} onClick={() => setPage(page - 1)}>
              上一页
            </button>
            <span className="ink-3">
              {page + 1}/{pages}（{total} 单）
            </span>
            <button className="btn" disabled={page + 1 >= pages} onClick={() => setPage(page + 1)}>
              下一页
            </button>
          </div>
        </>
      )}
    </Card>
  );
}

