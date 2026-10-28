// flows.tsx —— 复盘页面件库（T-0173 棘轮拆分：历史订单卡/流水筛常量）。
// T-0220b：历史订单首页=ev_position 推帧分片订阅（一跳零重拉；载荷粒度
// 5 行/页——原 10 行翻页粒度同票对齐）；翻页视图=unary 参数取（页参数=
// 视图态，非推送面）。
import { useEffect, useState } from "react";
import { client } from "../api";
import { hydrate, useSlice } from "../store";
import { Card, EmptyState, ErrorState, Skeleton } from "./ui";
import { pbDate } from "./fmt";
import type { ListClosedPositionsResponse } from "../../gen/dashboardv1/core_pb";

const PAGE = 5;

export function ClosedPositionsCard() {
  const [page, setPage] = useState(1);
  const [fetched, setFetched] = useState<ListClosedPositionsResponse | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [tick, setTick] = useState(0);
  const live = useSlice("closedPositions");
  const home = page === 1; // 首页=推帧分片；翻页=unary 取页

  useEffect(() => {
    if (home) {
      hydrate("closedPositions", () =>
        client.listClosedPositions({ limit: PAGE, offset: 0 })
      );
    }
  }, [home]);
  useEffect(() => {
    if (home) return;
    let alive = true;
    client
      .listClosedPositions({ limit: PAGE, offset: (page - 1) * PAGE })
      .then((d) => {
        if (!alive) return;
        setFetched(d);
        setErr(null);
      })
      .catch((e: unknown) => {
        if (alive) setErr(e instanceof Error ? e.message : String(e));
      });
    return () => {
      alive = false;
    };
  }, [home, page, tick]);

  const c = home
    ? { data: live.data, err: live.err, loading: live.loading }
    : { data: fetched, err, loading: fetched === null && err === null };
  const total = c.data?.total ?? 0;
  const pages = Math.max(1, Math.ceil(total / PAGE));
  const retry = () => {
    if (home) {
      hydrate("closedPositions", () =>
        client.listClosedPositions({ limit: PAGE, offset: 0 })
      );
    } else {
      setTick((t) => t + 1); // 翻页视图重取（key 指纹=home:page:tick）
    }
  };
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
        <ErrorState msg={c.err} onRetry={retry} />
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
            <button className="btn" disabled={page === 1} onClick={() => setPage(page - 1)}>
              上一页
            </button>
            <span className="ink-3">
              {page}/{pages}（{total} 单）
            </span>
            <button className="btn" disabled={page + 1 > pages} onClick={() => setPage(page + 1)}>
              下一页
            </button>
          </div>
        </>
      )}
    </Card>
  );
}

