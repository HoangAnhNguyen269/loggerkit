package zapx

import (
	"context"
	"errors"

	"github.com/elastic/go-elasticsearch/v8/esutil"
)

// esIndexer interface for testing
type esIndexer interface {
	Add(ctx context.Context, item esutil.BulkIndexerItem) error
	Close(ctx context.Context) error
}

// MockIndexer for testing
type MockIndexer struct {
	addErrors    []error
	addCallCount int
	closeError   error
	closed       bool
}

func NewMockIndexer(addErrors []error) *MockIndexer {
	return &MockIndexer{
		addErrors: addErrors,
	}
}

func (m *MockIndexer) Add(ctx context.Context, item esutil.BulkIndexerItem) error {
	defer func() { m.addCallCount++ }()

	if m.addCallCount < len(m.addErrors) {
		return m.addErrors[m.addCallCount]
	}

	// Simulate successful add by calling OnSuccess if no error
	if item.OnSuccess != nil {
		item.OnSuccess(ctx, item, esutil.BulkIndexerResponseItem{Status: 201})
	}

	return nil
}

func (m *MockIndexer) Close(ctx context.Context) error {
	m.closed = true
	return m.closeError
}

func (m *MockIndexer) SetCloseError(err error) {
	m.closeError = err
}

func (m *MockIndexer) IsClosed() bool {
	return m.closed
}

func (m *MockIndexer) GetAddCallCount() int {
	return m.addCallCount
}

// MockFailingIndexer always fails Add operations
type MockFailingIndexer struct {
	*MockIndexer
}

func NewMockFailingIndexer() *MockFailingIndexer {
	return &MockFailingIndexer{
		MockIndexer: &MockIndexer{
			addErrors: []error{errors.New("mock indexer add error")},
		},
	}
}

func (m *MockFailingIndexer) Add(ctx context.Context, item esutil.BulkIndexerItem) error {
	m.addCallCount++

	// Always call OnFailure
	if item.OnFailure != nil {
		item.OnFailure(ctx, item, esutil.BulkIndexerResponseItem{Status: 503}, errors.New("mock failure"))
	}

	return errors.New("mock indexer add error")
}
