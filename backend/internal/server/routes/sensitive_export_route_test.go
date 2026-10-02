package routes

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// 账号和代理导出包含明文凭证，路由调整时不能绕过 step-up 二次验证。
func TestSensitiveExportsRemainStepUpProtected(t *testing.T) {
	source, err := os.ReadFile("admin.go")
	require.NoError(t, err)
	routes := string(source)
	require.Contains(t, routes, `accounts.GET("/data", gin.HandlerFunc(stepUpAuth), h.Admin.Account.ExportData)`)
	require.Contains(t, routes, `proxies.GET("/data", gin.HandlerFunc(stepUpAuth), h.Admin.Proxy.ExportData)`)
}
