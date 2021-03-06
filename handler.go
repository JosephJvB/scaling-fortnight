package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jaf-unwrapped.com/users/clients"
	"jaf-unwrapped.com/users/models"
)

var (
	adminSpotifyId string
	auth           clients.IAuth
	ddb            clients.IDdb
)

func init() {
	log.SetPrefix("LoadUsers:")
	log.SetFlags(0)
	adminSpotifyId = os.Getenv("AdminSpotifyId")
	auth = clients.NewAuth()
	ddb = clients.NewDdb()
}

func HandleLambdaEvent(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Println(request)
	log.Println("request.HTTPMethod:", request.HTTPMethod)
	log.Println("request.Body:", request.Body)
	if request.HTTPMethod == "OPTIONS" {
		return models.NewBasicResponse(200, ""), nil
	}

	authHeader, ok := request.Headers["Authorization"]
	if !ok || authHeader == "" {
		msg := "Invalid request, missing Authorization header"
		return models.NewBasicResponse(400, msg), nil
	}
	var token string
	s := strings.Split(authHeader, " ")
	if len(s) == 2 {
		token = s[1]
	}
	if token == "" {
		msg := "Invalid request, invalid Authorization header"
		return models.NewBasicResponse(400, msg), nil
	}

	claims, err := auth.Decode(token)
	if err != nil {
		msg := "Invalid request, failed to decode bearer token"
		return models.NewBasicResponse(400, msg), nil
	}

	if claims.Data.SpotifyId != adminSpotifyId {
		msg := "Invalid request, Unauthorized user, not joe!"
		return models.NewBasicResponse(400, msg), nil
	}
	// https://stackoverflow.com/questions/36051177/date-now-equivalent-in-go
	now := time.Now().UTC().UnixNano() / 1e6
	if claims.Data.Expires < now {
		msg := "Invalid request, token expired"
		return models.NewBasicResponse(400, msg), nil
	}

	users, err := ddb.GetUsers()
	if err != nil {
		msg := "Failed to get users from ddb " + err.Error()
		return models.NewBasicResponse(400, msg), nil
	}

	nextClaims := models.JWTClaims{
		Data: models.JWTData{
			Expires:   now * 1000,
			SpotifyId: claims.Data.SpotifyId,
		},
	}
	token, err = auth.Encode(nextClaims)
	if err != nil {
		msg := "Failed to encode token " + err.Error()
		return models.NewBasicResponse(500, msg), nil
	}

	return models.NewUserResponse(
		users,
		token,
	), nil
}

func main() {
	lambda.Start(HandleLambdaEvent)
}
