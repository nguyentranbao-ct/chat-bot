package mongodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/sync/errgroup"
)

// keep the baseRepo implementation in sync with IRepository interface
var _ IRepository[IEntity] = (*baseRepo[IEntity])(nil)

type IEntity interface {
	CollectionName() string
	GetUpdates() any
	GetObjectID() models.ObjectID
}

type PaginateWithTotal[E any] struct {
	Total int64
	Data  []E
}

type IRepository[E IEntity] interface {
	Insert(ctx context.Context, entity E, opts ...*options.InsertOneOptions) (string, error)
	InsertMany(ctx context.Context, entities []E, opts ...*options.InsertManyOptions) ([]string, error)
	Find(ctx context.Context, filter bson.M, opts ...*options.FindOptions) ([]E, error)
	FindByID(ctx context.Context, docID string) (*E, error)
	FindOne(ctx context.Context, filter bson.M, opts ...*options.FindOneOptions) (*E, error)
	UpdateOne(ctx context.Context, filter bson.M, entity E, opts ...*options.FindOneAndUpdateOptions) (*E, error)
	UpdateOneByDoc(ctx context.Context, filter E, entity E, opts ...*options.FindOneAndUpdateOptions) (*E, error)
	UpsertOne(ctx context.Context, filter bson.M, entity E, upsertOpts UpsertOpts[E], opts ...*options.FindOneAndUpdateOptions) (*E, error)
	UpdateOneWithUnset(ctx context.Context, filter bson.M, entity E, unset bson.M, opts ...*options.FindOneAndUpdateOptions) (*E, error)
	UpdateMany(ctx context.Context, filter bson.M, data any, opts ...*options.UpdateOptions) error
	UpdateByID(ctx context.Context, id string, entity E) error
	BulkUpdateByIDs(ctx context.Context, docs []E) error
	PartialBulkUpdateByIDs(ctx context.Context, params []PartialBulkUpdateItem) error
	DeleteOne(ctx context.Context, filter bson.M) error
	DeleteOneByDoc(ctx context.Context, doc E) error
	DeleteMany(ctx context.Context, filter bson.M) (int64, error)
	Count(ctx context.Context, filter bson.M, opts ...*options.CountOptions) (int64, error)
	Paginate(ctx context.Context, filter bson.M, limit int64, skip int64, opts ...*options.FindOptions) ([]E, error)
	PaginateWithTotal(ctx context.Context, filter bson.M, limit int64, skip int64, opts ...*options.FindOptions) (*PaginateWithTotal[E], error)
	Iterate(ctx context.Context, filter bson.M, fn func(E) error, opts ...*options.FindOptions) error
}

type baseRepo[E IEntity] struct {
	coll *mongo.Collection
}

func newBaseRepo[E IEntity](dbc *mongo.Database) baseRepo[E] {
	var entity E
	return baseRepo[E]{
		coll: dbc.Collection(entity.CollectionName()),
	}
}

// this is a helper function to get the collection, but only for scripting purposes
func (r *baseRepo[E]) GetCollection() *mongo.Collection {
	return r.coll
}

func (r *baseRepo[E]) Insert(ctx context.Context, entity E, opts ...*options.InsertOneOptions) (string, error) {
	result, err := r.coll.InsertOne(ctx, entity, opts...)
	if err != nil {
		return "", fmt.Errorf("insert one: %w", err)
	}

	oid, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("invalid inserted id: %T %+v", result.InsertedID, result.InsertedID)
	}

	return oid.Hex(), nil
}

func (r *baseRepo[E]) InsertMany(ctx context.Context, entities []E, opts ...*options.InsertManyOptions) ([]string, error) {
	docs := make([]any, 0, len(entities))
	for _, e := range entities {
		docs = append(docs, e)
	}
	result, err := r.coll.InsertMany(ctx, docs, opts...)
	if err != nil {
		return nil, fmt.Errorf("insert many: %w", err)
	}
	ids := make([]string, len(result.InsertedIDs))
	for i, id := range result.InsertedIDs {
		oid, ok := id.(primitive.ObjectID)
		if !ok {
			return nil, fmt.Errorf("invalid inserted id: %T %+v", id, id)
		}
		ids[i] = oid.Hex()
	}

	return ids, nil
}

