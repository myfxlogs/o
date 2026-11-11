// floorOffice.tsx —— 写实办公室版交易厅（业主令 11-11：要真办公室质感）。
// 大厅=木地开放区+水泥区；左墙大落地窗+档案室门，右墙 NAV 大屏/venue 屏排/
// 机房钢门；件件独立可交互（hot() 包裹），数据绑定位与旧景一致。
import { P, pts, IsoBox, Person } from "../components/iso";
import { plainText } from "../components/plain";
import { Desk, OfficeChair, Monitor, Plant, WaterCooler, PendantLamp, WallWindowL, ServerRack, DoorSprite } from "./officeParts";
import { GhostRow, VenueScreens, type Tip } from "./floorFocus";

type Hot = (tt: Omit<Tip, "x" | "y">, x: number, y: number, z?: number, cls?: string) => Record<string, unknown>;

type Note = { escId: unknown; open: boolean; note: string };
type Probe = { escClass: string; escScope: string; open: boolean; lastEsc?: unknown };
type Sess = { running?: boolean };

// 木地/水泥地两分区+木纹线（参考图左木右水泥）。
function OfficeFloor(props: { w: number; d: number; split: number }) {
  const { w, d, split } = props;
  return (
    <g>
      <polygon points={pts([P(0, 0), P(w, 0), P(w, split), P(0, split)])} fill="#b0834f" stroke="#8f6a3d" strokeWidth="1.4" />
      {Array.from({ length: Math.ceil(split) + 1 }, (_, i) => (
        <line key={"w" + i} x1={P(0, i * 1.0)[0]} y1={P(0, i * 1.0)[1]} x2={P(w, i * 1.0)[0]} y2={P(w, i * 1.0)[1]} stroke="#9c7443" strokeWidth="0.6" />
      ))}
      <polygon points={pts([P(0, split), P(w, split), P(w, d), P(0, d)])} fill="#9aa0a8" stroke="#7d838c" strokeWidth="1.4" />
      {Array.from({ length: Math.ceil(d - split) + 1 }, (_, i) => split + i).map((yy) => (
        <line key={"c" + yy} x1={P(0, yy)[0]} y1={P(0, yy)[1]} x2={P(w, yy)[0]} y2={P(w, yy)[1]} stroke="#878d96" strokeWidth="0.5" />
      ))}
      {Array.from({ length: w + 1 }, (_, i) => (
        <line key={"g" + i} x1={P(i, split)[0]} y1={P(i, split)[1]} x2={P(i, d)[0]} y2={P(i, d)[1]} stroke="#878d96" strokeWidth="0.5" />
      ))}
    </g>
  );
}

