package edges

import (
	"context"
	"sync"
	"time"
)

// FakeObjectStore is an in-memory S3 stand-in. Tests drive the media flow by
// presigning (which just records the key), marking bytes present to simulate
// broom's upload, then confirming — exactly the reserve → upload → HEAD → available
// sequence, with no network.
type FakeObjectStore struct {
	mu sync.Mutex

	// PresignURL is returned verbatim from PresignPut when set, so tests can
	// assert broom receives a specific upload URL.
	PresignURL string

	// BucketErr, when set, makes HeadBucket fail so tests can drive the degraded
	// branch of the health gauge.
	BucketErr error

	present   map[string]bool
	presigned []string
}

func NewFakeObjectStore() *FakeObjectStore {
	return &FakeObjectStore{present: map[string]bool{}}
}

func (f *FakeObjectStore) PresignPut(_ context.Context, key, _ string, _ time.Duration) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.presigned = append(f.presigned, key)
	if f.PresignURL != "" {
		return f.PresignURL, nil
	}
	return "https://s3.fake.local/upload/" + key, nil
}

func (f *FakeObjectStore) HeadObject(_ context.Context, key string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.present[key], nil
}

func (f *FakeObjectStore) HeadBucket(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.BucketErr
}

func (f *FakeObjectStore) DeleteObject(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.present, key)
	return nil
}

// PutBytes simulates broom completing a presigned upload, so a later HeadObject
// reports the key as present.
func (f *FakeObjectStore) PutBytes(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.present[key] = true
}

// FakeSourceClient is a scriptable Steam/GitHub client. Tests register a result
// (or error) per source to drive the poller through changed / unchanged (304) /
// unreachable cases.
type FakeSourceClient struct {
	mu      sync.Mutex
	results map[string]FetchResult
	errs    map[string]error
}

func NewFakeSourceClient() *FakeSourceClient {
	return &FakeSourceClient{
		results: map[string]FetchResult{},
		errs:    map[string]error{},
	}
}

// SetResult scripts the result Fetch returns for a source.
func (f *FakeSourceClient) SetResult(source string, result FetchResult) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.results[source] = result
	delete(f.errs, source)
}

// SetError scripts a network failure for a source, standing in for an
// unreachable third party.
func (f *FakeSourceClient) SetError(source string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs[source] = err
	delete(f.results, source)
}

func (f *FakeSourceClient) Fetch(_ context.Context, source, _, _ string) (FetchResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.errs[source]; err != nil {
		return FetchResult{}, err
	}
	return f.results[source], nil
}

// FakeTelemetry captures every health gauge value so tests can assert the
// service reported healthy or degraded without a live OTLP backend.
type FakeTelemetry struct {
	mu       sync.Mutex
	health   []bool
	shutdown bool
}

func NewFakeTelemetry() *FakeTelemetry { return &FakeTelemetry{} }

func (f *FakeTelemetry) RecordHealth(_ context.Context, healthy bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.health = append(f.health, healthy)
}

func (f *FakeTelemetry) Shutdown(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.shutdown = true
	return nil
}

// LastHealth reports the most recently recorded gauge value and whether any was
// recorded at all.
func (f *FakeTelemetry) LastHealth() (value bool, recorded bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.health) == 0 {
		return false, false
	}
	return f.health[len(f.health)-1], true
}

// NewFakes returns a Set wired entirely to the in-memory fakes, the standard
// starting point for a black-box test.
func NewFakes() Set {
	return Set{
		ObjectStore:  NewFakeObjectStore(),
		SourceClient: NewFakeSourceClient(),
		Telemetry:    NewFakeTelemetry(),
	}
}
