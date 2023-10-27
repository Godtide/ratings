package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Godtide/rating/config"
	"github.com/Godtide/rating/handlers"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/labstack/gommon/random"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	//CorrelationID is a request id unique to the request being made
	CorrelationID = "X-Correlation-ID"
)

var (
	c             *mongo.Client
	db            *mongo.Database
	prodCol       *mongo.Collection
	rewardCol     *mongo.Collection
	usersCol      *mongo.Collection
	userRewardCol *mongo.Collection
	walletCol     *mongo.Collection
	cfg           config.Properties
)

func init() {
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("Configuration cannot be read : %v", err)
	}
	ctx := context.Background()
	connectURI := fmt.Sprintf("mongodb://%s:%s", cfg.DBHost, cfg.DBPort)
	c, err := mongo.Connect(ctx, options.Client().ApplyURI(connectURI))
	if err != nil {
		log.Fatalf("Unable to connect to database : %v", err)
	}
	db = c.Database(cfg.DBName)
	prodCol = db.Collection(cfg.ProductCollection)
	usersCol = db.Collection(cfg.UsersCollection)
	walletCol = db.Collection(cfg.WalletCollection)
	userRewardCol = db.Collection(cfg.UsersRewardCollection)
	rewardCol = db.Collection(cfg.RewardCollection)

	isUserIndexUnique := true
	indexModel := mongo.IndexModel{
		Keys: bson.M{"username": 1},
		Options: &options.IndexOptions{
			Unique: &isUserIndexUnique,
		},
	}
	_, err = usersCol.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Fatalf("Unable to create an index : %+v", err)
	}
}

func addCorrelationID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// generate correlation id
		id := c.Request().Header.Get(CorrelationID)
		var newID string
		if id == "" {
			//generate a random number
			newID = random.String(12)
		} else {
			newID = id
		}
		c.Request().Header.Set(CorrelationID, newID)
		c.Response().Header().Set(CorrelationID, newID)
		return next(c)
	}
}

func main() {
	e := echo.New()
	e.Logger.SetLevel(log.DEBUG)
	e.Pre(middleware.RemoveTrailingSlash())
	e.Pre(addCorrelationID)

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `${time_rfc3339_nano} ${remote_ip} ${header:X-Correlation-ID} ${host} ${method} ${uri} ${user_agent} ` +
			`${status} ${error} ${latency_human}` + "\n",
	}))

	uh := &handlers.UsersHandler{UserCol: usersCol, WalletCol: walletCol}
	us := &handlers.UserRewardHandler{
		UserRewardCol: userRewardCol,
	    RewardCol: rewardCol,
	    WalletCol: walletCol,
		Wallet: handlers.Wallet{PrivateKey: cfg.MasterPrivateKey, PublicKey: cfg.MasterPublicKey},
		Apikey : cfg.ApiKey,
	    ContractAdrress : cfg.ContractAdrress,
	}
	ar := &handlers.RewardHandler{UserRewardCol: userRewardCol, RewardCol: rewardCol}

	e.POST("/users", uh.CreateUser)
	e.POST("/admin/reward", ar.CreateRewards)
	e.POST("/reward/create", us.CreateUserRewards)
	e.POST("/reward/claim", us.ClaimReward)
	e.GET("/rewards", h.GetRewards)

	e.Logger.Infof("Listening on %s:%s", cfg.Host, cfg.Port)
	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)))

}
