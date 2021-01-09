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

	_, err := db.Collection(users).Indexes().CreateMany(ctx, []m.IndexModel{{
		Keys: bson.M{
			"v": 1,
		},
		Options: options.Index().
			SetUnique(true).
			SetPartialFilterExpression(bson.M{
				"v": bson.M{
					"$exists": true,
				},
			}),
	}, {
		Keys: bson.M{
			"y": 1,
		},
		Options: options.Index().
			SetUnique(true).
			SetPartialFilterExpression(bson.M{
				"y": bson.M{
					"$exists": true,
				},
			}),
	}, {
		Keys: bson.M{
			"t": 1,
		},
	}})
	logger.Must(err)

	_, err = db.Collection(roles).Indexes().CreateOne(ctx, m.IndexModel{
		Keys: bson.D{{
			Key:   "c",
			Value: 1,
		}, {
			Key:   "u",
			Value: 1,
		}, {
			Key:   "g",
			Value: 1,
		}},
		Options: options.Index().SetUnique(true),
	})
	logger.Must(err)

	const hours24InSeconds = 86400
	_, err = db.Collection(companyVerifyPending).Indexes().CreateMany(ctx, []m.IndexModel{{
		Keys: bson.D{{
			Key:   "c",
			Value: 1,
		}, {
			Key:   "u",
			Value: 1,
		}},
		Options: options.Index().SetUnique(true),
	}, {
		Keys: bson.M{
			"ca": 1,
		},
		Options: options.Index().SetExpireAfterSeconds(hours24InSeconds),
	}})
	logger.Must(err)

	_, err = db.Collection(companyVerifySuccess).Indexes().CreateOne(ctx, m.IndexModel{
		Keys: bson.M{
			"c": 1,
		},
		Options: options.Index().SetUnique(true),
	})
	logger.Must(err)
}
