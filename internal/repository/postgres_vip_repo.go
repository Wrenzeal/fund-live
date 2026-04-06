package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ domain.VIPRepository = (*PostgresVIPRepository)(nil)

type PostgresVIPRepository struct {
	db *gorm.DB
}

func NewPostgresVIPRepository(db *gorm.DB) *PostgresVIPRepository {
	return &PostgresVIPRepository{db: db}
}

func (r *PostgresVIPRepository) GetMembership(ctx context.Context, userID string) (*domain.VIPMembership, error) {
	var record database.UserMembership
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get vip membership: %w", result.Error)
	}

	return &domain.VIPMembership{
		ID:           record.ID,
		UserID:       record.UserID,
		PlanCode:     record.PlanCode,
		PlanName:     record.PlanName,
		BillingCycle: domain.VIPBillingCycle(record.BillingCycle),
		ActivatedAt:  record.ActivatedAt,
		ExpiresAt:    record.ExpiresAt,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}, nil
}

func (r *PostgresVIPRepository) SaveMembership(ctx context.Context, membership *domain.VIPMembership) error {
	if membership == nil {
		return nil
	}

	record := &database.UserMembership{
		ID:           membership.ID,
		UserID:       membership.UserID,
		PlanCode:     membership.PlanCode,
		PlanName:     membership.PlanName,
		BillingCycle: string(membership.BillingCycle),
		ActivatedAt:  membership.ActivatedAt,
		ExpiresAt:    membership.ExpiresAt,
		CreatedAt:    membership.CreatedAt,
		UpdatedAt:    membership.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"plan_code",
			"plan_name",
			"billing_cycle",
			"activated_at",
			"expires_at",
			"updated_at",
		}),
	}).Create(record).Error; err != nil {
		return fmt.Errorf("failed to save vip membership: %w", err)
	}

	return nil
}

func (r *PostgresVIPRepository) DeleteMembership(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&database.UserMembership{}).Error; err != nil {
		return fmt.Errorf("failed to delete vip membership: %w", err)
	}
	return nil
}

func (r *PostgresVIPRepository) GetDailyUsage(ctx context.Context, userID, usageDate string) (*domain.VIPDailyUsage, error) {
	parsedDate, err := time.Parse("2006-01-02", usageDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vip usage date: %w", err)
	}

	var record database.VIPUsageDaily
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND usage_date = ?", userID, parsedDate).
		First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get vip daily usage: %w", result.Error)
	}

	return &domain.VIPDailyUsage{
		UserID:                record.UserID,
		UsageDate:             record.UsageDate.Format("2006-01-02"),
		SectorAnalysisUsed:    record.SectorAnalysisUsed,
		PortfolioAnalysisUsed: record.PortfolioAnalysisUsed,
		CreatedAt:             record.CreatedAt,
		UpdatedAt:             record.UpdatedAt,
	}, nil
}

func (r *PostgresVIPRepository) SaveDailyUsage(ctx context.Context, usage *domain.VIPDailyUsage) error {
	if usage == nil {
		return nil
	}

	parsedDate, err := time.Parse("2006-01-02", usage.UsageDate)
	if err != nil {
		return fmt.Errorf("failed to parse vip usage date: %w", err)
	}

	record := &database.VIPUsageDaily{
		UserID:                usage.UserID,
		UsageDate:             parsedDate,
		SectorAnalysisUsed:    usage.SectorAnalysisUsed,
		PortfolioAnalysisUsed: usage.PortfolioAnalysisUsed,
		CreatedAt:             usage.CreatedAt,
		UpdatedAt:             usage.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "usage_date"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"sector_analysis_used",
			"portfolio_analysis_used",
			"updated_at",
		}),
	}).Create(record).Error; err != nil {
		return fmt.Errorf("failed to save vip daily usage: %w", err)
	}

	return nil
}

func (r *PostgresVIPRepository) DeleteDailyUsages(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&database.VIPUsageDaily{}).Error; err != nil {
		return fmt.Errorf("failed to delete vip daily usages: %w", err)
	}
	return nil
}

func (r *PostgresVIPRepository) SaveTask(ctx context.Context, task *domain.VIPTaskRecord) error {
	if task == nil {
		return nil
	}

	record := &database.AnalysisTask{
		ID:               task.ID,
		UserID:           task.UserID,
		Type:             string(task.Type),
		TargetType:       string(task.TargetType),
		TargetID:         task.TargetID,
		TargetName:       task.TargetName,
		Status:           string(task.Status),
		ProgressText:     task.ProgressText,
		TemplateReportID: task.TemplateReportID,
		ReportID:         task.ReportID,
		CreatedAt:        task.CreatedAt,
		StartedAt:        task.StartedAt,
		CompletedAt:      task.CompletedAt,
		FailedAt:         task.FailedAt,
		UpdatedAt:        task.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status",
			"progress_text",
			"report_id",
			"started_at",
			"completed_at",
			"failed_at",
			"updated_at",
		}),
	}).Create(record).Error; err != nil {
		return fmt.Errorf("failed to save analysis task: %w", err)
	}

	return nil
}

