// iso.tsx —— 等距投影绘件（/floor 交易厅用，后续 3D 换渲染层同数据绑定位）。
// P=投影/pts=多点串/IsoBox=三面盒。

export const T = 30; // 半瓦宽 px
export const P = (x: number, y: number, z = 0): [number, number] => [
  (x - y) * T,
  (x + y) * (T / 2) - z * T,
];
export const pts = (arr: [number, number][]) => arr.map((p) => p.join(",")).join(" ");

// 等距盒：顶面+前 y 面+右 x 面（视角从 +x+y 看入角）
export function IsoBox(props: {
  x: number; y: number; w: number; d: number; h: number; z0?: number;
  top: string; front: string; right: string; dashed?: boolean; op?: number;
}) {
  const { x, y, w, d, h, z0 = 0 } = props;
  const top = pts([P(x, y, z0 + h), P(x + w, y, z0 + h), P(x + w, y + d, z0 + h), P(x, y + d, z0 + h)]);
  const front = pts([P(x, y + d, z0), P(x + w, y + d, z0), P(x + w, y + d, z0 + h), P(x, y + d, z0 + h)]);
  const right = pts([P(x + w, y, z0), P(x + w, y + d, z0), P(x + w, y + d, z0 + h), P(x + w, y, z0 + h)]);
  const dash = props.dashed ? { strokeDasharray: "4 3" } : {};
  const st = { fillOpacity: props.op ?? 1, ...dash };
  return (
    <g>
      <polygon points={top} fill={props.top} stroke="#0b0e14" strokeWidth="0.6" style={st} />
      <polygon points={front} fill={props.front} stroke="#0b0e14" strokeWidth="0.6" style={st} />
      <polygon points={right} fill={props.right} stroke="#0b0e14" strokeWidth="0.6" style={st} />
    </g>
  );
}

