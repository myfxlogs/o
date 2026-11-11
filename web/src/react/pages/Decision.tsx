// Decision.tsx —— 决策径页（dashboard-spec v2 三径③：升件→处置链尾流+
// 条件请求榜+教训矿面——「它怎么想」可见；每行指回 ops_event 事实 id，
// 判词 open/age 全后端出）。
import { client } from "../api";
import { useLive } from "../hooks";
import { EmptyState, ErrorState, Skeleton, StateDot } from "../components/ui";
import { AGE } from "../components/panel";
import { pbDate } from "../components/fmt";
import { PagerBar, usePager } from "../components/pager";
import { plainKind, plainText } from "../components/plain";

// ageTxt 秒级龄人话（<120min→Nm / <48h→N.Nh / else→N.Nd——审计 P2：裸分钟不可读）。
const ageTxt = (s: number) =>
  s < 7200 ? `${Math.floor(s / 60)}m` : s < 172800 ? `${(s / 3600).toFixed(1)}h` : `${(s / 86400).toFixed(1)}d`;

export function DecisionPage() {
  // 业主令 11-11：分页防撑高——取数窗 20→100 行,前端切片每页 10
  const t = useLive(["ev_pipeline"], () => client.getDecisionTrail({ limit: 100 }));

  const trailPager = usePager(t.data?.trail ?? [], 10);
  const reqPager = usePager(t.data?.condreqs ?? [], 10);

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
          何事要裁 · LIVE
        </div>
        <h1 className="v2-title">
          {openN === 0 ? "都处理完了" : `最近 ${(d.trail ?? []).length} 条上报，${openN} 条待处理`}
        </h1>
        <p className="v2-lede">
          教训矿 {Number(m?.engPending ?? 0n) + Number(m?.econPending ?? 0n)} 条候选待裁定
        </p>
      </section>

      {/* 升件→处置链——unified panel：open 行灯黄，已销行灯绿 */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">上报 · 处理记录</h2>
          <PagerBar {...trailPager} onPage={trailPager.setPage} />
        </div>
        {(d.trail ?? []).length === 0 ? (
          <EmptyState title="暂无上报" hint="系统上报落账后在此显示" />
        ) : (
          <div className="v2-rows">
            {trailPager.slice.map((r) => (
              <div key={String(r.escId)} className="v2-row">
                <StateDot
                  state={r.open ? 1 : 0}
                  breathe={r.open}
                  title={r.open ? "待处理" : `已处理 #${r.dispId}`}
                />
                <div className="v2-row-main">
                  <span className="v2-row-title">
                    {plainKind(r.escClass, r.escScope, r.note)}
                    <span className="ink-3 v2-row-id">#{String(r.escId)}</span>
                  </span>
                  <span className="v2-row-note" title={r.note}>{plainText(r.note)}</span>
                  <span className="v2-row-sub ink-3" title={r.verdict}>
                    {r.escTs ? AGE(pbDate(r.escTs)!) : "—"} 上报
                    {r.open
                      ? ` · 已等 ${ageTxt(Number(r.ageS))}`
                      : ` · 等了 ${ageTxt(Number(r.ageS))}${r.dispTs ? `，${AGE(pbDate(r.dispTs)!)} 处理完` : ""} · ${plainText(r.verdict)}`}
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
          <PagerBar {...reqPager} onPage={reqPager.setPage} />
        </div>
        {(d.condreqs ?? []).length === 0 ? (
          <EmptyState title="暂无待办请求" hint="你设的条件请求会在此显示" />
        ) : (
          <div className="v2-rows">
            {reqPager.slice.map((r) => (
              <div key={String(r.id)} className="v2-row">
                <StateDot state={r.open ? 1 : 3} title={r.open ? "待处理" : "已完成"} />
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

      {/* 教训矿面——canon 入库数=产出主数，裁定/待审作副数 */}
      <div className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">教训矿</h2>
          <span className="ink-3">踩过的坑 → 经验条目</span>
        </div>
        <div className="v2-funnel">
          <div className="v2-funnel-lv">
            <span className="v2-funnel-k ink-3">工程教训 · 已入库</span>
            <span className="v2-funnel-n num">{String(m?.engCanon ?? 0n)}</span>
            <span className="v2-funnel-note ink-3">
              已裁定 {String(m?.engVerdicted ?? 0n)} · 待审{" "}
              {String(m?.engPending ?? 0n)}
            </span>
          </div>
          <div className="v2-funnel-lv">
            <span className="v2-funnel-k ink-3">经济教训 · 已入库</span>
            <span className="v2-funnel-n num">{String(m?.econCanon ?? 0n)}</span>
            <span className="v2-funnel-note ink-3">
              已裁定 {String(m?.econVerdicted ?? 0n)} · 待审{" "}
              {String(m?.econPending ?? 0n)}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
