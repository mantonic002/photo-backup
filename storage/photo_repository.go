package storage

import (
	"context"
	"log"
	"photo-backup/model"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type PhotoDB interface {
	Connect(connectionString, databaseName, collectionName string) error
	Close() error
	SavePhoto(photo model.PhotoDB) error
	SearchPhotos(query string) ([]model.PhotoDB, error)
}

type MongoPhotoDB struct {
	mongoClient      *mongo.Client
	connectionString string
	databaseName     string
	collectionName   string
}

func (db *MongoPhotoDB) Connect(connectionString, databaseName, collectionName string) error {
	var err error
	db.connectionString = connectionString
	db.databaseName = databaseName
	db.collectionName = collectionName

	db.mongoClient, err = mongo.Connect(options.Client().ApplyURI(connectionString))
	if err != nil {
		return err
	}

	err = db.mongoClient.Ping(context.TODO(), nil)
	if err != nil {
		return err
	}

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
	collection := db.mongoClient.Database(db.databaseName).Collection(db.collectionName)
	_, err := collection.InsertOne(context.TODO(), photo)
	if err != nil {
		return err
	}
	log.Println("Photo saved to MongoDB:", photo.FilePath)
	return nil
}

func (db *MongoPhotoDB) SearchPhotos(query string) ([]model.PhotoDB, error) {
	return nil, nil
}
