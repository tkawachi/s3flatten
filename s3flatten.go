package main

import (
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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

	sess := session.Must(session.NewSession())
	svc := s3.New(sess)

	listInput := s3.ListObjectsInput{} // TODO specify bucket, prefix

	err := svc.ListObjectsPages(&listInput, func(page *s3.ListObjectsOutput, isLast bool) bool {
		return true
	})
	if err != nil {
		log.Fatal("Failed to list objects:", err)
	}
}
