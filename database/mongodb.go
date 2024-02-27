package database

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Feedback struct {
	UserID   int64  `json:"user_id"`
	FolkTale string `json:"folk_tale"`
	Message  string `json:"message"`
}

type Database struct {
	client         *mongo.Client
	feedbackCollec *mongo.Collection
}

func NewDatabase(connectionString, dbName, feedbackColName string) (*Database, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(connectionString))
	if err != nil {
		return nil, err
	}

	err = client.Connect(context.Background())
	if err != nil {
		return nil, err
	}

	db := client.Database(dbName)
	feedbackCol := db.Collection(feedbackColName)

	return &Database{
		client:         client,
		feedbackCollec: feedbackCol,
	}, nil
}

func (db *Database) Close() {
	if db.client != nil {
		err := db.client.Disconnect(context.Background())
		if err != nil {
			log.Println("Error disconnecting from MongoDB:", err)
		}
	}
}

func (db *Database) SaveFeedback(userID int64, folkTale string, message string) error {
	feedback := Feedback{
		UserID:   userID,
		FolkTale: folkTale,
		Message:  message,
	}

	_, err := db.feedbackCollec.InsertOne(context.Background(), feedback)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) GetFeedbacksByFolkTale(folkTale string) ([]Feedback, error) {
	cursor, err := db.feedbackCollec.Find(context.Background(), bson.M{"folktale": folkTale})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var feedbacks []Feedback
	for cursor.Next(context.Background()) {
		var feedback Feedback
		if err := cursor.Decode(&feedback); err != nil {
			return nil, err
		}
		feedbacks = append(feedbacks, feedback)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	log.Println(feedbacks)

	return feedbacks, nil
}
