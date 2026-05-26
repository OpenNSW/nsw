package profile

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/OpenNSW/nsw/backend/internal/auth"
	"github.com/OpenNSW/nsw/backend/internal/profile/company"
	"github.com/OpenNSW/nsw/backend/internal/profile/user"
)

// Handler handles profile HTTP requests.
type Handler struct {
	userSvc    user.Service
	companySvc company.Service
}

// NewHandler creates a new profile Handler.
func NewHandler(userSvc user.Service, companySvc company.Service) *Handler {
	return &Handler{
		userSvc:    userSvc,
		companySvc: companySvc,
	}
}

// CompanySummary represents trimmed company details.
type CompanySummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UserProfile represents the populated user profile containing nested company details.
type UserProfile struct {
	ID          string          `json:"id"`
	Email       string          `json:"email"`
	PhoneNumber string          `json:"phoneNumber"`
	Data        json.RawMessage `json:"data"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	Company     *CompanySummary `json:"company,omitempty"`
}

// HandleGetProfile handles GET /api/v1/profile.
func (h *Handler) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.GetAuthContext(ctx)
	if authCtx == nil || authCtx.User == nil {
		slog.Warn("unauthorized profile access attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	uRecord, err := h.userSvc.GetUser(authCtx.User.ID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			slog.Debug("user record not found", "userId", authCtx.User.ID)
			http.Error(w, "user profile not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to retrieve user profile", "userId", authCtx.User.ID, "error", err)
		http.Error(w, "failed to retrieve user profile", http.StatusInternalServerError)
		return
	}

	var compSummary *CompanySummary
	if uRecord.OUHandle != "" {
		companyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		comp, err := h.companySvc.GetCompanyByOUHandle(companyCtx, uRecord.OUHandle)
		if err != nil {
			if errors.Is(err, company.ErrCompanyNotFound) {
				slog.Debug("company record not found for user", "ouHandle", uRecord.OUHandle)
			} else {
				slog.Error("failed to retrieve company profile", "ouHandle", uRecord.OUHandle, "error", err)
				http.Error(w, "failed to retrieve company profile", http.StatusInternalServerError)
				return
			}
		} else {
			compSummary = &CompanySummary{
				ID:   comp.ID,
				Name: comp.Name,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(UserProfile{
		ID:          uRecord.ID,
		Email:       uRecord.Email,
		PhoneNumber: uRecord.PhoneNumber,
		Data:        uRecord.Data,
		CreatedAt:   uRecord.CreatedAt,
		UpdatedAt:   uRecord.UpdatedAt,
		Company:     compSummary,
	}); err != nil {
		slog.Error("failed to encode profile response", "error", err)
	}
}
