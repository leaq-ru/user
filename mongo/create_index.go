package mongo

import (
	"context"
	"github.com/nnqq/scr-user/logger"
	"go.mongodb.org/mongo-driver/bson"
	m "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func createIndex(db *m.Database) {
	ctx := context.Background()

	_, err := db.Collection(users).Indexes().CreateOne(ctx, m.IndexModel{
		Keys: bson.M{
			"v": 1,
		},
		Options: options.Index().SetUnique(true),
	})
	logger.Must(err)

	_, err = db.Collection(roles).Indexes().CreateOne(ctx, m.IndexModel{
		Keys: bson.D{{
			Key:   "c",
			Value: 1,
		}, {
			Key:   "u",
			Value: 1,
		}},
		Options: options.Index().SetUnique(true),
	})
	logger.Must(err)

	_, err = db.Collection(companyOwnTokens).Indexes().CreateOne(ctx, m.IndexModel{
		Keys: bson.M{
			"c": 1,
		},
		Options: options.Index().SetUnique(true),
	})
	logger.Must(err)
}
