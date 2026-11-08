// classify.go —— MT5 venue 拒码分类面（自 mtexec.go 拆件——棘轮 D-588）：
// APIError 终局位+歧义/瞬态码集+inst 级终局码集——untradeable 联动分类锚
// 与 Resolve 仲裁分流同源于此。
package mtexec

import (
	"errors"
	"fmt"

	mt5pb "o/internal/gen/mtapi/mt5"
)

// APIError venue 业务拒（mtapi Error 载荷——Code=mt5pb 枚举串、Msg=venue
// 拒文；TerminalInst=inst 级终局位——T-0145 untradeable 联动分类锚）。
type APIError struct {
	Code         string
	Msg          string
	TerminalInst bool
}

func (e *APIError) Error() string { return fmt.Sprintf("mt5 %s: %s", e.Code, e.Msg) }

// TerminalInstErr mt5 腿 inst 级终局判定（T-0145——SYMBOL_NOT_FOUND/
// INVALID_SYMBOL/TRADE_DISABLED 三码钦定入册锚；瞬态族（MARKET_CLOSED/
// REQUOTE/REQUEST_ERROR 等）/参数族（INVALID_VOLUME/STOPS/PRICE）/资金族
// （NO_MONEY）皆禁入册——市闭=会话复开非永拒，参数=我方病灶非 venue 事实）。
func TerminalInstErr(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.TerminalInst
}

// ambiguousCodes 歧义/瞬态码集——OrderSend 回报此族=单或已建未定态→
// State=unknown 走 Resolve 双面查定真态（真户实证族：REQUEST_ERROR=
// MarketOpenWaiterAsync 网关确认超时栈——退避重试即过，实未成交；但
// 超时窗内可能已成交=歧态势，Resolve 仲裁唯一正径）。价漂/重报族
// （REQUOTE*/PRICE_CHANGED）同归歧义——拒因是价非单，旧码躺 absent
// 吞歧义=裸腿险。
var ambiguousCodes = map[mt5pb.ErrorCode]bool{
	mt5pb.ErrorCode_REQUEST_ERROR: true, mt5pb.ErrorCode_REQUEST_TIMEOUT: true,
	mt5pb.ErrorCode_REQUEST_ACCEPTED: true, mt5pb.ErrorCode_REQUEST_ON_WAY: true,
	mt5pb.ErrorCode_REQUEST_RETURNED_IN_QUEUE:           true,
	mt5pb.ErrorCode_REQUEST_REJECTED_TO_ABSENCE_CONNECT: true,
	mt5pb.ErrorCode_REQUOTE:                             true, mt5pb.ErrorCode_REQUOTE_EX: true,
	mt5pb.ErrorCode_PRICE_CHANGED: true, mt5pb.ErrorCode_NO_PRICES: true,
	mt5pb.ErrorCode_TIMEOUT: true, mt5pb.ErrorCode_SERVER_BUSY: true,
	mt5pb.ErrorCode_ANOTHER_PROCESS_IN_PROGRESS: true,
	mt5pb.ErrorCode_NO_CONNECTION:               true, mt5pb.ErrorCode_CONNECTION_NOT_PERMITTED: true,
}

// terminalInstCodes inst 级终局码集（untradeable 入册锚——三码钦定：
// 符号不存在/非法=目录终局；TRADE_DISABLED=broker 关停非瞬态市闭）。
var terminalInstCodes = map[mt5pb.ErrorCode]bool{
	mt5pb.ErrorCode_SYMBOL_NOT_FOUND: true, mt5pb.ErrorCode_INVALID_SYMBOL: true,
	mt5pb.ErrorCode_TRADE_DISABLED: true,
}

// apiErr mtapi Error 载荷→APIError（终局位/歧义判定同源——Code.String()
// 枚举串为唯一语义载体，禁字符串匹配拒文）。
func apiErr(ge *mt5pb.Error) *APIError {
	code := ge.GetCode()
	return &APIError{Code: code.String(), Msg: ge.GetMessage(),
		TerminalInst: terminalInstCodes[code]}
}

// PbAPIError mt5pb.Error 载荷→*APIError 导出口（T-0145 返修①——活读面
// venue payload 拒（SymbolParams.GetError 等）err 化走终局分类同锚，
// 取用点免重复终局码表）。
func PbAPIError(ge *mt5pb.Error) *APIError { return apiErr(ge) }
