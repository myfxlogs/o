// TrxVoteEngine.tsx —— T-0184 投票引擎状态卡（只读显态面：在驶态/下窗
// 倒计时/待领取/今日改投×上限；启停=app_config 业主径不在前端鉴权面
// 内——降级只读+「config 热改径」注，鉴权面落位前零 mutating 控件）。
import { useCallback, useEffect, useState } from "react";
import { client } from "../api";
import { StateDot } from "./ui";
import type { GetTrxVoteStatusResponse } from "../../gen/dashboardv1/core_pb";

const fmtCountdown = (ms: number) => {
  if (ms <= 0) return "即刻";
  const m = Math.floor(ms / 60000);
  const h = Math.floor(m / 60);
  return h > 0 ? `${h}h${m % 60}m` : `${m}m`;
};

// TrxVoteEngineRows 投票引擎行组（业主㉓：撤独立卡——内容并入对冲结构卡
// TRX 账户块；灯语同款：在驶=绿呼吸/停驶=黄静止待令）。
export function TrxVoteEngineRows() {
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
  if (err) {
    return <div className="muted" style={{ fontSize: 12 }}>投票引擎读败：{err}</div>;
  }
  if (!st) return null; // 首拍未落=静默（对冲卡自身骨架已占位）
  const next = st.nextWindow ? new Date(Number(st.nextWindow.seconds) * 1000 + st.nextWindow.nanos / 1e6) : null;
  const remain = next ? next.getTime() - now : 0;
  return (
    <div role="group" aria-label="投票引擎">
      <div>
        {/* 灯语同款：在驶=绿呼吸，停驶=黄静止待令 */}
        <StateDot state={st.enabled ? 0 : 1} breathe={st.enabled} title={st.enabled ? "引擎在驶" : "引擎停驶待令"} />{" "}
        投票引擎 <span className={st.enabled ? "u-good" : "u-orange"}>{st.enabled ? "在驶" : "停驶"}</span>
        {next && <span className="muted"> · 下窗 {fmtCountdown(remain)}（{next.toISOString().slice(11, 16)} UTC）</span>}
      </div>
      <div className="muted" style={{ fontSize: 12 }}>
        待领取 {st.pendingTrx.toFixed(2)} TRX · 今日改投 {st.rebalanceToday}/{st.rebalanceCap} · 今日领奖 {st.claimToday} 笔
      </div>
    </div>
  );
}
