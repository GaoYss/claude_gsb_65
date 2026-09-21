package repair

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"
)

// timeLayout 是导出文件中的统一时间格式。
const timeLayout = "2006-01-02 15:04:05"

// ExportCSV 将当前操作者数据范围内的维修记录写为 CSV。
// 数据范围与列表/详情/看板完全一致(维修人员仅本人记录), 由 ListForExport 保证。
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
		"维修单号", "故障单号", "路灯编号", "维修人员", "维修班组", "联系电话",
		"开工时间", "完工时间", "状态", "维修结果", "维修内容", "耗材", "费用(元)", "耗时(分钟)", "备注",
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	formatTime := func(value *time.Time) string {
		if value == nil {
			return ""
		}
		return value.Format(timeLayout)
	}

	for _, item := range items {
		duration := ""
		if item.DurationMinutes != nil {
			duration = strconv.FormatInt(*item.DurationMinutes, 10)
		}
		row := []string{
			item.RepairNo,
			item.FaultNo,
			item.LampCode,
			item.Repairman,
			item.RepairTeam,
			item.ContactPhone,
			item.StartedAt.Format(timeLayout),
			formatTime(item.FinishedAt),
			StatusLabel(item.Status),
			ResultLabel(item.Result),
			item.Content,
			item.Materials,
			strconv.FormatFloat(item.Cost, 'f', 2, 64),
			duration,
			item.Remark,
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("写出维修导出文件失败: %w", err)
	}
	return nil
}