func (r *PostgresVIPRepository) ListTasks(ctx context.Context, userID string) ([]domain.VIPTaskRecord, error) {
	var records []database.AnalysisTask
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to list analysis tasks: %w", err)
	}

	result := make([]domain.VIPTaskRecord, 0, len(records))
	for _, record := range records {
		result = append(result, domain.VIPTaskRecord{
			ID:               record.ID,
			UserID:           record.UserID,
			Type:             domain.VIPTaskType(record.Type),
			TargetType:       domain.VIPTargetType(record.TargetType),
			TargetID:         record.TargetID,
			TargetName:       record.TargetName,
			Status:           domain.VIPTaskStatus(record.Status),
			ProgressText:     record.ProgressText,
			TemplateReportID: record.TemplateReportID,
			ReportID:         record.ReportID,
			CreatedAt:        record.CreatedAt,
			StartedAt:        record.StartedAt,
			CompletedAt:      record.CompletedAt,
			FailedAt:         record.FailedAt,
			UpdatedAt:        record.UpdatedAt,
		})
	}

	return result, nil
}

func (r *PostgresVIPRepository) DeleteTasks(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&database.AnalysisTask{}).Error; err != nil {
		return fmt.Errorf("failed to delete analysis tasks: %w", err)
	}
	return nil
}

func (r *PostgresVIPRepository) SaveReport(ctx context.Context, report *domain.VIPStoredReport) error {
	if report == nil {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record := &database.AnalysisReport{
			ID:          report.ID,
			UserID:      report.UserID,
			TaskID:      report.TaskID,
			PayloadJSON: report.PayloadJSON,
			GeneratedAt: report.GeneratedAt,
			CreatedAt:   report.CreatedAt,
			UpdatedAt:   report.UpdatedAt,
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"task_id",
				"payload_json",
				"generated_at",
				"updated_at",
			}),
		}).Create(record).Error; err != nil {
			return fmt.Errorf("failed to save analysis report: %w", err)
		}

		if err := tx.Where("report_id = ?", report.ID).Delete(&database.AnalysisReportSource{}).Error; err != nil {
			return fmt.Errorf("failed to delete analysis report sources: %w", err)
		}

		if len(report.Sources) == 0 {
			return nil
		}

		dbSources := make([]database.AnalysisReportSource, 0, len(report.Sources))
		for _, source := range report.Sources {
			publishedAt, err := time.Parse(time.RFC3339, source.PublishedAt)
			if err != nil {
				return fmt.Errorf("failed to parse analysis report source time: %w", err)
			}

			dbSources = append(dbSources, database.AnalysisReportSource{
				ID:          source.ID,
				ReportID:    report.ID,
				Title:       source.Title,
				Type:        string(source.Type),
				Publisher:   source.Publisher,
				PublishedAt: publishedAt,
				URL:         source.URL,
				Snippet:     source.Snippet,
			})
		}

		if err := tx.Create(&dbSources).Error; err != nil {
			return fmt.Errorf("failed to save analysis report sources: %w", err)
		}
		return nil
	})
}

func (r *PostgresVIPRepository) GetReportByID(ctx context.Context, reportID string) (*domain.VIPStoredReport, error) {
	var record database.AnalysisReport
	result := r.db.WithContext(ctx).Where("id = ?", reportID).First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get analysis report: %w", result.Error)
	}

	var dbSources []database.AnalysisReportSource
	if err := r.db.WithContext(ctx).
		Where("report_id = ?", reportID).
		Order("created_at ASC").
		Find(&dbSources).Error; err != nil {
		return nil, fmt.Errorf("failed to list analysis report sources: %w", err)
	}

	sources := make([]domain.VIPReportSource, 0, len(dbSources))
	for _, source := range dbSources {
		sources = append(sources, domain.VIPReportSource{
			ID:          source.ID,
			Title:       source.Title,
			Type:        domain.VIPSourceType(source.Type),
			Publisher:   source.Publisher,
			PublishedAt: source.PublishedAt.UTC().Format(time.RFC3339),
			URL:         source.URL,
			Snippet:     source.Snippet,
		})
	}

	return &domain.VIPStoredReport{
		ID:          record.ID,
		UserID:      record.UserID,
		TaskID:      record.TaskID,
		PayloadJSON: record.PayloadJSON,
		GeneratedAt: record.GeneratedAt,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
		Sources:     sources,
	}, nil
}

