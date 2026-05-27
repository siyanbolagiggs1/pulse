package submissions

import (
	"errors"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/middleware"
	"github.com/pulse/api/internal/utils"
)

// POST /api/submissions
func handleCreateSubmission(c *gin.Context) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	promoterID := middleware.GetUserID(c)
	sub, err := createSubmission(c.Request.Context(), promoterID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrAccountSuspended):
			utils.Fail(c, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrRateLimited):
			utils.Fail(c, http.StatusTooManyRequests, err.Error())
		case errors.Is(err, ErrCampaignNotActive), errors.Is(err, ErrCampaignExpired), errors.Is(err, ErrCampaignFull):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrPlatformMismatch), errors.Is(err, ErrEligibility):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrAlreadySubmitted), errors.Is(err, ErrDuplicateRepostURL):
			utils.Fail(c, http.StatusConflict, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, "Failed to create submission")
		}
		return
	}

	utils.OK(c, http.StatusCreated, "Submission created", toSubmissionResponse(sub))
}

// POST /api/submissions/upload
func handleUploadScreenshot(c *gin.Context) {
	file, header, err := c.Request.FormFile("screenshot")
	if err != nil {
		utils.Fail(c, http.StatusBadRequest, "screenshot file is required")
		return
	}
	defer file.Close()

	const maxSize = 10 << 20 // 10 MB
	if header.Size > maxSize {
		utils.Fail(c, http.StatusBadRequest, "screenshot must be under 10 MB")
		return
	}

	url, err := saveScreenshot(file, header)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to save screenshot")
		return
	}

	utils.OK(c, http.StatusOK, "Screenshot uploaded", UploadResponse{URL: url})
}

// GET /api/submissions
func handleGetSubmissions(c *gin.Context) {
	var q SubmissionListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Invalid query parameters", err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	submissions, total, err := getSubmissions(c.Request.Context(), userID, role, q)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch submissions")
		return
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 50 {
		q.Limit = 20
	}

	resp := make([]SubmissionResponse, 0, len(submissions))
	for i := range submissions {
		resp = append(resp, toSubmissionResponse(&submissions[i]))
	}

	meta := SubmissionListMeta{
		Total: total,
		Page:  q.Page,
		Limit: q.Limit,
		Pages: int64(math.Ceil(float64(total) / float64(q.Limit))),
	}

	utils.OKWithMeta(c, http.StatusOK, "", resp, meta)
}

// GET /api/submissions/:id
func handleGetSubmission(c *gin.Context) {
	id := c.Param("id")
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	sub, err := getSubmission(c.Request.Context(), id, userID, role)
	if err != nil {
		if errors.Is(err, ErrSubmissionNotFound) {
			utils.Fail(c, http.StatusNotFound, "Submission not found")
			return
		}
		utils.Fail(c, http.StatusInternalServerError, "Failed to fetch submission")
		return
	}

	utils.OK(c, http.StatusOK, "", toSubmissionResponse(sub))
}

// POST /api/submissions/:id/approve
func handleApproveSubmission(c *gin.Context) {
	id := c.Param("id")
	adminID := middleware.GetUserID(c)

	sub, err := approveSubmission(c.Request.Context(), adminID, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubmissionNotFound):
			utils.Fail(c, http.StatusNotFound, "Submission not found")
		case errors.Is(err, ErrNotReviewable):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, "Failed to approve submission")
		}
		return
	}

	utils.OK(c, http.StatusOK, "Submission approved", toSubmissionResponse(sub))
}

// POST /api/submissions/:id/reject
func handleRejectSubmission(c *gin.Context) {
	var req RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailWithErrors(c, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	id := c.Param("id")
	adminID := middleware.GetUserID(c)

	sub, err := rejectSubmission(c.Request.Context(), adminID, id, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubmissionNotFound):
			utils.Fail(c, http.StatusNotFound, "Submission not found")
		case errors.Is(err, ErrNotReviewable):
			utils.Fail(c, http.StatusBadRequest, err.Error())
		default:
			utils.Fail(c, http.StatusInternalServerError, "Failed to reject submission")
		}
		return
	}

	utils.OK(c, http.StatusOK, "Submission rejected", toSubmissionResponse(sub))
}