// 主厅整景（svg 内层，不含 defs——defs 在文件尾共享）。
export function MainHall(props: {
  live: boolean; probes: Probe[]; openProbes: number; notes: Note[]; openCount: number;
  s?: Sess; vens: [string, number][]; hot: Hot; toRoom: (r: "archive" | "server") => void;
  hover: (tt: Tip) => void; unhover: () => void;
}) {
  const { hot, live } = props;
  const W = 14, D = 10, Z = 3.1, SPLIT = 5.6;
  const door = (k: string, x: number, y: number, z: number, title: string, body: string[], to: () => void) => ({
    className: "iso-hot",
    onMouseEnter: () => { const [tx, ty] = P(x, y, z); props.hover({ k, x: tx, y: ty, title, body, to: "" }); },
    onMouseLeave: props.unhover,
    onClick: () => { props.unhover(); to(); },
  });
  return (
    <g>
      <OfficeFloor w={W} d={D} split={SPLIT} />
      {/* 左墙（窗墙）+右墙（屏墙）——亮灰办公室调 */}
      <polygon points={pts([P(0, 0), P(0, D), P(0, D, Z), P(0, 0, Z)])} fill="#cfcac0" stroke="#8a8478" strokeWidth="1.2" />
      <polygon points={pts([P(0, 0), P(W, 0), P(W, 0, Z), P(0, 0, Z)])} fill="#ddd8cf" stroke="#8a8478" strokeWidth="1.2" />

      {/* 大落地窗×2（gx=0 面）——哨兵排守窗前 */}
      <WallWindowL y0={0.9} y1={4.1} z0={0.95} z1={2.95} />
      <WallWindowL y0={4.4} y1={6.95} z0={0.95} z1={2.95} />

      {/* 哨兵——窗下人形排，探针态=灯色 */}
      {props.probes.slice(0, 6).map((p, i) => {
        const gy = 1.7 + i * 0.55, col = p.open ? "#ff6b6b" : p.lastEsc ? "#ffd76a" : "#6fd08c";
        return (
          <g key={p.escClass + "/" + p.escScope}
            {...hot({ k: "probe" + i, title: plainText(p.escClass + "/" + p.escScope), body: [p.open ? "正在报警" : "静默在岗"], to: "/device" }, 0.9, gy, 0, p.open ? "fx-breathe" : "")}>
            <Person x={0.9} y={gy} tone={col} />
          </g>
        );
      })}
      {props.probes.length > 6 && <text x={P(0.9, 5.2)[0]} y={P(0.9, 5.2)[1] - 8} className="iso-cap">+{props.probes.length - 6}</text>}

      {/* UTC 钟——左墙门上段 */}
      <g {...hot({ k: "utc", title: "UTC 钟", body: ["机器时间"], to: "" }, 0, 7.5, 2.55)}>
        <ellipse cx={P(0, 7.5, 2.55)[0]} cy={P(0, 7.5, 2.55)[1]} rx="11" ry="6" fill="#f4f1e8" stroke="#6a7280" strokeWidth="1" />
      </g>
      {/* 北京钟——右墙 */}
      <g {...hot({ k: "bj", title: "北京钟", body: ["你的时间"], to: "" }, 11.9, 0, 2.45)}>
        <ellipse cx={P(11.9, 0, 2.45)[0]} cy={P(11.9, 0, 2.45)[1]} rx="11" ry="6" fill="#f4f1e8" stroke="#6a7280" strokeWidth="1" />
      </g>

      {/* 档案室门——左墙门洞+Kenney 门件（点进屋） */}
      <g {...door("door", 0, 8.9, 1.1, "门 · 通档案室", ["点进去——canon 书架+日记钉墙"], () => props.toRoom("archive"))}>
        <polygon points={pts([P(0, 7.9, 0), P(0, 9.9, 0), P(0, 9.9, 2.2), P(0, 7.9, 2.2)])} fill="#3a332a" stroke="#4a3a29" strokeWidth="1" />
        <DoorSprite x={0.12} y={8.9} w={64} />
      </g>

      {/* 机房钢门——右墙尽端（点进机房） */}
      <g {...door("door2", 13.3, 0, 1.1, "钢门 · 通机房", ["点进去——常驻器官机柜排"], () => props.toRoom("server"))}>
        <polygon points={pts([P(12.6, 0, 0), P(14, 0, 0), P(14, 0, 2.2), P(12.6, 0, 2.2)])} fill="#7d848f" stroke="#565c66" strokeWidth="1" />
        <line x1={P(13.3, 0, 0)[0]} y1={P(13.3, 0, 0)[1]} x2={P(13.3, 0, 2.2)[0]} y2={P(13.3, 0, 2.2)[1]} stroke="#565c66" strokeWidth="0.9" />
        <text x={P(13.3, 0, 2.35)[0]} y={P(13.3, 0, 2.35)[1]} className="iso-cap" textAnchor="middle">机房</text>
      </g>

      {/* NAV 大屏——右墙挂屏（读数叠层在 Floor.tsx） */}
      <g {...hot({ k: "nav", title: "收银大屏 · NAV+持仓+历史", body: ["点开看净值/在仓/平仓全谱"], to: "/" }, 5.8, 0, 1.95)}>
        <polygon points={pts([P(4.6, 0, 1.3), P(7.0, 0, 1.3), P(7.0, 0, 2.6), P(4.6, 0, 2.6)])} fill="#20242e" stroke="#4a5160" strokeWidth="1.6" />
        <polygon points={pts([P(4.72, 0, 1.42), P(6.88, 0, 1.42), P(6.88, 0, 2.48), P(4.72, 0, 2.48)])} fill="#0d1420" />
      </g>

      {/* 日记钉墙——右墙软木钉板 */}
      <g {...hot({ k: "wall", title: "日记钉墙 · 共享大脑", body: props.notes.length ? props.notes.map((r) => (r.open ? "🔴 " : "") + plainText(r.note).slice(0, 40)) : ["墙上干净——无新钉"], to: "/decision" }, 9.1, 0, 1.5)}>
        <polygon points={pts([P(7.6, 0, 1.05), P(10.7, 0, 1.05), P(10.7, 0, 1.95), P(7.6, 0, 1.95)])} fill="#b08a5e" stroke="#8f6a3d" strokeWidth="1" />
        {props.notes.slice(0, 3).map((r, i) => {
          const x = 7.8 + i * 0.95, face = pts([P(x, 0, 1.2), P(x + 0.68, 0, 1.2), P(x + 0.68, 0, 1.8), P(x, 0, 1.8)]);
          const [px, py] = P(x + 0.34, 0, 1.74);
          return (
            <g key={String(r.escId)}>
              <polygon points={face} fill="#f0ead8" stroke="#a89e80" strokeWidth="0.5" transform={`rotate(${i % 2 ? 3 : -2} ${px} ${py})`} />
              <circle cx={px} cy={py} r="2.6" fill={r.open ? "#ff6b6b" : "#3ddc84"} stroke="#4a4f5a" strokeWidth="0.6" />
            </g>
          );
        })}
      </g>

      {/* venue 屏排——右墙顶带 */}
      <VenueScreens hot={hot} vens={props.vens} />

      {/* 身高尺——右墙角 */}
      <g {...hot({ k: "ruler", title: "身高尺", body: ["现位：婴→少年（A-D 在建）", "判据：点破率归零 + 年化≥8%"], to: "/" }, 0.35, 0, 1.5)}>
        <line x1={P(0.35, 0, 0.4)[0]} y1={P(0.35, 0, 0.4)[1]} x2={P(0.35, 0, 2.5)[0]} y2={P(0.35, 0, 2.5)[1]} stroke="#6a7280" strokeWidth="1.4" />
        {[{ z: 0.75, lb: "婴" }, { z: 1.5, lb: "少年" }, { z: 2.3, lb: "成年" }].map((m) => (
          <g key={m.lb}>
            <line x1={P(0.35, 0, m.z)[0] - 5} y1={P(0.35, 0, m.z)[1]} x2={P(0.35, 0, m.z)[0] + 5} y2={P(0.35, 0, m.z)[1]} stroke="#6a7280" strokeWidth="1.2" />
            <text x={P(0.35, 0, m.z)[0] - 9} y={P(0.35, 0, m.z)[1] + 3} className="iso-cap" textAnchor="end">{m.lb}</text>
          </g>
        ))}
        <path d={`M ${P(0.35, 0, 0.95)[0] + 8} ${P(0.35, 0, 0.95)[1]} l 8 -5 l 0 10 Z`} fill="#d9a62e" />
      </g>

      {/* 白板——立式写字板（工单行钉上） */}
      <g {...hot({ k: "board", title: "白板 · 待办", body: props.openCount ? [`${props.openCount} 件钉在板上——点开看全谱`] : ["板上干净——急件落账后钉在这里"], to: "/decision" }, 2.6, 1.6, 1.2)}>
        <IsoBox x={1.7} y={1.5} w={0.12} d={0.12} h={0.7} top="#8f8b80" front="#7a766b" right="#6a665c" />
        <IsoBox x={3.4} y={1.5} w={0.12} d={0.12} h={0.7} top="#8f8b80" front="#7a766b" right="#6a665c" />
        <IsoBox x={1.4} y={1.45} w={2.4} d={0.12} h={1.3} z0={0.62} top="#f5f2ea" front="#e8e4d8" right="#d8d3c4" />
        {Array.from({ length: Math.min(props.openCount, 4) }).map((_, i) => (
          <line key={i} x1={P(1.65, 1.57, 1.7 - i * 0.24)[0]} y1={P(1.65, 1.57, 1.7 - i * 0.24)[1]}
            x2={P(3.55, 1.57, 1.7 - i * 0.24)[0]} y2={P(3.55, 1.57, 1.7 - i * 0.24)[1]} stroke="#7a6a4d" strokeWidth="1.6" />
        ))}
        {props.openCount === 0 && <text x={P(2.6, 1.57, 1.35)[0]} y={P(2.6, 1.57, 1.35)[1]} className="iso-cap dim">板上干净</text>}
      </g>

      {/* 值班台——白桌+显示器+转椅（在班出人形） */}
      <g {...hot({ k: "desk", title: "值班台 · 唤知会话", body: [props.s?.running ? "在班——正在干活" : "下班了——空椅待命"], to: "/device" }, 7.6, 4.8, 0.9)}>
        <ellipse cx={P(7.7, 5.4)[0]} cy={P(7.7, 5.4)[1]} rx="70" ry="22" fill="#3a3f4b" opacity="0.18" />
        <Desk x={6.3} y={3.9} w={2.7} d={1.35} />
        <Monitor x={7.3} y={4.05} />
        <OfficeChair x={7.65} y={5.7} />
        {props.s?.running && <Person x={7.65} y={5.25} tone="#5b8def" breathe />}
      </g>

      {/* 采集簿记台——常驻器官二席（空椅=无人值守也干活） */}
      <g {...hot({ k: "desk2", title: "采集簿记台", body: ["常驻器官台——collector/projector 没人坐也在转"], to: "/device" }, 10.6, 4.8, 0.9)}>
        <Desk x={9.8} y={3.9} w={2.2} d={1.35} />
        <Monitor x={10.6} y={4.05} />
        <OfficeChair x={10.9} y={5.7} tone="#4a5160" />
      </g>

      {/* 毛坯工位——前排水泥区 */}
      <GhostRow hot={hot} />

      {/* 软装——绿植×2 + 饮水机 */}
      <Plant x={0.9} y={5.1} s={1.1} />
      <Plant x={13.2} y={1.0} s={0.9} />
      <g {...hot({ k: "wc", title: "饮水机", body: ["它不喝——但你看着顺眼就行"], to: "" }, 13.5, 6.7, 0.7)}>
        <WaterCooler x={13.5} y={6.7} />
      </g>

      {/* 吊灯×2——活性灯 */}
      <PendantLamp x={7.7} y={4.6} z={2.65} on={live} />
      <PendantLamp x={4} y={8.4} z={2.65} on={live} />
    </g>
  );
}

