package user

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	VkID      uint32             `bson:"v,omitempty"`
	Token     string             `bson:"t,omitempty"`
	FirstName string             `bson:"f,omitempty"`
	LastName  string             `bson:"l,omitempty"`
	Photo     string             `bson:"p,omitempty"`
	PhotoRec  string             `bson:"pr,omitempty"`
}