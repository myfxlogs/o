// api.ts —— O Connect 直连件（T-0156：REST shim 撤——proto 单源 buf regen，
// CoreService 同源直连；组件零感同签名——T-0113 铺路兑现）。
// 读面 RPC 合并窗（T-0269 事故修复——双取/轰炸放大器病灶根修）：
//   · 同 method+args 在飞=共享同一 promise（挂载波/同帧波多卡并发合一——
//     /trx getHedgeStructure 三卡同帧三取合一，e2e 实测 ×3→×1）；
//   · 成功落定后 MERGE_MS 内同键再取=直返末值（挂载取数↔紧跟的首波 ev_*
//     帧重取合流——e2e 实测双取病灶；事件驱动律不变：窗后新帧照取）；
//   · 连续败=指数降频（2^failN 秒封顶 30s——unary 死路径轰炸同退避律，
//     降频内同键再调直返同错不新发包）。
// 取舍落码：合并窗内帧语义=「该波取数已被近一次取覆盖」——展示值最坏滞后
// ~MERGE_MS（ev_* 周期面下帧自愈；低频面同刻多卡共享即本票钦定用途），
// 换取断流风暴与重取风暴的传输面安静。写径（trxAccountRefresh）与流
//（watch）不入窗。
import { createClient, type CallOptions, type Client } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { CoreService } from "../gen/dashboardv1/core_pb";

export function errText(e: unknown): string {
  return e instanceof Error ? e.message : String(e)
}

const MERGE_MS = 4_500; // 落定合并窗——盖挂载→首帧波间隔（e2e 实测 ~3s）又短于周期帧拍
const FAIL_CAP_MS = 30_000;

type Ent = {
  inflight?: Promise<unknown>;
  okAt?: number;
  val?: unknown;
  err?: unknown;
  failAt?: number;
  failN?: number;
};
const tab = new Map<string, Ent>();

function coalesce<T>(key: string, fn: () => Promise<T>): Promise<T> {
  const now = Date.now();
  const e = tab.get(key);
  if (e?.inflight) return e.inflight as Promise<T>;
  if (e?.okAt !== undefined && now - e.okAt < MERGE_MS) return Promise.resolve(e.val as T);
  if (
    e?.failAt !== undefined &&
    now - e.failAt < Math.min(2 ** (e.failN ?? 0) * 1_000, FAIL_CAP_MS)
  )
    return Promise.reject(e.err);
  const p = fn().then(
    (v) => {
      tab.set(key, { okAt: Date.now(), val: v, failN: 0 });
      return v;
    },
    (err: unknown) => {
      const prev = tab.get(key);
      tab.set(key, { failAt: Date.now(), err, failN: (prev?.failN ?? 0) + 1 });
      throw err;
    }
  );
  tab.set(key, { inflight: p, failN: e?.failN ?? 0 });
  // 陈条清面——键域=method+args（长 session 翻页/筛参累积有界化）。
  if (tab.size > 64) {
    for (const [k, x] of tab) {
      const last = Math.max(x.okAt ?? 0, x.failAt ?? 0);
      if (!x.inflight && now - last > FAIL_CAP_MS) tab.delete(k);
    }
  }
  return p;
}

type UnaryFn<Req, Res> = (req: Req, opt?: CallOptions) => Promise<Res>;
function deduped<Req, Res>(name: string, fn: UnaryFn<Req, Res>): UnaryFn<Req, Res> {
  return (req, opt) => coalesce(`${name} ${JSON.stringify(req)}`, () => fn(req, opt));
}

// 同源直连（Connect 协议 POST /o.dashboard.v1.CoreService/<Method>）。
const raw = createClient(CoreService,
  createConnectTransport({ baseUrl: window.location.origin }));

export const client: Client<typeof CoreService> = {
  ...raw,
  ping: deduped("Ping", raw.ping),
  overview: deduped("Overview", raw.overview),
  getTrxAccount: deduped("GetTrxAccount", raw.getTrxAccount),
  listTrxTxs: deduped("ListTrxTxs", raw.listTrxTxs),
  getTrxYieldRadar: deduped("GetTrxYieldRadar", raw.getTrxYieldRadar),
  getTrxYieldCurve: deduped("GetTrxYieldCurve", raw.getTrxYieldCurve),
  getHedgeStructure: deduped("GetHedgeStructure", raw.getHedgeStructure),
  getTrxVoteStatus: deduped("GetTrxVoteStatus", raw.getTrxVoteStatus),
  listHedgeFlow: deduped("ListHedgeFlow", raw.listHedgeFlow),
  getSystemHealth: deduped("GetSystemHealth", raw.getSystemHealth),
  getCockpit: deduped("GetCockpit", raw.getCockpit),
  getHostMetrics: deduped("GetHostMetrics", raw.getHostMetrics),
  listReports: deduped("ListReports", raw.listReports),
  getEquityCurve: deduped("GetEquityCurve", raw.getEquityCurve),
  listCashflows: deduped("ListCashflows", raw.listCashflows),
  listClosedPositions: deduped("ListClosedPositions", raw.listClosedPositions),
  getHeadless: deduped("GetHeadless", raw.getHeadless),
  getPipeline: deduped("GetPipeline", raw.getPipeline),
};
