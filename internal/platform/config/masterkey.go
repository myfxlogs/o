package config

import (
	"bytes"
	"fmt"
	"os"
)

// MasterKeyPath 主权钥文件路径——env PHASE_MASTER_KEY_FILE（白名单引导件，仅指路径
// 不载密），缺省 /run/secrets/phase_master_key（compose 挂载位，600 权限）。
func MasterKeyPath() string {
	if p := os.Getenv("PHASE_MASTER_KEY_FILE"); p != "" {
		return p
	}
	return "/run/secrets/phase_master_key"
}

// ReadMasterKey 读主权钥（主机密钥文件=唯一合法存放——不入库不入 env）。
// 缺件/空件=error：调用面遇密文行 fail-closed。
func ReadMasterKey() ([]byte, error) { return ReadMasterKeyFrom(MasterKeyPath()) }

// ReadMasterKeyFrom 指定路径版（工具件入参；语义同 ReadMasterKey）。
func ReadMasterKeyFrom(p string) ([]byte, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("master key 读败 %s: %w", p, err)
	}
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil, fmt.Errorf("master key 文件 %s 为空", p)
	}
	return b, nil
}
