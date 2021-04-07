package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pborman/getopt/v2"
)

var verbose bool

type s3Path struct {
	bucket string
	prefix string
}

type copyResult struct {
	srcKey string
	err    error
}

type s3Flatten struct {
	client      *s3.Client
	srcPath     *s3Path
	dstPath     *s3Path
	delimiter   string
	suffix      string
	concurrency int
}

func fanOut(input <-chan string) (<-chan string, <-chan string) {
	out1 := make(chan string)
	out2 := make(chan string)
	go func() {
		for {
			s, ok := <-input
			if ok {
				out1 <- s
				out2 <- s
			} else {
				// input is closed
				close(out1)
				close(out2)
				break
			}
		}
	}()
	return out1, out2
}

func debug(v ...interface{}) {
	if verbose {
		log.Println(v...)
	}
}

func parseS3Path(s string) (*s3Path, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse as S3 path: %s", s)
	}
	if u.Scheme != "s3" {
		return nil, fmt.Errorf("S3 path should start with `s3`: %s", s)
	}
	prefix := u.Path
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	prefix = strings.TrimPrefix(prefix, "/")
	return &s3Path{
		bucket: u.Host,
		prefix: prefix,
	}, nil
}

func genDstKey(srcKey, srcKeyPrefix, dstKeyPrefix, delimiter string) (string, error) {
	if !strings.HasPrefix(srcKey, srcKeyPrefix) {
		return "", fmt.Errorf("srcKeyPrefix not match. srcKey: %s, srcKeyPrefix: %s", srcKey, srcKeyPrefix)
	}
	return (dstKeyPrefix + strings.Replace(srcKey[len(srcKeyPrefix):], "/", delimiter, -1)), nil
}

func (sf *s3Flatten) listObjects(ctx context.Context, listedCh chan<- string) {
	defer close(listedCh)
	listInput := s3.ListObjectsV2Input{
		Bucket: aws.String(sf.srcPath.bucket),
		Prefix: aws.String(sf.srcPath.prefix),
	}
	paginator := s3.NewListObjectsV2Paginator(sf.client, &listInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			log.Fatal(err)
		}
		for _, value := range page.Contents {
			if strings.HasSuffix(*value.Key, sf.suffix) {
				listedCh <- *value.Key
			}
		}
	}
}

func (sf *s3Flatten) copyObject(inCh <-chan string, outCh chan<- copyResult) {
	for {
		srcKey, ok := <-inCh
		if !ok {
			return
		}
		dstKey, err := genDstKey(srcKey, sf.srcPath.prefix, sf.dstPath.prefix, sf.delimiter)
		if err != nil {
			outCh <- copyResult{srcKey, err}
			continue
		}
		input := &s3.CopyObjectInput{
			CopySource: aws.String(sf.srcPath.bucket + "/" + srcKey),
			Bucket:     aws.String(sf.dstPath.bucket),
			Key:        aws.String(dstKey),
		}
		debug("Starting copy:", srcKey)
		startTime := time.Now()
		_, err = sf.client.CopyObject(context.Background(), input)
		if err == nil {
			debug("Finished copy:", srcKey, time.Since(startTime))
		}
		outCh <- copyResult{srcKey, err}
	}
}

func watchComplete(listedCh <-chan string, copyOutCh <-chan copyResult) error {
	startTime := time.Now()
	cnt := 0
	logStat := func() {
		elapsed := time.Since(startTime)
		// objects processed per second
		speed := float32(cnt) * 1000 / float32(elapsed/time.Millisecond)
		log.Printf("Copied %d items in %v, %.2f items/sec", cnt, elapsed, speed)
	}
	defer logStat()
	completed := make(map[string]bool)
	listedChClosed := false
	isFinished := func() bool {
		return listedChClosed && len(completed) == 0
	}
	logTick := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-logTick.C:
			logStat()
		case srcKey, ok := <-listedCh:
			if ok {
				completed[srcKey] = false
			} else {
				listedChClosed = true
				if isFinished() {
					return nil
				}
			}
		case result := <-copyOutCh:
			delete(completed, result.srcKey)
			if result.err != nil {
				return result.err
			}
			cnt++
			if isFinished() {
				return nil
			}
		}
	}
}

func getBucketRegion(ctx context.Context, client *s3.Client, bucket string) (string, error) {
	region, err := manager.GetBucketRegion(ctx, client, bucket)
	if err != nil {
		var bnf manager.BucketNotFound
		if errors.As(err, &bnf) {
			return "", fmt.Errorf("unable to find bucket's region: %s", bucket)
		}
		return "", err
	}
	debug("Source bucket's region:", region)
	return region, nil
}

func createS3Client(ctx context.Context, srcBucket string) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	// If there is no region in the configuration, set region so that the HEAD request
	// of GetBucketRegion() will not fail.
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	client := s3.NewFromConfig(cfg)
	srcRegion, err := getBucketRegion(ctx, client, srcBucket)
	if err != nil {
		return nil, err
	}
	// re-create S3 client with source bucket's region
	cfg.Region = srcRegion
	return s3.NewFromConfig(cfg), nil
}

func mustS3FlattenFromArgs(ctx context.Context) *s3Flatten {
	helpFlag := getopt.BoolLong("help", 'h', "display help")
	delimiterFlag := getopt.StringLong("delimiter", 'd', "-", "Delimiter to replace '/' with to flatten path.")
	suffixFlag := getopt.StringLong("suffix", 's', "", "Copy only objects which has this suffix in key")
	concurrencyFlag := getopt.IntLong("concurrency", 'c', 128, "Number of goroutine for COPY operation")
	getopt.FlagLong(&verbose, "verbose", 'v', "verbose output")
	getopt.SetParameters("s3://src-bucket/path/to/src/ s3://dest-bucket/path/to/dest/")
	getopt.Parse()
	args := getopt.Args()
	if *helpFlag || len(args) != 2 {
		getopt.Usage()
		os.Exit(1)
	}
	srcPath, err := parseS3Path(args[0])
	if err != nil {
		log.Fatal(err)
	}
	dstPath, err := parseS3Path(args[1])
	if err != nil {
		log.Fatal(err)
	}
	if srcPath.bucket == dstPath.bucket && strings.HasPrefix(dstPath.prefix, srcPath.prefix) {
		log.Fatal("Destination path must not be located under source path.")
	}

	debug("Delimiter:", *delimiterFlag)
	debug("Source path:", srcPath)
	debug("Destination path:", dstPath)
	debug("Concurrency:", *concurrencyFlag)

	client, err := createS3Client(ctx, srcPath.bucket)
	if err != nil {
		log.Fatal(err)
	}

	return &s3Flatten{
		client:      client,
		srcPath:     srcPath,
		dstPath:     dstPath,
		delimiter:   *delimiterFlag,
		suffix:      *suffixFlag,
		concurrency: *concurrencyFlag,
	}
}

func main() {
	ctx := context.Background()
	sf := mustS3FlattenFromArgs(ctx)
	//
	// listObjects() --listedCh--+--listedCh1-------------------> watchComplete()
	//                           |                                        ^
	//                           |                                        |
	//                           +--listedCh2--> copyObject() --copyOutCh-+
	//
	copyOutCh := make(chan copyResult)
	listedCh := make(chan string, 1000) // Buffer to read a page ahead
	listedCh1, listedCh2 := fanOut(listedCh)

	go sf.listObjects(ctx, listedCh)

	for i := 0; i < sf.concurrency; i++ {
		go sf.copyObject(listedCh2, copyOutCh)
	}

	err := watchComplete(listedCh1, copyOutCh)
	if err != nil {
		log.Fatal(err)
	}
}
