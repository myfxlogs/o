// audit_1112_collect_test.go —— 审计/采集桥 RED 锚（audit/1112-collect）：
// 断言正确语义——修复落库后自然转绿，门转正为回归锚。
// 显跑：AUDIT_RED=1 go test ./internal/projector
//
// 订阅集与镜像集双端同集不变量——collector ActiveSetUnion 持 fin_proposal
// 提案臂（fin_proposed/fin_armed），projector refreshSets 无对偶臂=提案键
// tick 被谓词滤在桥外：armed 提案等入场判据读 SG md_ticks（LatestPx 鲜度
// 闸 fail-closed），独键提案（无策略/无 position/subreq 罩）数据到岸不过桥
// =armed 窗口静默饿毙（订得到但过不了桥——bridge-predicate 判例同族）。
package projector

import (
	"context"
	"os"
	"testing"
)

func auditRedGate(t *testing.T) {
	t.Helper()
	if os.Getenv("AUDIT_RED") != "1" {
		t.Skip("audit RED 锚——修复期显跑: AUDIT_RED=1 go test -run " + t.Name() + " ./internal/projector；修好后去门转正为回归锚")
	}
}

// TestArmedProposalMirrorArm SG 独键 armed 提案（仅 fin_proposal 行罩）的
// inst tick 应过谓词投影——现 refreshSets 缺提案臂=本断言 RED。
func TestArmedProposalMirrorArm(t *testing.T) {
	auditRedGate(t)
	ctx := context.Background()
	r, md, sg := runner(t)
	// SG 侧独键提案：eval+proposal（fin_armed），无 fin_strategy/fin_position/
	// fin_sub_request 行——离开提案臂即入不了活跃集。
	if _, err := sg.Exec(ctx, `INSERT INTO fin_opp_eval(id,ts,inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES (9001,now(),'NEWIN-USDT-SWAP','funding-hedge','short_perp_long_spot',0.08,'proposed','[]','{}')`); err != nil {
		t.Fatal(err)
	}
	if _, err := sg.Exec(ctx, `INSERT INTO fin_proposal(id,eval_id,inst_id,direction,net_apr,created_at,state)
		VALUES (9101,9001,'NEWIN-USDT-SWAP','short_perp_long_spot',0.08,now(),'fin_armed')`); err != nil {
		t.Fatal(err)
	}
	// MD 正本侧：collector 依提案臂在订该键（ActiveSetUnion 持 fin_proposal
	// 臂）——tick 行实存 md_ticks，只欠桥。
	if _, err := md.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last,recv_ts)
		VALUES (now(),'okx','NEWIN-USDT-SWAP',123.4,now())`); err != nil {
		t.Fatal(err)
	}
	if err := r.Reconcile(ctx, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n := count(t, sg, "SELECT count(*) FROM md_ticks WHERE inst_id='NEWIN-USDT-SWAP'"); n != 1 {
		t.Fatalf("armed 提案独键 tick 应过桥投影（订阅端提案臂在订）: %d——桥端谓词缺 fin_proposal 臂=armed 窗口鲜度闸无源可喂", n)
	}
}
