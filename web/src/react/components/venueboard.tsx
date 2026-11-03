// venueboard.tsx —— 找平台榜卡（T-0305：scoutv1.Board 只读面消费——
// metrics.boards=venue×族榜行；measuring 所前排；纯渲染零新状态；行数闸拆分）。
import { scoutClient } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton } from "./ui";

// T-0305：找平台卡——scoutv1.Board 只读面消费（metrics.boards=venue×族
// 榜行；measuring 所前排；纯渲染零新状态）。
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
  const rows = (r.data?.rows ?? [])
    .slice()
    .sort((a, b) => (a.state === "measuring" ? 0 : 1) - (b.state === "measuring" ? 0 : 1));
  return (
    <Card title="找平台">
      {r.loading && !r.data ? (
        <Skeleton rows={3} />
      ) : r.err && !r.data ? (
        <ErrorState msg={r.err} onRetry={r.reload} />
      ) : rows.length === 0 ? (
        <EmptyState title="scout 榜空" hint="venue 度量臂日拍后出行" />
      ) : (
        <table className="rows">
          <thead>
            <tr><th>venue</th><th>族</th><th>净APR</th><th>持续</th><th>基线</th><th>态</th><th>测于</th></tr>
          </thead>
          <tbody>
            {rows.flatMap((v) => {
              const m = (v.metrics ?? {}) as { boards?: ScoutBoardRow[]; measured_at?: string };
              const at = m.measured_at ? String(m.measured_at).slice(0, 10) : "—";
              const bs = m.boards ?? [];
              const vn = v.venue.length > 12 ? v.venue.slice(0, 12) + "…" : v.venue;
              if (bs.length === 0) {
                return [
                  <tr key={v.venue}><td>{vn}</td><td>—</td><td>—</td><td>—</td><td>—</td>
                    <td className="ink-3">{v.state}</td><td className="ink-3">{at}</td></tr>,
                ];
              }
              return bs.map((b) => (
                <tr key={v.venue + ":" + (b.family ?? "")}>
                  <td>{vn}</td>
                  <td>{b.family ?? "—"}</td>
                  <td className="num mono">{b.net_apr_mid !== undefined ? (b.net_apr_mid * 100).toFixed(2) + "%" : "—"}</td>
                  <td className="num mono">{b.persist !== undefined ? (b.persist * 100).toFixed(0) + "%" : "—"}</td>
                  <td className="ink-3">{b.baseline_source ?? "—"}</td>
                  <td className="ink-3">{v.state}</td>
                  <td className="ink-3">{at}</td>
                </tr>
              ));
            })}
          </tbody>
        </table>
      )}
    </Card>
  );
}

