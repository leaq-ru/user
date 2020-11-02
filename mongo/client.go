package mongo

import (
	"context"
	"github.com/nnqq/scr-user/config"
	"github.com/nnqq/scr-user/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"time"
)

var Users *mongo.Collection

const users = "users"

func init() {
	const timeout = 10
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().
		SetWriteConcern(writeconcern.New(
			writeconcern.WMajority(),
			writeconcern.J(true),
		)).
		SetReadConcern(readconcern.Majority()).
		SetReadPreference(readpref.SecondaryPreferred()).
		ApplyURI(config.Env.MongoDB.URL))
	logger.Must(err)

	err = client.Ping(ctx, nil)
	logger.Must(err)

	db := client.Database(config.ServiceName)
	createIndex(db)

	Users = db.Collection(users)
}
