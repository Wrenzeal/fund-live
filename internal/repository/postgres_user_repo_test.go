package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openPostgresUserRepoTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	adminCfg := database.DefaultConfig()
	adminCfg.DBName = "postgres"
	adminDB, err := gorm.Open(postgres.Open(adminCfg.DSN()), &gorm.Config{})
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}

	tempDBName := fmt.Sprintf("fund_user_repo_test_%d", time.Now().UnixNano())
	if err := adminDB.Exec(`CREATE DATABASE ` + tempDBName).Error; err != nil {
		t.Skipf("create temp database failed: %v", err)
	}

	cfg := database.DefaultConfig()
	cfg.DBName = tempDBName
	cfg.AutoMigrate = false
	db, err := database.InitDB(cfg, database.AllModels()...)
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}

	cleanup := func() {
		_ = database.Close()
		_ = adminDB.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = ? AND pid <> pg_backend_pid()`, tempDBName).Error
		_ = adminDB.Exec(`DROP DATABASE IF EXISTS ` + tempDBName).Error
	}

	return db, cleanup
}

func TestPostgresUserRepositoryDeleteWatchlistGroupDeletesChildren(t *testing.T) {
	db, cleanup := openPostgresUserRepoTestDB(t)
	defer cleanup()

	repo := NewPostgresUserRepository(db)
	ctx := context.Background()
	now := time.Now()
	group := &domain.UserWatchlistGroup{
		ID:          "wlg_test",
		UserID:      "user-1",
		Name:        "测试分组",
		Description: "删除分组回归测试",
		Accent:      "cyan",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := repo.SaveWatchlistGroup(ctx, group); err != nil {
		t.Fatalf("SaveWatchlistGroup() error = %v", err)
	}
	if err := repo.SaveWatchlistFund(ctx, &domain.UserWatchlistFund{
		GroupID:   group.ID,
		FundID:    "005827",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("SaveWatchlistFund() error = %v", err)
	}

	if err := repo.DeleteWatchlistGroup(ctx, group.UserID, group.ID); err != nil {
		t.Fatalf("DeleteWatchlistGroup() error = %v", err)
	}

	deletedGroup, err := repo.GetWatchlistGroupByID(ctx, group.UserID, group.ID)
	if err != nil {
		t.Fatalf("GetWatchlistGroupByID() error = %v", err)
	}
	if deletedGroup != nil {
		t.Fatalf("group still exists after delete: %+v", deletedGroup)
	}

	funds, err := repo.ListWatchlistFunds(ctx, group.UserID, group.ID)
	if err != nil {
		t.Fatalf("ListWatchlistFunds() error = %v", err)
	}
	if len(funds) != 0 {
		t.Fatalf("funds len = %d, want 0", len(funds))
	}
}
