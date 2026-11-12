// floorOffice.tsx —— 控制室模拟盘版交易厅（业主令 11-12：删美工全语义化）。
// 底板=等距线框网格;全元素=令牌(Cell):左列感知/白板/会话/成长/双钟,
// 中央器官工位阵+常驻机排,右列钱径/venue/钉墙/档案机房链;票签=搬运令牌。
import { P } from "../components/iso";
import { plainText } from "../components/plain";
import { IsoGrid, ZoneCap, Cell, Packet } from "./floorTokens";
import { GhostRow, VenueCol, type Fact, type Tip } from "./floorFocus";

type Hot = (tt: Omit<Tip, "x" | "y">, x: number, y: number, z?: number, cls?: string) => Record<string, unknown>;

type Note = { escId: unknown; open: boolean; note: string };
type Probe = { escClass: string; escScope: string; open: boolean; lastEsc?: unknown };
type Sess = { running?: boolean; spawns24h?: unknown; fails24h?: unknown };

// 主厅整景（svg 内层，不含 defs——defs 在文件尾共享）。
export function MainHall(props: {
  live: boolean; probes: Probe[]; openProbes: number; notes: Note[]; openCount: number;
  facts: Fact[];
  queue: { id: string; note: string; open: boolean }[];
  s?: Sess; vens: [string, number][]; hot: Hot; toRoom: (r: "archive" | "server") => void;
  hover: (tt: Tip) => void; unhover: () => void;
  engCanon: number; econCanon: number;
  walk?: { path: [number, number][]; tone: string; k: number }; walkDone: () => void;
}) {
  const { hot, live } = props;
  const W = 18, D = 12;
  const roomLink = (k: string, x: number, y: number, title: string, body: string[], to: () => void) => ({
    className: "iso-hot",
    onMouseEnter: () => { const [tx, ty] = P(x, y); props.hover({ k, x: tx, y: ty, title, body, to: "" }); },
    onMouseLeave: props.unhover,
    onClick: () => { props.unhover(); to(); },
  });
  return (
    <g>
      <IsoGrid w={W} d={D} />

      {/* ── 左列：感知哨兵 → 白板 → 会话 → 成长 → 双钟 ── */}
      <ZoneCap x={0.6} y={0.55} label="感知哨兵" />
      {props.probes.slice(0, 6).map((p, i) => {
        const col = p.open ? "#ff6b6b" : p.lastEsc ? "#ffd76a" : "#6fd08c";
        return (
          <g key={p.escClass + "/" + p.escScope}
            {...hot({ k: "probe" + i, title: plainText(p.escClass + "/" + p.escScope), body: [p.open ? "正在报警" : "静默在岗"], to: "/device" }, 1.65, 1.25 + i * 0.8, 0, p.open ? "fx-breathe" : "")}>
            <Cell x={0.6} y={0.9 + i * 0.8} w={2.1} d={0.62} tone={col}
              lines={[plainText(p.escClass).slice(0, 12)]} fs={8} />
          </g>
        );
      })}
      {props.probes.length > 6 && <text x={P(1.65, 0.9 + 6 * 0.8)[0]} y={P(1.65, 0.9 + 6 * 0.8)[1]} className="iso-cap">+{props.probes.length - 6}</text>}

      <g {...hot({ k: "board", title: "白板 · 待办", body: props.openCount ? [`${props.openCount} 件钉在板上——点开看全谱`] : ["板上干净——急件落账后钉在这里"], to: "/decision" }, 1.65, 5.9)}>
        <Cell x={0.6} y={5.4} w={2.1} d={1.0} tone="#d9a62e" lines={["白板", `${props.openCount} 件钉着`]} fs={9} />
      </g>

      <g {...hot({ k: "desk", title: "会话席 · 唤知会话", body: [props.s?.running ? "在班——正在干活" : "下班了——空席待命"], to: "/device" }, 1.65, 7.25)}>
        <Cell x={0.6} y={6.75} w={2.1} d={1.05} tone={props.s?.running ? "#5b8def" : "#5a6274"}
          lines={["会话席", props.s?.running ? "在班" : "下班", `24h唤${String(props.s?.spawns24h ?? "—")}败${String(props.s?.fails24h ?? "—")}`]} fs={9} />
      </g>

      <g {...hot({ k: "ruler", title: "成长位", body: ["现位：婴→少年（A-D 在建）", "判据：点破率归零 + 年化≥8%"], to: "/" }, 1.65, 8.6)}>
        <Cell x={0.6} y={8.15} w={2.1} d={0.72} tone="#8a93a3" lines={["成长 · 婴→少年"]} fs={8} />
      </g>

      <g {...hot({ k: "utc", title: "UTC 钟", body: ["机器时间"], to: "" }, 1.65, 9.55)}>
        <Cell x={0.6} y={9.2} w={2.1} d={0.68} tone="#4a5160" />
      </g>
      <g {...hot({ k: "bj", title: "北京钟", body: ["你的时间"], to: "" }, 1.65, 10.45)}>
        <Cell x={0.6} y={10.1} w={2.1} d={0.68} tone="#4a5160" />
      </g>

      {/* ── 顶排：常驻机令牌（机房全谱在子房） ── */}
      <ZoneCap x={4.0} y={0.55} label="常驻机" />
      {RACKS.map((nm, i) => (
        <g key={nm} {...hot({ k: "srv:" + nm, title: `机柜 · ${nm}`, body: ["常驻器官一件"], to: "/device" }, 4.75 + i * 1.65, 1.45)}>
          <Cell x={4.0 + i * 1.65} y={0.95} w={1.5} d={0.72} tone={live ? "#3ddc84" : "#5a6274"}
            lines={[nm]} fs={8} dim={!live} />
        </g>
      ))}

      {/* ── 中央上区：待办明细（openEsc/openReq 全谱直读,最值钱的信息位） ── */}
      <ZoneCap x={3.4} y={2.05} label="待办明细" />
      {props.queue.length === 0 && (
        <text x={P(3.6, 2.65)[0]} y={P(3.6, 2.65)[1]} className="iso-cap dim" fontSize="9">板上干净——零积压</text>
      )}
      {props.queue.slice(0, 9).map((r, i) => (
        <g key={r.id} {...hot({ k: "board", title: "待办明细", body: [plainText(r.note).slice(0, 80)], to: "/decision" }, 3.6, 2.55 + i * 0.5)}>
          <text x={P(3.6, 2.55 + i * 0.5)[0]} y={P(3.6, 2.55 + i * 0.5)[1]}
            className="iso-cap" fontSize="8.5" fill={r.open ? "#ff9b9b" : "#9aa7bd"}>
            {r.open ? "●" : "○"} #{r.id} {plainText(r.note).slice(0, 46)}
          </text>
        </g>
      ))}
      {props.queue.length > 9 && (
        <text x={P(3.6, 2.55 + 9 * 0.5)[0]} y={P(3.6, 2.55 + 9 * 0.5)[1]} className="iso-cap dim" fontSize="8.5">…另 {props.queue.length - 9} 件,进「何事要裁」看全谱</text>
      )}

      {/* ── 中央：器官工位阵 ── */}
      <ZoneCap x={3.4} y={7.1} label="器官工位" />
      <GhostRow hot={hot} facts={props.facts} />

      {/* ── 右列：钱径 → venue → 钉墙 → 档案/机房链 ── */}
      <ZoneCap x={14.6} y={0.55} label="钱径" />
      <g {...hot({ k: "nav", title: "收银 · NAV+持仓+历史", body: ["点开看净值/在仓/平仓全谱"], to: "/" }, 16.1, 1.7)}>
        <Cell x={14.6} y={0.9} w={3.0} d={1.5} tone="#3ddc84" lines={["NAV"]} fs={9} />
      </g>

      <VenueCol hot={hot} vens={props.vens} />

      <g {...hot({ k: "wall", title: "日记钉墙 · 共享大脑", body: props.notes.length ? props.notes.map((r) => (r.open ? "🔴 " : "") + plainText(r.note).slice(0, 40)) : ["墙上干净——无新钉"], to: "/decision" }, 16.1, 6.35)}>
        <Cell x={14.6} y={5.85} w={3.0} d={1.0} tone="#b08a5e" lines={["钉墙", `${props.notes.length} 条`]} fs={9} />
      </g>

      <g {...roomLink("door", 16.1, 7.55, "档案室", ["canon 书架 + 日记钉墙——点进屋"], () => props.toRoom("archive"))}>
        <Cell x={14.6} y={7.05} w={3.0} d={1.0} tone="#c9a86a"
          lines={["档案室 →", `eng ${props.engCanon} · econ ${props.econCanon}`]} fs={9} />
      </g>
      <g {...roomLink("door2", 16.1, 8.75, "机房", ["常驻器官机柜排——点进屋"], () => props.toRoom("server"))}>
        <Cell x={14.6} y={8.25} w={3.0} d={1.0} tone="#7fa8d0" lines={["机房 →", `${RACKS.length} 常驻`]} fs={9} />
      </g>

      {/* 票签——领命/移交事实触发的搬运令牌（顶层渲染不被遮） */}
      {props.walk && <Packet key={props.walk.k} path={props.walk.path} tone={props.walk.tone} id={props.walk.k} onDone={props.walkDone} />}
    </g>
  );
}

