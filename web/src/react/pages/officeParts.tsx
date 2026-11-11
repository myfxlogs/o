// officeParts.tsx —— 写实派办公家具基件（业主令 11-11）。
// 家具=Kenney Furniture Kit CC0 精灵图（等距 SE 角度，独立件可交互）；
// 非家具件仍自绘等距盒。热区由调用方 hot() 包裹，数据绑定位不变。
import { P, pts, IsoBox } from "../components/iso";
import deskUrl from "../assets/furniture/desk.png";
import chairUrl from "../assets/furniture/chairDesk.png";
import screenUrl from "../assets/furniture/computerScreen.png";
import plantUrl from "../assets/furniture/pottedPlant.png";
import plantSUrl from "../assets/furniture/plantSmall1.png";
import fridgeUrl from "../assets/furniture/kitchenFridge.png";
import ceilLampUrl from "../assets/furniture/lampSquareCeiling.png";
import doorUrl from "../assets/furniture/doorway.png";
import trashUrl from "../assets/furniture/trashcan.png";
import booksUrl from "../assets/furniture/books.png";

// 精灵件——底中锚点=地板落点（x,y,z 投影），w=目标屏宽 px。
const SPR = {
  desk: { u: deskUrl, w: 116, h: 122 },
  chair: { u: chairUrl, w: 60, h: 97 },
  screen: { u: screenUrl, w: 47, h: 59 },
  plant: { u: plantUrl, w: 28, h: 85 },
  plantS: { u: plantSUrl, w: 14, h: 18 },
  fridge: { u: fridgeUrl, w: 72, h: 146 },
  ceilLamp: { u: ceilLampUrl, w: 24, h: 39 },
  door: { u: doorUrl, w: 59, h: 148 },
  trash: { u: trashUrl, w: 33, h: 64 },
  books: { u: booksUrl, w: 26, h: 27 },
} as const;
export function Sprite(props: { name: keyof typeof SPR; x: number; y: number; z?: number; w: number }) {
  const m = SPR[props.name];
  const [px, py] = P(props.x, props.y, props.z ?? 0);
  const h = props.w * m.h / m.w;
  return <image href={m.u} x={px - props.w / 2} y={py - h} width={props.w} height={h} />;
}

// 白面桌——Kenney 木桌精灵（w/d 参数弃用，视觉宽由调用方 w 定）。
export function Desk(props: { x: number; y: number; w?: number; d?: number }) {
  return <Sprite name="desk" x={props.x + (props.w ?? 2.6) / 2} y={props.y + (props.d ?? 1.35) / 2} w={112} />;
}

// 转椅——Kenney 办公椅精灵。
export function OfficeChair(props: { x: number; y: number; tone?: string }) {
  void props.tone;
  return <Sprite name="chair" x={props.x} y={props.y} w={48} />;
}

// 显示器——Kenney 显示器精灵（z0=落桌高）。
export function Monitor(props: { x: number; y: number; z0?: number }) {
  return <Sprite name="screen" x={props.x} y={props.y} z={props.z0 ?? 0.72} w={34} />;
}

// 绿植——Kenney 盆栽精灵（大株/小株）。
export function Plant(props: { x: number; y: number; s?: number }) {
  return <Sprite name={(props.s ?? 1) > 0.95 ? "plant" : "plantS"} x={props.x} y={props.y} w={(props.s ?? 1) > 0.95 ? 40 : 34} />;
}

// 茶水间角——冰箱+咖啡机（饮水机替位，Kenney 无独立水机）。
export function WaterCooler(props: { x: number; y: number }) {
  return (
    <g>
      <Sprite name="fridge" x={props.x} y={props.y} w={52} />
      <Sprite name="trash" x={props.x - 0.8} y={props.y + 0.3} w={26} />
    </g>
  );
}

// 吊灯——吊线+Kenney 方顶灯精灵+光锥（live 时发光）。
export function PendantLamp(props: { x: number; y: number; z: number; on: boolean; id?: string }) {
  const [cx, cy] = P(props.x, props.y, props.z);
  const [bx, by] = P(props.x, props.y, 0);
  return (
    <g>
      <line x1={cx} y1={cy - 60} x2={cx} y2={cy} stroke="#5a5f6b" strokeWidth="1.1" />
      {props.on && <ellipse cx={cx} cy={cy + 20} rx="40" ry="17" fill="url(#lampGlow)" />}
      <Sprite name="ceilLamp" x={props.x} y={props.y} z={props.z - 0.42} w={26} />
      {props.on && <ellipse cx={bx} cy={by} rx="30" ry="12" fill="#ffd76a" opacity="0.08" />}
    </g>
  );
}

// 木门板件——Kenney doorway 精灵（墙上锚：底中=门底中点投影位）。
export function DoorSprite(props: { x: number; y: number; w?: number }) {
  return <Sprite name="door" x={props.x} y={props.y} w={props.w ?? 62} />;
}

// 书架/书籍精灵——档案室复用。
export function BookcaseSprite(props: { x: number; y: number; w?: number }) {
  return <Sprite name="books" x={props.x} y={props.y} w={props.w ?? 26} />;
}

// 左墙大窗——玻璃面+横竖棂条（贴 gx=0 墙；y 段自定）。
export function WallWindowL(props: { y0: number; y1: number; z0: number; z1: number }) {
  const { y0, y1, z0, z1 } = props;
  const ym = (y0 + y1) / 2, zm = (z0 + z1) / 2;
  return (
    <g>
      <polygon points={pts([P(0, y0, z0), P(0, y1, z0), P(0, y1, z1), P(0, y0, z1)])} fill="url(#sky)" stroke="#8f959e" strokeWidth="1.6" />
      <line x1={P(0, ym, z0)[0]} y1={P(0, ym, z0)[1]} x2={P(0, ym, z1)[0]} y2={P(0, ym, z1)[1]} stroke="#8f959e" strokeWidth="1" />
      <line x1={P(0, y0, zm)[0]} y1={P(0, y0, zm)[1]} x2={P(0, y1, zm)[0]} y2={P(0, y1, zm)[1]} stroke="#8f959e" strokeWidth="1" />
      <polygon points={pts([P(0, y0, z0 - 0.06), P(0, y1, z0 - 0.06), P(0, y1, z0), P(0, y0, z0)])} fill="#b9bec7" />
    </g>
  );
}

// 机柜——深柜体+两列 LED（绿=活/灰=断），机房用。
export function ServerRack(props: { x: number; y: number; h?: number; on: boolean; warn?: boolean }) {
  const { x, y } = props, h = props.h ?? 2.1;
  const led = props.on ? (props.warn ? "#ffb454" : "#3ddc84") : "#5a6274";
  return (
    <g>
      <IsoBox x={x} y={y} w={0.75} d={0.6} h={h} top="#2c3140" front="#232836" right="#1c2130" />
      {[0.35, 0.75, 1.15, 1.55, 1.9].map((z, i) => (
        <g key={i}>
          <line x1={P(x + 0.06, y + 0.6, z)[0]} y1={P(x + 0.06, y + 0.6, z)[1]} x2={P(x + 0.7, y + 0.6, z)[0]} y2={P(x + 0.7, y + 0.6, z)[1]} stroke="#12161f" strokeWidth="1" />
          <circle cx={P(x + 0.12, y + 0.6, z - 0.12)[0]} cy={P(x + 0.12, y + 0.6, z - 0.12)[1]} r="1.6" fill={led} className={props.on ? "fx-breathe" : ""} />
        </g>
      ))}
    </g>
  );
}
