// plain.ts —— 显示层人话字典：ops_event 原文 append-only 不动，
// 渲染时映射；悬浮 title 露原文（诚实层——机器方言是事实账，人读面才翻译）。

// 事件分类键 → 人话（esc_class，开放集——未录键原样透传）。
const CLASS_CN: Record<string, string> = {
  数据缺: "数据断供",
  唤知: "值守唤醒",
  部署: "部署",
  商业产出: "业务产出",
  观测盲: "观测盲区",
  探测: "探测器",
  钱径: "钱径",
};

// 事件对象键 → 人话（esc_scope，开放集——尾缀 #N 票锚先剥，未录键原样透传）。
const SCOPE_CN: Record<string, string> = {
  nav_stale: "NAV 净值停更",
  pair_zero: "配对恒零",
  stuck_esc: "卡件复燃",
  deploy_blocked: "部署受阻",
  intake_dry: "研究断粮",
};

// plainKind 行题人话——「数据缺/nav_stale」→「数据断供 · NAV 净值停更」；
// 空 class=业主令行（note 词头辨）→「业主令」。
export function plainKind(cls: string, scope: string, note: string): string {
  if (!cls) return note.startsWith("业主令") ? "业主令" : "系统上报";
  const c = CLASS_CN[cls] ?? cls;
  const s = scope.replace(/#\d+$/, "");
  const face = /^face:service:(.+)$/.exec(s);
  const sc = SCOPE_CN[s] ?? (face ? `新增服务面 ${face[1]}` : s);
  return `${c} · ${sc}`;
}

// 机器方言短语 → 人话（先长后短有序替换——「领命在案」先于「在案」）。
const PHRASES: [RegExp, string][] = [
  [/v=\d+\s*/g, ""], // 版本戳裸串
  [/^\S{1,12}\/\S{1,40}:\s*/, ""], // note 头 class/scope 复述（行题已显）
  [/销案后复燃|复燃|复岗火|复火/g, "再次触发"],
  [/销——/g, "处理完："],
  [/销案|销账/g, "结案"],
  [/转工单|立票/g, "已转工单"],
  [/领命在案/g, "已受理"],
  [/复拍复流/g, "恢复采集"],
  [/复拍/g, "恢复采集"],
  [/复绿/g, "已恢复"],
  [/归唤知会话|唤知会话/g, "值守会话"],
  [/坑处置/g, "故障处置"],
  [/日拍/g, "每日采集"],
  [/末拍/g, "末次"],
  [/下拍/g, "下一轮"],
  [/新拍/g, "新一轮"],
  [/首拍/g, "首次"],
  [/拍/g, "轮"],
  [/落账/g, "入账"],
  [/在案/g, "已记录"],
  [/判词/g, "结论"],
  [/升件/g, "上报"],
];

// plainText 注文/判词人话——原文不入 DOM 文本但挂 title 可查。
export function plainText(s: string): string {
  let t = s;
  for (const [re, to] of PHRASES) t = t.replace(re, to);
  return t;
}
