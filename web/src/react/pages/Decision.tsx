// Decision.tsx —— 决策径页（dashboard-spec v2 三径③：升件→处置链尾流+
// 条件请求榜+教训矿面——「它怎么想」可见；每行指回 ops_event 事实 id，
// 判词 open/age 全后端出）。
import { client } from "../api";
import { useLive } from "../hooks";
import { EmptyState, ErrorState, Skeleton, StateDot } from "../components/ui";
import { AGE } from "../components/panel";
import { pbDate } from "../components/fmt";

export function DecisionPage() {
  const t = useLive(["ev_pipeline"], () => client.getDecisionTrail({ limit: 20 }));

  if (t.loading && !t.data) return <Skeleton rows={6} />;
  if (t.err && !t.data) return <ErrorState msg={t.err} onRetry={t.reload} />;
  const d = t.data!;
  const openN = (d.trail ?? []).filter((r) => r.open).length;
  const m = d.mine;

  return (
    <div className="page v2">
      <section className="v2-hero">
        <div className="v2-eyebrow">
          <span className="v2-livedot" />
          决策径 · LIVE
        </div>
        <h1 className="v2-title">{openN === 0 ? "处置面清爽" : `${openN} 件未决`}</h1>
        <p className="v2-lede">
          近 {(d.trail ?? []).length} 升件 · 教训矿 pending{" "}
          {Number(m?.engPending ?? 0n) + Number(m?.econPending ?? 0n)}
        </p>
      </section>

      {/* 升件→处置链——unified panel：open 行灯黄，已销行灯绿 */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">升件 · 处置链</h2>
          <span className="ink-3">ops_event 事实链</span>
        </div>
        {(d.trail ?? []).length === 0 ? (
          <EmptyState title="无升件" hint="探测升件落地后在此亮行" />
        ) : (
          <div className="v2-rows">
            {(d.trail ?? []).map((r) => (
              <div key={String(r.escId)} className="v2-row">
                <StateDot
                  state={r.open ? 1 : 0}
                  breathe={r.open}
                  title={r.open ? "未决" : `处置 #${r.dispId}`}
                />
                <div className="v2-row-main">
                  <span className="v2-row-title">
                    {r.escClass}/{r.escScope}
                    <span className="ink-3 v2-row-id">#{String(r.escId)}</span>
                  </span>
                  <span className="v2-row-note">{r.note}</span>
                  <span className="v2-row-sub ink-3">
                    {r.escTs ? AGE(pbDate(r.escTs)!) : "—"} 升
                    {r.open
                      ? ` · 候 ${Math.floor(Number(r.ageS) / 60)}m`
                      : ` · ${Math.floor(Number(r.ageS) / 60)}m 后销 · ${r.verdict}`}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 条件请求榜——业主域请求开合态 */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">条件请求</h2>
          <span className="ink-3">30d</span>
        </div>
        {(d.condreqs ?? []).length === 0 ? (
          <EmptyState title="无条件请求" hint="业主条件请求落地后在此亮行" />
        ) : (
          <div className="v2-rows">
            {(d.condreqs ?? []).map((r) => (
              <div key={String(r.id)} className="v2-row">
                <StateDot state={r.open ? 1 : 3} title={r.open ? "未决" : "已结"} />
                <div className="v2-row-main">
                  <span className="v2-row-title">
                    请求<span className="ink-3 v2-row-id">#{String(r.id)}</span>
                  </span>
                  <span className="v2-row-note">{r.note}</span>
                  <span className="v2-row-sub ink-3">
                    {r.ts ? AGE(pbDate(r.ts)!) : "—"}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 教训矿面——eng/econ 状态计数 */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">教训矿</h2>
          <span className="ink-3">fin_*_lesson</span>
        </div>
        <div className="v2-funnel">
          <div className="v2-funnel-lv">
            <span className="v2-funnel-k ink-3">工程</span>
            <span className="v2-funnel-n num">{String(m?.engPending ?? 0n)}</span>
            <span className="v2-funnel-note ink-3">
              pending · canon {String(m?.engCanon ?? 0n)} · verdicted{" "}
              {String(m?.engVerdicted ?? 0n)}
            </span>
          </div>
          <div className="v2-funnel-lv">
            <span className="v2-funnel-k ink-3">经济</span>
            <span className="v2-funnel-n num">{String(m?.econPending ?? 0n)}</span>
            <span className="v2-funnel-note ink-3">pending</span>
          </div>
        </div>
      </div>
    </div>
  );
}
