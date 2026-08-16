// Package edges defines the three outward edges that leave custodian's box —
// object storage (S3), the third-party source clients (Steam/GitHub), and the
// telemetry sink (OTLP). Each is an interface injected at construction so that
// tests can drive the whole service against fakes and never touch the network.
// Real implementations live beside their fakes; the fakes are the reference
// behaviour every test relies on.
package edges

import (
	"context"
	"time"
)

// ObjectStore is custodian's view of S3: presign an upload so bytes go straight
// from broom to the bucket, HEAD an object to confirm those bytes landed, HEAD
// the bucket as one input to the health gauge, and delete an object on a media
// delete. Custodian is never on the byte path for uploads, but it is the only
// holder of S3 credentials — so removing bytes is custodian's job, not broom's.
type ObjectStore interface {
	PresignPut(ctx context.Context, key, contentType string, expires time.Duration) (url string, err error)
	HeadObject(ctx context.Context, key string) (exists bool, err error)
	HeadBucket(ctx context.Context) error
	DeleteObject(ctx context.Context, key string) error
}

// FetchResult is one poll of a third-party source. When NotModified is true the
// source returned 304 and Data is meaningless; the caller keeps the last stored
// row. ETag, when present, is sent back as If-None-Match on the next poll.
type FetchResult struct {
	NotModified bool
	ETag        string
	Data        map[string]any
}

// SourceClient fetches the current state of one third-party source. The
// credential is passed per-call, resolved from the process environment like
// every other secret; rotation is replace-and-restart. A network failure is an
// error, not a result — an unreachable source must never flip the health gauge.
type SourceClient interface {
	Fetch(ctx context.Context, source, credential, etag string) (FetchResult, error)
}

// Telemetry is the OTLP sink. Custodian records the health gauge through it and
// shuts it down cleanly on exit; the concrete backend is a swappable config URL.
type Telemetry interface {
	RecordHealth(ctx context.Context, healthy bool)
	Shutdown(ctx context.Context) error
}

// Set bundles the three edges so construction takes one argument, not three.
type Set struct {
	ObjectStore  ObjectStore
	SourceClient SourceClient
	Telemetry    Telemetry
}
