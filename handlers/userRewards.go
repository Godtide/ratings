package handlers

import (
	"github.com/Godtide/rating/dbiface"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/net/context"
	"net/http"
	"net/url"
	"time"
)

//UserReward describes reward accrued to a user
type UserReward struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id"validate:"omitempty"`
	UserId    primitive.ObjectID `json:"user_id, omitempty" bson:"user_id, omitempty"`
	RewardId  primitive.ObjectID `json:"user_id, omitempty" bson:"user_id, omitempty"`
	status    string             `json:"status,omitempty" bson:"status" validate:"required"` //open, redeemed, expired
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt" validate:"required"`
	expiresAt time.Time          `json:"expiresAt" bson:"expiresAt" validate:"required"` //expiry in days, gets deleted after expiry
}

//UserRewardHandler a user_reward handler
type UserRewardHandler struct {
	UserRewardCol dbiface.CollectionAPI
	RewardCol     dbiface.CollectionAPI
	WalletCol     dbiface.CollectionAPI
	Wallet        Wallet
	Apikey        string 
	ContractAdrress string
}

func insertUserReward(ctx context.Context, userReward UserReward, collection dbiface.CollectionAPI) (interface{}, *echo.HTTPError) {
	userReward.ID = primitive.NewObjectID()
	userReward.CreatedAt = time.Now()

	insertID, err := collection.InsertOne(ctx, userReward)
	if err != nil {
		log.Errorf("Unable to insert to Database:%v", err)
		return nil,
			echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "unable to insert to database"})
	}
	return insertID.InsertedID, nil
}

//CreateRewards create rewards on mongodb 
func (r *UserRewardHandler) CreateUserRewards(c echo.Context) error {
	var reward UserReward
	c.Echo().Validator = &rewardValidator{validator: v}
	if err := c.Bind(&reward); err != nil {
		log.Errorf("Unable to bind : %v", err)
		return c.JSON(http.StatusUnprocessableEntity, errorMessage{Message: "unable to parse request payload"})
	}
	if err := c.Validate(reward); err != nil {
		log.Errorf("Unable to validate the userReward %+v %v", reward, err)
		return c.JSON(http.StatusBadRequest, errorMessage{Message: "unable to validate request payload"})
	}
	IDs, httpError := insertUserReward(context.Background(), reward, r.UserRewardCol)
	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}
	return c.JSON(http.StatusCreated, IDs)
}

func findUserRewards(ctx context.Context, q url.Values, collection dbiface.CollectionAPI) ([]UserReward, *echo.HTTPError) {
	var userRewards []UserReward
	filter := make(map[string]interface{})
	for k, v := range q {
		filter[k] = v[0]
	}
	if filter["_id"] != nil {
		docID, err := primitive.ObjectIDFromHex(filter["_id"].(string))
		if err != nil {
			log.Errorf("Unable to convert to Object ID : %v", err)
			return userRewards,
				echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "unable to convert to ObjectID"})
		}
		filter["_id"] = docID
	}
	cursor, err := collection.Find(ctx, bson.M(filter))
	if err != nil {
		log.Errorf("Unable to find the userReward : %v", err)
		return userRewards,
			echo.NewHTTPError(http.StatusNotFound, errorMessage{Message: "unable to find the userReward"})
	}
	err = cursor.All(ctx, &userRewards)
	if err != nil {
		log.Errorf("Unable to read the cursor : %v", err)
		return userRewards,
			echo.NewHTTPError(http.StatusUnprocessableEntity, errorMessage{Message: "unable to parse retrieved userRewards"})
	}
	return userRewards, nil
}

//GetRewards gets a list of reward
func (r *UserRewardHandler) GetUserRewards(c echo.Context) error {
	userRewards, httpError := findUserRewards(context.Background(), c.QueryParams(), r.UserRewardCol)
	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}
	return c.JSON(http.StatusOK, userRewards)
}

func findUserReward(ctx context.Context, id string, collection dbiface.CollectionAPI) (UserReward, *echo.HTTPError) {
	var reward UserReward
	docID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Errorf("Unable to convert to Object ID : %v", err)
		return reward,
			echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "unable to convert to ObjectID"})
	}
	res := collection.FindOne(ctx, bson.M{"_id": docID})
	err = res.Decode(&reward)
	if err != nil {
		log.Errorf("Unable to find the reward : %v", err)
		return reward,
			echo.NewHTTPError(http.StatusNotFound, errorMessage{Message: "unable to find the reward"})
	}
	return reward, nil
}

//GetUserReward gets a single userReward
func (r *UserRewardHandler) GetUserReward(c echo.Context) error {
	reward, httpError := findUserReward(context.Background(), c.Param("id"), r.UserRewardCol)
	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}
	return c.JSON(http.StatusOK, reward)
}

//claim rewards
func (r *UserRewardHandler) ClaimReward(c echo.Context) error {
	var (
		wallet Wallet
		txHash string
	)
	userReward, httpError := findUserReward(context.Background(), c.Param("id"), r.UserRewardCol)
	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}
	reward, httpError := findReward(context.Background(), userReward.RewardId.String(), r.RewardCol)
	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}

	wallet, httpError = findWallet(context.Background(), userReward.UserId.String(), r.WalletCol)
	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}
	var redeemableAmount = reward.Points * reward.AmountRedeemable
	txHash, _ = transferRewards(r.Wallet, wallet.PublicKey, string(redeemableAmount), r.ApiKey, r.ContractAdrress)

	return c.JSON(http.StatusOK, txHash)
}

func deleteUserReward(ctx context.Context, id string, collection dbiface.CollectionAPI) (int64, *echo.HTTPError) {
	docID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Errorf("Unable convert to ObjectID : %v", err)
		return 0,
			echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "unable to convert to ObjectID"})
	}
	res, err := collection.DeleteOne(ctx, bson.M{"_id": docID})
	if err != nil {
		log.Errorf("Unable to delete the userRewards : %v", err)
		return 0,
			echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "unable to delete the userRewards"})
	}
	return res.DeletedCount, nil
}

//DeleteUserReward gets a single UserReward
func (r *UserRewardHandler) DeleteUserReward(c echo.Context) error {
	delCount, httpError := deleteUserReward(context.Background(), c.Param("id"), h.Col)
	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}
	return c.JSON(http.StatusOK, delCount)
}
