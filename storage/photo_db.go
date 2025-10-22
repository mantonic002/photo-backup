package storage

import (
	"context"
	"math"
	"photo-backup/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type PhotoDB interface {
	Connect(ctx context.Context, logger *zap.Logger, connectionString, databaseName, collectionName string) error
	Close(ctx context.Context) error
	SavePhoto(ctx context.Context, photo model.PhotoDB) (*model.PhotoDB, error)
	DeletePhoto(ctx context.Context, id string) (*model.PhotoDB, error)
	GetPhoto(ctx context.Context, id string) (*model.PhotoDB, error)
	GetPhotos(ctx context.Context, lastIdString string, limit int64) ([]model.PhotoDB, error)
	SearchPhotosByLocation(ctx context.Context,  lastIdString string, limit int64, latMin float64, latMax float64,  longMin float64, longMax float64) ([]model.PhotoDB, error)
}

type MongoPhotoDB struct {
	mongoClient      *mongo.Client
	collection       *mongo.Collection
	connectionString string
	databaseName     string
	collectionName   string
	Log              *zap.Logger
}

func (db *MongoPhotoDB) Connect(ctx context.Context, logger *zap.Logger, connectionString, databaseName, collectionName string) error {
	var err error
	db.connectionString = connectionString
	db.databaseName = databaseName
	db.collectionName = collectionName

	db.Log = logger

	db.mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(connectionString))
	if err != nil {
		db.Log.Error("failed to connect to MongoDB", zap.Error(err), zap.String("connection_string", connectionString))
		return err
	}

	err = db.mongoClient.Ping(ctx, nil)
	if err != nil {
		db.Log.Error("failed to ping MongoDB", zap.Error(err))
		return err
	}

	db.collection = db.mongoClient.Database(db.databaseName).Collection(db.collectionName)

	db.Log.Info("connected to MongoDB", zap.String("database", databaseName), zap.String("collection", collectionName))
	return nil
}

func (db *MongoPhotoDB) Close(ctx context.Context) error {
	if db.mongoClient != nil {
		err := db.mongoClient.Disconnect(ctx)
		if err != nil {
			db.Log.Error("failed to disconnect from MongoDB", zap.Error(err))
			return err
		}
		db.Log.Info("disconnected from MongoDB")
	}
	return nil
}

func (db *MongoPhotoDB) SavePhoto(ctx context.Context, photo model.PhotoDB) (*model.PhotoDB, error) {
	savedPhoto, err := db.collection.InsertOne(ctx, photo)
	if err != nil {
		db.Log.Error("failed to save photo to MongoDB", zap.Error(err), zap.String("file_path", photo.FilePath))
		return nil, err
	}
	oid, ok := savedPhoto.InsertedID.(primitive.ObjectID)
	if !ok {
		db.Log.Error("invalid ObjectID returned from MongoDB insert", zap.Any("inserted_id", savedPhoto.InsertedID))
		return nil, mongo.ErrInvalidIndexValue
	}
	photo.ID = oid
	db.Log.Info("photo saved to MongoDB", zap.String("file_path", photo.FilePath), zap.String("photo_id", oid.Hex()))
	return &photo, nil
}

func (db *MongoPhotoDB) DeletePhoto(ctx context.Context, id string) (*model.PhotoDB, error) {
	var photo model.PhotoDB

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		db.Log.Error("invalid photo ID format", zap.Error(err), zap.String("id", id))
		return nil, err
	}

	filter := bson.D{{Key: "_id", Value: oid}}
	err = db.collection.FindOneAndDelete(ctx, filter).Decode(&photo)
	if err != nil {
		db.Log.Error("failed to delete photo from MongoDB", zap.Error(err), zap.String("photo_id", id))
		return nil, err
	}
	db.Log.Info("photo deleted from MongoDB", zap.String("photo_id", id))
	return &photo, nil
}

