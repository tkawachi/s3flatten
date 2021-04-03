package main

import (
	"log"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func main() {
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
