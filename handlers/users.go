package handlers

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"time"

	"github.com/Godtide/rating/config"
	"github.com/Godtide/rating/dbiface"
	"github.com/dgrijalva/jwt-go"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

//User represents a user
type User struct {
	ID       primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Email    string             `json:"username" bson:"username" validate:"required,email"`
	Password string             `json:"password,omitempty" bson:"password" validate:"required,min=8,max=300"`
	IsAdmin  bool               `json:"isadmin,omitempty" bson:"isadmin"`
}

//UsersHandler users handler
type UsersHandler struct {
	UserCol   dbiface.CollectionAPI
	WalletCol dbiface.CollectionAPI
}

type errorMessage struct {
	Message string `json:"message"`
}

var (
	prop config.Properties
)

func isCredValid(givenPwd, storedPwd string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(storedPwd), []byte(givenPwd)); err != nil {
		return false
	}
	return true
}

func insertUser(ctx context.Context, user User, collection dbiface.CollectionAPI) (User, *echo.HTTPError) {
	var newUser User

	res := collection.FindOne(ctx, bson.M{"username": user.Email})

	err := res.Decode(&newUser)

	if err != nil && err != mongo.ErrNoDocuments {
		log.Errorf("Unable to decode retrieved user: %v", err)
		return newUser,
			echo.NewHTTPError(http.StatusUnprocessableEntity, errorMessage{Message: "Unable to decode retrieved user"})
	}
	if newUser.Email != "" {

		log.Errorf("User by %s already exists", user.Email)
		return newUser,
			echo.NewHTTPError(http.StatusBadRequest, errorMessage{Message: "User already exists"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 8)
	if err != nil {
		log.Errorf("Unable to hash the password: %v", err)
		return user,
			echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "Unable to process the password"})
	}
	user.Password = string(hashedPassword)

	_, err = collection.InsertOne(ctx, user)

	newRes := collection.FindOne(ctx, bson.M{"username": user.Email})
	err = newRes.Decode(&newUser)

	if err != nil {
		log.Errorf("Unable to insert the user :%+v", err)
		return user,
			echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "Unable to create the user"})
	}
	return newUser, nil
}

//CreateUser creates a user
func (h *UsersHandler) CreateUser(c echo.Context) error {
	var (
		user    User
		resUser User
	)
	c.Echo().Validator = &userValidator{validator: v}
	if err := c.Bind(&user); err != nil {
		log.Errorf("Unable to bind to user struct.")
		return c.JSON(http.StatusUnprocessableEntity,
			errorMessage{Message: "Unable to parse the request payload."})
	}
	if err := c.Validate(user); err != nil {
		log.Errorf("Unable to validate the requested body.")
		return c.JSON(http.StatusBadRequest,
			errorMessage{Message: "Unable to validate request body"})
	}
	resUser, httpError := insertUser(context.Background(), user, h.UserCol)
	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}

	fullWallet, httpError := createUserWallet(context.Background(), resUser.ID, h.WalletCol)

	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}

	return c.JSON(http.StatusCreated, fullWallet)
}
