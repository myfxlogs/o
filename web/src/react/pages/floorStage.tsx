// floorStage.tsx —— 全屏全息舱(floor v3 业主令:章鱼升整版,
// AI 舱图当底,活件全在浮层——零装饰像素铁律照守:每个覆盖件
// 绑活数据可点穿,底图只作氛围舞台,状态从不画在图上)。
// 几何: 位标=底图(1792×1006)像素坐标→stage 等比缩,改图先改本表。
//   中央全息球=脑(会话席+自态+记忆)  球座环=喙(NAV 钱芯)
//   环台9晶球=9器官  站位机器人×6=常驻机  后墙4屏=四面板
//   桌面件=待办浮件  底缘=事实尾流  顶角=警报灯
import { useEffect, useRef, useState } from "react";
import botStandUrl from "../assets/wall-station.png";
import rackUrl from "../assets/server-rack.png";
import globeUrl from "../assets/holo-globe.png";
import rimUrl from "../assets/rim-front.png";
import hallUrl from "../assets/room-shell.jpg";
import { plainText } from "../components/plain";
import type { Fact, Tip } from "./floorFocus";

export type StageOrgan = { i: number; name: string; st: string; hits: number; flash: boolean; anchor?: string };
export type StageVen = { name: string; usd: number; dud: boolean };
export type StageHost = { name: string; cpu: number; memPct: number; stale: boolean;
  cores: number; memUsedMb: number; memTotalMb: number; diskUsedGb: number; diskTotalGb: number;
  netRxKbps: number; netTxKbps: number; uptimeHours: number };
export type StageProps = {
  organs: StageOrgan[]; vens: StageVen[]; queue: { id: string; note: string }[];
  hosts?: StageHost[];
  probes: { escClass?: string; escScope?: string; open?: boolean }[];
  machines: { name: string; ok: boolean }[];
  nav?: { total: number; today: number; cum: number };
  awake: boolean; brainFlash: boolean; claim?: string; lastAct?: string;
  counts: { done: number; build: number; pin: number; openProbes: number };
  facts: Fact[]; live: boolean; utc: string; bj: string;
  onPick: (t: Tip) => void; onNav: (to: string) => void;
};

const fmtM = (v: number) => "$" + v.toLocaleString("zh-CN", { maximumFractionDigits: 0 });
// 圆桌9座·站位6机·脑/喙/屏/件/流 —— 底图%位表(改图先改表)
// SEAT_POS——9晶球悬浮台缘上方(台心50,66 椭圆内缩圈:rx≈30 ry≈16)
// θ=170°..10° 左缘→前中→右缘;尺度按θ景深(前大侧小)
const SEAT_POS = [
  [12.5, 59, 0.7], [17.5, 64, 0.8], [26.5, 68.5, 0.9], [38, 71.5, 1.0],
  [50, 72.3, 1.05], [62, 71.5, 1.0], [73.5, 68.5, 0.9], [82.5, 64, 0.8], [87.5, 59, 0.7],
];
const MACH_POS = [[6, 44], [14, 44], [22, 45], [30, 45], [68, 45], [76, 45], [84, 44], [90, 44]];
// 物理主机机柜位——左右墙脚地板线,真硬件件(非人形:器/服/机三形分家)
const HOST_POS = [[4, 47], [96, 47], [40, 46], [60, 46]];
// 台面工单板位——环台面环带散件,一待办一板(原图台面散件=活工单)
const PAD_POS = [[50, 70], [65.4, 67.4], [34.6, 67.4], [70.8, 64.5], [29.2, 64.5], [73.6, 60.9], [26.4, 60.9], [50, 74]];
const SCREEN_POS = { A: [9, 15], B: [24, 15], C: [70, 15], D: [86, 15] } as const;

