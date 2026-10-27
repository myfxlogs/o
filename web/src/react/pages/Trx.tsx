// Trx.tsx —— TRX 自营盘页（T-0170：三卡原逻辑迁统一 SPA——收益率/对冲结构/
// 统一流水；boot 踢一拍账户刷新，帧驱动照旧 topic=trx）。
import { useEffect } from "react";
import { client } from "../api";
import { TrxYieldRadar } from "../components/TrxYieldRadar";
import { TrxHedgeCard } from "../components/TrxHedgeCard";
import { TrxVoteEngine } from "../components/TrxVoteEngine";
import { HedgeFlow } from "../components/HedgeFlow";
import { ErrorBoundary } from "../components/ErrorBoundary";

export function TrxPage() {
  useEffect(() => {
    void client.trxAccountRefresh({}).catch(() => {});
  }, []);
  return (
    <div className="page">
      {/* 业主令：收益率雷达置顶；卡级错误边界——单卡故障不连坐 */}
      <ErrorBoundary label="收益率雷达">
        <TrxYieldRadar />
      </ErrorBoundary>
      <ErrorBoundary label="投票引擎">
        <TrxVoteEngine />
      </ErrorBoundary>
      <ErrorBoundary label="对冲结构">
        <TrxHedgeCard />
      </ErrorBoundary>
      <ErrorBoundary label="统一流水">
        <HedgeFlow />
      </ErrorBoundary>
    </div>
  );
}
