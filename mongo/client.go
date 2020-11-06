package mongo

import (
	"context"
	"github.com/nnqq/scr-user/config"
	"github.com/nnqq/scr-user/logger"
	m "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"time"
)

var (
	Client           *m.Client
	Users            *m.Collection
	Roles            *m.Collection
	OrgVerifyPending *m.Collection
	OrgVerifySuccess *m.Collection
)

const (
	users            = "users"
	roles            = "roles"
	orgVerifyPending = "org_verify_pending"
	orgVerifySuccess = "org_verify_success"
)

func init() {
	const timeout = 10
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	client, err := m.Connect(ctx, options.Client().
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

	Client = db.Client()
	Users = db.Collection(users)
	Roles = db.Collection(roles)
	OrgVerifyPending = db.Collection(orgVerifyPending)
	OrgVerifySuccess = db.Collection(orgVerifySuccess)
}
