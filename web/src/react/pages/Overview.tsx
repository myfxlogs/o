// Overview.tsx —— 总览（T-0170 第一性首屏：30 秒答三问——赚亏多少/仓什么
// 状态/系统活着没；hero 大数字+在持+健康汇总+最近动作，数据全走 Connect）。
import { client } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton, StateDot, UtcClock } from "../components/ui";
import { EarnToggle } from "../components/EarnToggle";
import { OpenPositionsCard } from "../components/flows";
import { fmtUsd } from "../components/fmt";
import { useState } from "react";
import type { MoneyRow } from "../../gen/dashboardv1/core_pb";
import { VENUE_CN } from "../components/labels";

const money = (v: number) =>
  "$" +
  v.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });


// MoneyTree 钱在哪树（业主回单②：按平台树状——okx 多户收拢树下，点击展开
// 看明细行；总额=树头常显不藏）。
function MoneyTree(props: { rows: MoneyRow[] }) {
  const [open, setOpen] = useState<Record<string, boolean>>({});
  const groups = new Map<string, MoneyRow[]>();
  for (const m of props.rows) {
    const v = m.venue || "其他";
    if (!groups.has(v)) groups.set(v, []);
    groups.get(v)!.push(m);
  }
  return (
    <div className="tree">
      {[...groups.entries()].map(([venue, rs]) => {
        const total = rs.reduce((a, r) => a + r.usd, 0);
        const isOpen = !!open[venue];
        return (
          <div key={venue} className="tree-node">
            <button
              className="tree-head"
              onClick={() => setOpen((o) => ({ ...o, [venue]: !o[venue] }))}
              aria-expanded={isOpen}
            >
              <span className={"caret" + (isOpen ? " open" : "")}>▸</span>
              <span className="tree-name">{VENUE_CN[venue] ?? venue}</span>
              <span className="ink-3 tree-n">{rs.length} 行</span>
              <span className="num mono tree-total">{fmtUsd(total)}</span>
            </button>
            {isOpen && (
              <table className="rows">
                <tbody>
                  {rs.map((m, i) => (
                    <tr key={i}>
                      <td>{m.label}</td>
                      <td className="num mono">{m.note ? m.note : fmtUsd(m.usd)}</td>
                      <td className="ink-3">{m.section}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        );
      })}
    </div>
  );
}

function pnlCls(v: number): string {
  return v > 0 ? "u-good" : v < 0 ? "u-bad" : "";
}

export function OverviewPage() {
  const ov = useLive(["ev_position", "ev_cashflow", "ev_account"], () =>
    client.overview({})
  );
  const health = useLive(["ev_account"], () => client.getSystemHealth({}));

  if (ov.loading && !ov.data) return <Skeleton rows={6} />;
  if (ov.err && !ov.data)
    return <ErrorState msg={ov.err} onRetry={ov.reload} />;
  const o = ov.data!;

  const worst = Math.max(
    0,
    ...(health.data?.components ?? []).map((c) => c.state)
  );
  const doing = o.doing ?? [];

  return (
    <div className="page">
      {/* Hero：三问之一——赚亏多少 */}
      <section className="hero">
        <div className="hero-item">
          <span className="hero-k">净值</span>
          <span className="hero-v">{money(o.navTotal)}</span>
        </div>
        <div className="hero-item">
          <span className="hero-k">今日</span>
          <span className={"hero-v " + pnlCls(o.pnlToday)}>
            {o.pnlToday >= 0 ? "+" : ""}
            {money(o.pnlToday)}
          </span>
        </div>
        <div className="hero-item">
          <span className="hero-k">累计</span>
          <span className={"hero-v " + pnlCls(o.pnlCum)}>
            {o.pnlCum >= 0 ? "+" : ""}
            {money(o.pnlCum)}
          </span>
        </div>
      </section>

      {/* 三问之三——系统一句话健康（点进系统页） */}
      <a className="health-strip" href="/device">
        <StateDot state={worst} />
        <span>
          {worst === 0
            ? "系统全部组件正常"
            : worst === 1
              ? "系统有组件节拍偏慢——点看明细"
              : "系统有组件断流/缺席——点看明细"}
        </span>
        <span className="ink-3">›</span>
      </a>

      {/* 三问之二——在持订单卡（业主令两瓦片合一+全宽：exec 持仓结构表+腿明细
          +venue 直读补充行+待命部署折行；腿表九列要全宽，出瀑布单列） */}
      {ov.err ? <ErrorState msg={ov.err} onRetry={ov.reload} /> : null}
      <OpenPositionsCard doing={doing} />

      {/* 最近动作+钱在哪同行双列（业主令：两卡同排不拆行——duo 栅格强制并排，
          窄屏回落单列） */}
      <div className="duo">
      <Card title="最近动作" extra={<UtcClock />}>
        {(o.recent ?? []).length === 0 ? (
          <EmptyState title="账面平静" hint="现金流落地后在此亮行" />
        ) : (
          <div className="feed">
            {(o.recent ?? []).map((s, i) => (
              <div key={i} className="feed-row mono">
                {s}
              </div>
            ))}
          </div>
        )}
      </Card>

      <Card title="钱在哪" extra={<span className="ink-3">{fmtUsd(o.navTotal)}</span>}>
        {(o.money ?? []).length === 0 ? (
          <EmptyState title="读件缺——venue 行缺席" hint="账户面恢复后这里列出每一块钱" />
        ) : (
          <MoneyTree rows={o.money ?? []} />
        )}
        {/* T-0342：Earn 自动申购开关——贴 Earn 展示处（钱在哪卡） */}
        <EarnToggle />
      </Card>
      </div>
    </div>
  );
}
