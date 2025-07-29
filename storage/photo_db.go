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
	Connect(connectionString, databaseName, collectionName string) error
	Close() error
	SavePhoto(photo model.PhotoDB) error
	GetPhoto(id string) (*model.PhotoDB, error)
	SearchPhotosByLocation(long float64, lat float64, dist int) (*[]model.PhotoDB, error)
}

type MongoPhotoDB struct {
	mongoClient      *mongo.Client
	collection       *mongo.Collection
	connectionString string
	databaseName     string
	collectionName   string
}

func (db *MongoPhotoDB) Connect(connectionString, databaseName, collectionName string) error {
	var err error
	db.connectionString = connectionString
	db.databaseName = databaseName
	db.collectionName = collectionName

	db.mongoClient, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(connectionString))
	if err != nil {
		return err
	}

	err = db.mongoClient.Ping(context.TODO(), nil)
	if err != nil {
		return err
	}

	db.collection = db.mongoClient.Database(db.databaseName).Collection(db.collectionName)

	log.Println("Connected to MongoDB")
	return nil
}

func (db *MongoPhotoDB) Close() error {
	if db.mongoClient != nil {
		err := db.mongoClient.Disconnect(context.TODO())
		if err != nil {
			return err
		}
		log.Println("Disconnected from MongoDB")
	}
	return nil
}

func (db *MongoPhotoDB) SavePhoto(photo model.PhotoDB) error {
	_, err := db.collection.InsertOne(context.TODO(), photo)
	if err != nil {
		return err
	}
	log.Println("Photo saved to MongoDB:", photo.FilePath)
	return nil
}

func (db *MongoPhotoDB) GetPhoto(id string) (*model.PhotoDB, error) {
	var photo model.PhotoDB

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	filter := bson.D{{Key: "_id", Value: oid}}
	err = db.collection.FindOne(context.TODO(), filter).Decode(&photo)

	if err != nil {
		log.Printf("Error getting photo info from MongoDB: %v", err)
		return nil, err
	}

	return &photo, nil
}

func (db *MongoPhotoDB) SearchPhotosByLocation(long float64, lat float64, dist int) (*[]model.PhotoDB, error) {
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

	output, err := db.collection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	if err = output.All(context.TODO(), &photos); err != nil {
		return nil, err
	}

	for _, photo := range photos {
		log.Println(photo)
	}

	return &photos, nil
}