func (r *baseRepo[E]) Find(ctx context.Context, filter bson.M, opts ...*options.FindOptions) ([]E, error) {
	cursor, err := r.coll.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	var entities []E
	if err := cursor.All(ctx, &entities); err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *baseRepo[E]) FindByID(ctx context.Context, docID string) (*E, error) {
	id := models.ObjectID(docID)
	filter := bson.M{"_id": id}
	entity := new(E)
	err := r.coll.FindOne(ctx, filter).Decode(&entity)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entity, models.ErrNotFound
	}
	if err != nil {
		return entity, err
	}
	return entity, nil
}

func (r *baseRepo[E]) FindOne(ctx context.Context, filter bson.M, opts ...*options.FindOneOptions) (*E, error) {
	var entity E
	err := r.coll.FindOne(ctx, filter, opts...).Decode(&entity)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *baseRepo[E]) UpdateOne(ctx context.Context, filter bson.M, entity E, opts ...*options.FindOneAndUpdateOptions) (*E, error) {
	update := bson.M{
		"$set": entity.GetUpdates(),
	}
	updateOpt := options.
		FindOneAndUpdate().
		SetReturnDocument(options.After)
	opts = append(opts, updateOpt)

	var updatedEntity E
	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts...).Decode(&updatedEntity)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &updatedEntity, nil
}

func (r *baseRepo[E]) UpdateOneByDoc(ctx context.Context, filter E, entity E, opts ...*options.FindOneAndUpdateOptions) (*E, error) {
	update := bson.M{
		"$set": entity.GetUpdates(),
	}
	updateOpt := options.
		FindOneAndUpdate().
		SetReturnDocument(options.After)
	opts = append(opts, updateOpt)

	var updatedEntity E
	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts...).Decode(&updatedEntity)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &updatedEntity, nil
}

func (r *baseRepo[E]) UpdateOneWithUnset(ctx context.Context, filter bson.M, entity E, unset bson.M, opts ...*options.FindOneAndUpdateOptions) (*E, error) {
	update := bson.M{
		"$set":   entity.GetUpdates(),
		"$unset": unset,
	}
	updateOpt := options.
		FindOneAndUpdate().
		SetReturnDocument(options.After)
	opts = append(opts, updateOpt)

	var updatedEntity E
	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts...).Decode(&updatedEntity)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &updatedEntity, nil
}

type UpsertOpts[E IEntity] struct {
	SetOnInsert bson.M
	Unset       bson.M
}

func (r *baseRepo[E]) UpsertOne(ctx context.Context, filter bson.M, entity E, upsertOpts UpsertOpts[E], opts ...*options.FindOneAndUpdateOptions) (*E, error) {
	update := bson.M{
		"$set": entity,
	}
	if upsertOpts.SetOnInsert != nil {
		update["$setOnInsert"] = upsertOpts.SetOnInsert
	}
	if upsertOpts.Unset != nil {
		update["$unset"] = upsertOpts.Unset
	}
	var updatedEntity E
	upsertOpt := options.
		FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	opts = append(opts, upsertOpt)
	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts...).Decode(&updatedEntity)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &updatedEntity, nil
}

func (r *baseRepo[E]) UpdateMany(ctx context.Context, filter bson.M, data any, opts ...*options.UpdateOptions) error {
	_, err := r.coll.UpdateMany(ctx, filter, data, opts...)
	return err
}

