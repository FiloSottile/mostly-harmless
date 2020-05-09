package main

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	res, _ := json.MarshalIndent(request, "", "\t")
	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(res),
	}, nil
}

func main() {
	lambda.Start(handler)
}
