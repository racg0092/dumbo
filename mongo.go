package dumbo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	opt "go.mongodb.org/mongo-driver/mongo/options"
)

// Mongodb store for session
type MongoStore struct {
	Database       string // database to use
	Collection     string // collection to use
	connection_str string // URI connection string
}

// Generates a new mongo db storage for the session
func NewMongoStore(db string, collection string, connection string) (MongoStore, error) {
	if connection == "" {
		return MongoStore{}, errors.New("connection is required")
	}
	return MongoStore{
		Database:       db,
		Collection:     collection,
		connection_str: connection,
	}, nil
}

func (ms MongoStore) connect() (*mongo.Client, error) {
	if ms.connection_str == "" {
		return nil, errors.New("connection string is </nil>")
	}
	ctx := context.Background()
	client, err := mongo.Connect(ctx, opt.Client().ApplyURI(ms.connection_str))
	if err != nil {
		return nil, err
	}
	return client, nil
}

// Save to store
func (ms MongoStore) Save(sess *Session) error {
	client, err := ms.connect()
	if err != nil {
		return err
	}
	ctx := context.TODO()
	defer client.Disconnect(ctx)

	collection := client.Database(ms.Database).Collection(ms.Collection)

	sr := collection.FindOne(ctx, primitive.D{{"_id", sess.ID}})
	var sessr Session

	err = sr.Decode(&sessr)
	if err == mongo.ErrNoDocuments {
		_, err = collection.InsertOne(ctx, sess)
		if err != nil {
			return err
		}
		return nil
	}

	if err != nil {
		return err
	}

	filter := primitive.D{{"_id", sess.ID}}
	update := bson.M{
		"$set": bson.M{
			"values":  sess.Values,
			"expires": sess.Expires,
		},
	}

	updr, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if updr.MatchedCount <= 0 {
		return errors.New("no doc matched")
	}

	if updr.ModifiedCount <= 0 {
		return errors.New("no doc modified")
	}

	return nil
}

// Delete from store
func (ms MongoStore) Delete(id string) error {
	client, err := ms.connect()
	if err != nil {
		return err
	}
	ctx := context.TODO()
	defer client.Disconnect(ctx)

	collection := client.Database(ms.Database).Collection(ms.Collection)
	filter := primitive.D{{"_id", id}}
	_, err = collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// Read from store
func (ms MongoStore) Read(id string) (*Session, error) {
	client, err := ms.connect()
	if err != nil {
		return nil, err
	}
	ctx := context.TODO()
	defer client.Disconnect(ctx)
	collection := client.Database(ms.Database).Collection(ms.Collection)
	filter := primitive.D{{"_id", id}}
	sr := collection.FindOne(ctx, filter)
	var session Session
	err = sr.Decode(&session)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if expired := session.Expires.Before(now); expired {
		go ms.Delete(id)
		return nil, ErrSessionExpired
	}

	return &session, nil
}
