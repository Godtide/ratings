package handlers

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/sha3"
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

//Wallet describes an user wallet to manage keys
type Wallet struct {
	ID         primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	UserId     primitive.ObjectID `json:"user_id, omitempty" bson:"user_string, omitempty"`
	PrivateKey string             `json:"private_key" bson:"private_key" validate:"required"`
	PublicKey  string             `json:"public_key" bson:"public_key" validate:"required"`
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

func (u User) createToken() (string, error) {
	if err := cleanenv.ReadEnv(&prop); err != nil {
		log.Errorf("Configuration cannot be read : %v", err)
	}
	claims := jwt.MapClaims{}
	claims["authorized"] = u.IsAdmin
	claims["user_id"] = u.Email
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := at.SignedString([]byte(prop.JwtTokenSecret))
	if err != nil {
		log.Errorf("Unable to generate the token :%v", err)
		return "", err
	}
	return token, nil
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
		return newUser,
			echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "Unable to process the password"})
	}
	user.Password = string(hashedPassword)
	_, err = collection.InsertOne(ctx, user)
	if err != nil {
		log.Errorf("Unable to insert the user :%+v", err)
		return newUser,
			echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "Unable to create the user"})
	}
	return user, nil
}

//CreateUser creates a user
func (h *UsersHandler) CreateUser(c echo.Context) error {
	var (
		user       User
		partWallet Wallet
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

	partWallet, err := createWallet()
	if err != nil {
		log.Errorf("Unable to create wallet :%+v", err)
		return c.JSON(http.StatusBadRequest,
			errorMessage{Message: "Unable to create wallet"})
	}

	wallet, httpError := insertWallet(context.Background(), Wallet{
		UserId:     resUser.ID,
		PrivateKey: partWallet.PrivateKey,
		PublicKey:  partWallet.PublicKey,
	}, h.WalletCol)

	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}

	token, err := user.createToken()
	if err != nil {
		log.Errorf("Unable to generate the token.")
		return echo.NewHTTPError(http.StatusInternalServerError,
			errorMessage{Message: "Unable to generate the token"})
	}
	c.Response().Header().Set("x-auth-token", "Bearer "+token)
	return c.JSON(http.StatusCreated, wallet)
}

func authenticateUser(ctx context.Context, reqUser User, collection dbiface.CollectionAPI) (User, *echo.HTTPError) {
	var storedUser User //user in db
	// check whether the user exists or not
	res := collection.FindOne(ctx, bson.M{"username": reqUser.Email})
	err := res.Decode(&storedUser)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Errorf("Unable to decode retrieved user: %v", err)
		return storedUser,
			echo.NewHTTPError(http.StatusUnprocessableEntity, errorMessage{Message: "Unable to decode retrieved user"})
	}
	if err == mongo.ErrNoDocuments {
		log.Errorf("User %s does not exist.", reqUser.Email)
		return storedUser,
			echo.NewHTTPError(http.StatusNotFound, errorMessage{Message: "User does not exist"})
	}
	//validate the password
	if !isCredValid(reqUser.Password, storedUser.Password) {
		return storedUser,
			echo.NewHTTPError(http.StatusUnauthorized, errorMessage{Message: "Credentials invalid"})
	}
	return User{Email: storedUser.Email}, nil
}

func findUser(ctx context.Context, username string, collection dbiface.CollectionAPI) (User, error) {
	var user User
	res := collection.FindOne(ctx, bson.M{"username": username})
	err := res.Decode(&user)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Errorf("Unable to decode retrieved user: %v", err)
		return user,
			echo.NewHTTPError(http.StatusUnprocessableEntity, errorMessage{Message: "Unable to decode retrieved user"})
	}
	if user.Email == "" {
		log.Printf("User by %s exists", username)

		return user,
			echo.NewHTTPError(http.StatusUnprocessableEntity, errorMessage{Message: "user with email doesn't exist"})
	}
	return user, nil
}

//func //(w *WalletHandler) createWallet(c echo.Context) error {

func createWallet() (Wallet, error) {

	//respUser, err := findUser(context.Background(), c.Param("email"), w.UserCol)

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	privateKeyBytes := crypto.FromECDSA(privateKey)
	fmt.Println(hexutil.Encode(privateKeyBytes)[2:])
	// 0xfad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	fmt.Println(hexutil.Encode(publicKeyBytes)[4:])
	// 0x049a7df67f79246283fdc93af76d4f8cdd62c4886e8cd870944e817dd0b97934fdd7719d0810951e03418205868a5c1b40b192451367f28e0088dd75e15de40c05

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	fmt.Println(address)
	// 0x96216849c49358B10257cb55b28eA603c874b05E

	hash := sha3.NewLegacyKeccak256()
	hash.Write(publicKeyBytes[1:])
	fmt.Println(hexutil.Encode(hash.Sum(nil)[12:]))
	// 0x96216849c49358b10257cb55b28ea603c874b05e

	return Wallet{
		//ID: ,
		//UserId:     respUser.ID,
		PrivateKey: hexutil.Encode(hash.Sum(nil)[12:]),
		PublicKey:  address,
	}, nil
	//return w, nil
}

func insertWallet(ctx context.Context, wallet Wallet, collection dbiface.CollectionAPI) (interface{}, *echo.HTTPError) {
	var insertedId interface{}
	insertedId, err := collection.InsertOne(ctx, wallet)
	if err != nil {
		log.Errorf("Unable to insert to Database:%v", err)
		return nil,
			echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "unable to insert to database"})
	}
	return insertedId, nil
}

// AuthnUser authenticates a user
func (h *UsersHandler) AuthnUser(c echo.Context) error {
	var user User
	c.Echo().Validator = &userValidator{validator: v}
	if err := c.Bind(&user); err != nil {
		log.Errorf("Unable to bind to user struct.")
		return c.JSON(http.StatusUnprocessableEntity,
			errorMessage{Message: "Unable to parse the request payload."})
	}
	if err := c.Validate(user); err != nil {
		log.Errorf("Unable to validate the requested body.")
		return c.JSON(http.StatusBadRequest,
			errorMessage{Message: "Unable to validate request payload"})
	}
	user, httpError := authenticateUser(context.Background(), user, h.UserCol)
	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}
	token, err := user.createToken()
	if err != nil {
		log.Errorf("Unable to generate the token.")
		return c.JSON(http.StatusInternalServerError,
			errorMessage{Message: "Unable to generate the token"})
	}
	c.Response().Header().Set("x-auth-token", "Bearer "+token)
	return c.JSON(http.StatusOK, User{Email: user.Email})
}
