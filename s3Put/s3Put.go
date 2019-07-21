package s3Put

import (
	"bytes"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// AddFileToS3 will upload a single file to S3, it requires an aws region,
// s3 bucket and the file to be copied. It will set file info like content
// type and encryption on the uploaded file. The S3 library expects aws
// credentials to be configured in $HOME/.aws just as the aws cli uses.
// See https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html
// for more information.
func AddFileToS3(s3Region, s3Bucket, fileDir string) error {

	// Create aws session
	s, err := session.NewSession(&aws.Config{Region: aws.String(s3Region)})
	if err != nil {
		log.Fatal(err)
	}

	// Open file for use
	file, err := os.Open(fileDir)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Get file size and read the file content into a buffer
	fileInfo, _ := file.Stat()
	var size int64 = fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	// Config Settings: Choose bucket, filename, content-type etc for uplaoded file
	_, err = s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(s3Bucket),
		Key:                  aws.String(fileDir),
		ACL:                  aws.String("private"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	return err
}