// 机房——常驻器官机柜令牌排（/floor 子房）。
export const RACKS = ["o-core", "PG", "collector", "projector", "sweeper", "eventbus"];
export function ServerRoom(props: { live: boolean; warn: boolean; hot: Hot; onExit: () => void }) {
  const { hot, live } = props;
  const W = 12, D = 8;
  return (
    <svg viewBox="-390 -60 960 480" className="floor-svg">
      <IsoGrid w={W} d={D} />
      <ZoneCap x={0.8} y={1.4} label="常驻器官机柜" />
      {RACKS.map((nm, i) => {
        const x = 1.2 + (i % 3) * 3.4, y = 2.2 + Math.floor(i / 3) * 2.4;
        const tone = !live ? "#5a6274" : props.warn && nm === "collector" ? "#ffb454" : "#3ddc84";
        return (
          <g key={nm} {...hot({ k: "srv:" + nm, title: `机柜 · ${nm}`, body: ["常驻器官一件"], to: "/device" }, x + 1.4, y + 0.8)}>
            <Cell x={x} y={y} w={2.8} d={1.5} tone={tone} lines={[nm, live ? "运行中" : "断流"]} fs={10}
              dim={!live} />
          </g>
        );
      })}
      <text x={P(1.2, 7.2)[0]} y={P(1.2, 7.2)[1]} className="iso-cap iso-hot" onClick={props.onExit}>← 回厅</text>
    </svg>
  );
}
