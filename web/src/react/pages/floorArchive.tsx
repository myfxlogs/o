// floorArchive.tsx —— 档案室（/floor 子房间，点主厅门进入）。
// 全自写等距景：左墙=工程 canon 书架，右墙=经济 canon 架+日记钉墙，
// 中央阅档台+吊灯，前门回主厅。canon 数=书脊实数，钉=ops_event 最新。
import { P, pts, IsoBox } from "../components/iso";
import { PersonSprite } from "./officeParts";
import { plainText } from "../components/plain";
import type { Tip } from "./floorFocus";

export function ArchiveRoom(props: {
  live: boolean;
  engCanon: number;
  econCanon: number;
  notes: { escId: bigint; note: string; open: boolean }[];
  hot: (tt: Omit<Tip, "x" | "y">, x: number, y: number, z?: number, cls?: string) => Record<string, unknown>;
  onExit: () => void;
}) {
  const { hot, live } = props;
  const W = 12, D = 8, WZ = 3;
  const eng = Math.min(props.engCanon, 24);
  const econ = Math.min(props.econCanon, 12);

  return (
    <svg viewBox="-400 -140 800 480" className="floor-svg">
      <defs>
        <radialGradient id="aGlow">
          <stop offset="0%" stopColor="#c9a86a" stopOpacity="0.5" />
          <stop offset="100%" stopColor="#c9a86a" stopOpacity="0" />
        </radialGradient>
      </defs>
      {/* 地板 */}
      <polygon points={pts([P(0, 0), P(W, 0), P(W, D), P(0, D)])} fill="#26221c" stroke="#171c26" strokeWidth="1.5" />
      {/* 两面墙——档案室色温偏暖木 */}
      <polygon points={pts([P(0, 0), P(0, D), P(0, D, WZ), P(0, 0, WZ)])} fill="#3a342a" stroke="#171c26" strokeWidth="1.2" />
      <polygon points={pts([P(0, 0), P(W, 0), P(W, 0, WZ), P(0, 0, WZ)])} fill="#443d30" stroke="#171c26" strokeWidth="1.2" />
      <text x={P(6, 0, 2.8)[0]} y={P(6, 0, 2.8)[1]} className="iso-cap" textAnchor="middle" fill="#c9a86a" fontSize="12">档 案 室</text>

      {/* 左墙工程书架——三格通高，canon 实数书脊 */}
      <g {...hot({ k: "shelf", title: "工程 canon 架", body: [`${props.engCanon} 盒在架`], to: "/decision" }, 0.5, 3, 1.4)}>
        <IsoBox x={0.15} y={0.6} w={0.7} d={6.4} h={2.6} top="#5a4632" front="#4a3a29" right="#3d2f21" />
        {[0.85, 1.7].map((z) => (
          <line key={z} x1={P(0.85, 0.65, z)[0]} y1={P(0.85, 0.65, z)[1]} x2={P(0.85, 6.95, z)[0]} y2={P(0.85, 6.95, z)[1]} stroke="#2b2117" strokeWidth="1.4" />
        ))}
        {Array.from({ length: eng }).map((_, i) => {
          const gy = 0.75 + (i % 8) * 0.72, gz = 0.85 + Math.floor(i / 8) * 0.85;
          const col = i % 4 === 0 ? "#7aa2f7" : i % 4 === 1 ? "#3ddc84" : i % 4 === 2 ? "#bb9af7" : "#ffd76a";
          return <polygon key={i} points={pts([P(0.85, gy, gz), P(0.85, gy + 0.5, gz), P(0.85, gy + 0.5, gz + 0.62), P(0.85, gy, gz + 0.62)])} fill={col} stroke="#0b0e14" strokeWidth="0.4" />;
        })}
      </g>

      {/* 右墙经济架（小）+ 日记钉墙（大） */}
      <g {...hot({ k: "econshelf", title: "经济 canon 架", body: [`${props.econCanon} 盒在架`], to: "/decision" }, 11, 0.4, 1.2)}>
        <IsoBox x={11.15} y={0.15} w={0.7} d={2.2} h={2.4} top="#5a4632" front="#4a3a29" right="#3d2f21" />
        {econ === 0 ? (
          <text x={P(11.15, 1.2, 1.2)[0]} y={P(11.15, 1.2, 1.2)[1]} className="iso-cap dim">空</text>
        ) : (
          Array.from({ length: econ }).map((_, i) => {
            const gy = 0.3 + i * 0.6;
            return <polygon key={i} points={pts([P(11.15, gy, 1.55), P(11.15, gy + 0.42, 1.55), P(11.15, gy + 0.42, 2.1), P(11.15, gy, 2.1)])} fill="#e8b34d" stroke="#0b0e14" strokeWidth="0.4" />;
          })
        )}
      </g>
      {/* 钉墙——右墙中后段，ops_event 六钉 */}
      <g {...hot({ k: "wall", title: "日记钉墙 · ops_event", body: props.notes.length ? props.notes.map((r) => plainText(r.note).slice(0, 36)) : ["墙上干净"], to: "/decision" }, 9, 0.3, 1.6)}>
        {props.notes.slice(0, 6).map((r, i) => {
          const x = 5.4 + i * 0.95;
          const [px, py] = P(x + 0.33, 0, 1.72);
          return (
            <g key={String(r.escId)}>
              <polygon points={pts([P(x, 0, 1.1), P(x + 0.66, 0, 1.1), P(x + 0.66, 0, 1.78), P(x, 0, 1.78)])} fill="#e8e2d0" stroke="#8a8268" strokeWidth="0.5" transform={`rotate(${i % 2 ? 2 : -3} ${px} ${py})`} />
              <circle cx={px} cy={py} r="2.4" fill={r.open ? "#ff6b6b" : "#3ddc84"} stroke="#0b0e14" strokeWidth="0.5" />
            </g>
          );
        })}
      </g>

      {/* 阅档台 */}
      <g {...hot({ k: "adesk", title: "阅档台", body: ["史官/会话翻账的位置——经济案都在这读"], to: "/decision" }, 6, 4.5, 0.7)}>
        <IsoBox x={5} y={4} w={2} d={1.2} h={0.75} top="#6b543c" front="#5a4632" right="#4a3a29" />
        <IsoBox x={6.1} y={4.15} w={0.7} d={0.45} h={0.06} z0={0.75} top="#e8e2d0" front="#d9d2bc" right="#c9c1a8" />
      </g>
      <PersonSprite n="idle_3" x={5.4} y={3.6} h={46} tone={live ? "#c9a86a" : undefined} />

      {/* 吊灯 */}
      {(() => { const [lx, ly] = P(6, 4); return (
        <>
          <line x1={lx} y1={-130} x2={lx} y2={ly - 60} stroke="#6b5a42" strokeWidth="1.1" />
          {live && <ellipse cx={lx} cy={ly - 42} rx="40" ry="17" fill="url(#aGlow)" />}
          <path d={`M ${lx - 12} ${ly - 60} L ${lx + 12} ${ly - 60} L ${lx + 5} ${ly - 46} L ${lx - 5} ${ly - 46} Z`} fill={live ? "#e8b34d" : "#4d5a75"} stroke="#0b0e14" strokeWidth="0.7" />
        </>
      ); })()}

      {/* 回厅门——前右角 */}
      <g onClick={props.onExit} className="iso-hot">
        <polygon points={pts([P(W, 5.4, 0), P(W, 7.6, 0), P(W, 7.6, 2.1), P(W, 5.4, 2.1)])} fill="#1d2432" stroke="#11161f" strokeWidth="0.8" />
        <polygon points={pts([P(W, 5.65, 0.05), P(W, 7.35, 0.05), P(W, 7.35, 1.95), P(W, 5.65, 1.95)])} fill="#55617a" stroke="#11161f" strokeWidth="0.6" />
        <text x={P(W, 6.5, 2.3)[0]} y={P(W, 6.5, 2.3)[1]} className="iso-cap" textAnchor="middle">← 回厅</text>
      </g>
    </svg>
  );
}
