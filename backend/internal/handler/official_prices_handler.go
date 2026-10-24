package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// OfficialPricesHandler 的写接口由 admin 路由组保护，公开接口只能读取已发布报价。
type OfficialPricesHandler struct {
	service *service.OfficialPricesService
}

func NewOfficialPricesHandler(s *service.OfficialPricesService) *OfficialPricesHandler {
	return &OfficialPricesHandler{service: s}
}

func (h *OfficialPricesHandler) Public(c *gin.Context) {
	result, err := h.service.Public(c.Request.Context(), c.Query("home") == "true")
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "订阅价格暂不可用")
		return
	}
	// 开关关闭后新的请求立即生效，避免 CDN 缓存继续展示已关闭的数据。
	c.Header("Cache-Control", "no-store")
	response.Success(c, result)
}

func (h *OfficialPricesHandler) Status(c *gin.Context) {
	status, err := h.service.Status(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "读取订阅价格配置失败")
		return
	}
	response.Success(c, status)
}

func (h *OfficialPricesHandler) Save(c *gin.Context) {
	var cfg service.OfficialPricesConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.Error(c, http.StatusBadRequest, "订阅价格配置格式不正确")
		return
	}
	if err := h.service.SaveConfig(c.Request.Context(), cfg); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	h.Status(c)
}

func (h *OfficialPricesHandler) Refresh(c *gin.Context) {
	if err := h.service.Refresh(c.Request.Context()); err != nil {
		response.Error(c, http.StatusBadGateway, err.Error())
		return
	}
	h.Status(c)
}
