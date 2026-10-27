// Trx.tsx —— TRX 自营盘页（T-0170：三卡原逻辑迁统一 SPA——收益率/对冲结构/
// 统一流水；boot 踢一拍账户刷新，帧驱动照旧 topic=trx）。
import { useEffect } from "react";
import { client } from "../api";
import { TrxYieldRadar } from "../components/TrxYieldRadar";
import { TrxHedgeCard } from "../components/TrxHedgeCard";
import { HedgeFlow } from "../components/HedgeFlow";

export function TrxPage() {
  useEffect(() => {
    void client.trxAccountRefresh({}).catch(() => {});
  }, []);
  return (
    <div className="page">
      <TrxYieldRadar />
      <TrxHedgeCard />
      <HedgeFlow />
    </div>
  );
}
