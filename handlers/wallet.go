package handlers

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/Godtide/rating/dbiface"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/sha3"
	"golang.org/x/net/context"
	"math/big"
	"net/http"
)

//Wallet describes a user wallet to manage keys
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

//find user wallets

func findWallet(ctx context.Context, userId string, collection dbiface.CollectionAPI) (Wallet, *echo.HTTPError) {
	var wallet Wallet
	docID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		log.Errorf("Unable to convert to Object ID : %v", err)
		return wallet,
			echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "unable to convert to ObjectID"})
	}
	res := collection.FindOne(ctx, bson.M{"user_id": docID})
	err = res.Decode(&wallet)
	if err != nil {
		log.Errorf("Unable to find the wallet : %v", err)
		return wallet,
			echo.NewHTTPError(http.StatusNotFound, errorMessage{Message: "unable to find the wallet"})
	}
	return wallet, nil
}

//GetWallet gets a single wallet by userId
func (h *WalletHandler) GetWallet(c echo.Context) error {
	wallet, httpError := findWallet(context.Background(), c.Param("id"), h.WalletCol)
	if httpError != nil {
		return c.JSON(httpError.Code, httpError.Message)
	}
	return c.JSON(http.StatusOK, wallet)
}

func transferRewards(wallet Wallet, userPublicKey string, rewardAmount string, key string, contractAddress string) (string, error) {

	var provider string = "https://optimism-mainnet.infura.io/v3/"

	client, err := ethclient.Dial(provider + key)
	if err != nil {
		log.Fatal(err)
	}

	privateKey, err := crypto.HexToECDSA(wallet.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}
	fromAddress := common.HexToAddress(wallet.PublicKey)

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal(err)
	}

	value := big.NewInt(0) // in wei (0 eth)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	toAddress := common.HexToAddress(userPublicKey)
	tokenAddress := common.HexToAddress(contractAddress)

	transferFnSignature := []byte("transfer(address,uint256)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(transferFnSignature)
	methodID := hash.Sum(nil)[:4]
	fmt.Println(hexutil.Encode(methodID)) 

	paddedAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	fmt.Println(hexutil.Encode(paddedAddress))

	amount := new(big.Int)
	amount.SetString(rewardAmount+"e18", 10) 
	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)
	fmt.Println(hexutil.Encode(paddedAmount))

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)

	gasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{
		To:   &toAddress,
		Data: data,
	})
	if err != nil {
		log.Fatal(err)
	}

	tx := types.NewTransaction(nonce, tokenAddress, value, gasLimit, gasPrice, data)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("tx sent: %s", signedTx.Hash().Hex())

	return signedTx.Hash().Hex(), nil

}
