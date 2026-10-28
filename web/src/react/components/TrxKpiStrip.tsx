// TrxKpiStrip.tsx —— TRX 核心指标条（T-0220c 三档层级首档：组合 7d/30d 年化+
// 待领取+强平距离——大数字 2em+呼吸灯伴随+独占首行）。灯语零新前端判据：
// 强平距离灯随服务端 platformAlertState（normal=绿缓息/warn=黄/breach=红静止）；
// 年化/待领取=正负号显色（现役 u-good 先例）。挂载取数+trx 信号帧重取
// （径同现役卡——曲线/引擎/结构面未载荷化）。
import { useCallback, useEffect, useState } from "react";
import { client } from "../api";
import { onData } from "../stream";
import { StateDot } from "./ui";

type KpiState = {
  annualized7d: number;
  annualized30d: number;
  sample7d: number;
  sample30d: number;
  pendingTrx: number;
  distPct: number | null;
  alert: string;
};

const fmtPct = (v: number) => (v > 0 ? `${v.toFixed(2)}%` : "—");

function Kpi(props: {
  label: string;
  value: string;
  unit?: string;
  cls?: string;
  dot: { state: number; breathe: boolean };
  dotTitle: string;
  note?: string;
}) {
  return (
    <section className="card kpi" role="group" aria-label={props.label}>
      <div className="kpi-head">
        <span className="kpi-k">{props.label}</span>
        <StateDot state={props.dot.state} breathe={props.dot.breathe} title={props.dotTitle} />
      </div>
      <div className={"kpi-v" + (props.cls ? " " + props.cls : "")}>
        {props.value}
        {props.unit && <span className="kpi-unit">{props.unit}</span>}
      </div>
      {props.note && <div className="kpi-note">{props.note}</div>}
    </section>
  );
}

export function TrxKpiStrip() {
  const [kpi, setKpi] = useState<KpiState | null>(null);
  const [err, setErr] = useState(false);

  const fetchAll = useCallback(async () => {
    try {
      const [curve, vote, hedge] = await Promise.all([
        client.getTrxYieldCurve({ days: 30 }),
        client.getTrxVoteStatus({}),
        client.getHedgeStructure({}),
      ]);
      setKpi({
        annualized7d: curve.annualized7d,
        annualized30d: curve.annualized30d,
        sample7d: curve.sampleDays7d,
        sample30d: curve.sampleDays30d,
        pendingTrx: vote.pendingTrx,
        distPct: hedge.structure?.legShort?.distPct ?? null,
        alert: hedge.structure?.platformAlertState || "normal",
      });
      setErr(false);
    } catch {
      setErr(true);
    }
  }, []);

  useEffect(() => {
    void fetchAll();
    return onData((v) => {
      if ((v as { topic: string }).topic === "trx") void fetchAll();
    });
  }, [fetchAll]);

  if (err && !kpi) {
    return (
      <div className="trx-kpi-strip">
        <section className="card kpi kpi-err">核心指标读败——下一流帧自动回填</section>
      </div>
    );
  }
  if (!kpi) return null; // 首拍未落=不占位闪（次级瓦片骨架已示加载）
  const cold7d = kpi.sample7d > 0 && kpi.sample7d < 7;
  const cold30d = kpi.sample30d > 0 && kpi.sample30d < 30;
  const alertState = kpi.alert === "breach" ? 2 : kpi.alert === "warn" ? 1 : 0;
  return (
    <div className="trx-kpi-strip">
      <Kpi
        label="组合 7d 年化"
        value={fmtPct(kpi.annualized7d)}
        cls={kpi.annualized7d > 0 ? "u-good" : ""}
        dot={{ state: kpi.annualized7d > 0 ? 0 : 1, breathe: kpi.annualized7d > 0 }}
        dotTitle="组合年化（能量+投票+隔夜息）"
        note={cold7d ? `样本不足（${kpi.sample7d}d）` : undefined}
      />
      <Kpi
        label="组合 30d 年化"
        value={fmtPct(kpi.annualized30d)}
        cls={kpi.annualized30d > 0 ? "u-good" : ""}
        dot={{ state: kpi.annualized30d > 0 ? 0 : 1, breathe: kpi.annualized30d > 0 }}
        dotTitle="组合年化（能量+投票+隔夜息）"
        note={cold30d ? `样本不足（${kpi.sample30d}d）` : undefined}
      />
      <Kpi
        label="待领取"
        value={kpi.pendingTrx.toFixed(2)}
        unit="TRX"
        cls={kpi.pendingTrx > 0 ? "u-good" : ""}
        dot={{ state: kpi.pendingTrx > 0 ? 0 : 1, breathe: kpi.pendingTrx > 0 }}
        dotTitle="奖励待领取（rewardNum 折值）"
      />
      <Kpi
        label="强平距离"
        value={kpi.distPct != null ? `${kpi.distPct.toFixed(1)}%` : "—"}
        cls={kpi.distPct != null && kpi.distPct <= 0 ? "u-bad" : ""}
        dot={{ state: alertState, breathe: alertState !== 2 }}
        dotTitle={`平台预警态：${kpi.alert}`}
      />
    </div>
  );
}
