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
	status    string             `         json:"status,omitempty" bson:"status" validate:"required"` //open, redeemed, expired
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt" validate:"required"`
	expiresAt time.Time          `json:"expiresAt" bson:"expiresAt" validate:"required"` //expiry in days, gets deleted after expiry
}

//RewardHandler a user_reward handler
type UserRewardHandler struct {
	UserReward dbiface.CollectionAPI
}

func insertUserReward(ctx context.Context, reward UserReward, collection dbiface.CollectionAPI) (interface{}, *echo.HTTPError) {
	reward.ID = primitive.NewObjectID()
	insertID, err := collection.InsertOne(ctx, reward)
	if err != nil {
		log.Errorf("Unable to insert to Database:%v", err)
		return nil,
			echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "unable to insert to database"})
	}
	return insertID.InsertedID, nil
}

//CreateRewards create rewards on mongodb database
func (r *RewardHandler) CreateUserRewards(c echo.Context) error {
	var reward UserReward
	c.Echo().Validator = &rewardValidator{validator: v}
	if err := c.Bind(&reward); err != nil {
		log.Errorf("Unable to bind : %v", err)
		return c.JSON(http.StatusUnprocessableEntity, errorMessage{Message: "unable to parse request payload"})
	}
	if err := c.Validate(reward); err != nil {
		log.Errorf("Unable to validate the product %+v %v", reward, err)
		return c.JSON(http.StatusBadRequest, errorMessage{Message: "unable to validate request payload"})
	}
	IDs, httpError := insertUserReward(context.Background(), reward, r.Reward)
	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}
	return c.JSON(http.StatusCreated, IDs)
}

func findUserRewards(ctx context.Context, q url.Values, collection dbiface.CollectionAPI) ([]UserReward, *echo.HTTPError) {
	var rewards []UserReward
	filter := make(map[string]interface{})
	for k, v := range q {
		filter[k] = v[0]
	}
	if filter["_id"] != nil {
		docID, err := primitive.ObjectIDFromHex(filter["_id"].(string))
		if err != nil {
			log.Errorf("Unable to convert to Object ID : %v", err)
			return rewards,
				echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "unable to convert to ObjectID"})
		}
		filter["_id"] = docID
	}
	cursor, err := collection.Find(ctx, bson.M(filter))
	if err != nil {
		log.Errorf("Unable to find the rewards : %v", err)
		return rewards,
			echo.NewHTTPError(http.StatusNotFound, errorMessage{Message: "unable to find the rewards"})
	}
	err = cursor.All(ctx, &rewards)
	if err != nil {
		log.Errorf("Unable to read the cursor : %v", err)
		return rewards,
			echo.NewHTTPError(http.StatusUnprocessableEntity, errorMessage{Message: "unable to parse retrieved products"})
	}
	return rewards, nil
}

//GetRewards gets a list of reward
func (h *RewardHandler) GetUserRewards(c echo.Context) error {
	products, httpError := findUserRewards(context.Background(), c.QueryParams(), h.Reward)
	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}
	return c.JSON(http.StatusOK, products)
}

func findUserReward(ctx context.Context, id string, collection dbiface.CollectionAPI) (Reward, *echo.HTTPError) {
	var reward Reward
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

//GetProduct gets a single reward
func (h *RewardHandler) GetUserReward(c echo.Context) error {
	reward, httpError := findUserReward(context.Background(), c.Param("id"), h.Reward)
	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}
	return c.JSON(http.StatusOK, reward)
}
