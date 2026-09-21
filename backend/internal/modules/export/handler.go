package export

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"streetlight/internal/httpx"
	"streetlight/internal/modules/auth"
	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/repair"
)

// Handler 处理数据导出请求。
type Handler struct {
	service *Service
	guard   *auth.Guard
}

// NewHandler 构造导出处理器。
func NewHandler(service *Service, guard *auth.Guard) *Handler {
	return &Handler{service: service, guard: guard}
}

// ExportFaults 导出故障 CSV。
func (h *Handler) ExportFaults(c *gin.Context) {
	var query fault.ListQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": err.Error()})
		return
	}

	fileName := fmt.Sprintf("faults-%s.csv", time.Now().Format("20060102-150405"))
	setCSVHeaders(c, fileName)
	// Excel 识别 UTF-8 中文需要 BOM。
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	count, err := h.service.WriteFaultCSV(c.Request.Context(), query, c.Writer)
	if err != nil {
		// 响应已开始写出, 无法再改成 JSON 错误, 只能中断并记录。
		c.Abort()
		return
	}
	// 导出是敏感的数据外发动作, 即使为 GET 也必须留痕。
	h.guard.Audit().Record(c, auth.Entry{
		Actor:       auth.CurrentUser(c),
		Action:      "export.fault",
		Resource:    "export",
		Description: fmt.Sprintf("导出故障数据 %d 条", count),
		Result:      auth.AuditResultAllowed,
	})
}

// ExportRepairs 导出维修 CSV。
func (h *Handler) ExportRepairs(c *gin.Context) {
	var query repair.ListQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": "INVALID_ARGUMENT", "message": err.Error()})
		return
	}

	fileName := fmt.Sprintf("repairs-%s.csv", time.Now().Format("20060102-150405"))
	setCSVHeaders(c, fileName)
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	count, err := h.service.WriteRepairCSV(c.Request.Context(), query, c.Writer)
	if err != nil {
		c.Abort()
		return
	}
	h.guard.Audit().Record(c, auth.Entry{
		Actor:       auth.CurrentUser(c),
		Action:      "export.repair",
		Resource:    "export",
		Description: fmt.Sprintf("导出维修数据 %d 条", count),
		Result:      auth.AuditResultAllowed,
	})
}

func setCSVHeaders(c *gin.Context, fileName string) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("X-Export-Filename", fileName)
}
