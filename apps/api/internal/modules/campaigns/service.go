package campaigns

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrCampaignNotFound    = errors.New("campaign not found")
	ErrInsufficientBalance = errors.New("wallet balance is insufficient for this campaign budget")
	ErrInvalidDates        = errors.New("end date must be after start date")
	ErrCannotDelete        = errors.New("only draft, paused, or active campaigns can be deleted")
	ErrNotOwner            = errors.New("you do not own this campaign")
)

func createCampaign(ctx context.Context, businessID string, req CreateCampaignRequest) (*models.Campaign, error) {
	if !req.EndDate.After(req.StartDate) {
		return nil, ErrInvalidDates
	}

	bizObjID, err := bson.ObjectIDFromHex(businessID)
	if err != nil {
		return nil, ErrCampaignNotFound
	}

	// Check and lock wallet balance.
	walletCol := database.GetCollection(models.WalletsCollection)
	var wallet models.Wallet
	if err := walletCol.FindOne(ctx, bson.M{"userId": bizObjID}).Decode(&wallet); err != nil {
		return nil, errors.New("wallet not found")
	}
	if wallet.AvailableBalance < req.Budget {
		return nil, ErrInsufficientBalance
	}

	// Deduct from available, add to pending (budget is locked until submissions are processed).
	now := time.Now().UTC()
	_, err = walletCol.UpdateOne(ctx,
		bson.M{"_id": wallet.ID},
		bson.M{"$inc": bson.M{
			"availableBalance": -req.Budget,
			"pendingBalance":   req.Budget,
			"totalSpent":       req.Budget,
		}, "$set": bson.M{"updatedAt": now}},
	)
	if err != nil {
		return nil, err
	}

	// Record the lock transaction.
	tx := models.Transaction{
		WalletID:     wallet.ID,
		UserID:       bizObjID,
		Type:         models.TxCampaignLock,
		Amount:       req.Budget,
		BalanceAfter: wallet.AvailableBalance - req.Budget,
		Description:  "Budget locked for campaign: " + req.Title,
		CreatedAt:    now,
	}
	if _, err := database.GetCollection(models.TransactionsCollection).InsertOne(ctx, tx); err != nil {
		// Non-fatal: wallet is already updated; log but don't fail the campaign creation.
		_ = err
	}

	maxParticipants := req.MaxParticipants
	if maxParticipants == 0 && req.BaseRepostRate > 0 {
		maxParticipants = int(math.Floor(req.Budget / req.BaseRepostRate))
	}

	campaign := &models.Campaign{
		BusinessID:          bizObjID,
		Title:               req.Title,
		Description:         req.Description,
		TargetURL:           req.TargetURL,
		MediaAssets:         req.MediaAssets,
		Platform:            req.Platform,
		Budget:              req.Budget,
		RemainingBudget:     req.Budget,
		BaseRepostRate:      req.BaseRepostRate,
		MinFollowers:        req.MinFollowers,
		MinEngagementRate:   req.MinEngagementRate,
		MinInfluenceScore:   req.MinInfluenceScore,
		MaxParticipants:     maxParticipants,
		CurrentParticipants: 0,
		Status:              models.CampaignStatusActive,
		StartDate:           req.StartDate,
		EndDate:             req.EndDate,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	col := database.GetCollection(models.CampaignsCollection)
	result, err := col.InsertOne(ctx, campaign)
	if err != nil {
		// Rollback wallet lock on failure.
		_, _ = walletCol.UpdateOne(ctx,
			bson.M{"_id": wallet.ID},
			bson.M{"$inc": bson.M{
				"availableBalance": req.Budget,
				"pendingBalance":   -req.Budget,
				"totalSpent":       -req.Budget,
			}, "$set": bson.M{"updatedAt": time.Now().UTC()}},
		)
		return nil, err
	}

	if err := col.FindOne(ctx, bson.M{"_id": result.InsertedID}).Decode(campaign); err != nil {
		return nil, err
	}

	return campaign, nil
}

func getCampaigns(ctx context.Context, q CampaignListQuery) ([]models.Campaign, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 50 {
		q.Limit = 20
	}

	filter := bson.M{}
	if q.Platform != "" {
		filter["platform"] = q.Platform
	}
	if q.Status != "" {
		filter["status"] = q.Status
	} else {
		// Default marketplace view shows only active campaigns.
		filter["status"] = models.CampaignStatusActive
	}

	col := database.GetCollection(models.CampaignsCollection)
	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	sortField := "createdAt"
	sortDir := int32(-1)
	if q.Sort == "payout_asc" {
		sortField, sortDir = "baseRepostRate", 1
	} else if q.Sort == "payout_desc" {
		sortField, sortDir = "baseRepostRate", -1
	}

	skip := int64((q.Page - 1) * q.Limit)
	cursor, err := col.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: sortField, Value: sortDir}}).
			SetSkip(skip).
			SetLimit(int64(q.Limit)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var campaigns []models.Campaign
	if err := cursor.All(ctx, &campaigns); err != nil {
		return nil, 0, err
	}

	return campaigns, total, nil
}

