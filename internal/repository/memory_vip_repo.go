package repository

import (
	"context"
	"strings"
	"sync"

	"github.com/RomaticDOG/fund/internal/domain"
)

var _ domain.VIPRepository = (*MemoryVIPRepository)(nil)

type MemoryVIPRepository struct {
	mu              sync.RWMutex
	memberships     map[string]domain.VIPMembership
	dailyUsage      map[string]map[string]domain.VIPDailyUsage
	tasks           map[string]map[string]domain.VIPTaskRecord
	reportsByID     map[string]domain.VIPStoredReport
	reportIDsByUser map[string]map[string]struct{}
	ordersByID      map[string]domain.VIPOrder
	orderIDByNo     map[string]string
	orderIDsByUser  map[string]map[string]struct{}
}

func NewMemoryVIPRepository() *MemoryVIPRepository {
	return &MemoryVIPRepository{
		memberships:     make(map[string]domain.VIPMembership),
		dailyUsage:      make(map[string]map[string]domain.VIPDailyUsage),
		tasks:           make(map[string]map[string]domain.VIPTaskRecord),
		reportsByID:     make(map[string]domain.VIPStoredReport),
		reportIDsByUser: make(map[string]map[string]struct{}),
		ordersByID:      make(map[string]domain.VIPOrder),
		orderIDByNo:     make(map[string]string),
		orderIDsByUser:  make(map[string]map[string]struct{}),
	}
}

func (r *MemoryVIPRepository) GetMembership(ctx context.Context, userID string) (*domain.VIPMembership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.memberships[userID]
	if !ok {
		return nil, nil
	}
	copyRecord := record
	return &copyRecord, nil
}

func (r *MemoryVIPRepository) SaveMembership(ctx context.Context, membership *domain.VIPMembership) error {
	if membership == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	copyRecord := *membership
	r.memberships[membership.UserID] = copyRecord
	return nil
}

func (r *MemoryVIPRepository) DeleteMembership(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.memberships, userID)
	return nil
}

func (r *MemoryVIPRepository) GetDailyUsage(ctx context.Context, userID, usageDate string) (*domain.VIPDailyUsage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	byDate := r.dailyUsage[userID]
	if byDate == nil {
		return nil, nil
	}
	record, ok := byDate[usageDate]
	if !ok {
		return nil, nil
	}
	copyRecord := record
	return &copyRecord, nil
}

func (r *MemoryVIPRepository) SaveDailyUsage(ctx context.Context, usage *domain.VIPDailyUsage) error {
	if usage == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.dailyUsage[usage.UserID]; !ok {
		r.dailyUsage[usage.UserID] = make(map[string]domain.VIPDailyUsage)
	}

	copyRecord := *usage
	r.dailyUsage[usage.UserID][usage.UsageDate] = copyRecord
	return nil
}

func (r *MemoryVIPRepository) DeleteDailyUsages(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.dailyUsage, userID)
	return nil
}

func (r *MemoryVIPRepository) SaveTask(ctx context.Context, task *domain.VIPTaskRecord) error {
	if task == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tasks[task.UserID]; !ok {
		r.tasks[task.UserID] = make(map[string]domain.VIPTaskRecord)
	}

	copyRecord := *task
	r.tasks[task.UserID][task.ID] = copyRecord
	return nil
}

func (r *MemoryVIPRepository) ListTasks(ctx context.Context, userID string) ([]domain.VIPTaskRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	byID := r.tasks[userID]
	result := make([]domain.VIPTaskRecord, 0, len(byID))
	for _, task := range byID {
		result = append(result, task)
	}
	return result, nil
}

func (r *MemoryVIPRepository) DeleteTasks(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tasks, userID)
	return nil
}

func (r *MemoryVIPRepository) SaveReport(ctx context.Context, report *domain.VIPStoredReport) error {
	if report == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	copyRecord := *report
	copyRecord.Sources = append([]domain.VIPReportSource(nil), report.Sources...)
	r.reportsByID[report.ID] = copyRecord

	if _, ok := r.reportIDsByUser[report.UserID]; !ok {
		r.reportIDsByUser[report.UserID] = make(map[string]struct{})
	}
	r.reportIDsByUser[report.UserID][report.ID] = struct{}{}
	return nil
}

func (r *MemoryVIPRepository) GetReportByID(ctx context.Context, reportID string) (*domain.VIPStoredReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.reportsByID[reportID]
	if !ok {
		return nil, nil
	}
	copyRecord := record
	copyRecord.Sources = append([]domain.VIPReportSource(nil), record.Sources...)
	return &copyRecord, nil
}

func (r *MemoryVIPRepository) DeleteReports(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	reportIDs := r.reportIDsByUser[userID]
	for reportID := range reportIDs {
		delete(r.reportsByID, reportID)
	}
	delete(r.reportIDsByUser, userID)
	return nil
}

func (r *MemoryVIPRepository) SaveOrder(ctx context.Context, order *domain.VIPOrder) error {
	if order == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	copyRecord := *order
	r.ordersByID[order.ID] = copyRecord
	r.orderIDByNo[order.OrderNo] = order.ID
	if _, ok := r.orderIDsByUser[order.UserID]; !ok {
		r.orderIDsByUser[order.UserID] = make(map[string]struct{})
	}
	r.orderIDsByUser[order.UserID][order.ID] = struct{}{}
	return nil
}

func (r *MemoryVIPRepository) GetOrderByID(ctx context.Context, orderID string) (*domain.VIPOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.ordersByID[strings.TrimSpace(orderID)]
	if !ok {
		return nil, nil
	}
	copyRecord := record
	return &copyRecord, nil
}

func (r *MemoryVIPRepository) GetOrderByOrderNo(ctx context.Context, orderNo string) (*domain.VIPOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orderID, ok := r.orderIDByNo[strings.TrimSpace(orderNo)]
	if !ok {
		return nil, nil
	}
	record, ok := r.ordersByID[orderID]
	if !ok {
		return nil, nil
	}
	copyRecord := record
	return &copyRecord, nil
}

func (r *MemoryVIPRepository) DeleteOrders(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	orderIDs := r.orderIDsByUser[userID]
	for orderID := range orderIDs {
		record, ok := r.ordersByID[orderID]
		if ok {
			delete(r.orderIDByNo, record.OrderNo)
		}
		delete(r.ordersByID, orderID)
	}
	delete(r.orderIDsByUser, userID)
	return nil
}
