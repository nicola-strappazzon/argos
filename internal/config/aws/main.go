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

	opts := session.Options{
		Config: aws.Config{
			Region: aws.String(region),
		},
		SharedConfigState: session.SharedConfigEnable,
	}

	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		opts.Profile = profile
	}

	return session.NewSessionWithOptions(opts)
}
