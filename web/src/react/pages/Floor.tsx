// Floor.tsx —— 交易大厅壳层（业主令 11-11 活物世界观第三版·写实办公室）。
// 三房间：大厅(MainHall·写实家具)/档案室(ArchiveRoom)/机房(ServerRoom)，
// 门点切换非聚焦；悬停详情/点击聚焦到中央=floorFocus 承接，数据全接 RPC。
import { useEffect, useMemo, useState } from "react";
import { client } from "../api";
import { useLive } from "../hooks";
import { onLive } from "../stream";
import { Skeleton, ErrorState } from "../components/ui";
import { P } from "../components/iso";
import { FocusBody, type Tip } from "./floorFocus";
import { ArchiveRoom } from "./floorArchive";
import { MainHall, ServerRoom } from "./floorOffice";

const fmtMoney = (v: number) =>
  "$" + v.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

type Room = "floor" | "archive" | "server";
const VB: Record<Room, { x: number; y: number; w: number; h: number }> = {
  floor: { x: 470, y: 150, w: 940, h: 560 },
  archive: { x: 400, y: 140, w: 800, h: 480 },
  server: { x: 380, y: 140, w: 760, h: 440 },
};

export function FloorPage() {
  const ov = useLive(["ev_position", "ev_account"], () => client.overview({}));
  const g = useLive(["ev_account", "ev_position", "ev_pipeline"], () => client.getFaceGrid({}));
  const t = useLive(["ev_pipeline"], () => client.getDecisionTrail({ limit: 12 }));
  const [live, setLive] = useState(false);
  const [now, setNow] = useState(() => new Date());
  const [tip, setTip] = useState<Tip | null>(null);
  const [focus, setFocus] = useState<Tip | null>(null);
  const [room, setRoom] = useState<Room>("floor");
  useEffect(() => onLive((v) => setLive(!!v)), []);
  useEffect(() => {
    const i = setInterval(() => setNow(new Date()), 1000);
    const on = (e: KeyboardEvent) => {
      if (e.key === "Escape") setFocus((f) => (f ? null : (setRoom("floor"), null)));
    };
    window.addEventListener("keydown", on);
    return () => { clearInterval(i); window.removeEventListener("keydown", on); };
  }, []);

  const d = g.data, o = ov.data, s = d?.session;
  const openEsc = useMemo(() => (t.data?.trail ?? []).filter((r) => r.open), [t.data]);
  const openReq = useMemo(() => (t.data?.condreqs ?? []).filter((r) => r.open), [t.data]);
  const vens = useMemo(() => { // venue 屏排——money 行按 venue 分组合计
    const m = new Map<string, number>();
    for (const r of o?.money ?? []) m.set(r.venue, (m.get(r.venue) ?? 0) + (r.usd || 0));
    return [...m.entries()]; }, [o]);
  const probes = d?.probes ?? [];
  const openProbes = probes.filter((p) => p.open).length;
  const engCanon = Number(t.data?.mine?.engCanon ?? 0n);
  const econCanon = Number(t.data?.mine?.econCanon ?? 0n);
  const wallNotes = (t.data?.trail ?? []).slice(0, 3);
  const utc = now.toLocaleTimeString("zh-CN", { timeZone: "UTC", hour12: false }),
    bj = now.toLocaleTimeString("zh-CN", { timeZone: "Asia/Shanghai", hour12: false });

  const hot = (tt: Omit<Tip, "x" | "y">, x: number, y: number, z = 0, cls = "") => {
    const [px, py] = P(x, y, z);
    return {
      onMouseEnter: () => setTip({ ...tt, x: px, y: py }),
      onMouseLeave: () => setTip(null),
      onClick: () => { setTip(null); setFocus({ ...tt, x: px, y: py }); },
      className: "iso-hot" + (cls ? " " + cls : ""),
    };
  };

  if (g.loading && !g.data) return <Skeleton rows={6} />;
  if (g.err && !g.data) return <ErrorState msg={g.err} onRetry={g.reload} />;

  const vb = VB[room];

  return (
    <div className="page v2">
      <div className={`floor-scene${live ? " live" : " dark"}${focus ? " focused" : ""}`} style={{ aspectRatio: `${vb.w}/${vb.h}` }}>
        {room === "archive" ? (
          <ArchiveRoom live={live} engCanon={engCanon} econCanon={econCanon}
            notes={t.data?.trail ?? []} hot={hot} onExit={() => setRoom("floor")} />
        ) : room === "server" ? (
          <ServerRoom live={live} warn={openProbes > 0} hot={hot} onExit={() => setRoom("floor")} />
        ) : (
        <svg viewBox="-470 -150 940 560" className="floor-svg">
          <defs>
            <linearGradient id="sky" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#a8c4de" />
              <stop offset="100%" stopColor="#d8e4f0" />
            </linearGradient>
            <radialGradient id="lampGlow">
              <stop offset="0%" stopColor="#ffd76a" stopOpacity="0.55" />
              <stop offset="100%" stopColor="#ffd76a" stopOpacity="0" />
            </radialGradient>
          </defs>
          <MainHall live={live} probes={probes} openProbes={openProbes}
            notes={wallNotes} openCount={openEsc.length + openReq.length}
            trail={t.data?.trail ?? []} s={s} vens={vens} hot={hot} toRoom={setRoom}
            hover={setTip} unhover={() => setTip(null)} />
        </svg>
        )}

        {/* 悬停详情卡 */}
        {tip && !focus && (
          <div className="iso-tip" style={{ left: `${((tip.x + vb.x) / vb.w * 100).toFixed(2)}%`, top: `${((tip.y + vb.y) / vb.h * 100).toFixed(2)}%` }}>
            <div className="iso-tip-t">{tip.title}</div>
            {tip.body.map((b, i) => <div key={i} className="iso-tip-b">{b}</div>)}
            <div className="iso-tip-to">点击→到中央</div>
          </div>
        )}

        {/* 聚焦——物件到中央，景压暗让位 */}
        {focus && (<>
          <div className="iso-focus-back" onClick={() => setFocus(null)} />
          <div className="iso-focus">
            <button className="iso-focus-x" onClick={() => setFocus(null)}>×</button>
            <FocusBody f={focus} openEsc={openEsc} openReq={openReq} trail={t.data?.trail ?? []}
              probes={probes} openProbes={openProbes} engCanon={engCanon} econCanon={econCanon}
              nav={o ? { total: o.navTotal, today: o.pnlToday, cum: o.pnlCum } : undefined}
              money={o?.money ?? []} live={live} session={s} utc={utc} bj={bj} />
          </div>
        </>)}

        {/* 双钟读数+NAV 屏数（仅主厅——viewBox 归一按当前房间） */}
        {room === "floor" && (<>
          {([["utc", P(0, 7.5, 2.55), utc], ["bj", P(11.9, 0, 2.45), bj]] as const).map(([k, [px, py], v]) => (
            <div key={k} className="iso-clock" style={{ left: `${((px + vb.x) / vb.w * 100).toFixed(2)}%`, top: `${((py + vb.y) / vb.h * 100).toFixed(2)}%` }}>{v.slice(0, 5)}</div>))}
          <div className="iso-nav num" style={{ left: `${((P(5.8, 0, 1.95)[0] + vb.x) / vb.w * 100).toFixed(2)}%`, top: `${((P(5.8, 0, 1.95)[1] + vb.y) / vb.h * 100).toFixed(2)}%` }}>
            {o ? fmtMoney(o.navTotal) : "—"}
          </div>
        </>)}
        <div className="iso-state">{room === "floor" ? (live ? "厅灯亮——数据流活着" : "厅灯暗——数据流断了") : room === "archive" ? "档案室——ESC 或点门回厅" : "机房——ESC 或点门回厅"}</div>
      </div>
    </div>
  );
}
