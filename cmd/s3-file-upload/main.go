package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gopkg.in/yaml.v2"
)

// Config and it's fields are exported to receive the contents of a YAML
// configuration file
type Conf struct {
	SecretAccessKey string `yaml:"secret_access_key"`
	AccessKeyID     string `yaml:"access_key_id"`
	Region          string `yaml:"region"`
	BucketName      string `yaml:"bucket_name"`
}

func validateConf(conf Conf, configFilename string) (*Conf, error) {
	if conf.SecretAccessKey == "" {
		return nil, fmt.Errorf("required key: `secret_access_key` is missing from file: %q", configFilename)
	}

	if conf.AccessKeyID == "" {
		return nil, fmt.Errorf("required key: `access_key_id` is missing from file: %q", configFilename)
	}

	if conf.Region == "" {
		return nil, fmt.Errorf("required key: `region` is missing from file: %q", configFilename)
	}

	if conf.BucketName == "" {
		return nil, fmt.Errorf("required key: `bucket_name` is missing from file: %q", configFilename)
	}
	return &conf, nil
}

func unmarshalConf(configFilename string) (*Conf, error) {
	if configFilename == "" {
		return nil, errors.New("no config file specified")
	}

	configData, err := ioutil.ReadFile(configFilename)
	if configFilename == "" {
		return nil, fmt.Errorf("failed to load config file: %q due to: %s", configFilename, err)
	}

	var conf Conf
	err = yaml.UnmarshalStrict(configData, &conf)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal YAML from file: %q due to: %s", configFilename, err)
	}
	return validateConf(conf, configFilename)
}

func makeS3Client(c *Conf) (*s3.Client, error) {
	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	awsConfig.Credentials = credentials.NewStaticCredentialsProvider(
		c.AccessKeyID, c.SecretAccessKey, "")
	awsConfig.Region = c.Region
	return s3.NewFromConfig(awsConfig), nil
}

func listBucketContents(c *Conf, client *s3.Client) error {
	input := &s3.ListObjectsV2Input{Bucket: aws.String(c.BucketName)}
	output, err := client.ListObjectsV2(context.Background(), input)
	if err != nil {
		return err
	}

	if len(output.Contents) > 0 {
		for _, object := range output.Contents {
			fmt.Printf("%s %d\n", aws.ToString(object.Key), object.Size)
		}
	} else {
		fmt.Printf("bucket: %q contains 0 files\n", c.BucketName)
	}
	return nil
}

func putFile(c *Conf, client *s3.Client, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	input := &s3.PutObjectInput{
		Bucket: aws.String(c.BucketName),
		Key:    aws.String(filename),
		Body:   file,
	}
	_, err = client.PutObject(context.Background(), input)
	if err != nil {
		return err
	}
	return nil
}

func deleteFile(c *Conf, client *s3.Client, filename string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(c.BucketName),
		Key:    aws.String(filename),
	}
	_, err := client.DeleteObject(context.Background(), input)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	configFilename := flag.String("config", "", "Path to the YAML configuration file")
	putFilename := flag.String("put-file", "", "Path to file you want uploaded to the S3 bucket")
	deleteFilename := flag.String("delete-file", "", "Name of the file you want deleted from the S3 bucket")
	listBucket := flag.Bool("list-bucket", false, "List contents of the S3 bucket to stdout")
	flag.Parse()

	conf, err := unmarshalConf(*configFilename)
	if err != nil {
		log.Fatalf("failed to load S3 config: %s", err)
	}

	var client *s3.Client
	if *putFilename != "" || *deleteFilename != "" || *listBucket {
		client, err = makeS3Client(conf)
		if err != nil {
			log.Fatalf("failed to make S3 client: %s", err)
		}
	} else {
		fmt.Println("no action requested, nothing to do")
		os.Exit(0)
	}

	if *putFilename != "" {
		err := putFile(conf, client, *putFilename)
		if err != nil {
			log.Fatalf("failed to upload file: %q to bucket: %q due to: %s", *putFilename, conf.BucketName, err)
		}
	}

	if *deleteFilename != "" {
		err := deleteFile(conf, client, *deleteFilename)
		if err != nil {
			log.Fatalf("failed to delete file: %q from bucket: %q due to: %s", *deleteFilename, conf.BucketName, err)
		}
	}

	if *listBucket {
		err := listBucketContents(conf, client)
		if err != nil {
			log.Fatalf("failed to list to contents of bucket: %q due to: %s", conf.BucketName, err)
		}
	}
}
