package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/textract"
	"github.com/aws/aws-sdk-go-v2/service/textract/types"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}

	client := sqs.NewFromConfig(cfg)

	queueUrl := os.Getenv("COPILOT_QUEUE_URI")

	input := &sqs.ReceiveMessageInput{
		QueueUrl:            &queueUrl,
		MaxNumberOfMessages: 1,
		MessageAttributeNames: []string{
			"All",
		},
		WaitTimeSeconds: 10,
	}

	for {
		response, err := client.ReceiveMessage(context.TODO(), input)
		if err != nil {
			log.Println(err)
			continue
		}

		for _, msg := range response.Messages {
			log.Println("Received a message")
			process(&cfg, msg.Body)

			input := &sqs.DeleteMessageInput{
				QueueUrl:      &queueUrl,
				ReceiptHandle: msg.ReceiptHandle,
			}
			_, err := client.DeleteMessage(context.TODO(), input)
			if err != nil {
				log.Fatalf("Error deleting processed message: %v", err)
			}
		}
	}

}

type IncomingMessage struct {
	MessageId string
	Message   string
}

type StatusMessage struct {
	ImageID string
	Status  string
}

var topic = os.Getenv("IMAGE_STATUS_SNS_TOPIC")
var receiptUploadBucket = os.Getenv("RECEIPT_UPLOAD_BUCKET")

func process(cfg *aws.Config, message *string) {
	var parsedMessage IncomingMessage
	json.Unmarshal([]byte(*message), &parsedMessage)

	log.Printf("Processing %s", parsedMessage.Message)

	snsClient := sns.NewFromConfig(*cfg)
	textractClient := textract.NewFromConfig(*cfg)

	// regex the id out of the S3 path
	// the key is "uploads/{id}/image.png"
	re := regexp.MustCompile(`uploads/(.*?)/image.png$`)
	match := re.FindStringSubmatch(parsedMessage.Message)
	log.Printf("Matched: %v", match)
	key := match[0]
	imageId := match[1]

	sendStatus(snsClient, imageId, "Started")

	// Send the S3 path to textract to detect text in the image
	input := &textract.DetectDocumentTextInput{
		Document: &types.Document{
			S3Object: &types.S3Object{
				Bucket: &receiptUploadBucket,
				Name:   &key,
			},
		},
	}
	detected, err := textractClient.DetectDocumentText(context.TODO(), input)
	if err != nil {
		log.Printf("Textract failed: %v", err)
		sendStatus(snsClient, imageId, "Failed")
		return
	}

	// analyze the results from textract
	log.Printf("Textract finished")
	sendStatus(snsClient, imageId, "Text Detected")
	for _, block := range detected.Blocks {
		if block.BlockType == types.BlockTypeLine {
			log.Println(*block.Text)
		}
	}
	// simulate a time consuming task to analyze the text that's been detected
	time.Sleep(3 * time.Second)

	sendStatus(snsClient, imageId, "Accepted")
}

func sendStatus(client *sns.Client, imageId string, status string) {
	log.Printf("Sending status message via SNS Topic: %s", topic)
	u, err := json.Marshal(StatusMessage{ImageID: imageId, Status: status})
	if err != nil {
		panic(err)
	}
	message := string(u)

	input := &sns.PublishInput{
		Message:  &message,
		TopicArn: &topic,
	}

	log.Printf("Message: %s", message)
	snsResult, err := client.Publish(context.TODO(), input)
	if err != nil {
		log.Printf("Error publishing message: %v", err)
	}
	log.Printf("Message ID: %s", *snsResult.MessageId)
}
