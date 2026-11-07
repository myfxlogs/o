// Onboard.tsx —— 接入页（T-0430 入口归位：broker 接入卡自复盘页第三卡挪
// 顶导航「接入」语义位——venue 接入与复盘语义不搭，业主「完全错误」判；
// venue 板块/侦察榜仍在复盘页不动）。
import { BrokerOnboardCard } from "../components/brokerboard";

export function OnboardPage() {
  return (
    <div className="page">
      <BrokerOnboardCard />
    </div>
  );
}