func (r *PostgresVIPRepository) DeleteReports(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reportIDs []string
		if err := tx.Model(&database.AnalysisReport{}).
			Where("user_id = ?", userID).
			Pluck("id", &reportIDs).Error; err != nil {
			return fmt.Errorf("failed to list report ids for deletion: %w", err)
		}

		if len(reportIDs) > 0 {
			if err := tx.Where("report_id IN ?", reportIDs).Delete(&database.AnalysisReportSource{}).Error; err != nil {
				return fmt.Errorf("failed to delete analysis report sources: %w", err)
			}
		}

		if err := tx.Where("user_id = ?", userID).Delete(&database.AnalysisReport{}).Error; err != nil {
			return fmt.Errorf("failed to delete analysis reports: %w", err)
		}
		return nil
	})
}

func (r *PostgresVIPRepository) SaveOrder(ctx context.Context, order *domain.VIPOrder) error {
	if order == nil {
		return nil
	}

	record := &database.VIPOrder{
		ID:                  order.ID,
		UserID:              order.UserID,
		OrderNo:             order.OrderNo,
		PlanCode:            order.PlanCode,
		PlanName:            order.PlanName,
		BillingCycle:        string(order.BillingCycle),
		AmountFen:           order.AmountFen,
		Currency:            order.Currency,
		Status:              string(order.Status),
		PaymentChannel:      string(order.PaymentChannel),
		PaymentScene:        string(order.PaymentScene),
		Description:         order.Description,
		CodeURL:             order.CodeURL,
		WechatTransactionID: order.WechatTransactionID,
		WechatPrepayID:      order.WechatPrepayID,
		ErrorCode:           order.ErrorCode,
		ErrorMessage:        order.ErrorMessage,
		NotifyID:            order.NotifyID,
		NotifyPayload:       order.NotifyPayload,
		ExpiresAt:           order.ExpiresAt,
		PaidAt:              order.PaidAt,
		CreatedAt:           order.CreatedAt,
		UpdatedAt:           order.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status",
			"code_url",
			"wechat_transaction_id",
			"wechat_prepay_id",
			"error_code",
			"error_message",
			"notify_id",
			"notify_payload",
			"expires_at",
			"paid_at",
			"updated_at",
		}),
	}).Create(record).Error; err != nil {
		return fmt.Errorf("failed to save vip order: %w", err)
	}

	return nil
}

func (r *PostgresVIPRepository) GetOrderByID(ctx context.Context, orderID string) (*domain.VIPOrder, error) {
	var record database.VIPOrder
	result := r.db.WithContext(ctx).Where("id = ?", orderID).First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get vip order by id: %w", result.Error)
	}
	return r.toDomainVIPOrder(&record), nil
}

func (r *PostgresVIPRepository) GetOrderByOrderNo(ctx context.Context, orderNo string) (*domain.VIPOrder, error) {
	var record database.VIPOrder
	result := r.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get vip order by order no: %w", result.Error)
	}
	return r.toDomainVIPOrder(&record), nil
}

func (r *PostgresVIPRepository) DeleteOrders(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&database.VIPOrder{}).Error; err != nil {
		return fmt.Errorf("failed to delete vip orders: %w", err)
	}
	return nil
}

func (r *PostgresVIPRepository) toDomainVIPOrder(record *database.VIPOrder) *domain.VIPOrder {
	if record == nil {
		return nil
	}

	return &domain.VIPOrder{
		ID:                  record.ID,
		UserID:              record.UserID,
		OrderNo:             record.OrderNo,
		PlanCode:            record.PlanCode,
		PlanName:            record.PlanName,
		BillingCycle:        domain.VIPBillingCycle(record.BillingCycle),
		AmountFen:           record.AmountFen,
		Currency:            record.Currency,
		Status:              domain.VIPOrderStatus(record.Status),
		PaymentChannel:      domain.VIPPaymentChannel(record.PaymentChannel),
		PaymentScene:        domain.VIPPaymentScene(record.PaymentScene),
		Description:         record.Description,
		CodeURL:             record.CodeURL,
		WechatTransactionID: record.WechatTransactionID,
		WechatPrepayID:      record.WechatPrepayID,
		ErrorCode:           record.ErrorCode,
		ErrorMessage:        record.ErrorMessage,
		NotifyID:            record.NotifyID,
		NotifyPayload:       record.NotifyPayload,
		ExpiresAt:           record.ExpiresAt,
		PaidAt:              record.PaidAt,
		CreatedAt:           record.CreatedAt,
		UpdatedAt:           record.UpdatedAt,
	}
}
