package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/davecgh/go-spew/spew"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func main() {
	os.Setenv("OTEL_SERVICE_NAME", "lambdasawa")
	os.Setenv("AWS_ACCESS_KEY_ID", "localstack")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "localstack")

	ctx := context.Background()

	cleanup, err := initTracerProvider(ctx)
	if err != nil {
		panic(err)
	}
	defer cleanup(ctx)

	tracer = otel.Tracer("test-tracer")

	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	if err := run(ctx); err != nil {
		log.Panic(err)
	}
}

type cleanup = func(ctx context.Context)

func initTracerProvider(ctx context.Context) (cleanup, error) {
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)

	cleanup := func(ctx context.Context) {
		err := tracerProvider.Shutdown(ctx)
		if err != nil {
			log.Fatalf("error shutting down tracer provider: %v", err)
		}
	}

	otel.SetTracerProvider(tracerProvider)

	return cleanup, nil
}

func run(ctx context.Context) error {
	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint:         aws.String("http://localhost:4566"),
		Region:           aws.String("us-east-1"),
		S3ForcePathStyle: aws.Bool(true),
	}))

	{
		service := s3.New(sess)
		instrument(service.Client)

		buckets, err := service.ListBucketsWithContext(ctx, &s3.ListBucketsInput{})
		if err != nil {
			return err
		}
		spew.Dump(buckets)
	}

	{
		service := dynamodb.New(sess)
		instrument(service.Client)

		tables, err := service.ListTablesWithContext(ctx, &dynamodb.ListTablesInput{})
		if err != nil {
			return err
		}
		spew.Dump(tables)

		putItemOutput, err := service.PutItemWithContext(ctx, &dynamodb.PutItemInput{
			TableName: aws.String("not-found"),
			Item:      map[string]*dynamodb.AttributeValue{"id": {S: aws.String("1")}},
		})
		if err != nil {
			return err
		}
		spew.Dump(putItemOutput)
	}

	return nil
}

func instrument(client *client.Client) {
	serviceName := client.ServiceName

	client.Handlers.Send.PushFront(func(r *request.Request) {
		operationName := r.Operation.Name

		ctx, _ := tracer.Start(r.Context(), fmt.Sprintf("%s:%s", serviceName, operationName))

		r.SetContext(ctx)
	})

	client.Handlers.Complete.PushBack(func(r *request.Request) {
		span := trace.SpanFromContext(r.Context())
		defer span.End()

		if r.Error != nil {
			span.SetStatus(codes.Error, r.Error.Error())
			span.RecordError(r.Error)
		}
	})
}
