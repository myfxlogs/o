// iso.tsx —— 等距投影绘件（/floor 交易厅用，后续 3D 换渲染层同数据绑定位）。
// P=投影/pts=多点串。

export const T = 30; // 半瓦宽 px
export const P = (x: number, y: number, z = 0): [number, number] => [
  (x - y) * T,
  (x + y) * (T / 2) - z * T,
];
export const pts = (arr: [number, number][]) => arr.map((p) => p.join(",")).join(" ");