// 机房——常驻器官机柜排（o-core/PG/collector/projector/sweeper/eventbus）。
const RACKS = ["o-core", "PG", "collector", "projector", "sweeper", "eventbus"];
export function ServerRoom(props: { live: boolean; warn: boolean; hot: Hot; onExit: () => void }) {
  const { hot, live } = props;
  const W = 12, D = 8, Z = 2.7;
  return (
    <svg viewBox="-380 -140 760 440" className="floor-svg">
      <polygon points={pts([P(0, 0), P(W, 0), P(W, D), P(0, D)])} fill="#565d68" stroke="#3d424c" strokeWidth="1.4" />
      {Array.from({ length: W + 1 }, (_, i) => (
        <line key={"x" + i} x1={P(i, 0)[0]} y1={P(i, 0)[1]} x2={P(i, D)[0]} y2={P(i, D)[1]} stroke="#4c525c" strokeWidth="0.5" />
      ))}
      {Array.from({ length: D + 1 }, (_, i) => (
        <line key={"y" + i} x1={P(0, i)[0]} y1={P(0, i)[1]} x2={P(W, i)[0]} y2={P(W, i)[1]} stroke="#4c525c" strokeWidth="0.5" />
      ))}
      <polygon points={pts([P(0, 0), P(0, D), P(0, D, Z), P(0, 0, Z)])} fill="#4a505c" stroke="#343a44" strokeWidth="1.2" />
      <polygon points={pts([P(0, 0), P(W, 0), P(W, 0, Z), P(0, 0, Z)])} fill="#545a66" stroke="#343a44" strokeWidth="1.2" />

      {/* 机柜排——右墙沿 */}
      {RACKS.map((nm, i) => {
        const x = 0.9 + i * 1.85;
        return (
          <g key={nm} {...hot({ k: "srv:" + nm, title: `机柜 · ${nm}`, body: ["常驻器官一件"], to: "/device" }, x + 0.4, 0.8, 1.2)}>
            <ServerRack x={x} y={0.15} on={live} warn={props.warn && nm === "collector"} />
            <text x={P(x + 0.37, 0.82)[0]} y={P(x + 0.37, 0.82)[1] + 8} className="iso-cap" textAnchor="middle">{nm}</text>
          </g>
        );
      })}

      {/* 回厅门——左墙 */}
      <g className="iso-hot" onClick={props.onExit}>
        <polygon points={pts([P(0, 4.6, 0), P(0, 6.6, 0), P(0, 6.6, 2.1), P(0, 4.6, 2.1)])} fill="#7d848f" stroke="#565c66" strokeWidth="1" />
        <text x={P(0, 5.6, 0.9)[0] + 14} y={P(0, 5.6, 0.9)[1]} className="iso-cap">← 回厅</text>
      </g>

      {/* 吸顶灯带 */}
      {[3, 8].map((x) => {
        const [cx, cy] = P(x, 4, Z);
        return <ellipse key={x} cx={cx} cy={cy} rx="26" ry="7" fill={live ? "#e8ecf2" : "#6a7280"} stroke="#3d424c" strokeWidth="0.7" />;
      })}
      <text x={P(10.5, 6.5)[0]} y={P(10.5, 6.5)[1]} className="iso-cap dim">机房 · 常驻器官机柜</text>
    </svg>
  );
}


