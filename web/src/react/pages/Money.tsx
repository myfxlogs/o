// Money.tsx —— 钱径页（v2 三径①，单源律：本页只答「商业链路卡哪」——
// 漏斗四级+拒因+哑弹为本页独有；NAV/今日累计归总览、净值曲线/现金流归
// 复盘、在持订单归总览，指路不复读。判词/转化全后端出，前端零计算）。
import { client } from "../api";
import { useLive } from "../hooks";
import { EmptyState, ErrorState, Skeleton } from "../components/ui";
import { AGE } from "../components/panel";
import { pbDate, fmtUsd } from "../components/fmt";

export function MoneyPage() {
  const fn = useLive(["ev_pipeline", "ev_position"], () =>
    client.getFunnel({ windowDays: 30 })
  );

  if (fn.loading && !fn.data) return <Skeleton rows={6} />;
  if (fn.err && !fn.data) return <ErrorState msg={fn.err} onRetry={fn.reload} />;
  const f = fn.data!;
  const lv = (k: string) => f.levels?.find((l) => l.key === k);
  const posN = lv("position")?.n ?? -1n;

  return (
    <div className="page v2">
      <section className="v2-hero">
        <div className="v2-eyebrow">
          <span className="v2-livedot" />
          链在哪堵 · LIVE
        </div>
        <h1 className="v2-title">
          {posN >= 0n ? `${posN} 笔仓位（30d）` : "产出链 30d"}
        </h1>
        <p className="v2-lede num">
          评估 {lv("eval")?.n.toString() ?? "—"} → 提案 {lv("proposal")?.n.toString() ?? "—"} → 策略{" "}
          {lv("strategy")?.n.toString() ?? "—"} → 仓位 {lv("position")?.n.toString() ?? "—"}
        </p>
      </section>

      {/* 漏斗四级——窗内吞吐+现态在册（note=服务端判词注） */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">评估漏斗</h2>
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

      {/* 哑弹榜——live≥7d 零仓（live_dud 探针同源判据） */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">哑弹榜</h2>
          <span className="ink-3">{(f.duds ?? []).length} 枚</span>
        </div>
        {(f.duds ?? []).length === 0 ? (
          <EmptyState title="无哑弹" hint="live 超 7 天零仓策略在此显形" />
        ) : (
          <div className="v2-rows">
            {(f.duds ?? []).map((d) => (
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

      {/* 指路行——单源件他页寻（不复读） */}
      <div className="v2-linkrow ink-3">
        在持订单与 NAV 见 <a href="/">机赚如何</a> · 净值曲线与现金流见{" "}
        <a href="/review">钱怎么走的</a>
      </div>
    </div>
  );
}
