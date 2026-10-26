// trx.tsx —— TRX 页 arb TrxPage 壳逐字借形（裸 div 三区零页头——撤 .page
// 包裹+h1）：收益率（赚多少）→对冲结构（安全吗）→流水（发生了什么）。
// 监控提醒区撤（T-0156：watch_rule 后端面未落位——假功能下架，UI 不得
// 承诺系统能力外的事；后端落位后同 proto 恢复）。
// D-254 on-open kick=trxAccountRefresh 双踢（体感优化静默败）。
import { useEffect } from "react";
import { client } from "../api";
import { TrxYieldRadar } from "../components/TrxYieldRadar";
import { TrxHedgeCard } from "../components/TrxHedgeCard";
import { HedgeFlow } from "../components/HedgeFlow";

export default function TrxPage() {
  useEffect(() => {
    void client.trxAccountRefresh({}).catch(() => {});
  }, []);

  return (
    <div>
      <TrxYieldRadar />
      <TrxHedgeCard />
      <HedgeFlow />
    </div>
  );
}
