package edges

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// s3Store is custodian's production ObjectStore: the AWS SDK for Go v2 against a
// single media bucket, authenticated through the IMDS instance-profile chain so
// no long-lived AWS credentials ever touch the box (ADR-0001). Custodian is never
// on the byte path — it only presigns uploads and HEADs to observe them.
type s3Store struct {
	bucket  string
	client  *s3.Client
	presign *s3.PresignClient
}

// newS3Store builds the production store. LoadDefaultConfig resolves the region
// from the environment (AWS_REGION) and credentials from the default chain, which
// on the box is the IMDS instance profile — no static credentials anywhere.
func newS3Store(ctx context.Context, bucket string) (*s3Store, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return newS3StoreFromConfig(bucket, awsCfg), nil
}

// newS3StoreFromConfig builds a store over an already-resolved AWS config. The
// production path passes the instance-profile config from LoadDefaultConfig; a
// test passes static credentials and points optFns at a local endpoint.
func newS3StoreFromConfig(bucket string, awsCfg aws.Config, optFns ...func(*s3.Options)) *s3Store {
	client := s3.NewFromConfig(awsCfg, optFns...)
	return &s3Store{
		bucket:  bucket,
		client:  client,
		presign: s3.NewPresignClient(client),
	}
}

// PresignPut returns a presigned S3 PUT for the key and content-type, valid for
// the given window. broom uploads to it directly; custodian never sees the bytes.
func (s *s3Store) PresignPut(ctx context.Context, key, contentType string, expires time.Duration) (string, error) {
	req, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// HeadObject reports whether bytes for the key exist in the bucket. A 404 is a
// definite absence, not an error — it is the normal answer before broom uploads —
// so ConfirmMedia can flip to available only once bytes have landed. Any other
// failure is a real error.
func (s *s3Store) HeadObject(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// HeadBucket reaches the bucket as one input to the health gauge; the error, if
// any, passes straight through to the assessor.
func (s *s3Store) HeadBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	return err
}

// isNotFound recognises S3's two shapes of "no such object": the modelled
// NotFound error, and a bare 404 HEAD response the service returns without an
// error code.
func isNotFound(err error) bool {
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return true
	}
	var respErr *smithyhttp.ResponseError
	if errors.As(err, &respErr) {
		return respErr.HTTPStatusCode() == http.StatusNotFound
	}
	return false
}
