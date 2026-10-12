package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/watcharaY/portfolio-tracker/backend/internal/domain"
	"github.com/watcharaY/portfolio-tracker/backend/internal/usecase"
)

type dashboardHandler struct{ svc *usecase.DashboardService }

type assetClassResponse struct {
	Key           string       `json:"key"`
	Label         string       `json:"label"`
	Value         domain.Money `json:"value"`
	Cost          domain.Money `json:"cost"`
	PL            domain.Money `json:"pl"`
	PLPct         *float64     `json:"plPct"`
	AllocationPct *float64     `json:"allocationPct"`
}

type dashboardResponse struct {
	Total                     assetClassResponse   `json:"total"`
	Classes                   []assetClassResponse `json:"classes"`
	EstSavingsInterestPerYear domain.Money         `json:"estSavingsInterestPerYear"`
}

func toAssetClassResponse(a domain.AssetClassSummary) assetClassResponse {
	key := string(a.Class)
	if key == "" {
		key = "total"
	}
	return assetClassResponse{
		Key: key, Label: a.Label, Value: a.Value, Cost: a.Cost, PL: a.PL,
		PLPct: a.PLPercent, AllocationPct: a.Allocation,
	}
}

func (h *dashboardHandler) summary(c fiber.Ctx) error {
	s, err := h.svc.Summary(c.Context())
	if err != nil {
		return err
	}
	resp := dashboardResponse{
		Total:                     toAssetClassResponse(s.Total),
		Classes:                   make([]assetClassResponse, len(s.Classes)),
		EstSavingsInterestPerYear: s.EstSavingsInterestPerYear,
	}
	for i, cl := range s.Classes {
		resp.Classes[i] = toAssetClassResponse(cl)
	}
	return c.JSON(resp)
}
