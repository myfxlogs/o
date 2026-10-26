// api.ts —— O Connect 直连件（T-0156：REST shim 撤——proto 单源 buf regen，
// CoreService 同源直连；组件零感同签名——T-0113 铺路兑现）。
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { CoreService } from "../gen/dashboardv1/core_pb";

export function errText(e: unknown): string {
  return e instanceof Error ? e.message : String(e)
}

// 同源直连（Connect 协议 POST /o.dashboard.v1.CoreService/<Method>）。
export const client = createClient(CoreService,
  createConnectTransport({ baseUrl: window.location.origin }))
