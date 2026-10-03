package repository

import (
	"strings"
	"testing"
)

// 列表查询只加载摘要字段，避免请求和响应正文拖慢翻页。
func TestAuditLogSelectColumnsKeepBodiesInDetailOnly(t *testing.T) {
	if strings.Contains(auditLogListSelectColumns, "request_body") || strings.Contains(auditLogListSelectColumns, "response_body") {
		t.Fatal("audit log list query must not select payload bodies")
	}
	if !strings.Contains(auditLogDetailSelectColumns, "request_body") || !strings.Contains(auditLogDetailSelectColumns, "response_body") {
		t.Fatal("audit log detail query must select request and response bodies")
	}
}
