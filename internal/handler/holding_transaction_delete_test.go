package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

func TestDeleteHoldingTransactionAuthenticationAndOwnerScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryUserRepository()
	svc := service.NewUserPreferenceService(repository.NewMemoryFundRepository(), repo, repo, repo, repo)
	handler := NewUserHandler(svc, repo, domain.QuoteSourceSina)
	record := &domain.UserFundHoldingTransaction{ID: "tx", UserID: "owner", FundID: "005827", Type: domain.UserFundHoldingTransactionBuy}
	if err := repo.SaveFundHoldingTransaction(t.Context(), record); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.DELETE("/transactions/:transactionId", func(c *gin.Context) {
		if user := c.GetHeader("X-Test-User"); user != "" {
			c.Set("current_user", &domain.User{ID: user})
		}
	}, handler.DeleteFundHoldingTransaction)
	for _, tc := range []struct {
		user, id string
		want     int
	}{
		{"", "tx", 401}, {"other", "tx", 404}, {"owner", "missing", 404}, {"owner", "tx", 200}, {"owner", "tx", 404},
	} {
		req := httptest.NewRequest(http.MethodDelete, "/transactions/"+tc.id, nil)
		req.Header.Set("X-Test-User", tc.user)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != tc.want {
			t.Fatalf("user=%q id=%q status=%d want=%d body=%s", tc.user, tc.id, response.Code, tc.want, response.Body.String())
		}
	}
}

func TestDeleteLaterHistoryPreventsUnsafeRollbackThroughAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryUserRepository()
	svc := service.NewUserPreferenceService(repository.NewMemoryFundRepository(), repo, repo, repo, repo)
	handler := NewUserHandler(svc, repo, domain.QuoteSourceSina)
	holding := &domain.UserFundHolding{ID: "lot", UserID: "owner", FundID: "005827", Amount: decimal.NewFromInt(1200), Shares: decimal.NewFromInt(600)}
	if err := repo.SaveFundHolding(t.Context(), holding); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Add(-time.Hour)
	for i, id := range []string{"earlier", "later"} {
		record := &domain.UserFundHoldingTransaction{ID: id, UserID: "owner", HoldingID: "lot", FundID: "005827", Type: domain.UserFundHoldingTransactionBuy, Amount: decimal.NewFromInt(200), CreatedAt: now.Add(time.Duration(i) * time.Minute)}
		if err := repo.SaveFundHoldingTransaction(t.Context(), record); err != nil {
			t.Fatal(err)
		}
	}
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("current_user", &domain.User{ID: "owner"}) })
	router.DELETE("/transactions/:transactionId", handler.DeleteFundHoldingTransaction)
	router.GET("/transactions/:transactionId/rollback-preview", handler.PreviewFundHoldingTransactionRollback)
	router.POST("/transactions/:transactionId/rollback-apply", handler.ApplyFundHoldingTransactionRollback)
	deleted := httptest.NewRecorder()
	router.ServeHTTP(deleted, httptest.NewRequest(http.MethodDelete, "/transactions/later", nil))
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	preview := httptest.NewRecorder()
	router.ServeHTTP(preview, httptest.NewRequest(http.MethodGet, "/transactions/earlier/rollback-preview", nil))
	var payload struct {
		Data domain.UserFundHoldingTransactionRollbackPreview `json:"data"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if preview.Code != http.StatusOK || payload.Data.CanApplyAutomatically {
		t.Fatalf("unsafe preview: %s", preview.Body.String())
	}
	rejected := httptest.NewRecorder()
	router.ServeHTTP(rejected, httptest.NewRequest(http.MethodPost, "/transactions/earlier/rollback-apply", strings.NewReader(`{"reason":"test"}`)))
	if rejected.Code != http.StatusConflict {
		t.Fatalf("rollback status=%d body=%s", rejected.Code, rejected.Body.String())
	}
	after, _ := repo.GetFundHolding(t.Context(), "owner", "lot")
	if after == nil || !after.Amount.Equal(holding.Amount) || !after.Shares.Equal(holding.Shares) {
		t.Fatalf("history deletion changed position: %+v", after)
	}
}
