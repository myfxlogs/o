// noderole.go —— 节点角色闸（T-0104/D-0013）：O_NODE_ROLE=readonly=只读镜像
// ——okxexec 全件 NewClientReadOnly（venue 写径 do() 硬闸结构不可达）+mutating
// 岗不接线（earnW 归口/execution 执事/开仓缝/平仓缝全撤——盲库不动真钱）。
package main

import (
	"log"
	"os"

	"o/internal/trading/okxexec"
)

func nodeReadonly() bool {
	ro := os.Getenv("O_NODE_ROLE") == "readonly"
	if ro {
		log.Printf("[boot] 节点角色=只读镜像（D-0013）——venue 写径结构不可达+mutating 岗全撤")
	}
	return ro
}

func cliFactory(ro bool) func(okxexec.Credentials) *okxexec.Client {
	return func(creds okxexec.Credentials) *okxexec.Client {
		if ro {
			return okxexec.NewClientReadOnly(creds, false)
		}
		return okxexec.NewClient(creds, false)
	}
}
