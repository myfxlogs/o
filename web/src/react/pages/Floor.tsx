// Floor.tsx —— 它在干啥(floor v3 全屏全息舱:AI 舱图当舞台,
// 全部卡片装回舱内挂载点——脑/喙/9座/6机/4屏/尾流;
// 数据壳职责:拉数→态推导→喂 FloorStage,弹窗 FocusBody 照留)。
import { useEffect, useMemo, useState } from "react";
import { client } from "../api";
import { useLive } from "../hooks";
import { onLive } from "../stream";
import { Skeleton, ErrorState } from "../components/ui";
import { navigate } from "../router";
import { FocusBody, GHOSTS, ghostSt, type Tip } from "./floorFocus";
import { FloorStage } from "./floorStage";
import { useOctoFeed } from "./floorOcto";

const RACKS = ["o-core", "PG", "collector", "projector", "sweeper", "eventbus"];

export function FloorPage() {
  const ov = useLive(["ev_position", "ev_account"], () => client.overview({}));
  const g = useLive(["ev_account", "ev_position", "ev_pipeline"], () => client.getFaceGrid({}));
  const t = useLive(["ev_pipeline"], () => client.getDecisionTrail({ limit: 12 }));
  const hm = useLive(["ev_hostmetrics"], () => client.getHostMetrics({}));
  const [live, setLive] = useState(false);
  const [now, setNow] = useState(() => new Date());
  const [focus, setFocus] = useState<Tip | null>(null);
  useEffect(() => onLive((v) => setLive(!!v)), []);
  useEffect(() => {
    const i = setInterval(() => setNow(new Date()), 1000);
    const on = (e: KeyboardEvent) => { if (e.key === "Escape") setFocus(null); };
    window.addEventListener("keydown", on);
    return () => { clearInterval(i); window.removeEventListener("keydown", on); };
  }, []);

  const d = g.data, o = ov.data, s = d?.session;
  const facts = useMemo(() => t.data?.facts ?? [], [t.data]);
  const openEsc = useMemo(() => (t.data?.trail ?? []).filter((r) => r.open), [t.data]);
  const openReq = useMemo(() => (t.data?.condreqs ?? []).filter((r) => r.open), [t.data]);
  const probes = useMemo(() => d?.probes ?? [], [d]);

  // 器官态推导——同 facts 窗驱动
  const organs = useMemo(() => GHOSTS.map((gh, i) => ({
    gh, i, st: ghostSt(gh, facts),
    hits: facts.filter((f) => f.event !== "量表" && gh.keys.some((k) => f.note.includes(k))).length,
  })), [facts]);
  const octo = useOctoFeed(facts, organs);

  // venue 分腿——open 探针载 venue 名=断腿(整柱灰塌)
  const vens = useMemo(() => {
    const m = new Map<string, number>();
    for (const r of o?.money ?? []) m.set(r.venue, (m.get(r.venue) ?? 0) + (r.usd || 0));
    return [...m.entries()].map(([name, usd]) => ({
      name, usd,
      dud: probes.some((p) => p.open && `${p.escClass}/${p.escScope}`.toLowerCase().includes(name.toLowerCase())),
    }));
  }, [o, probes]);

  const queue = [...openEsc.map((r) => ({ id: String(r.escId), note: r.note })),
    ...openReq.map((r) => ({ id: String(r.id), note: "条件请求·待处理" }))];

  const utc = now.toLocaleTimeString("zh-CN", { timeZone: "UTC", hour12: false }),
    bj = now.toLocaleTimeString("zh-CN", { timeZone: "Asia/Shanghai", hour12: false });

  if (g.loading && !g.data) return <Skeleton rows={6} />;
  if (g.err && !g.data) return <ErrorState msg={g.err} onRetry={g.reload} />;

  return (
    <div className="page v2 fstage-page">
      <FloorStage
        organs={organs.map(({ gh, i, st, hits }) => ({ i, name: gh.name, st, hits, flash: octo.flash.has(i) }))}
        vens={vens}
        queue={queue}
        probes={probes}
        machines={RACKS.map((nm) => ({ name: nm, ok: live }))}
        hosts={(hm.data?.hosts ?? []).map((h) => ({
          name: h.host === "dev" ? "OCI" : h.host.toUpperCase(),
          cpu: h.cpuPct, memPct: h.memTotalMb ? (h.memUsedMb / h.memTotalMb) * 100 : 0,
          stale: h.stale,
        }))}
        nav={o ? { total: o.navTotal, today: o.pnlToday, cum: o.pnlCum } : undefined}
        awake={!!s?.running}
        brainFlash={octo.brainFlash}
        claim={s?.running ? octo.claim ?? "收棒中" : undefined}
        lastAct={octo.lastAct}
        counts={{
          done: organs.filter((x) => x.st === "done").length,
          build: organs.filter((x) => x.st === "build").length,
          pin: organs.filter((x) => x.st === "pin").length,
          openProbes: probes.filter((p) => p.open).length,
        }}
        facts={facts}
        live={live}
        utc={utc} bj={bj}
        onPick={setFocus}
        onNav={navigate}
      />

      {/* 聚焦弹窗——点挂载件开详情(原 FocusBody 全保留) */}
      {focus && (<>
        <div className="iso-focus-back" onClick={() => setFocus(null)} />
        <div className="iso-focus">
          <button className="iso-focus-x" onClick={() => setFocus(null)}>×</button>
          <FocusBody f={focus} openEsc={openEsc} openReq={openReq} trail={t.data?.trail ?? []} facts={facts}
            probes={probes} openProbes={probes.filter((p) => p.open).length}
            engCanon={Number(t.data?.mine?.engCanon ?? 0n)} econCanon={Number(t.data?.mine?.econCanon ?? 0n)}
            nav={o ? { total: o.navTotal, today: o.pnlToday, cum: o.pnlCum } : undefined}
            money={o?.money ?? []} live={live} session={s} utc={utc} bj={bj} />
        </div>
      </>)}
    </div>
  );
}
