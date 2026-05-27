package campaigns

import (
	"errors"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
	"github.com/pulse/api/internal/utils"
)

// POST /api/campaigns
func handleCreateCampaign(c *gin.Context) {
	var req CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	businessID := middleware.GetUserID(c)
	campaign, err := createCampaign(c.Request.Context(), businessID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidDates):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrInsufficientBalance):
			utils.Fail(c, http.StatusPaymentRequired, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, "Failed to create campaign")
		}
		return
	}

	utils.OK(c, http.StatusCreated, "Campaign created", toCampaignResponse(campaign))
}

// GET /api/campaigns
func handleGetCampaigns(c *gin.Context) {
	var q CampaignListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Invalid query parameters", err.Error())
		return
	}

	campaigns, total, err := getCampaigns(c.Request.Context(), q)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch campaigns")
		return
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 50 {
		q.Limit = 20
	}

	resp := make([]CampaignResponse, 0, len(campaigns))
	for i := range campaigns {
		resp = append(resp, toCampaignResponse(&campaigns[i]))
	}

	meta := CampaignListMeta{
		Total: total,
		Page:  q.Page,
		Limit: q.Limit,
		Pages: int64(math.Ceil(float64(total) / float64(q.Limit))),
	}

	utils.OKWithMeta(c, http.StatusOK, "", resp, meta)
}

// GET /api/campaigns/my
func handleGetMyCampaigns(c *gin.Context) {
	businessID := middleware.GetUserID(c)
	campaigns, err := getMyCampaigns(c.Request.Context(), businessID)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch campaigns")
		return
	}

	resp := make([]CampaignResponse, 0, len(campaigns))
	for i := range campaigns {
		resp = append(resp, toCampaignResponse(&campaigns[i]))
	}

	utils.OK(c, http.StatusOK, "", resp)
}

// GET /api/campaigns/:id
func handleGetCampaign(c *gin.Context) {
	id := c.Param("id")
	campaign, err := getCampaign(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCampaignNotFound) {
			utils.Fail(c, http.StatusNotFound, "Campaign not found")
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch campaign")
		return
	}

	utils.OK(c, http.StatusOK, "", toCampaignResponse(campaign))
}

// PATCH /api/campaigns/:id
func handleUpdateCampaign(c *gin.Context) {
	var req UpdateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	businessID := middleware.GetUserID(c)
	id := c.Param("id")

	campaign, err := updateCampaign(c.Request.Context(), businessID, id, req)
	if err != nil {
		if errors.Is(err, ErrCampaignNotFound) {
			utils.Fail(c, http.StatusNotFound, "Campaign not found")
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Failed to update campaign")
		return
	}

	utils.OK(c, http.StatusOK, "Campaign updated", toCampaignResponse(campaign))
}

// DELETE /api/campaigns/:id
func handleDeleteCampaign(c *gin.Context) {
	businessID := middleware.GetUserID(c)
	id := c.Param("id")

	if err := deleteCampaign(c.Request.Context(), businessID, id); err != nil {
		switch {
		case errors.Is(err, ErrCampaignNotFound):
			utils.Fail(c, http.StatusNotFound, "Campaign not found")
		case errors.Is(err, ErrCannotDelete):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, "Failed to delete campaign")
		}
		return
	}

	utils.OK(c, http.StatusOK, "Campaign deleted and budget refunded", nil)
}
