package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/uuid"
)

const (
	physicalResourceID string      = "PivotalCloudFoundry"
	Success            status      = "SUCCESS"
	Failed                         = "FAILED"
	Create             requestType = "Create"
	Update                         = "Update"
	Delete                         = "Delete"
)

type status string
type requestType string

type UpdateCustomResourceInput struct {
	RequestType       requestType
	LogicalResourceID string
	Status            status
	Reason            string
	QueueURL          string
}

type CustomResourceRequest struct {
	RequestType       string `json:"RequestType"`
	ResponseURL       string `json:"ResponseURL"`
	RequestID         string `json:"RequestId"`
	ResourceType      string `json:"ResourceType"`
	LogicalResourceID string `json:"LogicalResourceId"`
}

type CustomResourceResponse struct {
	Status             string `json:"Status"`
	Reason             string `json:"Reason"`
	PhysicalResourceID string `json:"PhysicalResourceId"`
	StackID            string `json:"StackId"`
	RequestID          string `json:"RequestId"`
	LogicalResourceID  string `json:"LogicalResourceId"`
}

func (c *Client) UpdateCustomResource(ctx context.Context, i UpdateCustomResourceInput) error {
	requests, err := c.receiveCustomResourceRequests(ctx, i.QueueURL)
	if err != nil {
		return err
	}

	for _, request := range *requests {
		if request.LogicalResourceID == i.LogicalResourceID &&
			request.RequestType == string(i.RequestType) {
			body, err := json.Marshal(CustomResourceResponse{
				Status:             string(i.Status),
				Reason:             i.Reason,
				PhysicalResourceID: physicalResourceID,
				StackID:            c.stackID,
				RequestID:          request.RequestID,
				LogicalResourceID:  request.LogicalResourceID,
			})
			hreq, err := http.NewRequest("PUT", request.ResponseURL, bytes.NewReader(body))
			if err != nil {
				return err
			}
			_, err = http.DefaultClient.Do(hreq)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) receiveCustomResourceRequests(ctx context.Context, queueURL string) (*[]CustomResourceRequest, error) {
	service := sqs.New(c.session)
	id := uuid.New().String()
	var max int64 = 10
	var visibility int64 = 1
	request := sqs.ReceiveMessageInput{
		QueueUrl:                &queueURL,
		ReceiveRequestAttemptId: &id,
		MaxNumberOfMessages:     &max,
		VisibilityTimeout:       &visibility,
	}
	response, err := service.ReceiveMessageWithContext(ctx, &request)
	if err != nil {
		return nil, err
	}
	var requests []CustomResourceRequest
	for _, msg := range response.Messages {
		var req CustomResourceRequest
		err := json.Unmarshal([]byte(*msg.Body), &req)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return &requests, nil
}
