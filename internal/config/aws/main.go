package awsconfig

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

func NewSession() (*session.Session, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		return nil, fmt.Errorf("AWS_REGION environment variable is not set")
	}
	return session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
}
