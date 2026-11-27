// profitbits.tsx —— 赚钱页两张 Overview 同源卡（O-41386：持仓盈亏卡=
// O-41385 openpositions.tsx 复用件——doing 补充行由本壳取 Overview 喂；
// venue 余额卡=Overview.money 同读面 MoneyTree——不重建第二套读面）。
import { client } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton } from "./ui";
import { MoneyTree } from "./moneytree";
import { fmtUsd } from "./fmt";
import { OpenPositionsCard } from "./openpositions";

// ProfitPositionsCard 在持订单卡壳——Overview.doing 补充行（venue 直读
// 面）与 ListOpenPositions 主表合一展示（两瓦片合一业主令形不变）；
// overview 读败=doing 空集降级，主表照常（补充行非关键径）。
export function ProfitPositionsCard() {
  const ov = useLive(["ev_position", "ev_cashflow", "ev_account"], () =>
    client.overview({})
  );
  return <OpenPositionsCard doing={ov.data?.doing ?? []} />;
}

// ProfitVenueCard venue 余额卡——Overview.money 三区直挂 MoneyTree
// （与总览页钱在哪同读面同渲染件——venue 树/读件缺行如实显）。
export function ProfitVenueCard() {
  const ov = useLive(["ev_position", "ev_cashflow", "ev_account"], () =>
    client.overview({})
  );
  const rows = ov.data?.money ?? [];
  return (
    <Card
      title="venue 余额"
      extra={<span className="ink-3">{fmtUsd(ov.data?.navTotal ?? 0)}</span>}
    >
      {ov.loading && !ov.data ? (
        <Skeleton rows={4} />
      ) : ov.err && !ov.data ? (
        <ErrorState msg={ov.err} onRetry={ov.reload} />
      ) : rows.length === 0 ? (
        <EmptyState title="读件缺——venue 行缺席" hint="账户面恢复后这里列出每一块钱" />
      ) : (
        <MoneyTree rows={rows} />
      )}
    </Card>
  );
}
