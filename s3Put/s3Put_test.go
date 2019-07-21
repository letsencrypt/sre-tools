package s3Put

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/awstesting/mock"
)

func TestAddFileToS3(t *testing.T) {
	awsConfig := &aws.Config{
		Region: aws.String("mock-region-1"),
	}
	s := mock.Session
	t.Logf("output: %v", s)
	c := mock.NewMockClient(awsConfig)
	t.Log(c.Handlers)
}
