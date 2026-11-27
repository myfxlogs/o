// Review.tsx —— 复盘页（T-0170：报告三粒度+净值曲线+现金流全量下钻——
// 数据全走 Connect 新契约 ListReports/GetEquityCurve/ListCashflows）。
import { useState } from "react";
import { client } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton } from "../components/ui";
import { ClosedPositionsCard } from "../components/flows";
import { VenueBoardCard } from "../components/venueboard";
import { fmtUsd, pbDate } from "../components/fmt";
import { METRIC_CN, METRIC_FALLBACK, cn } from "../components/labels";
import { EquityCard, FlowsCard } from "../components/flowcards";

const PERIODS: { k: string; label: string }[] = [
  { k: "daily", label: "日报" },
  { k: "weekly", label: "周报" },
  { k: "monthly", label: "月报" },
];

// T-0441：指标/类目/平台词表归一 labels.ts——未知键人话兜底不裸出机读枚举。
const metricTxt = (k: string) => cn(METRIC_CN, k, METRIC_FALLBACK);

function ReportsCard() {
  const [period, setPeriod] = useState("daily");
  const [page, setPage] = useState(0);
  const r = useLive(["ev_cashflow"],
    () => client.listReports({ period, n: 10, offset: page * 10 }),
    `${period}:${page}`
  );
  const total = r.data?.total ?? 0;
  const pages = Math.max(1, Math.ceil(total / 10));
  const go = (p: number) => {
    setPage(p);
  };
  return (
    <Card
      title="报告"
      extra={
        <span className="seg">
          {PERIODS.map((p) => (
            <button
              key={p.k}
              className={"btn seg-btn" + (period === p.k ? " on" : "")}
              onClick={() => {
                setPeriod(p.k);
                go(0);
              }}
            >
              {p.label}
            </button>
          ))}
        </span>
      }
    >
      {r.loading && !r.data ? (
        <Skeleton rows={4} />
      ) : r.err && !r.data ? (
        <ErrorState msg={r.err} onRetry={r.reload} />
      ) : (r.data?.reports ?? []).length === 0 ? (
        <EmptyState title="首份报告在路上" hint="下一个 UTC 日界自动出行，出行即推飞书" />
      ) : (
        <div className="feed">
          {(r.data?.reports ?? []).map((rep) => (
            <details key={rep.period + String(rep.periodStart?.seconds)} className="rep">
              <summary>
                {(pbDate(rep.periodStart)?.toISOString().slice(0, 10) ?? '—')}
                {rep.metrics.find((m) => m.k === "net")
                  ? ` — 净 ${fmtUsd(rep.metrics.find((m) => m.k === "net")!.v)}`
                  : ""}
              </summary>
              <table className="rows">
                <tbody>
                  {rep.metrics.map((m) => (
                    <tr key={m.k}>
                      <td title={m.k}>{metricTxt(m.k)}</td>
                      <td className="num mono">{m.v.toLocaleString("zh-CN", { maximumFractionDigits: 4 })}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {rep.note ? <div className="ink-3 note">{rep.note}</div> : null}
            </details>
          ))}
          <div className="pager">
            <button className="btn" disabled={page === 0} onClick={() => go(page - 1)}>
              上一页
            </button>
            <span className="ink-3">
              {page + 1}/{pages}（{total} 份）
            </span>
            <button className="btn" disabled={page + 1 >= pages} onClick={() => go(page + 1)}>
              下一页
            </button>
          </div>
        </div>
      )}
    </Card>
  );
}

export function ReviewPage() {
  return (
    <div className="page v2">
      <section className="v2-hero">
        <div className="v2-eyebrow">
          <span className="v2-livedot" />
          钱怎么走的 · LIVE
        </div>
        <h1 className="v2-title">收支复盘</h1>
        <p className="v2-lede">净值曲线→平台对账→期报→流水→已平仓——读序自上而下</p>
      </section>
      <EquityCard />
      <VenueBoardCard />
      <div className="duo">
        <ReportsCard />
        <FlowsCard />
      </div>
      <ClosedPositionsCard />
    </div>
  );
}
