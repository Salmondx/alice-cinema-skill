package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

const tableName = "alice-cinema-skill"

// Location contains information about users location
type Location struct {
	UserID     string `json:"userID"`
	InProgress bool   `json:"inProgress"`
	Completed  bool   `json:"completed"`
	Subway     string `json:"subway"`
	City       string `json:"city"`
}

// LocationStorage provides a storage for user location
type LocationStorage interface {
	Get(userID string) (*Location, error)
	Save(userID string, location *Location) error
}

// InMemoryStorage stores info in a map
type InMemoryStorage struct {
	store map[string]*Location
}

func NewStorage() *InMemoryStorage {
	return &InMemoryStorage{make(map[string]*Location)}
}

func (s *InMemoryStorage) Get(userID string) (*Location, error) {
	if loc, ok := s.store[userID]; ok {
		return loc, nil
	}
	return &Location{}, nil
}

func (s *InMemoryStorage) Save(userID string, location *Location) error {
	s.store[userID] = location
	return nil
}

// DynamoStorage stores info in AWS DynamoDB
type DynamoStorage struct {
	client *dynamodb.DynamoDB
}

func NewDynamoStorage() (*DynamoStorage, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-central-1"),
	})

	if err != nil {
		return nil, err
	}

	client := dynamodb.New(sess)
	return &DynamoStorage{client}, nil
}

func (d *DynamoStorage) Get(userID string) (*Location, error) {
	result, err := d.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"userID": {
				S: aws.String(userID),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	var location Location
	err = dynamodbattribute.UnmarshalMap(result.Item, &location)
	if err != nil {
		return nil, err
	}

	// No previous location found
	if location.UserID == "" {
		return &Location{}, nil
	}

	return &location, nil
}

func (d *DynamoStorage) Save(userID string, location *Location) error {
	location.UserID = userID

	av, err := dynamodbattribute.MarshalMap(location)
	if err != nil {
		return err
	}

	_, err = d.client.PutItem(
		&dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      av,
		})
	if err != nil {
		return err
	}

	return nil
}
