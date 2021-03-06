package company_verify

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

const MetaName = "leaq-verification"

type CompanyVerify struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	UserID      primitive.ObjectID `bson:"u,omitempty"`
	CompanyID   primitive.ObjectID `bson:"c,omitempty"`
	MetaContent string             `bson:"mc,omitempty"`
	CreatedAt   time.Time          `bson:"ca,omitempty"`
}
