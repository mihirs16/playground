package edges

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// The real S3 store is exercised the same way the OTLP sink is: against a local
// HTTP stand-in for the live backend, with static test credentials so no test
// ever reaches for the instance-profile chain or the network.
func testStore(bucket, endpoint string) *s3Store {
	awsCfg := aws.Config{
		Region:      "eu-west-2",
		Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
	}
	return newS3StoreFromConfig(bucket, awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
}

// PresignPut hands broom a PUT it can upload to directly: the method is PUT, the
// bucket and object key are in the path, and the URL expires after exactly the
// requested window. Content-type is carried on the PutObject input so a compliant
// upload lands with it; SigV4 presigning does not bind it into the signature.
func TestS3PresignPutIsAPutToKeyWithExpiry(t *testing.T) {
	store := testStore("media-bucket", "https://s3.eu-west-2.amazonaws.com")

	const (
		key         = "hero-shot"
		contentType = "image/png"
		expires     = 15 * time.Minute
	)

	rawURL, err := store.PresignPut(context.Background(), key, contentType, expires)
	if err != nil {
		t.Fatalf("presign put: %v", err)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("presigned url is not a url: %v", err)
	}
	if !strings.Contains(parsed.Path, "media-bucket") || !strings.Contains(parsed.Path, key) {
		t.Fatalf("presigned path %q missing bucket or key", parsed.Path)
	}

	query := parsed.Query()
	if got := query.Get("X-Amz-Expires"); got != "900" {
		t.Fatalf("X-Amz-Expires = %q, want 900 (the 15m window in seconds)", got)
	}
	if got := query.Get("X-Amz-Signature"); got == "" {
		t.Fatal("presigned url carries no signature")
	}
}

// HeadObject reflects real object presence: a 200 from S3 is present, and a 404 —
// no bytes uploaded yet — is absent, not an error. This is what lets ConfirmMedia
// flip to available only once bytes have actually landed.
func TestS3HeadObjectReflectsPresence(t *testing.T) {
	const uploadedKey = "hero-shot"

	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("method = %q, want HEAD", r.Method)
		}
		if strings.HasSuffix(r.URL.Path, "/"+uploadedKey) {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer sink.Close()

	store := testStore("media-bucket", sink.URL)

	present, err := store.HeadObject(context.Background(), uploadedKey)
	if err != nil {
		t.Fatalf("head uploaded object: %v", err)
	}
	if !present {
		t.Fatal("uploaded object reported absent")
	}

	present, err = store.HeadObject(context.Background(), "never-uploaded")
	if err != nil {
		t.Fatalf("head missing object returned an error, want absent-not-error: %v", err)
	}
	if present {
		t.Fatal("missing object reported present")
	}
}

// DeleteObject removes the bytes from the bucket: it issues a DELETE against the
// key so custodian — the only holder of S3 credentials — can clean up media broom
// can never reach directly. A failure surfaces rather than silently orphaning.
func TestS3DeleteObjectRemovesTheKey(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer sink.Close()

	if err := testStore("media-bucket", sink.URL).DeleteObject(context.Background(), "hero-shot"); err != nil {
		t.Fatalf("delete object: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method = %q, want DELETE", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/hero-shot") {
		t.Fatalf("delete path %q missing key", gotPath)
	}
}

func TestS3DeleteObjectSurfacesFailure(t *testing.T) {
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer sink.Close()

	if err := testStore("media-bucket", sink.URL).DeleteObject(context.Background(), "hero-shot"); err == nil {
		t.Fatal("delete against a failing bucket returned nil, want the failure surfaced")
	}
}

// HeadBucket reaches the real bucket: a reachable bucket is a nil error (a healthy
// input to the gauge), and an unreachable one surfaces the error so the gauge can
// go degraded.
func TestS3HeadBucketReachesBucket(t *testing.T) {
	reachable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("method = %q, want HEAD", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer reachable.Close()

	if err := testStore("media-bucket", reachable.URL).HeadBucket(context.Background()); err != nil {
		t.Fatalf("head reachable bucket: %v", err)
	}

	unreachable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer unreachable.Close()

	if err := testStore("media-bucket", unreachable.URL).HeadBucket(context.Background()); err == nil {
		t.Fatal("unreachable bucket returned nil error, want the failure surfaced to the gauge")
	}
}
