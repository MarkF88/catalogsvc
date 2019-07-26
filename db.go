package main

import (
	"fmt"
	"os"

	"github.com/globalsign/mgo"
	"github.com/sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"encoding/base64"
	"encoding/json"
	//"github.com/buger/jsonparser"
)

var (

	// Mongo stores the mongodb connection string information
	mongo *mgo.DialInfo

	db *mgo.Database

	collection *mgo.Collection
)

var res map[string]interface{}

func getSecret() {
	secretName := "catalogdb"
	region := "us-west-2"

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)

	if err != nil {
		logger.Errorf("Error from getSecret is %s", err.Error())
	}


	//Create a Secrets Manager client
	svc := secretsmanager.New(sess)
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	// In this sample we only handle the specific exceptions for the 'GetSecretValue' API.
	// See https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html

	result, err := svc.GetSecretValue(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
				case secretsmanager.ErrCodeDecryptionFailure:
				// Secrets Manager can't decrypt the protected secret text using the provided KMS key.
				fmt.Println(secretsmanager.ErrCodeDecryptionFailure, aerr.Error())

				case secretsmanager.ErrCodeInternalServiceError:
				// An error occurred on the server side.
				fmt.Println(secretsmanager.ErrCodeInternalServiceError, aerr.Error())

				case secretsmanager.ErrCodeInvalidParameterException:
				// You provided an invalid value for a parameter.
				fmt.Println(secretsmanager.ErrCodeInvalidParameterException, aerr.Error())

				case secretsmanager.ErrCodeInvalidRequestException:
				// You provided a parameter value that is not valid for the current state of the resource.
				fmt.Println(secretsmanager.ErrCodeInvalidRequestException, aerr.Error())

				case secretsmanager.ErrCodeResourceNotFoundException:
				// We can't find the resource that you asked for.
				fmt.Println(secretsmanager.ErrCodeResourceNotFoundException, aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secretString, decodedBinarySecret string
	if result.SecretString != nil {
		secretString = *result.SecretString
	} else {
		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			fmt.Println("Base64 Decode Error:", err)
			return
		}
		decodedBinarySecret = string(decodedBinarySecretBytes[:len])
	}

	fmt.Println("bs", decodedBinarySecret)

	bytes := []byte(secretString)

	json.Unmarshal([]byte(bytes), &res)

	os.Setenv("CATALOG_DB_PASSWORD_AWS", res["password"].(string))
	
	return
}

// ConnectDB accepts name of database and collection as a string
func ConnectDB(dbName string, collectionName string, logger *logrus.Logger) *mgo.Session {

	// Retrieve Secret from AWS Secrets Manager
	getSecret()	

	dbUsername := os.Getenv("CATALOG_DB_USERNAME")
	dbSecret := os.Getenv("CATALOG_DB_PASSWORD_AWS")

	// Get ENV variable or set to default value
	dbIP := GetEnv("CATALOG_DB_HOST", "0.0.0.0")
	dbPort := GetEnv("CATALOG_DB_PORT", "27017")

	mongoDBUrl := fmt.Sprintf("mongodb://%s:%s@%s:%s/?authSource=admin", dbUsername, dbSecret, dbIP, dbPort)

	Session, error := mgo.Dial(mongoDBUrl)

	if error != nil {
		fmt.Printf(error.Error())
		logger.Fatalf(error.Error())
		os.Exit(1)
	}

	db = Session.DB(dbName)

	error = db.Session.Ping()
	if error != nil {
		logger.Errorf("Unable to connect to database %s", dbName)
	}

	collection = db.C(collectionName)

	logger.Info("Connected to database and the collection")

	return Session
}

// CloseDB accepst Session as input to close Connection to the database
func CloseDB(s *mgo.Session, logger *logrus.Logger) {

	defer s.Close()
	logger.Info("Closed connection to db")
}
