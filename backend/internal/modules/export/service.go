package export

import (
	"context"
	"encoding/csv"
	"io"
	"strconv"
	"time"

	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/repair"
	"streetlight/pkg/pagination"
)

// maxExportRows 单次导出上限, 防止一次性拉取过多数据。
const maxExportRows = 10000

const exportTimeLayout = "2006-01-02 15:04:05"

const pageSize = 200

// Service 负责把业务模块的列表查询结果序列化为 CSV。
// 数据全部来自 fault/repair 的列表服务, 因此导出的数据范围与页面、接口完全一致。
type Service struct {
	faults  FaultLister
	repairs RepairLister
}

// NewService 构造导出服务。
func NewService(faults FaultLister, repairs RepairLister) *Service {
	return &Service{faults: faults, repairs: repairs}
}

// WriteFaultCSV 导出故障数据, 返回写入的数据行数(不含表头)。
func (s *Service) WriteFaultCSV(ctx context.Context, query fault.ListQuery, writer io.Writer) (int, error) {
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write([]string{
		"故障单号", "路灯编号", "所在道路", "故障类型", "紧急程度", "来源", "处理状态",
		"负责人", "上报人", "联系电话", "上报时间", "维修次数", "关闭时间", "关闭说明",
	}); err != nil {
		return 0, err
	}

	written := 0
	for page := 1; written < maxExportRows; page++ {
		query.Params = pagination.Params{Page: page, PageSize: pageSize, SortBy: "reported_at", Order: "desc"}
		items, total, _, err := s.faults.List(ctx, query)
		if err != nil {
			return written, err
		}
		for _, item := range items {
			if err := csvWriter.Write([]string{
				item.FaultNo, item.LampCode, item.RoadName, item.FaultType,
				fault.LevelLabel(item.FaultLevel), fault.SourceLabel(item.Source),
				fault.StatusLabel(item.Status), item.AssigneeName,
				item.Reporter, item.ReporterPhone,
				formatTime(item.ReportedAt), strconv.Itoa(item.RepairCount),
				formatTimePtr(item.ClosedAt), item.CloseRemark,
			}); err != nil {
				return written, err
			}
			written++
			if written >= maxExportRows {
				break
			}
		}
		if int64(written) >= total || len(items) < pageSize {
			break
		}
	}

	csvWriter.Flush()
	return written, csvWriter.Error()
}

// WriteRepairCSV 导出维修数据, 返回写入的数据行数(不含表头)。
func (s *Service) WriteRepairCSV(ctx context.Context, query repair.ListQuery, writer io.Writer) (int, error) {
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write([]string{
		"维修单号", "故障单号", "路灯编号", "负责人署名", "维修班组", "联系电话",
		"开工时间", "完工时间", "状态", "维修结果", "维修内容", "耗材", "费用(元)",
	}); err != nil {
		return 0, err
	}

	written := 0
	for page := 1; written < maxExportRows; page++ {
		query.Params = pagination.Params{Page: page, PageSize: pageSize, SortBy: "started_at", Order: "desc"}
		items, total, _, err := s.repairs.List(ctx, query)
		if err != nil {
			return written, err
		}
		for _, item := range items {
			if err := csvWriter.Write([]string{
				item.RepairNo, item.FaultNo, item.LampCode,
				item.Repairman, item.RepairTeam, item.ContactPhone,
				formatTime(item.StartedAt), formatTimePtr(item.FinishedAt),
				repair.StatusLabel(item.Status), repair.ResultLabel(item.Result),
				item.Content, item.Materials,
				strconv.FormatFloat(item.Cost, 'f', 2, 64),
			}); err != nil {
				return written, err
			}
			written++
			if written >= maxExportRows {
				break
			}
		}
		if int64(written) >= total || len(items) < pageSize {
			break
		}
	}

	csvWriter.Flush()
	return written, csvWriter.Error()
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(exportTimeLayout)
}

func formatTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatTime(*value)
}
