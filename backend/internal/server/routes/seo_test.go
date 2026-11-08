package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSEORoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterSEORoutes(router, nil)

	t.Run("robots", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
		req.Host = "example.com"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), "Sitemap: http://example.com/sitemap.xml")
		require.Contains(t, w.Body.String(), "Disallow: /admin/")
	})

	t.Run("sitemap uses forwarded https origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
		req.Host = "api.example.com"
		req.Header.Set("X-Forwarded-Proto", "https, http")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "application/xml; charset=utf-8", w.Header().Get("Content-Type"))
		require.Contains(t, w.Body.String(), "https://api.example.com/home")
		require.Contains(t, w.Body.String(), "https://api.example.com/official-prices/chatgpt")
		require.NotContains(t, w.Body.String(), "/model-plaza")
		require.True(t, strings.HasPrefix(w.Body.String(), "<?xml version=\"1.0\""))
	})
}
