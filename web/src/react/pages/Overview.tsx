// Overview.tsx —— 总览（T-0170 第一性首屏：30 秒答三问——赚亏多少/仓什么
// 状态/系统活着没；2026-11-11 v2 骨架迁：hero 判词大字→健康条→在持→
// 最近动作→钱在哪 线性读序，duo 双卡墙撤——数据全走 Connect）。
import { client } from "../api";
import { useLive } from "../hooks";
import { EmptyState, ErrorState, Skeleton, StateDot, UtcClock } from "../components/ui";
import { EarnToggle } from "../components/EarnToggle";
import { OpenPositionsCard } from "../components/flows";
import { fmtUsd, pbDate } from "../components/fmt";
import { AGE } from "../components/panel";
import { PagerBar, usePager } from "../components/pager";
import { useState } from "react";
import type { MoneyRow } from "../../gen/dashboardv1/core_pb";
import { VENUE_CN } from "../components/labels";

const money = (v: number) =>
  "$" +
  v.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

const signed = (v: number) => (v >= 0 ? "+" : "−") + money(Math.abs(v));


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

export function OverviewPage() {
  const ov = useLive(["ev_position", "ev_cashflow", "ev_account"], () =>
    client.overview({})
  );
  const health = useLive(["ev_account"], () => client.getSystemHealth({}));
  // 链堵面（业主令 11-11：链在哪堵并入本页——漏斗+哑弹榜单源迁此，钱径页退役）
  const fn = useLive(["ev_pipeline", "ev_position"], () =>
    client.getFunnel({ windowDays: 30 })
  );
  const dudPager = usePager(fn.data?.duds ?? [], 8);

  if (ov.loading && !ov.data) return <Skeleton rows={6} />;
  if (ov.err && !ov.data)
    return <ErrorState msg={ov.err} onRetry={ov.reload} />;
  const o = ov.data!;
  const f = fn.data;

  const worst = Math.max(
    0,
    ...(health.data?.components ?? []).map((c) => c.state)
  );
  const doing = o.doing ?? [];
  const healthTxt =
    worst === 0 ? "系统全绿" : worst === 1 ? "有组件节拍偏慢" : "有组件断流";

  return (
    <div className="page v2">
      {/* Hero：三问之一——赚亏多少（判词大字=NAV,副注=今日/累计/健康） */}
      <section className="v2-hero">
        <div className="v2-eyebrow">
          <span className="v2-livedot" />
          机赚如何 · LIVE
        </div>
        <h1 className="v2-title num">{money(o.navTotal)}</h1>
        <p className="v2-lede num">
          今日 {signed(o.pnlToday)} · 累计 {signed(o.pnlCum)} · {healthTxt}
        </p>
      </section>

      {/* 三问之三——系统一句话健康（点进哪病了） */}
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

      {/* 链在哪堵（业主令 11-11 并入——评估漏斗吞吐+拒因+哑弹榜分页榜） */}
      {f ? (
        <div className="v2-cols">
          <div className="v2-panel">
            <div className="v2-panel-head">
              <h2 className="v2-panel-title">链在哪堵 · 评估漏斗</h2>
              <span className="ink-3">{f.windowDays}d 窗</span>
            </div>
            <div className="v2-funnel">
              {(f.levels ?? []).map((l) => (
                <div key={l.key} className="v2-funnel-lv">
                  <span className="v2-funnel-k ink-3">{l.label}</span>
                  <span className={"v2-funnel-n num" + (l.n < 0n ? " ink-3" : "")}>
                    {l.n < 0n ? "缺" : Number(l.n).toLocaleString("zh-CN")}
                  </span>
                  <span className="v2-funnel-note ink-3">{l.note}</span>
                </div>
              ))}
            </div>
            {(f.rejects ?? []).length > 0 && (
              <div className="v2-funnel-rej ink-3">
                拒因：{(f.rejects ?? []).map((r) => `${r.reason}·${r.n}`).join("　")}
              </div>
            )}
          </div>

          <div className="v2-panel">
            <div className="v2-panel-head">
              <h2 className="v2-panel-title">哑弹榜</h2>
              <PagerBar {...dudPager} onPage={dudPager.setPage} />
            </div>
            {(f.duds ?? []).length === 0 ? (
              <EmptyState title="无哑弹" hint="live 超 7 天零仓策略在此显形" />
            ) : (
              <div className="v2-rows">
                {dudPager.slice.map((d) => (
                  <div key={String(d.id)} className="v2-row">
                    <div className="v2-row-main">
                      <span className="v2-row-title">
                        #{String(d.id)} {d.instId}
                      </span>
                      <span className="v2-row-sub ink-3">
                        {d.templateRef || "—"} · 名义 {fmtUsd(d.notionalUsd)} · 入场{" "}
                        {pbDate(d.enteredAt) ? AGE(pbDate(d.enteredAt)!) : "—"}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      ) : null}

      {/* 最近动作+钱在哪 1行两列（业主令 11-11：v2-cols 自适应多列栅格） */}
      <div className="v2-cols">
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">最近动作</h2>
          <UtcClock />
        </div>
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
      </div>

      {/* 钱在哪——venue 树（读序第五=证据层；Earn 开关贴展示处 T-0342） */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">钱在哪</h2>
          <span className="ink-3">{fmtUsd(o.navTotal)}</span>
        </div>
        {(o.money ?? []).length === 0 ? (
          <EmptyState title="读件缺——venue 行缺席" hint="账户面恢复后这里列出每一块钱" />
        ) : (
          <MoneyTree rows={o.money ?? []} />
        )}
        {/* T-0342：Earn 自动申购开关——贴 Earn 展示处（钱在哪卡） */}
        <EarnToggle />
      </div>
      </div>
    </div>
  );
}
