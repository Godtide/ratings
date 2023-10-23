
//
//import (
//	"context"
//	"crypto/ecdsa"
//	"fmt"
//
//	_ "github.com/Godtide/rating/config"
//	"github.com/Godtide/rating/dbiface"
//	"github.com/Godtide/rating/handlers"
//	"github.com/ethereum/go-ethereum/common/hexutil"
//	"github.com/ethereum/go-ethereum/crypto"
//	"github.com/labstack/echo/v4"
//	"github.com/labstack/gommon/log"
//	_ "go.mongodb.org/mongo-driver/bson"
//	_ "go.mongodb.org/mongo-driver/mongo"
//	"golang.org/x/crypto/sha3"
//	"net/http"
//)
//
////
//////Wallet describes an user wallet to manage keys
////type Wallet struct {
////	ID         primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
////	UserId     primitive.ObjectID `json:"user_id" bson:"user_string" validate:"required"`
////	PrivateKey string             `json:"private_key" bson:"private_key" validate:"required"`
////	PublicKey  string             `json:"public_key" bson:"public_key" validate:"required"`
////}
//
////WalletHandler handles a user wallet
//type WalletHandler struct {
//	UserCol   dbiface.CollectionAPI
//	WalletCol dbiface.CollectionAPI
//}
//
//func (w *WalletHandler) CreateWallet(c echo.Context) error {
//	finder := &handlers.UsersHandler{Col: w.UserCol}
//
//	respUser, err := finder.FindUser(context.Background(), c.Param("email"), w.UserCol)
//
//	privateKey, err := crypto.GenerateKey()
//	if err != nil {
//		log.Fatal(err)
//	}
//	privateKeyBytes := crypto.FromECDSA(privateKey)
//	fmt.Println(hexutil.Encode(privateKeyBytes)[2:])
//	// 0xfad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19
//
//	publicKey := privateKey.Public()
//	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
//	if !ok {
//		log.Fatal("error casting public key to ECDSA")
//	}
//
//	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
//	fmt.Println(hexutil.Encode(publicKeyBytes)[4:])
//	// 0x049a7df67f79246283fdc93af76d4f8cdd62c4886e8cd870944e817dd0b97934fdd7719d0810951e03418205868a5c1b40b192451367f28e0088dd75e15de40c05
//
//	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
//	fmt.Println(address)
//	// 0x96216849c49358B10257cb55b28eA603c874b05E
//
//	hash := sha3.NewLegacyKeccak256()
//	hash.Write(publicKeyBytes[1:])
//	fmt.Println(hexutil.Encode(hash.Sum(nil)[12:]))
//	// 0x96216849c49358b10257cb55b28ea603c874b05e
//
//	wallet, httpError := insertWallet(context.Background(), Wallet{
//		UserId:     respUser.ID,
//		PrivateKey: hexutil.Encode(hash.Sum(nil)[12:]),
//		PublicKey:  address,
//	}, w.WalletCol)
//	if httpError != nil {
//		return c.JSON(httpError.Code, httpError.Message)
//	}
//	return c.JSON(http.StatusCreated, wallet)
//}
//
////
////func insertWallet(ctx context.Context, wallet Wallet, collection dbiface.CollectionAPI) (interface{}, *echo.HTTPError) {
////	var insertedId interface{}
////	insertedId, err := collection.InsertOne(ctx, wallet)
////	if err != nil {
////		log.Errorf("Unable to insert to Database:%v", err)
////		return nil,
////			echo.NewHTTPError(http.StatusInternalServerError, errorMessage{Message: "unable to insert to database"})
////	}
////	return insertedId, nil
////}
