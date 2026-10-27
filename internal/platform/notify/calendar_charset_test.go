// calendar_charset_test.go —— T-0183 桩锚：windows-1252 声明样本解出条目
// （CharsetReader 接入实证——非 UTF-8 声明不再恒败）。
package notify

import (
	"bytes"
	"encoding/xml"
	"testing"

	"golang.org/x/net/html/charset"
)

func TestCharsetReaderWindows1252(t *testing.T) {
	body := []byte(`<?xml version="1.0" encoding="windows-1252"?>
<weekly><event><title>NFP</title><impact>High</impact></event></weekly>`)
	var w ffWeekly
	dec := xml.NewDecoder(bytes.NewReader(body))
	dec.CharsetReader = charset.NewReaderLabel
	if err := dec.Decode(&w); err != nil {
		t.Fatalf("windows-1252 声明应解出: %v", err)
	}
	if len(w.Events) != 1 || w.Events[0].Impact != "High" {
		t.Fatalf("条目缺: %+v", w)
	}
}
