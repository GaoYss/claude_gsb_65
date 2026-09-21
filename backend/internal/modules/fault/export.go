package fault

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"
)

// exportTimeLayout 是导出文件中的统一时间格式。
const exportTimeLayout = "2006-01-02 15:04:05"

// ListForExport 按查询条件导出全部(不分页)故障, 与列表使用同一过滤口径。
func (s *Service) ListForExport(ctx context.Context, query ListQuery) ([]Fault, error) {
	filter, err := buildFilter(query)
	if err != nil {
		return nil, err
	}
	return s.repo.ListAll(ctx, filter)
}

// ExportCSV 将当前查询条件下的故障写为 CSV。能否导出由路由上的 export:fault 权限统一拦截。
func (s *Service) ExportCSV(ctx context.Context, writer io.Writer, query ListQuery) error {
	items, err := s.ListForExport(ctx, query)
	if err != nil {
		return err
	}

	// UTF-8 BOM, 保证 Excel 直接打开时中文不乱码。
	if _, err := writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}

	csvWriter := csv.NewWriter(writer)
	header := []string{
		"故障单号", "路灯编号", "所在道路", "故障类型", "紧急程度", "来源",
		"故障描述", "上报人", "上报人电话", "上报时间", "处理状态",
		"维修次数", "关闭时间", "关闭说明",
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	formatTime := func(value *time.Time) string {
		if value == nil {
			return ""
		}
		return value.Format(exportTimeLayout)
	}

	for _, item := range items {
		row := []string{
			item.FaultNo,
			item.LampCode,
			item.RoadName,
			item.FaultType,
			LevelLabel(item.FaultLevel),
			SourceLabel(item.Source),
			item.Description,
			item.Reporter,
			item.ReporterPhone,
			item.ReportedAt.Format(exportTimeLayout),
			StatusLabel(item.Status),
			strconv.Itoa(item.RepairCount),
			formatTime(item.ClosedAt),
			item.CloseRemark,
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("写出故障导出文件失败: %w", err)
	}
	return nil
}
