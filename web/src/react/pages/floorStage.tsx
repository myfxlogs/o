// floorStage.tsx —— 全屏全息舱(floor v3 业主令:章鱼升整版,
// AI 舱图当底,活件全在浮层——零装饰像素铁律照守:每个覆盖件
// 绑活数据可点穿,底图只作氛围舞台,状态从不画在图上)。
// 几何: 位标=底图(1792×1006)像素坐标→stage 等比缩,改图先改本表。
//   中央全息球=脑(会话席+自态+记忆)  球座环=喙(NAV 钱芯)
//   圆桌9座=9器官  站位机器人×6=常驻机  后墙4屏=四面板
//   桌面件=待办浮件  底缘=事实尾流  顶角=警报灯
import { useEffect, useRef, useState } from "react";
import botSeatUrl from "../assets/bot-seat.png";
import botStandUrl from "../assets/bot-stand.png";
import globeUrl from "../assets/holo-globe.png";
import tableUrl from "../assets/conf-table.png";
import chairUrl from "../assets/robot-chair.png";
import hallUrl from "../assets/bridge-hall.jpg";
import frameUrl from "../assets/holo-dashboard.png";
import { plainText } from "../components/plain";
import type { Fact, Tip } from "./floorFocus";

export type StageOrgan = { i: number; name: string; st: string; hits: number; flash: boolean; anchor?: string };
export type StageVen = { name: string; usd: number; dud: boolean };
export type StageProps = {
  organs: StageOrgan[]; vens: StageVen[]; queue: { id: string; note: string }[];
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
// SEAT_POS 按器官 i 定点:[x%,y%,scale]——done 器居前座满尺,
// pin 器居后缘缩尺(景深+立项=快入座)
const SEAT_POS = [
  [76, 62, 0.85], [63, 71, 1.0], [37, 71, 1.0], [24, 62, 0.85], [20, 53, 0.7],
  [31, 49, 0.65], [68, 45, 0.65], [84, 54, 0.7], [50, 74, 1.0],
];
const MACH_POS = [[74, 38], [80, 40], [34, 38], [26, 35], [90, 39], [93, 33]];
const SCREEN_POS = { A: [11, 15], B: [26, 15], C: [68, 15], D: [85, 15] } as const;

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

  return (
    <div className="fstage" style={{ backgroundImage: `url(${hallUrl})` }}>
      {/* 顶左 HUD——计数+时钟(原 hero 压缩位) */}
      <div className="fs-hud">
        <span className={`fs-dot ${p.live ? "ok" : "bad"}`} />
        {p.counts.done}完工·{p.counts.build}在建·{p.counts.pin}立项·{p.queue.length}待办
        <span className="fs-hud-sub">UTC {p.utc.slice(0, 5)} · 北京 {p.bj.slice(0, 5)}</span>
      </div>
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
      </button>

      {/* 环桌——器官座围的主桌,中空圆心嵌 NAV 喙 */}
      <img src={tableUrl} className="fs-table" style={{ left: "50%", top: "64%" }} alt="" draggable={false} />

      {/* 桌心孔=喙·NAV 钱芯(钱从桌心入) */}
      <button className="fs-nav" style={{ left: "50%", top: "61.5%" }}
        onClick={() => p.onPick({ k: "nav", x: 0, y: 0, title: "NAV · 收银", body: [], to: "/" })}>
        <b>{p.nav ? fmtM(p.nav.total) : "—"}</b>
        <span>今日 {p.nav ? (p.nav.today >= 0 ? "+" : "−") + fmtM(Math.abs(p.nav.today)) : "—"}</span>
      </button>

      {/* 圆桌9座=器官——色=态,新行到=脉冲,点=详情 */}
      {p.organs.map((o, k) => (
        <button key={o.i}
          className={`fs-seatwrap st-${o.st}${o.flash || pulse.has(o.i) ? " pulse" : ""}`}
          style={{ left: `${SEAT_POS[k][0]}%`, top: `${SEAT_POS[k][1]}%`, width: `${8.6 * (SEAT_POS[k][2] ?? 1)}%` }}
          onClick={() => p.onPick({ k: "gh" + o.i, x: 0, y: 0, title: `工位 · ${o.name}`, body: [], to: "/decision" })}>
          {/* pin/ghost=空椅(座摆人未坐);build/done=坐机 */}
          <img src={o.st === "pin" || o.st === "ghost" ? chairUrl : botSeatUrl} alt={o.name}
            className={`fs-bot${o.st === "pin" || o.st === "ghost" ? " chair" : ""}${SEAT_POS[k][0] > 55 ? " flip" : ""}`} draggable={false} />
          <span className="fs-seat">
            <b>{o.name}</b>
            <span>{o.st === "done" ? "完工" : o.st === "build" ? "在建" : o.st === "pin" ? "立项" : "毛坯"}{o.hits ? ` ·×${o.hits}` : ""}</span>
          </span>
        </button>
      ))}

      {/* 站位机器人×6=常驻机——站姿机贴图,断流灰红 */}
      {p.machines.map((m, k) => (
        <button key={m.name} className={`fs-machwrap${m.ok ? "" : " off"}`}
          style={{ left: `${MACH_POS[k][0]}%`, top: `${MACH_POS[k][1]}%` }}
          onClick={() => p.onPick({ k: "srv:" + m.name, x: 0, y: 0, title: `机柜 · ${m.name}`, body: [], to: "/device" })}>
          <img src={botStandUrl} alt={m.name} className="fs-botm" draggable={false} />
          <span className="fs-mach"><span className={`fs-dot ${m.ok ? "ok" : "bad"}`} />{m.name}</span>
        </button>
      ))}

      {/* 屏A 钱径——venue 柱,断腿=整柱灰塌(业主裁:venue=外伸非围坐) */}
      <div className="fs-screen" style={{ left: `${SCREEN_POS.A[0]}%`, top: `${SCREEN_POS.A[1]}%` }}>
        <img src={frameUrl} className="fs-frame" alt="" draggable={false} />
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
        <img src={frameUrl} className="fs-frame" alt="" draggable={false} />
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
        <img src={frameUrl} className="fs-frame" alt="" draggable={false} />
        <div className="fs-scr-t">待办 {p.queue.length ? `· ${p.queue.length}` : "· 板上净"}</div>
        {p.queue.slice(0, 5).map((q) => (
          <button key={q.id} className="fs-queue" onClick={() => p.onNav("/decision")}>
            #{q.id} {plainText(q.note).slice(0, 26)}
          </button>
        ))}
      </div>
      {/* 屏D 自态账页(末行事实) */}
      <div className="fs-screen" style={{ left: `${SCREEN_POS.D[0]}%`, top: `${SCREEN_POS.D[1]}%` }}>
        <img src={frameUrl} className="fs-frame" alt="" draggable={false} />
        <div className="fs-scr-t">账页·末行</div>
        {trail.slice(0, 5).map((f) => (
          <div key={String(f.id)} className="fs-fact">{f.event} <span>{plainText(f.note).slice(0, 24)}</span></div>
        ))}
      </div>

      {/* 桌面待办浮件——桌前中部 */}
      {p.queue.length > 0 && (
        <button className="fs-doc" style={{ left: "42%", top: "63%" }} onClick={() => p.onNav("/decision")}>
          待办 ×{p.queue.length}
        </button>
      )}
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
