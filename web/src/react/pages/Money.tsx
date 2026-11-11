// Money.tsx —— 钱径页（dashboard-spec v2 三径①：净值/今日累计→漏斗四级→
// 哑弹榜——判词/转化全后端出，前端零计算零词表；unified panel+hairlines
// 承 apple-liquid-glass skill：同级行一面板，禁碎卡墙）。
import { client } from "../api";
import { useLive } from "../hooks";
import { Card, EmptyState, ErrorState, Skeleton } from "../components/ui";
import { OpenPositionsCard } from "../components/flows";
import { LineChart } from "../components/chart";
import { AGE } from "../components/panel";
import { pbDate, fmtUsd } from "../components/fmt";

const money = (v: number) =>
  (v >= 0 ? "+" : "") +
  "$" +
  v.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const pnlCls = (v: number) => (v > 0 ? "u-good" : v < 0 ? "u-bad" : "");

export function MoneyPage() {
  const ov = useLive(["ev_position", "ev_cashflow", "ev_account"], () =>
    client.overview({})
  );
  const eq = useLive(["ev_cashflow"], () => client.getEquityCurve({ days: 90 }));
  const fn = useLive(["ev_pipeline", "ev_position"], () =>
    client.getFunnel({ windowDays: 30 })
  );

  if (ov.loading && !ov.data) return <Skeleton rows={6} />;
  if (ov.err && !ov.data) return <ErrorState msg={ov.err} onRetry={ov.reload} />;
  const o = ov.data!;
  const pts = eq.data?.points ?? [];
  const f = fn.data;

  return (
    <div className="page v2">
      <section className="v2-hero">
        <div className="v2-eyebrow">
          <span className="v2-livedot" />
          钱径 · LIVE
        </div>
        <h1 className="v2-title">
          {fmtUsd(o.navTotal)}
        </h1>
        <p className="v2-lede num">
          今日 <span className={pnlCls(o.pnlToday)}>{money(o.pnlToday)}</span>
          {" · "}累计 <span className={pnlCls(o.pnlCum)}>{money(o.pnlCum)}</span>
        </p>
      </section>

      {/* 净值曲线——90d 采点（equity 读面同源） */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">净值曲线</h2>
          <span className="ink-3">90d</span>
        </div>
        {eq.err && !eq.data ? (
          <ErrorState msg={eq.err} onRetry={eq.reload} />
        ) : pts.length === 0 ? (
          <EmptyState title="采点缺" hint="equity 采点落地后在此亮线" />
        ) : (
          <LineChart
            labels={pts.map((p) => pbDate(p.day)?.toISOString().slice(5, 10) ?? "")}
            xs={pts.map((p) => (pbDate(p.day)?.getTime() ?? 0) / 1000)}
            ys={pts.map((p) => p.nav)}
            height={200}
            fmtY={(v) => "$" + v.toLocaleString("zh-CN", { maximumFractionDigits: 0 })}
          />
        )}
      </div>

      {/* 漏斗四级——窗内吞吐+现态在册（note=服务端判词注） */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">评估漏斗</h2>
          <span className="ink-3">{f?.windowDays ?? 30}d 窗</span>
        </div>
        {fn.err && !fn.data ? (
          <ErrorState msg={fn.err} onRetry={fn.reload} />
        ) : !f ? (
          <Skeleton rows={4} />
        ) : (
          <>
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
          </>
        )}
      </div>

      {/* 哑弹榜——live≥7d 零仓（live_dud 探针同源；unified panel+hairlines） */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">哑弹榜</h2>
          <span className="ink-3">{(f?.duds ?? []).length} 枚</span>
        </div>
        {(f?.duds ?? []).length === 0 ? (
          <EmptyState title="无哑弹" hint="live 超 7 天零仓策略在此显形" />
        ) : (
          <div className="v2-rows">
            {(f?.duds ?? []).map((d) => (
              <div key={String(d.id)} className="v2-row">
                <div className="v2-row-main">
                  <span className="v2-row-title">
                    #{d.id} {d.instId}
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

      {/* 在持——flows 件复用（doing 读面同源） */}
      <OpenPositionsCard doing={o.doing ?? []} />
    </div>
  );
}
