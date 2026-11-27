// StrategyPnlCard.tsx —— 策略盈亏榜（O-41386：ListStrategyPnl 新 RPC 消费——
// fin_pnl_net 钦定账本行 join 仓→策略聚合，服务端按 net 降序；
// realized/fees/funding 三分列+net 合计+关联仓数+末笔时；
// ev_cashflow 信号帧重取零轮询；空榜=如实空不兜底）。
import { client } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton } from "./ui";
import { pbDate } from "./fmt";

const fmtUsd = (v: number) =>
  (v > 0 ? "+" : v < 0 ? "−" : "") +
  "$" +
  Math.abs(v).toLocaleString("zh-CN", { maximumFractionDigits: 2 });
const pnlCls = (v: number) => (v > 0 ? "u-good" : v < 0 ? "u-bad" : "");

export function StrategyPnlCard() {
  const lv = useLive(["ev_cashflow"], () => client.listStrategyPnl({ days: 30 }));
  const rows = lv.data?.rows ?? [];
  return (
    <Card title="策略盈亏榜（30 天）">
      {lv.loading && !lv.data ? (
        <Skeleton rows={4} />
      ) : lv.err && !lv.data ? (
        <ErrorState msg={lv.err} onRetry={lv.reload} />
      ) : rows.length === 0 ? (
        <EmptyState
          title="榜上无策略账"
          hint="窗内无挂仓账行——账外腿/现货转换不入策略榜（如实空）"
        />
      ) : (
        <table className="rows">
          <thead>
            <tr>
              <th>族</th>
              <th className="num">实现</th>
              <th className="num">费</th>
              <th className="num">资费</th>
              <th className="num">净</th>
              <th className="num">仓数</th>
              <th>末笔</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={String(r.strategyRef)}>
                <td>{r.template || "—"}</td>
                <td className={`num mono ${pnlCls(r.realizedUsd)}`}>{fmtUsd(r.realizedUsd)}</td>
                <td className={`num mono ${pnlCls(r.feesUsd)}`}>{fmtUsd(r.feesUsd)}</td>
                <td className={`num mono ${pnlCls(r.fundingUsd)}`}>{fmtUsd(r.fundingUsd)}</td>
                <td className={`num mono ${pnlCls(r.netUsd)}`}>{fmtUsd(r.netUsd)}</td>
                <td className="num mono">{r.positions}</td>
                <td className="mono ink-3">
                  {pbDate(r.lastTs)?.toISOString().slice(5, 16).replace("T", " ") ?? "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </Card>
  );
}
