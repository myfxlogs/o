// metaseats-domtest.mjs —— T-0349 渲染面最小断言（repo 无 web 测试面——
// 取 react-dom/server renderToString DOM 断言 + esbuild（vite 内置 dep）
// 捆扎，零新依赖；取舍记 T-0349 结果段）。运行：npm test（web/ 下）。
// 断言四锚：白名单阳性（REGIME/VENUEWATCH 现表真形）· 工程席零现 ·
// 超龄剔行非红标 · 白名单外默认不显。
import { build } from "esbuild";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { pathToFileURL } from "node:url";
import { fileURLToPath } from "node:url";

const webRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const comp = join(webRoot, "src", "react", "components", "metaseats.tsx").replaceAll("\\", "/");

// 入口经 stdin 注入（esbuild 全捆扎组件+react-dom/server——自包含零外依，
// /tmp 落件不涉 node_modules 解析径）。
const entry = `
import { renderToStaticMarkup } from "react-dom/server";
import { createElement as h } from "react";
import { MetaSeats } from "${comp}";
export { renderToStaticMarkup, h, MetaSeats };
`;

const res = await build({
	stdin: { contents: entry, resolveDir: webRoot, loader: "ts", sourcefile: "metaseats-domtest-entry.ts" },
	bundle: true, write: false, format: "esm", platform: "node",
	jsx: "automatic", logLevel: "silent",
	// react-dom-server node 变体 CJS 动态 require 内建件——ESM 落件挂 createRequire 桥
	banner: { js: `import { createRequire } from "node:module"; const require = createRequire(import.meta.url);` },
});
const tmp = mkdtempSync(join(tmpdir(), "msdt-"));
const outFile = join(tmp, "entry.mjs");
writeFileSync(outFile, res.outputFiles[0].text);
const { renderToStaticMarkup, h, MetaSeats } = await import(pathToFileURL(outFile).href);

// fixture：现在起 ageS 秒前的 Timestamp（seconds=bigint——@bufbuild v2 型）
const nowS = Math.floor(Date.now() / 1000);
const row = (ticket, seat, state, ageS, withTs = true) => ({
	ticket, seat, state, model: "", phase: "",
	...(withTs ? { lastActivity: { seconds: BigInt(nowS - ageS) } } : {}),
});
const hlL = (s) => s; // 断言用恒等映射（组件对 hlL 输出仅做字面比对）
const fmtLagMin = (sec) => `${sec}s`;
const render = (rows) => renderToStaticMarkup(h(MetaSeats, { rows, hlL, fmtLagMin }));

let fail = 0, n = 0;
const check = (name, cond) => { n++; console.log(`${cond ? "PASS" : "FAIL"} ${name}`); if (!cond) fail++; };

// ① 白名单阳性——两业务席鲜行全渲（真表现形：REGIME=分析席/VENUEWATCH=venue哨兵）
const base = render([
	row("META-REGIME", "分析席", "在飞", 120),
	row("META-VENUEWATCH", "venue哨兵", "在飞", 600),
]);
check("白名单阳性:REGIME 渲", base.includes("META-REGIME"));
check("白名单阳性:VENUEWATCH 渲", base.includes("META-VENUEWATCH"));
check("白名单阳性:恰好两注行", (base.match(/kpi-note/g) || []).length === 2);

// ② 工程席零现——DM/VERDICT/GAUGES/GC/Dev-T-* 全不渲（真表僵尸样本 T-0329 对形）
const eng = render([
	row("META-DM", "决策席", "在飞", 10),
	row("META-VERDICT", "裁定席", "在飞", 10),
	row("META-GAUGES", "巡检闸", "在飞", 10),
	row("META-GC", "GC拍", "在飞", 10),
	row("META-Dev-T-0329", "施工席", "在飞", 61200), // 现表实证：17h alive 冻结
]);
check("工程席零现:五席全不渲", !/META-(DM|VERDICT|GAUGES|GC|Dev-T-0329)/.test(eng));
check("工程席零现:注行数为零", (eng.match(/kpi-note/g) || []).length === 0);

// ③ 超龄剔行非红标——REGIME 3600s>1800s 龄：行整剔（fixture 全 在飞 态，
//    输出若现 u-bad 即标红径残留）
const over = render([
	row("META-REGIME", "分析席", "在飞", 3600), // >1800s 超龄
	row("META-VENUEWATCH", "venue哨兵", "在飞", 600),
]);
check("超龄剔行:REGIME 超龄行不渲", !over.includes("META-REGIME"));
check("超龄剔行:剔行非标红（零 u-bad）", !over.includes("u-bad"));
check("超龄剔行:余 VENUEWATCH 一行", (over.match(/kpi-note/g) || []).length === 1);

// ④ 龄闸边界——恰等 cadence（3600=3600）不超不剔；超 1s 即剔
const edge = render([
	row("META-VENUEWATCH", "venue哨兵", "在飞", 3600),
]);
check("龄闸边界:恰等 cadence 渲", edge.includes("META-VENUEWATCH"));
const edge1 = render([
	row("META-VENUEWATCH", "venue哨兵", "在飞", 3601),
]);
check("龄闸边界:超 1s 即剔", !edge1.includes("META-VENUEWATCH"));

// ⑤ 白名单外默认不显——非 META 票与无 lastActivity 白名单行各对拍
const misc = render([
	row("BASESWP", "Dev", "在飞", 10),          // 非 META 票
	row("META-REGIME", "分析席", "在飞", 0, false), // 白名单缺时间戳=age0 渲（承旧语义）
]);
check("白名单外:非 META 票不渲", !misc.includes("BASESWP"));
check("缺时间戳:白名单行 age0 渲", misc.includes("META-REGIME"));

console.log(fail === 0 ? `DOM-ASSERT 全绿 ${n}/${n}` : `DOM-ASSERT 红 ${n - fail}/${n} 失败`);
process.exit(fail === 0 ? 0 : 1);
