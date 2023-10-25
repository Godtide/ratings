package handlers

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/Godtide/rating/dbiface"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/sha3"
	"golang.org/x/net/context"
	"net/http"
)

//Wallet describes an user wallet to manage keys
type Wallet struct {
	ID         primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	UserId     primitive.ObjectID `json:"user_id, omitempty" bson:"user_id, omitempty"`
	PrivateKey string             `json:"private_key" bson:"private_key" validate:"required"`
	PublicKey  string             `json:"public_key" bson:"public_key" validate:"required"`
}

type WalletHandler struct {
	WalletCol dbiface.CollectionAPI
}

func createUserWallet(ctx context.Context, userId primitive.ObjectID, collection dbiface.CollectionAPI) (interface{}, *echo.HTTPError) {
	partWallet, err := createWallet()
	if err != nil {
		log.Errorf("Unable to create wallet :%+v", err)
		return partWallet,
			echo.NewHTTPError(http.StatusBadRequest, errorMessage{Message: "error generating public and private keys public address "})
	}

	fullWallet, httpError := insertWallet(ctx, Wallet{
		UserId:     userId,
		PrivateKey: partWallet.PrivateKey,
		PublicKey:  partWallet.PublicKey,
	}, collection)

	if httpError != nil {
		return partWallet,
			echo.NewHTTPError(http.StatusBadRequest, errorMessage{Message: "error creating wallet"})
	}
	return fullWallet, nil
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

func createWallet() (Wallet, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	hash := sha3.NewLegacyKeccak256()
	fmt.Println(hexutil.Encode(hash.Sum(nil)[12:]))

	return Wallet{
		PrivateKey: hexutil.Encode(hash.Sum(nil)[12:]),
		PublicKey:  address,
	}, nil
}
