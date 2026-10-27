// TrxVoteEngine.tsx —— T-0184 投票引擎状态卡（只读显态面：在驶态/下窗
// 倒计时/待领取/今日改投×上限；启停=app_config 业主径不在前端鉴权面
// 内——降级只读+「config 热改径」注，鉴权面落位前零 mutating 控件）。
import { useCallback, useEffect, useState } from "react";
import { client } from "../api";
import { Skeleton } from "./ui";
import type { GetTrxVoteStatusResponse } from "../../gen/dashboardv1/core_pb";

const fmtCountdown = (ms: number) => {
  if (ms <= 0) return "即刻";
  const m = Math.floor(ms / 60000);
  const h = Math.floor(m / 60);
  return h > 0 ? `${h}h${m % 60}m` : `${m}m`;
};

export function TrxVoteEngine() {
  const [st, setSt] = useState<GetTrxVoteStatusResponse | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [now, setNow] = useState(Date.now());
  const load = useCallback(() => {
    void client.getTrxVoteStatus({})
      .then((r) => { setSt(r); setErr(null); })
      .catch((e: unknown) => setErr(String(e)));
  }, []);
  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    const t = setInterval(() => { load(); setNow(Date.now()); }, 30_000);
    return () => clearInterval(t);
  }, [load]);
  if (err) return <section className="card"><h3>投票引擎</h3><div className="dim">读败：{err}</div></section>;
  if (!st) return <section className="card"><h3>投票引擎</h3><Skeleton /></section>;
  const next = st.nextWindow ? new Date(Number(st.nextWindow.seconds) * 1000 + st.nextWindow.nanos / 1e6) : null;
  const remain = next ? next.getTime() - now : 0;
  return (
    <section className="card">
      <h3>投票引擎 <span className={st.enabled ? "ok" : "warn"}>{st.enabled ? "在驶" : "停驶"}</span></h3>
      <div className="grid-2">
        <div><div className="dim">下次领奖窗</div><div>{next ? `${fmtCountdown(remain)}（${next.toISOString().slice(11, 16)} UTC）` : "—"}</div></div>
        <div><div className="dim">待领取</div><div>{st.pendingTrx.toFixed(2)} TRX</div></div>
        <div><div className="dim">今日改投</div><div>{st.rebalanceToday} / {st.rebalanceCap} 次</div></div>
        <div><div className="dim">今日领奖</div><div>{st.claimToday} 笔</div></div>
      </div>
      <div className="dim" style={{ marginTop: 8, fontSize: 11 }}>启停=业主径 app_config trx.vote_enabled（鉴权面未落——config 热改径，操作留账）</div>
    </section>
  );
}
