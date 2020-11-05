package company_own_tokens

import "go.mongodb.org/mongo-driver/bson/primitive"

const MetaName = "leaq-verification"

type CompanyOwnTokens struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	CompanyID   primitive.ObjectID `bson:"c,omitempty"`
	MetaName    string             `bson:"mn,omitempty"`
	MetaContent string             `bson:"mc,omitempty"`
}
