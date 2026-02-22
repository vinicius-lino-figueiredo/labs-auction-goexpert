package auction

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

type CreateAuctionSuite struct {
	suite.Suite
}

func TestCreateAuctionSuite(t *testing.T) {
	suite.Run(t, new(CreateAuctionSuite))
}

func (s *CreateAuctionSuite) TestVerifyExpiredSendsUpdateCommand() {
	mt := mtest.New(s.T(), mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("MongoMock", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{{Key: "ok", Value: 1}, {Key: "nModified", Value: 1}})

		ar := &AuctionRepository{Collection: mt.Coll}
		ar.verifyExpired()

		events := mt.GetAllStartedEvents()
		s.Len(events, 1)
		s.Equal("update", events[0].CommandName)
	})
}

func (s *CreateAuctionSuite) TestWatcherFiresAutomatically() {
	mt := mtest.New(s.T(), mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("MongoMock", func(mt *mtest.T) {
		for i := 0; i < 10; i++ {
			mt.AddMockResponses(bson.D{{Key: "ok", Value: 1}, {Key: "nModified", Value: 1}})
		}

		auctionVerify = 50 * time.Millisecond

		ar := &AuctionRepository{Collection: mt.Coll}
		ar.startTimeoutWatcher()

		time.Sleep(200 * time.Millisecond)

		events := mt.GetAllStartedEvents()
		s.GreaterOrEqual(len(events), 2)
	})
}