func (r *baseRepo[E]) BulkUpdateByIDs(ctx context.Context, docs []E) error {
	models := make([]mongo.WriteModel, 0, len(docs))
	for _, doc := range docs {
		model := mongo.
			NewUpdateOneModel().
			SetFilter(bson.M{
				"_id": doc.GetObjectID(),
			}).
			SetUpdate(bson.M{
				"$set": doc.GetUpdates(),
			})
		models = append(models, model)
	}
	_, err := r.coll.BulkWrite(ctx, models)
	if err != nil {
		return fmt.Errorf("bulk write: %w", err)
	}
	return nil
}

func (r *baseRepo[E]) UpdateByID(ctx context.Context, docID string, entity E) error {
	result, err := r.coll.UpdateOne(ctx, bson.M{"_id": models.ObjectID(docID)}, bson.M{
		"$set": entity.GetUpdates(),
	})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrNotFound
	}

	return nil
}

func (r *baseRepo[E]) DeleteOne(ctx context.Context, filter bson.M) error {
	result, err := r.coll.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *baseRepo[E]) DeleteOneByDoc(ctx context.Context, doc E) error {
	result, err := r.coll.DeleteOne(ctx, doc)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *baseRepo[E]) DeleteMany(ctx context.Context, filter bson.M) (int64, error) {
	result, err := r.coll.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

func (r *baseRepo[E]) Count(ctx context.Context, filter bson.M, opts ...*options.CountOptions) (int64, error) {
	return r.coll.CountDocuments(ctx, filter, opts...)
}

func (r *baseRepo[E]) Paginate(ctx context.Context, filter bson.M, limit int64, skip int64, opts ...*options.FindOptions) ([]E, error) {
	opts = append(opts, options.Find().SetSkip(skip).SetLimit(limit))
	cursor, err := r.coll.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	var entities []E
	if err := cursor.All(ctx, &entities); err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *baseRepo[E]) PaginateWithTotal(ctx context.Context, filter bson.M, limit int64, skip int64, opts ...*options.FindOptions) (*PaginateWithTotal[E], error) {
	group, ctx := errgroup.WithContext(ctx)
	var entities []E
	var total int64
	var err error

	group.Go(func() error {
		opts = append(opts, options.Find().SetSkip(skip).SetLimit(limit))
		cursor, err := r.coll.Find(ctx, filter, opts...)
		if err != nil {
			return fmt.Errorf("find: %w", err)
		}
		if err := cursor.All(ctx, &entities); err != nil {
			return fmt.Errorf("cursor all: %w", err)
		}
		return nil
	})

	group.Go(func() error {
		total, err = r.coll.CountDocuments(ctx, filter)
		if err != nil {
			return fmt.Errorf("count documents: %w", err)
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}

	return &PaginateWithTotal[E]{Total: total, Data: entities}, nil
}

var ErrStop = errors.New("stop")

func (r *baseRepo[E]) Iterate(ctx context.Context, filter bson.M, fn func(E) error, opts ...*options.FindOptions) error {
	cursor, err := r.coll.Find(ctx, filter, opts...)
	if err != nil {
		return err
	}

	for cursor.Next(ctx) {
		var entity E
		if err := cursor.Decode(&entity); err != nil {
			return err
		}

		err := fn(entity)
		if errors.Is(err, ErrStop) {
			return nil
		}
		if err != nil {
			return err
		}
	}

	return nil
}

type PartialBulkUpdateItem struct {
	ID    models.ObjectID
	Set   bson.M
	Unset bson.M
}

func (r *baseRepo[E]) PartialBulkUpdateByIDs(ctx context.Context, params []PartialBulkUpdateItem) error {
	items := []mongo.WriteModel{}
	for _, p := range params {
		if len(p.Set) == 0 && len(p.Unset) == 0 {
			continue
		}

		update := bson.M{}
		if len(p.Set) > 0 {
			update["$set"] = p.Set
		}
		if len(p.Unset) > 0 {
			update["$unset"] = p.Unset
		}
		item := mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": p.ID}).
			SetUpdate(update)
		items = append(items, item)
	}
	_, err := r.coll.BulkWrite(ctx, items)
	if err != nil {
		return fmt.Errorf("bulk write: %w", err)
	}
	return nil
}
