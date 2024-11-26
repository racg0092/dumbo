package dumbo

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	opt "go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStore struct {
	Database       string
	Collection     string
	connection_str string
}

func (ms *MongoStore) SetConnectionString(cs string) {
	ms.connection_str = cs
}

func (ms MongoStore) connect() (*mongo.Client, error) {
	if ms.connection_str == "" {
		return nil, errors.New("connection string is </nil>")
	}
	ctx := context.TODO()
	client, err := mongo.Connect(ctx, opt.Client().ApplyURI(ms.connection_str))
	if err != nil {
		return nil, err
	}
	return client, nil
}

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

	sessr.Values = sess.Values

	filter := primitive.D{{"_id", sess.ID}}
	update := primitive.D{
		{"$set", primitive.D{
			{"values", sess.Values},
		}},
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
	return &session, nil
}
