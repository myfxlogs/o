// floorArchive.tsx —— 档案室（/floor 子房,点主厅档案链进屋）。
// 模拟盘令牌版:canon 两计数块+钉墙文本列——钉=ops_event 最新行,可直读。
import { P } from "../components/iso";
import { IsoGrid, ZoneCap, Cell } from "./floorTokens";
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
  const W = 12, D = 8;

  return (
    <svg viewBox="-390 -60 960 480" className="floor-svg">
      <IsoGrid w={W} d={D} />
      <ZoneCap x={0.8} y={1.0} label="档案室 · canon 与钉墙" />

      {/* canon 双架——计数令牌 */}
      <g {...hot({ k: "shelf", title: "工程 canon 架", body: [`${props.engCanon} 盒在架`], to: "/decision" }, 3.0, 2.8)}>
        <Cell x={1.6} y={1.6} w={2.8} d={2.0} tone={live ? "#c9a86a" : "#8a8268"}
          lines={["工程 canon", `${props.engCanon} 盒`]} fs={11} dim={!live} />
      </g>
      <g {...hot({ k: "econshelf", title: "经济 canon 架", body: [props.econCanon ? `${props.econCanon} 盒在架` : "空架——裁定件无一入库"], to: "/decision" }, 7.0, 2.8)}>
        <Cell x={5.6} y={1.6} w={2.8} d={2.0} tone={props.econCanon ? "#e8b34d" : "#5a6274"}
          lines={["经济 canon", `${props.econCanon} 盒`]} fs={11} dim={!props.econCanon} />
      </g>

      {/* 日记钉墙——ops_event 尾流令牌列(文本直读,最多 6 行) */}
      <g {...hot({ k: "wall", title: "日记钉墙 · ops_event", body: props.notes.length ? props.notes.map((r) => plainText(r.note).slice(0, 36)) : ["墙上干净"], to: "/decision" }, 9.6, 1.6)}>
        <Cell x={9.6} y={1.6} w={1.9} d={0.72} tone="#b08a5e" lines={["钉墙", `${props.notes.length} 条`]} fs={8.5} />
      </g>
      {props.notes.slice(0, 6).map((r, i) => (
        <text key={String(r.escId)} x={P(1.6, 4.4 + i * 0.5)[0]} y={P(1.6, 4.4 + i * 0.5)[1]}
          className="iso-cap" fontSize="8.5" fill={r.open ? "#ff9b9b" : "#9aa7bd"}>
          {r.open ? "●" : "○"} #{String(r.escId)} {plainText(r.note).slice(0, 52)}
        </text>
      ))}

      <text x={P(9.6, 7.4)[0]} y={P(9.6, 7.4)[1]} className="iso-cap iso-hot" onClick={props.onExit}>← 回厅</text>
    </svg>
  );
}
