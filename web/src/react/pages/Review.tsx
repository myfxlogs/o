// Review.tsx —— 复盘页（T-0170：报告三粒度+净值曲线+现金流全量下钻——
// 数据全走 Connect 新契约 ListReports/GetEquityCurve/ListCashflows）。
import { useState } from "react";
import { client } from "../api";
import { useSnap } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton } from "../components/ui";
import { LineChart } from "../components/chart";
import { fmtUsd, pbDate } from "../components/fmt";

const PERIODS: { k: string; label: string }[] = [
  { k: "daily", label: "日报" },
  { k: "weekly", label: "周报" },
  { k: "monthly", label: "月报" },
];

const METRIC_NAMES: Record<string, string> = {
  net: "净额",
  cum_nav: "累计净值",
  funding_income: "资金费收入",
  earn_balance: "Earn 停泊",
  opened: "开仓",
  closed: "平仓",
  proposed: "评",
  rejected: "拒",
  active: "在册策略",
  alerts: "风控拒",
  capital_flows: "资金流",
  annualized: "年化",
  capital_base: "基数",
  retired: "退役",
};

function ReportsCard() {
  const [period, setPeriod] = useState("daily");
  const [page, setPage] = useState(0);
  const r = useSnap(["ev_cashflow"], () =>
    client.listReports({ period, n: 10, offset: page * 10 })
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
                      <td>{METRIC_NAMES[m.k] ?? m.k}</td>
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

function EquityCard() {
  const c = useSnap(["ev_cashflow"], () => client.getEquityCurve({ days: 90 }));
  const pts = c.data?.points ?? [];
  return (
    <Card title="净值曲线（90 天）">
      {c.loading && !c.data ? (
        <Skeleton rows={3} />
      ) : c.err && !c.data ? (
        <ErrorState msg={c.err} onRetry={c.reload} />
      ) : (
        <LineChart
          xs={pts.map((p) => (() => { const d = pbDate(p.day); return d ? Math.floor(d.getTime()/1000) : 0; })())}
          ys={pts.map((p) => p.nav)}
          labels={pts.map((p) => pbDate(p.day)?.toISOString().slice(0, 10) ?? "")}
          fmtY={(v) => "$" + v.toLocaleString("zh-CN", { maximumFractionDigits: 0 })}
        />
      )}
    </Card>
  );
}

const FLOW_KINDS = ["", "interest", "funding_settle", "trade_pnl", "fee", "deposit", "withdraw"];

function FlowsCard() {
  const [kind, setKind] = useState("");
  const [page, setPage] = useState(0);
  const f = useSnap(["ev_cashflow"], () =>
    client.listCashflows({ kind, limit: 20, offset: page * 20 })
  );
  const total = f.data?.total ?? 0;
  const pages = Math.max(1, Math.ceil(total / 20));
  return (
    <Card
      title="现金流全量"
      extra={
        <span className="seg">
          {FLOW_KINDS.map((k) => (
            <button
              key={k}
              className={"btn seg-btn" + (kind === k ? " on" : "")}
              onClick={() => {
                setKind(k);
                setPage(0);
              }}
            >
              {k === "" ? "全部" : k}
            </button>
          ))}
        </span>
      }
    >
      {f.loading && !f.data ? (
        <Skeleton rows={6} />
      ) : f.err && !f.data ? (
        <ErrorState msg={f.err} onRetry={f.reload} />
      ) : (f.data?.rows ?? []).length === 0 ? (
        <EmptyState title="该筛无现金流" hint="换个类目，或等下一笔落地" />
      ) : (
        <>
          <table className="rows">
            <thead>
              <tr>
                <th>时间</th>
                <th>类</th>
                <th className="num">额</th>
                <th>venue</th>
              </tr>
            </thead>
            <tbody>
              {(f.data?.rows ?? []).map((row) => (
                <tr key={row.id}>
                  <td className="mono">{(pbDate(row.occurredAt)?.toISOString().slice(5, 16).replace("T", " ") ?? '—')}</td>
                  <td>{row.kind}</td>
                  <td className={"num mono " + (row.amount >= 0 ? "u-good" : "u-bad")}>
                    {row.amount >= 0 ? "+" : ""}
                    {row.amount.toFixed(4)} {row.ccy}
                  </td>
                  <td className="ink-3">{row.venue || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="pager">
            <button className="btn" disabled={page === 0} onClick={() => setPage(page - 1)}>
              上一页
            </button>
            <span className="ink-3">
              {page + 1}/{pages}（{total} 行）
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

export function ReviewPage() {
  return (
    <div className="page">
      <EquityCard />
      <div className="duo">
        <ReportsCard />
        <FlowsCard />
      </div>
    </div>
  );
}
