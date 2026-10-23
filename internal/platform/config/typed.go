package config

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// 标量契约（T-0018 裁决钉死）：app_config 值=JSON 原生类型——数字存数字、
// 布尔存布尔、字符串存字符串。写约定严格禁双编码；读器宽容（历史字符串
// 编码残留两形皆收）。类型不符=error（fail-closed 保留语义）。

// GetFloat 数字键（json.Number 或 numeric string 两形）。
func (c *Client) GetFloat(k string) (float64, error) {
	raw, err := c.Get(k)
	if err != nil {
		return 0, err
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, fmt.Errorf("%s: 非数字形", k)
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: 数字解析败 %w", k, err)
	}
	return v, nil
}

// GetBool 布尔键（jsonb bool 或 "true"/"false" 字符串两形）。
func (c *Client) GetBool(k string) (bool, error) {
	raw, err := c.Get(k)
	if err != nil {
		return false, err
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s == "true", nil
	}
	return false, fmt.Errorf("%s: 非布尔形", k)
}

// GetString 字符串键（只受字符串——数字存字符串=双编码病灶不误诊为合法）。
func (c *Client) GetString(k string) (string, error) {
	raw, err := c.Get(k)
	if err != nil {
		return "", err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("%s: 非字符串形", k)
	}
	return s, nil
}

// GetInt 整数键（json.Number 或 numeric string 两形；非整拒）。
func (c *Client) GetInt(k string) (int64, error) {
	f, err := c.GetFloat(k)
	if err != nil {
		return 0, err
	}
	return int64(f), nil
}
