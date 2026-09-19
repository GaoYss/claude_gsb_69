package status

import (
	"github.com/gin-gonic/gin"

	"streetlight/internal/httpx"
	"streetlight/internal/response"
)

// Handler 处理维修状态查询相关的 HTTP 请求。
type Handler struct {
	service *Service
}

// NewHandler 构造维修状态查询处理器。
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Overview 维修状态总览看板。
func (h *Handler) Overview(c *gin.Context) {
	result, err := h.service.Overview(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

// Lamps 路灯维修状态列表。
func (h *Handler) Lamps(c *gin.Context) {
	var query LampQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		response.Fail(c, err)
		return
	}
	items, total, page, err := h.service.Lamps(c.Request.Context(), query)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.NewPageData(items, total, page.Page, page.PageSize))
}

// Track 维修状态全链路追踪。
func (h *Handler) Track(c *gin.Context) {
	var query TrackQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		response.Fail(c, err)
		return
	}
	result, err := h.service.Track(c.Request.Context(), query)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}
