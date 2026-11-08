// venueboard.tsx —— 侦察（进水口）榜卡（T-0305：scoutv1.Board 只读面消费——
// metrics.boards=venue×族榜行；纯渲染零新状态；行数闸拆分。
// T-0361：业主视面降级——卡体 <details> 默认收合只露 venue 计数摘要
// （建议注册=唯一业主动作态置显眼色）；展开行按业主关心序排序+10 行分页；
// 正名「侦察（进水口）」与 flow_pulse scout 段「侦察」同词）。
import { useState } from "react";
import { scoutClient } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton } from "./ui";
import { SCOUT_BASE_SRC_CN, SCOUT_FAMILY_CN, SCOUT_STATE_CN, cn } from "./labels";

// T-0306/T-0441：裸值归中文映射归一 labels.ts——未知值人话兜底不吞不裸出。
const famTxt = (v?: string) => cn(SCOUT_FAMILY_CN, v, "未识别族");
const stateTxt = (v?: string) => cn(SCOUT_STATE_CN, v, "未识别态");
const baseSrcTxt = (v?: string) => cn(SCOUT_BASE_SRC_CN, v, "未识别源");

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
                  {stateTxt(s)} {n}
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
                  <td>{famTxt(b?.family)}</td>
                  <td className="num mono">{b?.net_apr_mid !== undefined ? (b.net_apr_mid * 100).toFixed(2) + "%" : "—"}</td>
                  <td className="num mono">{b?.persist !== undefined ? (b.persist * 100).toFixed(0) + "%" : "—"}</td>
                  <td className="ink-3">{baseSrcTxt(b?.baseline_source)}</td>
                  <td className="ink-3">{stateTxt(v.state)}</td>
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
