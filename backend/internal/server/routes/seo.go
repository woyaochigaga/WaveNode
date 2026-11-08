package routes

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

var seoOfficialPriceApps = []string{"chatgpt", "claude", "grok"}

// RegisterSEORoutes 注册与前端同源的爬虫元数据接口。
func RegisterSEORoutes(r *gin.Engine, settingService *service.SettingService) {
	r.GET("/robots.txt", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=300")
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte("User-agent: *\nAllow: /home\nAllow: /model-plaza\nAllow: /official-prices/\nDisallow: /dashboard\nDisallow: /admin/\nDisallow: /api/\nDisallow: /payment/\nDisallow: /auth/\nDisallow: /login\nDisallow: /register\nDisallow: /forgot-password\nDisallow: /reset-password\nDisallow: /setup\nDisallow: /key-usage\nSitemap: "+requestOrigin(c)+"/sitemap.xml\n"))
	})

	r.GET("/sitemap.xml", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=900")
		origin := requestOrigin(c)
		paths := []string{"/home"}
		for _, app := range seoOfficialPriceApps {
			paths = append(paths, "/official-prices/"+app)
		}
		if settingService != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
			runtime := settingService.GetModelPlazaRuntime(ctx)
			cancel()
			if runtime.Enabled && !runtime.RequireAuth {
				paths = append(paths, "/model-plaza")
			}
		}

		var builder strings.Builder
		builder.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
		for _, path := range paths {
			builder.WriteString("  <url><loc>")
			builder.WriteString(html.EscapeString(origin + path))
			builder.WriteString("</loc></url>\n")
		}
		builder.WriteString("</urlset>\n")
		c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(builder.String()))
	})
}

func requestOrigin(c *gin.Context) string {
	proto := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0])
	if proto != "http" && proto != "https" {
		proto = "http"
		if c.Request.TLS != nil {
			proto = "https"
		}
	}
	host := strings.TrimSpace(c.Request.Host)
	if host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("%s://%s", proto, host)
}
