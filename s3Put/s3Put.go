package s3Put

import (
	"bytes"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// iFile : Interface for fileMeta functions so we can override the struct methods
// for testing.
type iFile interface {
	open(p string)
	size() int64
	name() string
	reader() *bytes.Reader
}

// fileMeta : Struct to contain and organize information about the file to be uploaded to S3
type fileMeta struct {
	iFile
	path   string
	handle *os.File
}

// open : fileMeta method. Input a file path of file to open and sets the path
// and handle objects of the struct.
func (file fileMeta) open(p string) {
	file.path = p
	f, err := os.Open(file.path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.handle.Close()
	file.handle = f
}

// size : Method to return the size in bytes of the file as an int64.
func (file fileMeta) size() int64 {
	f, err := file.handle.Stat()
	if err != nil {
		log.Fatal(err)
	}
	var size int64 = f.Size()
	return size
}

// name : Method to return the name of the file as a string.
func (file fileMeta) name() string {
	f, err := file.handle.Stat()
	if err != nil {
		log.Fatal(err)
	}
	var name string = f.Name()
	return name
}

// reader : Method to return bytes.Reader content of the file.
// This can be used to pass to s3 as the body of the file.
func (file fileMeta) reader() *bytes.Reader {
	buffer := make([]byte, file.size())
	_, err := file.handle.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}
	return bytes.NewReader(buffer)
}

// createSession: Input aws-region, Outputs a session, error
// This can be replaced with a mock session for testing.
func createSession(s3Region string) (*session.Session, error) {
	return session.NewSession(&aws.Config{Region: aws.String(s3Region)})
}

// Function vars. Can be overridden for testing.
var (
	sessionNewSession = createSession
)

// AddFileToS3 will upload a single file to S3, it requires an aws region,
// s3 bucket and the file to be copied. The S3 library expects aws
// credentials to be configured in $HOME/.aws just as the aws cli uses.
// See https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html
// for more information.
func AddFileToS3(s3Region, s3Bucket, filePath string) error {

	// New aws session
	s, err := sessionNewSession(s3Region)
	if err != nil {
		return err
	}

	// Create and open new fileMeta for the input file to be uploaded to s3
	var f fileMeta
	f.open(filePath)
	putObjectS3(s, f, s3Bucket)
	return nil
}

func putObjectS3(s *session.Session, f iFile, s3Bucket string) error {
	// Config Settings: Choose bucket, filename, content-type etc for uplaoded file
	_, err := s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(s3Bucket),
		Key:           aws.String(f.name()),
		ACL:           aws.String("private"),
		Body:          f.reader(),
		ContentLength: aws.Int64(f.size()),
		//ContentType:          aws.String(http.DetectContentType(fileBytes)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	return err
}
