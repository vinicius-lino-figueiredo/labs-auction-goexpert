package auction

import (
	"context"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"log"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type m = map[string]any

// Por algum motivo é int64 e não time.Time
var auctionTimeout int64 = 1200

var auctionVerify = 5 * time.Second

func init() {
	env, ok := os.LookupEnv("AUCTION_TIMEOUT")
	if !ok {
		return
	}
	d, err := time.ParseDuration(env)
	if err != nil {
		panic(err)
	}
	auctionTimeout = int64(d.Seconds())

	env, ok = os.LookupEnv("AUCTION_VERIFY")
	if !ok {
		return
	}
	auctionVerify, err = time.ParseDuration(env)
	if err != nil {
		panic(err)
	}
	if auctionVerify == 0 {
		panic("AUCTION_VERIFY cannot be zero")
	}
}

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
	TTL         int64                           `bson:"ttl"`
}
type AuctionRepository struct {
	sync.Mutex
	Collection *mongo.Collection
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	ar := &AuctionRepository{
		Collection: database.Collection("auctions"),
	}
	ar.startTimeoutWatcher()
	return ar
}

func (ar *AuctionRepository) CreateAuction(ctx context.Context, auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	timestamp := auctionEntity.Timestamp.Unix()
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   timestamp,
		TTL:         timestamp + auctionTimeout,
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	return nil
}

func (ar *AuctionRepository) startTimeoutWatcher() {
	go func() {
		t := time.NewTicker(auctionVerify)
		for range t.C {
			go ar.verifyExpired()
		}
	}()
}

func (ar *AuctionRepository) verifyExpired() {
	// Impedindo chamadas concorrentes
	if !ar.TryLock() {
		return
	}
	defer ar.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err := ar.Collection.UpdateMany(ctx,
		m{"ttl": m{"$lte": time.Now().Unix()}, "status": auction_entity.Active},
		m{"$set": m{"status": auction_entity.Completed}},
	)
	if err != nil {
		log.Println("error updating timed out auctions:", err.Error())
	}
}
