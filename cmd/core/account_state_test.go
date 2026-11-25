package main

// account_state_test.go —— O-39440 账户生命周期面桩测（SetAccountState
// 迁移机全谱/零引用真删 vs 有史墓碑/结构子户与 frozen 域拒；HTTP 径
// 走 earnOriginGate 同源闸）。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const acctProc = "/o.dashboard.v1.CoreService/SetAccountState"

func acctPost(t *testing.T, mux http.Handler, acct, state, origin string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, acctProc,
		strings.NewReader(`{"account_id":"`+acct+`","state":"`+state+`"}`))
	req.Host = "ui.local"
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func seedAcctRow(t *testing.T, acct, state, subType string) {
	t.Helper()
	// brokerFaceMux 返回值在调用侧——本函数由各测内联 pool.Exec 走,
	// 见 TestSetAccountStateTransition。
	_ = acct
	_ = state
	_ = subType
}

// TestSetAccountStateTransition 迁移机全谱：active→disabled→active
// 往返/幂等/disabled→closed 零引用=物理删('deleted')；有史→'closed'
// 墓碑；closed 终态拒；frozen/结构子户/未知户/坏态全拒。
func TestSetAccountStateTransition(t *testing.T) {
	mux, _, pool, ctx := brokerFaceMux(t, nil, nil)
	seed := func(id, state, sub string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO fin_account(account_id,venue,account_no,alias,ccy,sub_type,state)
			VALUES($1,'tstacct','1','t','USDT',$2,$3)`, id, sub, state); err != nil {
			t.Fatalf("fin_account 造行 %s: %v", id, err)
		}
	}
	stateOf := func(id string) string {
		t.Helper()
		var s string
		if err := pool.QueryRow(ctx, `SELECT state FROM fin_account WHERE account_id=$1`, id).Scan(&s); err != nil {
			return "<gone>"
		}
		return s
	}
	seed("tstacct:1", "active", "main")
	defer pool.Exec(ctx, `DELETE FROM fin_account WHERE account_id LIKE 'tstacct:%'`)
	defer pool.Exec(ctx, `DELETE FROM fin_cashflow WHERE venue='tstacct'`)

	// 形闸/未知户/坏态
	if rec := acctPost(t, mux, "BAD!!", "disabled", "http://ui.local"); rec.Code == 200 {
		t.Fatal("坏 account_id 应拒")
	}
	if rec := acctPost(t, mux, "tstacct:ghost", "disabled", "http://ui.local"); rec.Code == 200 {
		t.Fatal("未知户应拒")
	}
	if rec := acctPost(t, mux, "tstacct:1", "bogus", "http://ui.local"); rec.Code == 200 {
		t.Fatal("坏 state 应拒")
	}
	// active→closed 跳档拒（先停用再删律）
	if rec := acctPost(t, mux, "tstacct:1", "closed", "http://ui.local"); rec.Code == 200 {
		t.Fatal("active→closed 跳档应拒")
	}
	// active→disabled→幂等→active 复活
	if rec := acctPost(t, mux, "tstacct:1", "disabled", "http://ui.local"); rec.Code != 200 ||
		!strings.Contains(rec.Body.String(), `"state":"disabled"`) {
		t.Fatalf("disable: %d %s", rec.Code, rec.Body)
	}
	if stateOf("tstacct:1") != "disabled" {
		t.Fatal("disabled 未落库")
	}
	if rec := acctPost(t, mux, "tstacct:1", "disabled", "http://ui.local"); rec.Code != 200 {
		t.Fatalf("幂等 disable: %d", rec.Code)
	}
	if rec := acctPost(t, mux, "tstacct:1", "active", "http://ui.local"); rec.Code != 200 ||
		!strings.Contains(rec.Body.String(), `"state":"active"`) {
		t.Fatalf("复活: %d %s", rec.Code, rec.Body)
	}
	// 再停→closed 零引用=物理删（'deleted' 回执+行灭）
	acctPost(t, mux, "tstacct:1", "disabled", "http://ui.local")
	rec := acctPost(t, mux, "tstacct:1", "closed", "http://ui.local")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"state":"deleted"`) {
		t.Fatalf("零引用 closed 应物理删: %d %s", rec.Code, rec.Body)
	}
	if stateOf("tstacct:1") != "<gone>" {
		t.Fatal("零引用户应物理 DELETE")
	}
	// 有史→closed 墓碑（fin_cashflow 引用在=不删只封态）
	seed("tstacct:2", "disabled", "main")
	if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,account_id,occurred_at)
		VALUES('fee','tstacct','USDT',1,1,'t','tstacct:2',now())`); err != nil {
		t.Fatal("cashflow 引用桩:", err)
	}
	rec = acctPost(t, mux, "tstacct:2", "closed", "http://ui.local")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"state":"closed"`) {
		t.Fatalf("有史 closed 应墓碑: %d %s", rec.Code, rec.Body)
	}
	if stateOf("tstacct:2") != "closed" {
		t.Fatal("有史户应落 closed 墓碑")
	}
	// closed 终态拒（任何迁移）
	if rec := acctPost(t, mux, "tstacct:2", "active", "http://ui.local"); rec.Code == 200 {
		t.Fatal("closed→active 终态应拒")
	}
	// frozen legacy 桶/结构子户非生命周期域
	seed("tstacct:legacy", "frozen", "main")
	if rec := acctPost(t, mux, "tstacct:legacy", "active", "http://ui.local"); rec.Code == 200 {
		t.Fatal("frozen 桶应拒")
	}
	// okx:trading=迁移种子结构子户（sub_type=trading）——直测拒不动行
	if rec := acctPost(t, mux, "okx:trading", "disabled", "http://ui.local"); rec.Code == 200 {
		t.Fatal("结构子户应拒")
	}
}
