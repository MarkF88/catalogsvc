package main

import (
	"fmt"
	"os"

	"github.com/globalsign/mgo"
	"github.com/sirupsen/logrus"
	"github.com/hashicorp/vault/api"
//	"encoding/base64"
	//"encoding/json"
)

var (

	// Mongo stores the mongodb connection string information
	mongo *mgo.DialInfo

	db *mgo.Database

	collection *mgo.Collection
)

var res map[string]string

var vaultAddress = os.Getenv("VAULT_ADDR")

var token = os.Getenv("TOKEN")

func getSecret() (string, string) {
	// Path to secret within Vault
	secretName := "secret/data/catalogdb"

	// Start a new connection with the vault
	client, err := api.NewClient(&api.Config{
		Address: vaultAddress,
	})

	// Set Token - This token can be based on the role and permissions set within vault
	client.SetToken(token)

	if err != nil {
		logger.Errorf("Error from getSecret is %s", err.Error())
	}

	// Read the secret
	res, err := client.Logical().Read("secret/data/catalogdb")

	if err !=nil {
		logger.Errorf("Error retrieving Secret from vault %s", err.Error())
	}
	
	vaultData := res.Data["data"]

	// Get the vaules for db username and password
	dbUser := vaultData.(map[string]interface{})["username"].(string)
	dbPass := vaultData.(map[string]interface{})["password"].(string)	
	
	return dbUser, dbPass
}

// ConnectDB accepts name of database and collection as a string
func ConnectDB(dbName string, collectionName string, logger *logrus.Logger) *mgo.Session {

	// Retrieve Username and Password from Hashi Vault
	dbUsername, dbSecret := getSecret()

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
