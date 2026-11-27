// flowcards.tsx —— 复盘/赚钱两页共享卡（O-41386 抽离复用：GetEquityCurve
// 净值曲线+ListCashflows 资金流水全量下钻——同吃一读面防第二套渲染；
// kind/venue 筛+分页自含）。
import { useState } from "react";
import { client } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton } from "./ui";
import { LineChart } from "./chart";
import { pbDate } from "./fmt";
import { CASHFLOW_KIND_CN, KIND_FALLBACK, VENUE_CN, cn } from "./labels";

// T-0441：指标/类目/平台词表归一 labels.ts——未知键人话兜底不裸出机读枚举。
const kindTxt = (k: string) => cn(CASHFLOW_KIND_CN, k, KIND_FALLBACK);
const venueTxt = (v: string) => cn(VENUE_CN, v, "其他");

export function EquityCard() {
  const c = useLive(["ev_cashflow"], () => client.getEquityCurve({ days: 90 }));
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

const FLOW_KINDS: [string, string][] = [
  ["", "全部"],
  ["interest", "活期利息"],
  ["funding_settle", "资金费"],
  ["trade_pnl", "平仓盈亏"],
  ["fee", "交易成本"],
  ["deposit", "入金"],
  ["withdraw", "出金"],
];
const FLOW_VENUES: [string, string][] = [["", "全部平台"], ["okx", "OKX"], ["mt5", "MT5"]];

export function FlowsCard() {
  const [kind, setKind] = useState("");
  const [venue, setVenue] = useState("");
  const [page, setPage] = useState(0);
  const f = useLive(["ev_cashflow"],
    () => client.listCashflows({ kind, venue, limit: 10, offset: page * 10 }),
    `${kind}:${venue}:${page}`
  );
  const total = f.data?.total ?? 0;
  const pages = Math.max(1, Math.ceil(total / 10));
  return (
    <Card
      title="现金流全量"
      extra={
        <span className="seg">
          {FLOW_KINDS.map(([k, cn]) => (
            <button
              key={k}
              className={"btn seg-btn" + (kind === k ? " on" : "")}
              onClick={() => {
                setKind(k);
                setPage(0);
              }}
            >
              {cn}
            </button>
          ))}
          <span className="ink-3">｜</span>
          {FLOW_VENUES.map(([v, cn]) => (
            <button
              key={v}
              className={"btn seg-btn" + (venue === v ? " on" : "")}
              onClick={() => {
                setVenue(v);
                setPage(0);
              }}
            >
              {cn}
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
                <th>平台</th>
              </tr>
            </thead>
            <tbody>
              {(f.data?.rows ?? []).map((row) => (
                <tr key={row.id}>
                  <td className="mono">{(pbDate(row.occurredAt)?.toISOString().slice(5, 16).replace("T", " ") ?? '—')}</td>
                  <td>{kindTxt(row.kind)}</td>
                  <td className={"num mono " + (row.amount >= 0 ? "u-good" : "u-bad")}>
                    {row.amount >= 0 ? "+" : ""}
                    {row.amount.toFixed(4)} {row.ccy}
                  </td>
                  <td className="ink-3">{venueTxt(row.venue)}</td>
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
