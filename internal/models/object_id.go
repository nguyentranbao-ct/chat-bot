package models

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ObjectID is used to seemlessly convert between string and primitive.ObjectID
//
//nolint:recvcheck // use pointer receiver to match bson.UnmarshalValue
type ObjectID string

func (o ObjectID) MarshalBSONValue() (bsontype.Type, []byte, error) {
	p, err := primitive.ObjectIDFromHex(string(o))
	if err != nil {
		return bson.TypeNull, nil, err
	}
	return bson.MarshalValue(p)
}

func (o *ObjectID) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	var p primitive.ObjectID
	err := bson.UnmarshalValue(t, data, &p)
	if err != nil {
		return err
	}
	*o = ObjectID(p.Hex())
	return nil
}

func (o ObjectID) String() string {
	return string(o)
}
