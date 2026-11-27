// Profit.tsx —— 赚钱页（O-41386 Trx 页同款薄壳组装：页壳只管组件编排+
// ErrorBoundary 隔件，组件各自 useLive/client 取数——KPI 条→持仓盈亏→
// 权益曲线→策略盈亏榜→资金流水→venue 余额，读序自上而下）。
import { ProfitKpiStrip } from "../components/ProfitKpiStrip";
import { StrategyPnlCard } from "../components/StrategyPnlCard";
import { ProfitPositionsCard, ProfitVenueCard } from "../components/profitbits";
import { EquityCard, FlowsCard } from "../components/flowcards";
import { ErrorBoundary } from "../components/ErrorBoundary";

export function ProfitPage() {
  return (
    <div className="page v2">
      <section className="v2-hero">
        <div className="v2-eyebrow">
          <span className="v2-livedot" />
          赚钱怎样 · LIVE
        </div>
        <h1 className="v2-title">赚钱怎样</h1>
      </section>
      {/* 首档：核心指标条——NAV/今日盈亏/30d净/年化率/部署率 */}
      <ErrorBoundary label="核心指标">
        <ProfitKpiStrip />
      </ErrorBoundary>
      {/* 次档：持仓盈亏（O-41385 复用件）→权益曲线 */}
      <ErrorBoundary label="持仓盈亏">
        <ProfitPositionsCard />
      </ErrorBoundary>
      <ErrorBoundary label="权益曲线">
        <EquityCard />
      </ErrorBoundary>
      {/* 三档：策略盈亏榜（ListStrategyPnl）→资金流水→venue 余额 */}
      <ErrorBoundary label="策略盈亏榜">
        <StrategyPnlCard />
      </ErrorBoundary>
      <ErrorBoundary label="资金流水">
        <FlowsCard />
      </ErrorBoundary>
      <ErrorBoundary label="venue 余额">
        <ProfitVenueCard />
      </ErrorBoundary>
    </div>
  );
}