export function FloorStage(p: StageProps) {
  const [pulse, setPulse] = useState<Set<number>>(new Set());
  const seenRef = useRef(new Map<number, bigint>());
  const initRef = useRef(false);
  // 器官域新行到达→对应座位脉冲(复用腕抽语义——新信号=座位一亮)
  useEffect(() => {
    if (!p.facts.length) return;
    const next = new Set<number>();
    for (const o of p.organs) {
      const hit = p.facts.find((f) => f.event !== "量表" && f.note.includes(o.name));
      if (!hit) continue;
      const prev = seenRef.current.get(o.i);
      if (initRef.current && (prev === undefined || hit.id > prev)) next.add(o.i);
      seenRef.current.set(o.i, prev !== undefined && prev > hit.id ? prev : hit.id);
    }
    initRef.current = true;
    if (next.size) {
      setPulse((s) => new Set([...s, ...next]));
      const t = window.setTimeout(() => setPulse((s) => { const m = new Set(s); next.forEach((i) => m.delete(i)); return m; }), 1200);
      return () => clearTimeout(t);
    }
  }, [p.facts, p.organs]);

  const openProbes = p.probes.filter((x) => x.open);
  const trail = p.facts.filter((f) => f.event !== "量表").slice(0, 10);
  const alarm = openProbes.length > 0 || p.queue.length > 6 || !p.live;
  // 器官单牌——流水线长在身上:待办单注文含器官名=该器有单;
  // 段位=注文末段工序词(认领→施工→修→戳→并→部署→销)
  const STAGE_RE = /认领|施工|修|戳|并|部署|销|收口/g;
  const organTicket = (o: StageOrgan) => {
    const key = o.name.replace(/^[A-Z] ?/, "");
    const q = p.queue.find((x) => x.note.includes(key));
    if (!q) return null;
    const hits = q.note.match(STAGE_RE);
    return { id: q.id, stage: hits ? hits[hits.length - 1] : null };
  };

  return (
    <div className="fstage" style={{ backgroundImage: `url(${hallUrl})` }}>
      {/* 顶左 HUD——计数+时钟(原 hero 压缩位) */}
      <div className="fs-hud">
        <span className={`fs-dot ${p.live ? "ok" : "bad"}`} />
        {p.counts.done}完工·{p.counts.build}在建·{p.counts.pin}立项·{p.queue.length}待办
        <span className="fs-hud-sub">UTC {p.utc.slice(0, 5)} · 北京 {p.bj.slice(0, 5)}</span>
      </div>
      {/* 部署基线——量表帧「部署=ver(age)」,并入脑文案行免撞碗沿 */}
      {/* 顶右警报灯——积压/断流红闪 */}
      <button className={`fs-alarm${alarm ? " on" : ""}`}
        onClick={() => p.onNav("/decision")}
        title={alarm ? `警 ${openProbes.length}探针·${p.queue.length}待办` : "盘面静"}>
        {alarm ? `⚠ ${openProbes.length + p.queue.length}` : "·"}
      </button>
      {/* 顶右纯景——投件归 /inbox 专页(业主判:floor 不载具) */}

      {/* 中央全息球=脑——在班泛光,球下在办+动向 */}
      <button className={`fs-brain${p.awake ? " awake" : ""}${p.brainFlash ? " flash" : ""}`}
        style={{ left: "50%", top: "24%" }}
        onClick={() => p.onPick({ k: "desk", x: 0, y: 0, title: "会话席", body: [], to: "/device" })}>
        <img src={globeUrl} alt="" className="fs-globe" draggable={false} />
        <span className="fs-brain-glow" />
      </button>
      <button className="fs-brain-txt" style={{ left: "50%", top: "44.5%" }}
        onClick={() => p.onPick({ k: "desk", x: 0, y: 0, title: "会话席", body: [], to: "/device" })}>
        <span className="fs-brain-line main">{p.awake ? p.claim ?? "在班·收棒中" : "脑·下班"}</span>
        {p.lastAct && <span className="fs-brain-line sub">{p.lastAct}</span>}
        {(() => {
          const f = p.facts.find((x) => x.event === "量表" && x.note.includes("部署="));
          const m = f?.note.match(/部署=([0-9a-f]{9,12})\(([^)]*)\)/);
          return m ? <span className="fs-brain-line dep">▣{m[1].slice(0, 9)}·{m[2]}前</span> : null;
        })()}
      </button>

      {/* 环台+光斗已烘进底图(原像素100%)——NAV 嵌台心孔 */}
      <button className="fs-nav" style={{ left: "50%", top: "57%" }}
        onClick={() => p.onPick({ k: "nav", x: 0, y: 0, title: "NAV · 收银", body: [], to: "/" })}>
        <b>{p.nav ? fmtM(p.nav.total) : "—"}</b>
        <span>今日 {p.nav ? (p.nav.today >= 0 ? "+" : "−") + fmtM(Math.abs(p.nav.today)) : "—"}</span>
      </button>

      {/* 环台9晶球=器官核——辉光色=态,呼吸灯节律=活性,新行到=球一搏,点=详情 */}
      {p.organs.map((o, k) => (
        <button key={o.i}
          className={`fs-seatwrap st-${o.st}${o.flash || pulse.has(o.i) ? " pulse" : ""}`}
          style={{ left: `${SEAT_POS[k][0]}%`, top: `${SEAT_POS[k][1]}%`, width: `${11.4 * (SEAT_POS[k][2] ?? 1)}%` }}
          onClick={() => p.onPick({ k: "gh" + o.i, x: 0, y: 0, title: `工位 · ${o.name}`, body: [], to: "/decision" })}>
          {/* 球下投影——悬浮纵深:球升影缩淡,球降影扩浓 */}
          <span className="fs-orbshadow" />
          {/* 悬浮晶球——态=辉光色(done青稳/build金急/pin虚形/ghost淡影) */}
          <span className="fs-orb" />
          {/* 单牌——该器身上有单即亮:#票号+当前段位(认领/施工/部署/销) */}
          {(() => { const t = organTicket(o); return t ? (
            <span className={`fs-tkt${t.stage ? ` sg-${t.stage}` : ""}`} title={p.queue.find(x => x.id === t.id)?.note}>
              #{t.id}{t.stage ? `·${t.stage}` : ""}
            </span>
          ) : null; })()}
          <span className="fs-seat">
            <b>{o.name}</b>
            <span>{o.st === "done" ? "完工" : o.st === "build" ? "在建" : o.st === "pin" ? "立项" : "毛坯"}{o.hits ? ` ·×${o.hits}` : ""}</span>
          </span>
        </button>
      ))}

      {/* 前台环缘遮罩——原图前唇盖座机下半身:机埋台缘凹槽非浮台面 */}
      <img src={rimUrl} className="fs-rim" style={{ left: "50%", top: "64%" }} alt="" draggable={false} />

      {/* 站位机器人×6=常驻机——站姿机贴图,断流灰红 */}
      {p.machines.map((m, k) => (
        <button key={m.name} className={`fs-machwrap${m.ok ? "" : " off"}`}
          style={{ left: `${MACH_POS[k][0]}%`, top: `${MACH_POS[k][1]}%` }}
          onClick={() => p.onPick({ k: "srv:" + m.name, x: 0, y: 0, title: `机柜 · ${m.name}`, body: [], to: "/device" })}>
          <img src={botStandUrl} alt={m.name} className="fs-botm" draggable={false} />
          <span className="fs-breath" />
          <span className="fs-mach"><span className={`fs-dot ${m.ok ? "ok" : "bad"}`} />{m.name}</span>
        </button>
      ))}

      {/* 物理主机=机柜——fin_host_metrics 真监控:cpu% 载点,stale 灰红 */}
      {(p.hosts ?? []).slice(0, HOST_POS.length).map((h, k) => (
        <button key={h.name} className={`fs-rackwrap${h.stale ? " off" : ""}`}
          style={{ left: `${HOST_POS[k][0]}%`, top: `${HOST_POS[k][1]}%` }}
          onClick={() => p.onPick({ k: "host:" + h.name, x: 0, y: 0, title: `主机 · ${h.name}`, body: [], to: "/device" })}>
          <img src={rackUrl} alt={h.name} className="fs-rack" draggable={false} />
          <span className="fs-rack-l"><span className={`fs-dot ${h.stale ? "bad" : "ok"}`} />{h.name} {h.cpu.toFixed(0)}%</span>
        </button>
      ))}

      {/* 屏A 钱径——venue 柱,断腿=整柱灰塌(业主裁:venue=外伸非围坐) */}
      <div className="fs-screen" style={{ left: `${SCREEN_POS.A[0]}%`, top: `${SCREEN_POS.A[1]}%` }}>
        <div className="fs-scr-t">钱径</div>
        {p.vens.map((v) => (
          <button key={v.name} className={`fs-ven${v.dud ? " dud" : ""}`}
            onClick={() => p.onPick({ k: "ven:" + v.name, x: 0, y: 0, title: `${v.name} 资金`, body: [`合计 ${fmtM(v.usd)}`], to: "/" })}>
            {v.name}<b>{fmtM(v.usd)}</b>
          </button>
        ))}
      </div>
      {/* 屏B 哨兵墙 */}
      <div className="fs-screen" style={{ left: `${SCREEN_POS.B[0]}%`, top: `${SCREEN_POS.B[1]}%` }}>
        <div className="fs-scr-t">哨兵 {openProbes.length ? `· ${openProbes.length}警` : "· 静"}</div>
        {p.probes.slice(0, 5).map((pr, i) => (
          <button key={i} className={`fs-probe${pr.open ? " on" : ""}`}
            onClick={() => p.onPick({ k: "probe" + i, x: 0, y: 0, title: plainText((pr.escClass ?? "") + "/" + (pr.escScope ?? "")), body: [pr.open ? "正在报警" : "静默在岗"], to: "/device" })}>
            {pr.open ? "▲" : "·"}{plainText(pr.escScope ?? "")}
          </button>
        ))}
      </div>
      {/* 屏C 待办+工单 */}
      <div className="fs-screen" style={{ left: `${SCREEN_POS.C[0]}%`, top: `${SCREEN_POS.C[1]}%` }}>
        <div className="fs-scr-t">待办 {p.queue.length ? `· ${p.queue.length}` : "· 板上净"}</div>
        {p.queue.slice(0, 5).map((q) => (
          <button key={q.id} className="fs-queue" onClick={() => p.onNav("/decision")}>
            #{q.id} {plainText(q.note).slice(0, 26)}
          </button>
        ))}
      </div>
      {/* 屏D 自态账页(末行事实) */}
      <div className="fs-screen" style={{ left: `${SCREEN_POS.D[0]}%`, top: `${SCREEN_POS.D[1]}%` }}>
        <div className="fs-scr-t">账页·末行</div>
        {trail.slice(0, 5).map((f) => (
          <div key={String(f.id)} className="fs-fact">{f.event} <span>{plainText(f.note).slice(0, 24)}</span></div>
        ))}
      </div>

      {/* 台面工单板——原图台面散件=活待办,一板一票可点穿 */}
      {p.queue.slice(0, 8).map((q, i) => (
        <button key={q.id} className="fs-pad"
          style={{ left: `${PAD_POS[i][0]}%`, top: `${PAD_POS[i][1]}%` }}
          onClick={() => p.onNav("/decision")} title={plainText(q.note).slice(0, 40)}>
          #{q.id}
        </button>
      ))}
      {/* 底缘事实尾流——拖尾横排 */}
      <div className="fs-trail">
        {trail.map((f) => (
          <span key={String(f.id)} className="fs-trail-i">
            <i>{f.event}</i>{plainText(f.note).slice(0, 20)}
          </span>
        ))}
      </div>
    </div>
  );
}