func getCampaign(ctx context.Context, id string) (*models.Campaign, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrCampaignNotFound
	}

	var campaign models.Campaign
	if err := database.GetCollection(models.CampaignsCollection).
		FindOne(ctx, bson.M{"_id": objID}).Decode(&campaign); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrCampaignNotFound
		}
		return nil, err
	}

	return &campaign, nil
}

func getMyCampaigns(ctx context.Context, businessID string) ([]models.Campaign, error) {
	objID, err := bson.ObjectIDFromHex(businessID)
	if err != nil {
		return nil, ErrCampaignNotFound
	}

	cursor, err := database.GetCollection(models.CampaignsCollection).
		Find(ctx, bson.M{"businessId": objID},
			options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
		)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var campaigns []models.Campaign
	if err := cursor.All(ctx, &campaigns); err != nil {
		return nil, err
	}

	return campaigns, nil
}

func updateCampaign(ctx context.Context, businessID, campaignID string, req UpdateCampaignRequest) (*models.Campaign, error) {
	bizObjID, err := bson.ObjectIDFromHex(businessID)
	if err != nil {
		return nil, ErrNotOwner
	}
	campObjID, err := bson.ObjectIDFromHex(campaignID)
	if err != nil {
		return nil, ErrCampaignNotFound
	}

	fields := bson.M{"updatedAt": time.Now().UTC()}
	if req.Title != "" {
		fields["title"] = req.Title
	}
	if req.Description != "" {
		fields["description"] = req.Description
	}
	if req.MediaAssets != nil {
		fields["mediaAssets"] = req.MediaAssets
	}
	if req.MinFollowers != nil {
		fields["minFollowers"] = *req.MinFollowers
	}
	if req.MinEngagementRate != nil {
		fields["minEngagementRate"] = *req.MinEngagementRate
	}
	if req.MinInfluenceScore != nil {
		fields["minInfluenceScore"] = *req.MinInfluenceScore
	}
	if req.EndDate != nil {
		fields["endDate"] = *req.EndDate
	}
	if req.Status != "" {
		fields["status"] = req.Status
	}

	var campaign models.Campaign
	err = database.GetCollection(models.CampaignsCollection).FindOneAndUpdate(
		ctx,
		bson.M{"_id": campObjID, "businessId": bizObjID},
		bson.M{"$set": fields},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&campaign)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrCampaignNotFound
		}
		return nil, err
	}

	return &campaign, nil
}

func deleteCampaign(ctx context.Context, businessID, campaignID string) error {
	bizObjID, err := bson.ObjectIDFromHex(businessID)
	if err != nil {
		return ErrNotOwner
	}
	campObjID, err := bson.ObjectIDFromHex(campaignID)
	if err != nil {
		return ErrCampaignNotFound
	}

	col := database.GetCollection(models.CampaignsCollection)
	var campaign models.Campaign
	if err := col.FindOne(ctx, bson.M{"_id": campObjID, "businessId": bizObjID}).Decode(&campaign); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrCampaignNotFound
		}
		return err
	}

	if campaign.Status == models.CampaignStatusCompleted {
		return ErrCannotDelete
	}

	// Refund remaining budget back to available balance.
	if campaign.RemainingBudget > 0 {
		now := time.Now().UTC()
		walletCol := database.GetCollection(models.WalletsCollection)
		_, _ = walletCol.UpdateOne(ctx,
			bson.M{"userId": bizObjID},
			bson.M{
				"$inc": bson.M{
					"availableBalance": campaign.RemainingBudget,
					"pendingBalance":   -campaign.RemainingBudget,
					"totalSpent":       -campaign.RemainingBudget,
				},
				"$set": bson.M{"updatedAt": now},
			},
		)

		// Record refund transaction.
		var wallet models.Wallet
		if err := walletCol.FindOne(ctx, bson.M{"userId": bizObjID}).Decode(&wallet); err == nil {
			tx := models.Transaction{
				WalletID:     wallet.ID,
				UserID:       bizObjID,
				Type:         models.TxRefund,
				Amount:       campaign.RemainingBudget,
				BalanceAfter: wallet.AvailableBalance,
				Description:  "Budget refunded from deleted campaign: " + campaign.Title,
				CreatedAt:    now,
			}
			_, _ = database.GetCollection(models.TransactionsCollection).InsertOne(ctx, tx)
		}
	}

	_, err = col.DeleteOne(ctx, bson.M{"_id": campObjID})
	return err
}
