// Trx.tsx —— TRX 自营盘页（T-0170 三卡迁统一 SPA；T-0220c 视觉层级三档重排：
// 核心指标条（大数字优先）→次级瓦片（雷达‖对冲双列）→明细层（统一流水全宽）；
// boot 踢一拍账户刷新，帧驱动照旧 topic=trx）。
import { useEffect } from "react";
import { client } from "../api";
import { TrxKpiStrip } from "../components/TrxKpiStrip";
import { TrxYieldRadar } from "../components/TrxYieldRadar";
import { TrxHedgeCard } from "../components/TrxHedgeCard";
import { HedgeFlow } from "../components/HedgeFlow";
import { ErrorBoundary } from "../components/ErrorBoundary";

export function TrxPage() {
  useEffect(() => {
    void client.trxAccountRefresh({}).catch(() => {});
  }, []);
  return (
    <div className="page">
      {/* 首档：核心指标条——组合年化×2/待领取/强平距离（大数字 2em+呼吸灯） */}
      <ErrorBoundary label="核心指标">
        <TrxKpiStrip />
      </ErrorBoundary>
      {/* 次档：收益率雷达‖对冲结构双列并排（.duo 先例，窄屏回落单列） */}
      <div className="duo">
        <ErrorBoundary label="收益率">
          <TrxYieldRadar />
        </ErrorBoundary>
        <ErrorBoundary label="对冲结构">
          <TrxHedgeCard />
        </ErrorBoundary>
      </div>
      {/* 三档：统一流水——全宽明细（13px） */}
      <ErrorBoundary label="统一流水">
        <HedgeFlow />
      </ErrorBoundary>
    </div>
  );
}
