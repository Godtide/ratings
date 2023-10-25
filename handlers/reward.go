package handlers

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

//Reward describes reward types available
type Reward struct {
	ID     primitive.ObjectID `json:"_id,omitempty" bson:"_id"validate:"required"`
	Type   string             `         json:"type" bson:"type" validate:"required"` //high. medium, low
	Points int8               `json:"points, omitempty" bson:"points" validate:"required"`
	//AmountRedeemable int8             `json:"redeemable" bson:"redeemable" validate:"required"`
	expiry    int8      `json:"expiry" bson:"expiry" validate:"required"` //expiry in days
	CreatedAt time.Time `json:"createdAt" bson:"createdAt" validate:"required"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt" validate:"required"`
	DeletedAt time.Time `json:"deletedAt" bson:"deletedAt" validate:"required"`
}

//UserReward describes reward accrued to a user
type UserReward struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id"validate:"omitempty"`
	UserId    primitive.ObjectID `json:"user_id, omitempty" bson:"user_id, omitempty"`
	status    string             `         json:"status,omitempty" bson:"status" validate:"required"` //open, redeemed, expired
	Points    int8               `json:"points, omitempty" bson:"points" validate:"required"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt" validate:"required"`
	expiresAt time.Time          `json:"expiresAt" bson:"expiresAt" validate:"required"` //expiry in days, gets deleted after expiry
}
