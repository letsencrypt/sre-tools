package s3Put

import (
	"bytes"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/awstesting/mock"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

// Mock s3 client with s3iface
type mockS3Client struct {
	s3iface.S3API
}

func (m *mockS3Client) uploadToS3(input *s3.New(s).putObjectS3(&s3.PutObjectInput()))

// Mock an aws api session
func mockNewSession(s3Region string) (*session.Session, error) {
	return mock.Session, nil
}

/*
// Return []byte to simulate a file read from disk for upload
func mockReadFile(mockFilePath string) ([]byte, error) {
	return []byte{'m', 'o', 'c', 'k', ' ', 'd', 'a', 't', 'a'}, nil
}
*/

func mockReadFileError(mockFilePath string) ([]byte, error) {
	return nil, errors.New("Mock: Could not read file")
}

func mockNewSessionError(s3Region string) (*session.Session, error) {
	return nil, errors.New("Mock: Could not open s3 session")
}

type mockFileMeta struct {
	//iFile
	path string
}

func (file mockFileMeta) open(p string) {

}

func (file mockFileMeta) size() int64 {
	return int64(123456)
}

func (file mockFileMeta) name() string {
	return "mock-file"
}

func (file mockFileMeta) reader() *bytes.Reader {
	return bytes.NewReader([]byte{'m', 'o', 'c', 'k', ' ', 'd', 'a', 't', 'a'})
}

type test struct {
	mockSession *session.Session
	mockPath    *bytes.Reader
}

func TestAddFileToS3(t *testing.T) {
	m := &mockFileMeta{
		path: "mock/path",
	}
	sessionNewSessionBak := sessionNewSession
	sessionNewSession = mockNewSession
	s, err := sessionNewSession("mock-region-1")
	if err != nil {
		t.Fatalf("Unexpected error: %q", err)
	}
	s3Bucket := "mock-s3-bucket"
	// table driven here
	// nil, m, "mock-s3-bucket"
	// s, m, nil
	// s, nil, "mock-s3-bucket"

	/*tests := []test{
		{input: nil, m, s3Bucket, want: []string{"", ","}},
		{input: s, nil, s3Bucket, want: []string{"", ","}},
		{input: s, m, nil, want: []string{"", ","}},
	}*/
	err = putObjectS3(s, m, s3Bucket)
	if err != nil {
		t.Fatalf("error: %q", err)
	}
	/*putObjectS3(nil, m, s3Bucket)
	if err != nil {
		t.Logf("error: %q", err)
	}*/
	//t.Logf("test data:\n  %q\n  %q\n  %q\n  %q", m.name(), m.reader(), m.size(), m.path)
	sessionNewSession = sessionNewSessionBak
}
