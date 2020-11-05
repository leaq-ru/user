package role

import "go.mongodb.org/mongo-driver/bson/primitive"

type Grant uint8

const (
	None Grant = iota
	Owner
	Admin
)

type Role struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    primitive.ObjectID `bson:"u,omitempty"`
	CompanyID primitive.ObjectID `bson:"c,omitempty"`
	Grant     Grant              `bson:"g,omitempty"`
}
