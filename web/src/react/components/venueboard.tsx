// venueboard.tsx —— 侦察（进水口）榜卡（T-0305：scoutv1.Board 只读面消费——
// metrics.boards=venue×族榜行；纯渲染零新状态；行数闸拆分。
// T-0361：业主视面降级——卡体 <details> 默认收合只露 venue 计数摘要
// （建议注册=唯一业主动作态置显眼色）；展开行按业主关心序排序+10 行分页；
// 正名「侦察（进水口）」与 flow_pulse scout 段「侦察」同词）。
import { useState } from "react";
import { scoutClient } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton } from "./ui";

// T-0306：裸值归中文映射（未知值原样透出不吞——家族/状态/基线源三表）。
const FAMILY_CN: Record<string, string> = {
  carry: "单所息差",
  xvenue_spread: "跨所价差",
  basis: "基差",
};
const STATE_CN: Record<string, string> = {
  measuring: "度量中",
  candidate: "候选",
  recommended: "建议注册",
  registered: "已注册",
  rejected: "已拒",
  kyc_blocked: "KYC受限",
  adopted: "已采纳",
  dismissed: "已驳回",
  parked: "已停放",
};
const BASE_SRC_CN: Record<string, string> = { earn: "Earn", okx_family: "OKX同族" };
const cn = (m: Record<string, string>, v?: string) => (v ? m[v] ?? v : "—");

// T-0361：业主关心序——建议注册>候选>度量中>其余。
const STATE_RANK = (s?: string) =>
  s === "recommended" ? 0 : s === "candidate" ? 1 : s === "measuring" ? 2 : 3;

const PAGE = 10;

type ScoutBoardRow = {
  family?: string;
  net_apr_mid?: number;
  persist?: number;
  baseline?: number;
  baseline_source?: string;
  top_n?: number;
};

export function VenueBoardCard() {
  const r = useLive(["ev_funding"], () => scoutClient.board({}));
  const [page, setPage] = useState(0);
  const rows = r.data?.rows ?? [];

  // 计数摘要按 venue 计非族行数（票面口径）。
  const counts = new Map<string, number>();
  for (const v of rows) counts.set(v.state ?? "", (counts.get(v.state ?? "") ?? 0) + 1);
  const summaryOrder = ["recommended", "candidate", "measuring"];
  const rest = [...counts.entries()].filter(([s]) => !summaryOrder.includes(s));
  const summarySegs = [
    ...summaryOrder.filter((s) => counts.has(s)).map((s) => [s, counts.get(s)!] as const),
    ...rest,
  ];

  // 展平 venue×族行后排序：态序先、同态净年化降序。
  const flat = rows.flatMap((v) => {
    const m = (v.metrics ?? {}) as { boards?: ScoutBoardRow[]; measured_at?: string };
    const at = m.measured_at ? String(m.measured_at).slice(0, 10) : "—";
    const vn = v.venue.length > 12 ? v.venue.slice(0, 12) + "…" : v.venue;
    const bs = m.boards ?? [];
    return (bs.length === 0 ? [undefined] : bs).map((b) => ({ v, vn, at, b }));
  });
  flat.sort(
    (x, y) =>
      STATE_RANK(x.v.state) - STATE_RANK(y.v.state) ||
      (y.b?.net_apr_mid ?? -Infinity) - (x.b?.net_apr_mid ?? -Infinity)
  );
  const pages = Math.max(1, Math.ceil(flat.length / PAGE));
  const safePage = Math.min(page, pages - 1);
  const view = flat.slice(safePage * PAGE, safePage * PAGE + PAGE);

  return (
    <Card title="侦察（进水口）">
      {r.loading && !r.data ? (
        <Skeleton rows={3} />
      ) : r.err && !r.data ? (
        <ErrorState msg={r.err} onRetry={r.reload} />
      ) : rows.length === 0 ? (
        <EmptyState title="侦察榜暂无数据" hint="交易所度量每日更新后展示" />
      ) : (
        <details className="vboard">
          <summary>
            {summarySegs.map(([s, n], i) =>
              s === "recommended" ? (
                <span key={s}>
                  {i > 0 ? " · " : ""}
                  <span className="chip chip-warn">建议注册 {n}</span>
                </span>
              ) : (
                <span key={s}>
                  {i > 0 ? " · " : ""}
                  {cn(STATE_CN, s)} {n}
                </span>
              )
            )}
          </summary>
          <table className="rows">
            <thead>
              <tr><th>平台</th><th>族</th><th>净年化</th><th>达标天比</th><th>基线源</th><th>状态</th><th>度量日</th></tr>
            </thead>
            <tbody>
              {view.map(({ v, vn, at, b }) => (
                <tr key={v.venue + ":" + (b?.family ?? "")}>
                  <td>{vn}</td>
                  <td>{cn(FAMILY_CN, b?.family)}</td>
                  <td className="num mono">{b?.net_apr_mid !== undefined ? (b.net_apr_mid * 100).toFixed(2) + "%" : "—"}</td>
                  <td className="num mono">{b?.persist !== undefined ? (b.persist * 100).toFixed(0) + "%" : "—"}</td>
                  <td className="ink-3">{cn(BASE_SRC_CN, b?.baseline_source)}</td>
                  <td className="ink-3">{cn(STATE_CN, v.state)}</td>
                  <td className="ink-3">{at}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {flat.length > PAGE && (
            <div className="pager">
              <button className="btn" disabled={safePage === 0} onClick={() => setPage(safePage - 1)}>
                上一页
              </button>
              <span>
                {safePage + 1}/{pages}（{flat.length} 行）
              </span>
              <button className="btn" disabled={safePage + 1 >= pages} onClick={() => setPage(safePage + 1)}>
                下一页
              </button>
            </div>
          )}
        </details>
      )}
    </Card>
  );
}
