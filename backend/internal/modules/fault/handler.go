package fault

import (
	"github.com/gin-gonic/gin"

	"streetlight/internal/httpx"
	"streetlight/internal/modules/auth"
	"streetlight/internal/response"
)

// Handler 处理故障登记相关的 HTTP 请求。
type Handler struct {
	service *Service
}

// NewHandler 构造故障登记处理器。
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List 查询故障列表。
func (h *Handler) List(c *gin.Context) {
	var query ListQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		response.Fail(c, err)
		return
	}
	items, total, page, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.NewPageData(items, total, page.Page, page.PageSize))
}

// Get 查询故障详情。
func (h *Handler) Get(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// Create 登记故障。
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, entity)
}

// Update 修改故障登记信息。
func (h *Handler) Update(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req UpdateRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// Close 关闭故障。
func (h *Handler) Close(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req CloseRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.Close(c.Request.Context(), id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// Delete 删除故障。
func (h *Handler) Delete(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}

// Metadata 返回故障字典。
func (h *Handler) Metadata(c *gin.Context) {
	response.OK(c, h.service.Metadata())
}

// Transition 例外流转故障状态(仅管理岗, 需 fault:bypass 授权)。
func (h *Handler) Transition(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req TransitionRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, previous, target, err := h.service.Transition(c.Request.Context(), id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Set(auth.ContextAuditNo, entity.FaultNo)
	c.Set(auth.ContextAuditDetail,
		"例外流转: "+StatusLabel(previous)+" → "+StatusLabel(target)+" 原因: "+req.Remark)
	response.OK(c, entity)
}

// Export 导出故障 CSV(需 export:fault 授权)。
func (h *Handler) Export(c *gin.Context) {
	var query ListQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		response.Fail(c, err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=faults.csv")
	if err := h.service.ExportCSV(c.Request.Context(), c.Writer, query); err != nil {
		if !c.Writer.Written() {
			response.Fail(c, err)
		}
		return
	}
}
