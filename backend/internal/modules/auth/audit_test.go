package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"streetlight/internal/modules/auth"
)

func newAuditDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&auth.User{}, &auth.AuditLog{}))
	require.NoError(t, auth.InstallAuditGuard(db))
	return db
}

// TestAuditLogAppendOnly 验证审计记录只能追加, 无法 UPDATE / DELETE,
// 且角色与权限快照被冗余保存, 后续改权限不影响历史。
func TestAuditLogAppendOnly(t *testing.T) {
	db := newAuditDB(t)
	repo := auth.NewAuditRepository(db)
	ctx := context.Background()

	entry := &auth.AuditLog{
		OccurredAt:         time.Now(),
		CreatedAt:          time.Now(),
		ActorID:            7,
		ActorUsername:      "repairman",
		ActorRole:          auth.RoleRepair,
		RoleSnapshot:       auth.RoleRepair,
		PermSnapshot:       "repair:view,repair:advance",
		Action:             "repair.finish",
		Resource:           "repair",
		ResourceID:         "WX202609210001",
		Result:             auth.AuditResultDenied,
		RequiredPermission: auth.PermRepairManage,
		Reason:             "该记录由其他维修人员负责",
	}
	require.NoError(t, repo.Append(ctx, entry))

	// 触发器必须拒绝改写与删除。
	err := db.Model(&auth.AuditLog{}).Where("id = ?", entry.ID).
		Update("result", auth.AuditResultAllowed).Error
	require.Error(t, err, "审计表 UPDATE 必须被触发器拒绝")

	err = db.Delete(&auth.AuditLog{}, entry.ID).Error
	require.Error(t, err, "审计表 DELETE 必须被触发器拒绝")

	// 原始记录原封不动。
	var saved auth.AuditLog
	require.NoError(t, db.First(&saved, entry.ID).Error)
	require.Equal(t, auth.AuditResultDenied, saved.Result)
	require.Equal(t, auth.PermRepairManage, saved.RequiredPermission)
	require.Equal(t, auth.RoleRepair, saved.RoleSnapshot)
	require.Contains(t, saved.PermSnapshot, auth.PermRepairAdvance)
}

// TestAuditLogQuery 验证审计可按结果等条件只读查询。
func TestAuditLogQuery(t *testing.T) {
	db := newAuditDB(t)
	repo := auth.NewAuditRepository(db)
	ctx := context.Background()

	for _, result := range []string{auth.AuditResultAllowed, auth.AuditResultDenied, auth.AuditResultDenied} {
		require.NoError(t, repo.Append(ctx, &auth.AuditLog{
			OccurredAt: time.Now(), CreatedAt: time.Now(),
			RoleSnapshot: "-", PermSnapshot: "-",
			Action: "fault.update", Resource: "fault", Result: result,
		}))
	}

	denied, total, err := repo.List(ctx, auth.AuditFilter{Result: auth.AuditResultDenied}, 0, 20)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, denied, 2)
}
