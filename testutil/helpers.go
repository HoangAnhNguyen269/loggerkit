package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

// CaptureStdout captures stdout during the execution of fn
func CaptureStdout(fn func()) (string, error) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stdout = w

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(&buf, r)
	}()

	fn()

	w.Close()
	os.Stdout = oldStdout
	wg.Wait()
	r.Close()

	return buf.String(), nil
}

// TempFile creates a temporary file for testing
func TempFile(t testing.TB, prefix, suffix string) (string, func()) {
	t.Helper()
	f, err := os.CreateTemp("", prefix+"*"+suffix)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	path := f.Name()
	f.Close()

	cleanup := func() {
		os.Remove(path)
	}

	return path, cleanup
}

// TempDir creates a temporary directory for testing
func TempDir(t testing.TB, prefix string) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return dir, cleanup
}

// ElasticsearchMockServer creates a mock Elasticsearch server
type ElasticsearchMockServer struct {
	*httptest.Server
	mu            sync.RWMutex
	responses     []MockResponse
	receivedDocs  []map[string]interface{}
	requestCount  int
	bulkResponses []MockBulkResponse
}

type MockResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

type MockBulkResponse struct {
	StatusCode int
	Items      []MockBulkItem
}

type MockBulkItem struct {
	Index MockBulkItemResult `json:"index"`
}

type MockBulkItemResult struct {
	Status int    `json:"status"`
	Error  string `json:"error,omitempty"`
}

// NewElasticsearchMock creates a new mock Elasticsearch server
func NewElasticsearchMock() *ElasticsearchMockServer {
	mock := &ElasticsearchMockServer{
		responses:     []MockResponse{},
		receivedDocs:  []map[string]interface{}{},
		bulkResponses: []MockBulkResponse{},
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		mock.requestCount++
		mock.mu.Unlock()

		switch r.URL.Path {
		case "/_bulk":
			mock.handleBulkRequest(w, r)
		default:
			mock.handleGenericRequest(w, r)
		}
	}))

	return mock
}

func (m *ElasticsearchMockServer) handleBulkRequest(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	lines := bytes.Split(body, []byte("\n"))

	// Parse bulk request
	for i := 0; i < len(lines)-1; i += 2 {
		if len(lines[i]) == 0 {
			continue
		}
		// Skip action line, parse doc line
		if i+1 < len(lines) && len(lines[i+1]) > 0 {
			var doc map[string]interface{}
			if err := json.Unmarshal(lines[i+1], &doc); err == nil {
				m.mu.Lock()
				m.receivedDocs = append(m.receivedDocs, doc)
				m.mu.Unlock()
			}
		}
	}

	m.mu.RLock()
	if len(m.bulkResponses) > 0 {
		resp := m.bulkResponses[0]
		if len(m.bulkResponses) > 1 {
			m.bulkResponses = m.bulkResponses[1:]
		}
		m.mu.RUnlock()

		w.WriteHeader(resp.StatusCode)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": resp.Items,
		})
		return
	}
	m.mu.RUnlock()

	// Default success response
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": []MockBulkItem{
			{Index: MockBulkItemResult{Status: 201}},
		},
	})
}

func (m *ElasticsearchMockServer) handleGenericRequest(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	if len(m.responses) > 0 {
		resp := m.responses[0]
		if len(m.responses) > 1 {
			m.responses = m.responses[1:]
		}
		m.mu.RUnlock()

		for k, v := range resp.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(resp.StatusCode)
		w.Write([]byte(resp.Body))
		return
	}
	m.mu.RUnlock()

	// Default response
	w.WriteHeader(200)
	w.Write([]byte(`{"version": {"number": "8.0.0"}}`))
}

// SetResponse sets the next response to return
func (m *ElasticsearchMockServer) SetResponse(statusCode int, body string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = append(m.responses, MockResponse{
		StatusCode: statusCode,
		Body:       body,
	})
}

// SetBulkResponse sets the next bulk response
func (m *ElasticsearchMockServer) SetBulkResponse(statusCode int, items []MockBulkItem) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bulkResponses = append(m.bulkResponses, MockBulkResponse{
		StatusCode: statusCode,
		Items:      items,
	})
}

// GetReceivedDocs returns all documents received by the mock server
func (m *ElasticsearchMockServer) GetReceivedDocs() []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]map[string]interface{}, len(m.receivedDocs))
	copy(result, m.receivedDocs)
	return result
}

// GetRequestCount returns the total number of requests received
func (m *ElasticsearchMockServer) GetRequestCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requestCount
}

// WaitForDocs waits for a specific number of documents to be received
func (m *ElasticsearchMockServer) WaitForDocs(count int, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			m.mu.RLock()
			docCount := len(m.receivedDocs)
			m.mu.RUnlock()
			if docCount >= count {
				return true
			}
		}
	}
}
