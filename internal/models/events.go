package models

import (
	"time"
)

type Event struct {
	ID        ObjectID `bson:"_id,omitempty"`
	ProjectID ObjectID `bson:"project_id,omitempty"`
	Name      string   `bson:"name,omitempty"`
	// Payload   map[string]any `bson:"payload,omitempty"`

	CreatedAt time.Time `bson:"created_at,omitempty"`
	UpdatedAt time.Time `bson:"updated_at,omitempty"`
}

func (Event) CollectionName() string {
	return "events"
}

func (n Event) GetObjectID() ObjectID {
	return n.ID
}

func (n Event) GetUpdates() any {
	// update everything except ID and CreatedAt
	// all fields are omitempty, so we don't need to check for empty value
	n.ID = ""
	n.CreatedAt = time.Time{}
	n.UpdatedAt = time.Now()
	return n
}
