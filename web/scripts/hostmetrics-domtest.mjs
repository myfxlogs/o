// hostmetrics-domtest.mjs —— T-0434 前端陈旧硬闸 DOM 断言（范式同
// quoteboard-domtest.mjs：renderToStaticMarkup+esbuild 捆扎，零新依赖）。
// 断言锚（票面断言④）：①stale 行=数值区置灰壳（hm-stale-body）+红标
// 「数据陈旧·非实时」（hm-stale-badge）+拍断语+停拍 hint——尸体一眼识破；
// ②客户端龄复核硬闸：服务端标 false 而 ts 龄>2min（断流僵尸帧）同被闸；
// ③鲜行零回归：无红标/无置灰壳/hm-fill 条照渲；④无行=显式「无数据」态。
// 运行：node scripts/hostmetrics-domtest.mjs（web/ 下）。
import { build } from "esbuild";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { pathToFileURL, fileURLToPath } from "node:url";

const webRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const comp = join(webRoot, "src", "react", "components", "hostmetrics.tsx").replaceAll("\\", "/");
const store = join(webRoot, "src", "react", "store.ts").replaceAll("\\", "/");

const entry = `
import { renderToStaticMarkup } from "react-dom/server";
import { createElement as h } from "react";
import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { HostMetricsCard } from "${comp}";
import { setHostMetrics } from "${store}";
export { renderToStaticMarkup, h, create, TimestampSchema, HostMetricsCard, setHostMetrics };
`;

const res = await build({
	stdin: { contents: entry, resolveDir: webRoot, loader: "ts", sourcefile: "hostmetrics-domtest-entry.ts" },
	bundle: true, write: false, format: "esm", platform: "node",
	jsx: "automatic", logLevel: "silent",
	banner: { js: `import { createRequire } from "node:module"; const require = createRequire(import.meta.url); globalThis.window ??= { location: { origin: "http://domtest" } };` },
});
const tmp = mkdtempSync(join(tmpdir(), "hmdt-"));
const outFile = join(tmp, "entry.mjs");
writeFileSync(outFile, res.outputFiles[0].text);
const { renderToStaticMarkup, h, create, TimestampSchema, HostMetricsCard, setHostMetrics } = await import(pathToFileURL(outFile).href);

const nowS = Math.floor(Date.now() / 1000);
const tsOf = (ageS) => create(TimestampSchema, { seconds: BigInt(nowS - ageS) });
const host = (name, { stale = false, ageS = 0 } = {}) => ({
	host: name, stale,
	ts: tsOf(ageS),
	cores: 4, cpuPct: 12.5, memTotalMb: 8192, memUsedMb: 4096,
	diskTotalGb: 450, diskUsedGb: 252, netRxKbps: 100, netTxKbps: 50,
	uptimeHours: 240,
});

let fails = 0;
const check = (name, cond) => {
	if (cond) console.log(`ok  ${name}`);
	else { console.error(`FAIL ${name}`); fails++; }
};
const render = (hosts) => {
	setHostMetrics({ hosts });
	return renderToStaticMarkup(h(HostMetricsCard));
};

// ① 服务端 stale 行：置灰壳+红标+拍断语+停拍 hint——尸体一眼识破。
const m1 = render([host("sg", { stale: true, ageS: 8 * 86400 })]);
check("stale 红标「数据陈旧·非实时」在", m1.includes("数据陈旧·非实时") && m1.includes("hm-stale-badge"));
check("stale 数值区置灰壳（hm-stale-body）在", m1.includes("hm-stale-body"));
check("stale 拍断语在", m1.includes("拍断"));
check("stale 停拍 hint 在", m1.includes("自报停拍"));
check("stale 行数值可留（磁盘数字在）", m1.includes("450") && m1.includes("252"));

// ② 客户端龄复核硬闸：服务端标 false 而 ts 龄 8 天（断流僵尸帧）同被闸。
const m2 = render([host("sg", { stale: false, ageS: 8 * 86400 })]);
check("僵尸帧（标 false 龄 8d）被红标闸住", m2.includes("数据陈旧·非实时") && m2.includes("hm-stale-body"));

// ③ 鲜行零回归：无红标/无置灰壳/条照渲/无拍断语。
const m3 = render([host("sg", { stale: false, ageS: 5 })]);
check("鲜行无红标", !m3.includes("数据陈旧·非实时") && !m3.includes("hm-stale-badge"));
check("鲜行无置灰壳", !m3.includes("hm-stale-body"));
check("鲜行 hm-fill 条照渲", m3.includes("hm-fill"));
check("鲜行无拍断语", !m3.includes("拍断"));

// ④ 无行=显式「无数据」态不空跑。
const m4 = render([]);
check("无行显式「无数据」态", m4.includes("无数据"));

if (fails > 0) {
	console.error(`❌ hostmetrics-domtest ${fails} 断言红`);
	process.exit(1);
}
console.log("✅ hostmetrics-domtest 全绿");
