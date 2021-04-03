package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pborman/getopt/v2"
)

func main() {
	helpFlag := getopt.BoolLong("help", 'h', "display help")
	delimiterFlag := getopt.StringLong("delimiter", 'd', "-", "Delimiter to replace '/' with to flatten path.")
	getopt.SetParameters("s3://src-bucket/path/to/src/ s3://dest-bucket/path/to/dest/")
	getopt.Parse()
	args := getopt.Args()
	if *helpFlag || len(args) != 2 {
		getopt.Usage()
		os.Exit(1)
	}
	log.Println(*delimiterFlag)

	log.Println(args)

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	client := s3.NewFromConfig(cfg)

	listInput := s3.ListObjectsV2Input{} // TODO specify bucket, prefix
	paginator := s3.NewListObjectsV2Paginator(client, &listInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Printf("error: %v", err)
			return
		}
		for _, value := range page.Contents {
			log.Println(*value.Key)
		}
	}
}
