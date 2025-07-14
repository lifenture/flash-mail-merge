package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"com/lifenture/flash-mail-merge/internal/docx"
	"com/lifenture/flash-mail-merge/internal/fields"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var body map[string]string
	headers := map[string]string{"Content-Type": "application/json"}

	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		log.Printf("failed to unmarshal request body: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    headers,
			Body:       `{"error": "Invalid input"}`,
		}, nil
	}

	encodedDocx, ok := body["docx"]
	if !ok {
		log.Println("'docx' not found in body")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    headers,
			Body:       `{"error": "'docx' key missing"}`,
		}, nil
	}

	docxBytes, err := base64.StdEncoding.DecodeString(encodedDocx)
	if err != nil {
		log.Printf("failed to decode base64 string: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    headers,
			Body:       `{"error": "Failed to decode base64 input"}`,
		}, nil
	}

	documentXML, err := docx.ReadDocumentXML(docxBytes)
	if err != nil {
		log.Printf("failed to read DOCX document: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"error": "Failed to process document"}`,
		}, nil
	}

	extractedFields, err := fields.Extract(documentXML)
	if err != nil {
		log.Printf("failed to extract fields: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"error": "Failed to extract fields"}`,
		}, nil
	}

	for _, name := range extractedFields {
		log.Printf("mergeField=%s", name)
	}

	response := map[string]interface{}{
		"fields": extractedFields,
		"count":  len(extractedFields),
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Printf("failed to marshal response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       `{"error": "Failed to create response"}`,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(responseBody),
	}, nil
}

func main() {
	lambda.Start(handler)
}