func (db *MongoPhotoDB) GetPhoto(ctx context.Context, id string) (*model.PhotoDB, error) {
	var photo model.PhotoDB

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		db.Log.Error("invalid photo ID format", zap.Error(err), zap.String("id", id))
		return nil, err
	}

	filter := bson.D{{Key: "_id", Value: oid}}
	err = db.collection.FindOne(ctx, filter).Decode(&photo)
	if err != nil {
		db.Log.Info("failed to get photo from MongoDB", zap.Error(err), zap.String("id", id))
		return nil, err
	}

	db.Log.Info("retrieved photo from MongoDB", zap.String("id", id), zap.String("file_path", photo.FilePath))
	return &photo, nil
}

func (db *MongoPhotoDB) GetPhotos(ctx context.Context, lastIdString string, limit int64) ([]model.PhotoDB, error) {
	var photos []model.PhotoDB

	filter := bson.M{}
	if lastIdString != "" {
		lastId, err := primitive.ObjectIDFromHex(lastIdString)
		if err != nil {
			db.Log.Info("invalid last ID format", zap.Error(err), zap.String("last_id", lastIdString))
			return nil, err
		}
		filter = bson.M{"_id": bson.M{"$lt": lastId}}
	}

	opts := options.Find().SetLimit(limit).SetSort(bson.M{"_id": -1})
	output, err := db.collection.Find(ctx, filter, opts)
	if err != nil {
		db.Log.Error("failed to query photos from MongoDB", zap.Error(err), zap.Int64("limit", limit))
		return nil, err
	}
	if err = output.All(ctx, &photos); err != nil {
		db.Log.Error("failed to decode photos from MongoDB", zap.Error(err))
		return nil, err
	}

	db.Log.Info("retrieved photos from MongoDB", zap.Int("count", len(photos)))
	return photos, nil
}

func (db *MongoPhotoDB) SearchPhotosByLocation(ctx context.Context,  lastIdString string, limit int64, latMin float64, latMax float64,  longMin float64, longMax float64) ([]model.PhotoDB, error) {
	var photos []model.PhotoDB

	// Ensure proper order of coordinates
    minLong := math.Min(longMin, longMax)
    maxLong := math.Max(longMin, longMax)
    minLat := math.Min(latMin, latMax)
    maxLat := math.Max(latMin, latMax)

    polygon := bson.A{
        bson.A{minLong, minLat}, // bottom-left
        bson.A{minLong, maxLat}, // top-left
        bson.A{maxLong, maxLat}, // top-right
        bson.A{maxLong, minLat}, // bottom-right
        bson.A{minLong, minLat}, // close the polygon
    }

    filter := bson.M{
        "lonlat": bson.M{
            "$geoWithin": bson.M{
                "$geometry": bson.M{
                    "type":        "Polygon",
                    "coordinates": bson.A{polygon},
                },
            },
        },
    }

	if lastIdString != "" {
		lastId, err := primitive.ObjectIDFromHex(lastIdString)
		if err != nil {
			db.Log.Info("invalid last ID format", zap.Error(err), zap.String("last_id", lastIdString))
			return nil, err
		}
		filter["_id"] = bson.M{"$lt": lastId}
	}

	opts := options.Find().SetLimit(limit).SetSort(bson.M{"_id": -1})
	output, err := db.collection.Find(ctx, filter, opts)
	if err != nil {
		db.Log.Error("failed to search photos by location", zap.Error(err), zap.Float64("latMin", latMin), zap.Float64("latMax", latMax), zap.Float64("longMin", longMin), zap.Float64("longMax", longMax))
		return nil, err
	}
	if err = output.All(ctx, &photos); err != nil {
		db.Log.Error("failed to decode photos from location search", zap.Error(err))
		return nil, err
	}

	db.Log.Info("retrieved photos by location", zap.Int("count", len(photos)), zap.Float64("latMin", latMin), zap.Float64("latMax", latMax), zap.Float64("longMin", longMin), zap.Float64("longMax", longMax))
	return photos, nil
}
