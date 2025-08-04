package storage

import (
	"context"
	"log"
	"photo-backup/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PhotoDB interface {
	Connect(ctx context.Context, connectionString, databaseName, collectionName string) error
	Close(ctx context.Context) error
	SavePhoto(ctx context.Context, photo model.PhotoDB) error
	GetPhoto(ctx context.Context, id string) (*model.PhotoDB, error)
	GetPhotos(ctx context.Context, lastIdString string, limit int64) ([]model.PhotoDB, error)
	SearchPhotosByLocation(ctx context.Context, long float64, lat float64, dist int) ([]model.PhotoDB, error)
}

type MongoPhotoDB struct {
	mongoClient      *mongo.Client
	collection       *mongo.Collection
	connectionString string
	databaseName     string
	collectionName   string
}

func (db *MongoPhotoDB) Connect(ctx context.Context, connectionString, databaseName, collectionName string) error {
	var err error
	db.connectionString = connectionString
	db.databaseName = databaseName
	db.collectionName = collectionName

	db.mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(connectionString))
	if err != nil {
		return err
	}

	err = db.mongoClient.Ping(ctx, nil)
	if err != nil {
		return err
	}

	db.collection = db.mongoClient.Database(db.databaseName).Collection(db.collectionName)

	log.Println("Connected to MongoDB")
	return nil
}

func (db *MongoPhotoDB) Close(ctx context.Context) error {
	if db.mongoClient != nil {
		err := db.mongoClient.Disconnect(ctx)
		if err != nil {
			return err
		}
		log.Println("Disconnected from MongoDB")
	}
	return nil
}

func (db *MongoPhotoDB) SavePhoto(ctx context.Context, photo model.PhotoDB) error {
	_, err := db.collection.InsertOne(ctx, photo)
	if err != nil {
		return err
	}
	log.Println("Photo saved to MongoDB:", photo.FilePath)
	return nil
}

func (db *MongoPhotoDB) GetPhoto(ctx context.Context, id string) (*model.PhotoDB, error) {
	var photo model.PhotoDB

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	filter := bson.D{{Key: "_id", Value: oid}}
	err = db.collection.FindOne(ctx, filter).Decode(&photo)

	if err != nil {
		log.Printf("Error getting photo info from MongoDB: %v", err)
		return nil, err
	}

	return &photo, nil
}

func (db *MongoPhotoDB) GetPhotos(ctx context.Context, lastIdString string, limit int64) ([]model.PhotoDB, error) {
	var photos []model.PhotoDB

	filter := bson.M{}
	if lastIdString != "" {
		lastId, err := primitive.ObjectIDFromHex(lastIdString)
		if err != nil {
			return nil, err
		}
		filter = bson.M{"_id": bson.M{"$gt": lastId}}
	}

	opts := options.Find().SetLimit(limit).SetSort(bson.M{"_id": 1})
	output, err := db.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	if err = output.All(ctx, &photos); err != nil {
		return nil, err
	}

	for _, photo := range photos {
		log.Println(photo)
	}

	return photos, nil
}

func (db *MongoPhotoDB) SearchPhotosByLocation(ctx context.Context, long float64, lat float64, dist int) ([]model.PhotoDB, error) {
	var photos []model.PhotoDB

	var geoPoint = model.GeoPoint{
		Type:        "Point",
		Coordinates: []float64{long, lat},
	}

	filter := bson.D{
		{Key: "lonlat", Value: bson.D{
			{Key: "$near", Value: bson.D{
				{Key: "$geometry", Value: geoPoint},
				{Key: "$maxDistance", Value: dist},
			}},
		}},
	}

	output, err := db.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	if err = output.All(ctx, &photos); err != nil {
		return nil, err
	}

	for _, photo := range photos {
		log.Println(photo)
	}

	return photos, nil
}
