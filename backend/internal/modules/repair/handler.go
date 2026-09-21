package repair

import (
	"strings"

	"github.com/gin-gonic/gin"

	"streetlight/internal/httpx"
	"streetlight/internal/modules/auth"
	"streetlight/internal/response"
)

// Handler 处理维修记录相关的 HTTP 请求。
type Handler struct {
	service *Service
}

// NewHandler 构造维修记录处理器。
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List 查询维修记录列表。
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

// ListByFault 查询指定故障的维修过程记录。
func (h *Handler) ListByFault(c *gin.Context) {
	faultID, err := httpx.ParseID(c, "faultId")
	if err != nil {
		response.Fail(c, err)
		return
	}
	items, err := h.service.ListByFault(c.Request.Context(), faultID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, items)
}

// Get 查询维修记录详情。
func (h *Handler) Get(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// Create 录入维修记录。
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

// Update 修改维修记录。
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

// Finish 完成维修。
func (h *Handler) Finish(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req FinishRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.Finish(c.Request.Context(), id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// Delete 删除维修记录。
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

// Metadata 返回维修字典。
func (h *Handler) Metadata(c *gin.Context) {
	meta, err := h.service.Metadata(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, meta)
}

// Statistics 维修统计。
func (h *Handler) Statistics(c *gin.Context) {
	statistics, err := h.service.Statistics(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, statistics)
}

// Assign 改派维修记录负责人(仅管理岗)。
func (h *Handler) Assign(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req AssignRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, previous, next, err := h.service.Assign(c.Request.Context(), id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Set(auth.ContextAuditNo, entity.RepairNo)
	c.Set(auth.ContextAuditDetail, "改派负责人: "+previous+" → "+next+
		" 班组: "+entity.RepairTeam+" 原因: "+strings.TrimSpace(req.Reason))
	response.OK(c, entity)
}

// Export 按当前操作者范围导出维修记录 CSV。
func (h *Handler) Export(c *gin.Context) {
	var query ListQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		response.Fail(c, err)
		return
	}
	// 导出范围与列表一致, 维修人员只能导出本人记录。
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=repairs.csv")
	if err := h.service.ExportCSV(c.Request.Context(), c.Writer, query); err != nil {
		if !c.Writer.Written() {
			response.Fail(c, err)
		}
		return
	}
}
