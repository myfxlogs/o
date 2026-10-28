// TrxHedgeCard.tsx —— 对冲结构卡（arb 同名件借形，D-212 对冲工作台）：
// 镜像双腿表（开仓价/现价/浮动盈亏/杠杆/强平价/距强平%）+净敞口徽章+carry
// 统计条+双账户块+↻ 刷新钮。arb 原型 setTimeout(2s) 等待拍=前端人为延时——
// NOX 改流帧驱动：kick trxAccountRefresh 后等 onData() trx 帧
// 重取 getHedgeStructure（流先零定时器令，票面明令）。
import { useCallback, useEffect, useState } from "react";
import { client } from "../api";
import { onData } from "../stream";
import { TrxVoteEngineRows } from "./TrxVoteEngine";
import { Skeleton } from "./ui";
import type { HedgeStructure } from "../../gen/dashboardv1/core_pb";
import { fmtUsd, fmtNum, fmtPrice, fmtThr, pnlClass, fmtSigned } from "./fmt";

export function TrxHedgeCard() {
  const [structure, setStructure] = useState<HedgeStructure | null>(null);
  const [trxRefreshing, setTrxRefreshing] = useState(false);

  const refetch = useCallback(async () => {
    try {
      const hs = await client.getHedgeStructure({});
      if (hs.structure) setStructure(hs.structure);
    } catch { /* 体感件静默败——下帧自然回填 */ }
  }, []);

  // 挂载首取+trx 流帧重取（arb setTimeout(2s) 等待拍→流帧驱动）。
  useEffect(() => {
    void refetch();
    return onData((v) => {
      if ((v as { topic: string }).topic === "trx") void refetch();
    });
  }, [refetch]);
  // T-0173 业主令：CMC 报价 30s 自动重抓（此前只靠 ev_cashflow 触发——
  // CMC 价格变动不触发 PG NOTIFY，卡片价格分钟级陈旧）
  useEffect(() => {
    const t = setInterval(() => void refetch(), 30_000);
    return () => clearInterval(t);
  }, [refetch]);

  const ready = structure && structure.mid > 0;
  const legSpot = structure?.legSpot;
  const legShort = structure?.legShort;
  const accts = structure?.accts;

  return (
    <section className="card">
      <div className="card-h">
        <h2>对冲结构</h2>
        <span style={{ marginLeft: "auto", display: "inline-flex", alignItems: "center", gap: 6 }}
          title={structure?.wsConnected ? "CMC WS 在线（LiveState 30s 新鲜度验证）" : "REST 兜底——CMC WS 断/陈"}>
          {/* WS 灯挂共用珠体（T-0222 统一）：在线=s0 breathe（breatheCalm 2.4s），
              断流=s1 still（静止珠——呼吸=活性，降级不装活）。 */}
          <span className={"dot " + (structure?.wsConnected ? "s0 breathe" : "s1 still")} />
          <span style={{ fontSize: 12, color: "var(--ink-2)" }}>
            {structure?.wsConnected ? "WS 在线" : "REST 兜底"}
          </span>
        </span>
        {ready && structure!.platformAlertState && structure!.platformAlertState !== "normal" && (
          <span className="ws-dot-wrap" title={`平台预警: ${structure!.platformAlertState}`}>
            <span className="ws-dot ws-dot-warn" />
            <span className="muted">{structure!.platformAlertState === "breach" ? "平台强平触碰" : "平台预警"}</span>
          </span>
        )}
      </div>
      {!ready && <Skeleton rows={4} />}
      {ready && (
        <>
          <div className="table-scroll scroll-cap">
            <table className="rows">
              <thead>
                <tr><th>腿</th><th>数量</th><th>开仓价</th><th>现价</th><th>浮动盈亏</th><th>杠杆</th><th>强平价</th><th>距强平%</th></tr>
              </thead>
              <tbody>
                <tr>
                  <td>{legSpot?.label ?? "质押现货"}</td>
                  <td>质押 {fmtNum(legSpot?.qtyStaked ?? 0)} TRX</td>
                  <td>{fmtPrice(structure!.baselinePrice)}</td>
                  <td>{fmtPrice(structure!.ask || structure!.mid)}</td>
                  <td className={pnlClass(legSpot?.pnlVsBase ?? 0, true)}>{fmtUsd(legSpot?.pnlVsBase ?? 0)}</td>
                  <td>—</td><td>—</td><td>—</td>
                </tr>
                <tr>
                  <td>{legShort?.label ?? "CMC 空单"}</td>
                  <td>{fmtNum(legShort?.qty ?? 0)} TRX</td>
                  <td>{fmtPrice(structure!.baselinePrice)}</td>
                  <td>{fmtPrice(structure!.ask || structure!.mid)}</td>
                  <td className={pnlClass(legShort?.pnlVsBase ?? 0, true)}>{fmtUsd(legShort?.pnlVsBase ?? 0)}</td>
                  <td>{(legShort?.lev ?? 0) > 0 ? `${legShort!.lev}×` : "—"}</td>
                  <td>{(legShort?.liq ?? 0) > 0 ? (
                    <span title={structure!.closeoutSource === "platform_rule_50"
                      ? `平台口径 · 50% 规则\nCMC Balance 面板标定：强平=保证金 50%，预警=保证金 80%\n强平净值 ${fmtUsd(structure!.closeoutLevelUsd)}\n预警净值 ${fmtUsd(structure!.warningLevelUsd)}`
                      : undefined}>{fmtThr(legShort!.liq)}</span>
                  ) : "—"}</td>
                  <td className={(legShort?.distPct ?? 0) < 0 ? "neg" : ""}>
                    {(legShort?.distPct ?? 0) > 0 ? legShort!.distPct.toFixed(2) + "%" : "—"}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div className="banner" role="status">
            <span>
              净敞口 <strong>{fmtSigned(structure!.netExposureTrx)} TRX</strong>
              （≈ {fmtUsd(structure!.netExposureUsd)} USD）
              <span className="muted" style={{ marginLeft: 8 }}>· 近似中性</span>
            </span>
          </div>
          {/* carry 统计条迁收益率卡（业主裁定——carry=收益事实非结构事实，归收益率卡域） */}
          {accts && (
            <div className="hedge-accts" role="group" aria-label="双账户摘要">
              <div className="hedge-acct-block">
                <h3 className="muted">TRX 账户</h3>
                <div>余额 {fmtNum(accts.trxBalance, 3)} TRX</div>
                {/* 待领取归投票引擎卡（同 rewardNum 源——T-0173 业主⑳能减尽减） */}
                <div>能量委托 {fmtNum(accts.energyDelegated, 3)} TRX</div>
                <div>票权 {fmtNum(accts.votes)}</div>
              </div>
              <div className="hedge-acct-block">
                <h3 className="muted">投票引擎</h3>
                <TrxVoteEngineRows />
                {accts.netLimit > 0 && (
                  <div className="muted" title={`带宽上限 ${fmtNum(accts.netLimit + accts.freeNetLimit, 0)} BP`}>
                    剩余带宽 {fmtNum((accts.freeNetLimit - accts.freeNetUsed) + (accts.netLimit - accts.netUsed), 0)} BP
                    {accts.bandwidthStatus === "red" && <span className="u-bad"> · 红</span>}
                    {accts.bandwidthStatus === "yellow" && <span className="u-orange"> · 黄</span>}
                  </div>
                )}
                {accts.energyLimit > 0 && (
                  <div className="muted" title={`能量上限 ${fmtNum(accts.energyLimit, 0)} Energy`}>
                    剩余能量 {fmtNum(accts.energyLimit - accts.energyUsed, 0)} Energy
                  </div>
                )}
              </div>
              <div className="hedge-acct-block">
                <h3 className="muted">CMC 账户</h3>
                <div>现金 {fmtUsd(accts.cmcCash)} USD</div>
                <div>净值 {fmtUsd(accts.cmcEquity)} USD</div>
                <div className="muted">强平净值 {fmtUsd(structure!.closeoutLevelUsd)} / 预警净值 {fmtUsd(structure!.warningLevelUsd)}</div>
                <div className="muted" title="平台口径 · 50% 规则：CMC Balance 面板标定，强平=保证金 50%，预警=保证金 80%；规则变更后需复核">口径：强平=保证金 50%，预警=80%（平台标定）</div>
              </div>
            </div>
          )}
        </>
      )}
    </section>
  );
}
