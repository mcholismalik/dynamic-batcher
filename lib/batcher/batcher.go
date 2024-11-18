package batcher

import (
	"container/heap"
	"context"
	"errors"
	"sync"
	"time"
)

// Request represents a single request to be processed.
type Request struct {
	ListenerKey     string
	UniqueKey       string
	AggregatedValue interface{}
	ResultChan      chan interface{}
}

// NewRequest creates a new Request instance.
func NewRequest(listenerKey, uniqueKey string, aggregatedValue interface{}) *Request {
	return &Request{
		ListenerKey:     listenerKey,
		UniqueKey:       uniqueKey,
		AggregatedValue: aggregatedValue,
		ResultChan:      make(chan interface{}, 1),
	}
}

// Listener defines how requests should be batched and processed.
type Listener struct {
	key     string
	maxReq  int
	maxWait time.Duration
	run     Runner
}

// NewListener creates a new Listener instance with provided options.
func NewListener(key string, opts ...ListenerOption) *Listener {
	l := &Listener{
		key:     key,
		maxReq:  10,
		maxWait: 20 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// ListenerOption defines a function type for Listener configuration.
type ListenerOption func(*Listener)

// WithMaxRequests sets the maximum number of requests before executing a batch.
func WithMaxRequests(maxReq int) ListenerOption {
	return func(l *Listener) {
		l.maxReq = maxReq
	}
}

// WithMaxWait sets the maximum wait time before executing a batch.
func WithMaxWait(maxWait time.Duration) ListenerOption {
	return func(l *Listener) {
		l.maxWait = maxWait
	}
}

// WithRunner sets the runner function for executing the batch.
func WithRunner(run Runner) ListenerOption {
	return func(l *Listener) {
		l.run = run
	}
}

// Runner defines a function that processes a batch of requests.
type Runner func(ctx context.Context, reqs []*Request, done chan struct{})

// Batch represents a collection of requests that should be processed together.
type Batch struct {
	maxReq       int
	maxWait      time.Duration
	run          Runner
	mutex        *sync.RWMutex
	requests     []*Request
	state        int
	lastUsed     time.Time
	heapIndex    int // Index of the batch in the heap
	mutexBatcher *sync.RWMutex
}

// BatchHeap is a min-heap to manage batches based on their last used time.
type BatchHeap []*Batch

func (bh BatchHeap) Len() int           { return len(bh) }
func (bh BatchHeap) Less(i, j int) bool { return bh[i].lastUsed.Before(bh[j].lastUsed) }
func (bh BatchHeap) Swap(i, j int) {
	bh[i], bh[j] = bh[j], bh[i]
	bh[i].heapIndex = i
	bh[j].heapIndex = j
}

func (bh *BatchHeap) Push(x interface{}) {
	n := len(*bh)
	b := x.(*Batch)
	b.heapIndex = n
	*bh = append(*bh, b)
}

func (bh *BatchHeap) Pop() interface{} {
	old := *bh
	n := len(old)
	b := old[n-1]
	b.heapIndex = -1 // safety
	*bh = old[0 : n-1]
	return b
}

// Batcher is the main structure that manages batching and batch cleanup.
type Batcher interface {
	AddListener(listener *Listener) error
	GetListener(key string) (*Listener, error)
	AddRequest(req *Request) error
}

// Batcher implementation that manages batches and requests.
type batcher struct {
	batches       map[string][]*Batch
	listeners     map[string]*Listener
	requestChan   chan *Request
	mutex         *sync.RWMutex
	heap          *BatchHeap
	interval      time.Duration
	staleDuration time.Duration
}

// NewBatcher creates and initializes a new batcher instance.
func NewBatcher(opts ...BatcherOption) *batcher {
	bh := &BatchHeap{}
	heap.Init(bh)
	b := &batcher{
		listeners:     make(map[string]*Listener),
		requestChan:   make(chan *Request),
		batches:       make(map[string][]*Batch),
		mutex:         new(sync.RWMutex),
		heap:          bh,
		interval:      1 * time.Second,
		staleDuration: 3 * time.Second,
	}

	for _, opt := range opts {
		opt(b)
	}

	b.Init()
	return b
}

// BatcherOption defines a function type for Batcher configuration.
type BatcherOption func(*batcher)

// WithCleanupInterval sets the interval for batch cleanup.
func WithCleanupInterval(interval time.Duration) BatcherOption {
	return func(b *batcher) {
		b.interval = interval
	}
}

// WithStaleDuration sets the duration before a batch is considered stale.
func WithStaleDuration(staleDuration time.Duration) BatcherOption {
	return func(b *batcher) {
		b.staleDuration = staleDuration
	}
}

// Init starts the batcher's internal processes for handling incoming requests.
func (b *batcher) Init() {
	go func() {
		for req := range b.requestChan {
			go b.AssignRequest(req)
		}
	}()
	go b.CleanupBatches()
}

// AddListener registers a new listener for batching requests.
func (b *batcher) AddListener(listener *Listener) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if _, exists := b.listeners[listener.key]; exists {
		return errors.New("listener conflict")
	}
	b.listeners[listener.key] = listener
	return nil
}

// GetListener retrieves a listener by its key.
func (b *batcher) GetListener(key string) (*Listener, error) {
	listener, exists := b.listeners[key]
	if !exists {
		return nil, errors.New("listener not found")
	}
	return listener, nil
}

// AddRequest adds a new request to the batcher's queue for processing.
func (b *batcher) AddRequest(req *Request) error {
	if _, err := b.GetListener(req.ListenerKey); err != nil {
		return err
	}
	b.requestChan <- req
	return nil
}

// AssignRequest assigns an incoming request to an appropriate batch.
func (b *batcher) AssignRequest(req *Request) *Batch {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	batch := b.getIdleBatch(req.UniqueKey)
	if batch == nil {
		batch = b.createNewBatch(req)
	}

	batch.mutex.Lock()
	batch.requests = append(batch.requests, req)
	if len(batch.requests) == batch.maxReq {
		batch.mutex.Unlock()
		go batch.Execute()
	} else {
		batch.mutex.Unlock()
		go batch.WaitAndExecute()
	}

	return batch
}

// getIdleBatch retrieves an idle batch for the given unique key.
func (b *batcher) getIdleBatch(key string) *Batch {
	for _, batch := range b.batches[key] {
		batch.mutex.Lock()
		if batch.state == 0 {
			batch.mutex.Unlock()
			return batch
		}
		batch.mutex.Unlock()
	}
	return nil
}

// createNewBatch creates a new batch and adds it to the heap.
func (b *batcher) createNewBatch(req *Request) *Batch {
	listener, _ := b.GetListener(req.ListenerKey)
	newBatch := &Batch{
		maxReq:       listener.maxReq,
		maxWait:      listener.maxWait,
		run:          listener.run,
		mutexBatcher: b.mutex,
		mutex:        new(sync.RWMutex),
		lastUsed:     time.Now(),
	}

	// Add the new batch to the batcher's map and heap.
	b.batches[req.UniqueKey] = append(b.batches[req.UniqueKey], newBatch)
	heap.Push(b.heap, newBatch)

	return newBatch
}

// CleanupBatches periodically removes stale batches from the heap and map.
func (b *batcher) CleanupBatches() {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for range ticker.C {
		b.mutex.Lock()
		for b.heap.Len() > 0 {
			staleBatch := heap.Pop(b.heap).(*Batch)
			staleBatch.mutex.Lock()

			// Check if the batch is stale and idle
			if time.Since(staleBatch.lastUsed) > b.staleDuration && staleBatch.state == 0 && len(staleBatch.requests) == 0 {
				staleBatch.mutex.Unlock()
				// Remove stale batch
				b.removeBatch(staleBatch)
			} else {
				staleBatch.mutex.Unlock()
				// If not stale, push it back onto the heap
				heap.Push(b.heap, staleBatch)
				break
			}
		}
		b.mutex.Unlock()
	}
}

// removeBatch removes a batch from the batches map.
func (b *batcher) removeBatch(batchToRemove *Batch) {
	for key, batches := range b.batches {
		for i, batch := range batches {
			if batch == batchToRemove {
				b.batches[key] = append(batches[:i], batches[i+1:]...)
				if len(b.batches[key]) == 0 {
					delete(b.batches, key)
				}
				return
			}
		}
	}
}

// Execute processes the batch immediately.
func (b *Batch) Execute() bool {
	b.mutexBatcher.Lock()
	if b.state == 1 {
		b.mutexBatcher.Unlock()
		return false
	}
	b.state = 1
	b.mutexBatcher.Unlock()

	done := make(chan struct{})
	go b.run(context.Background(), b.requests, done)
	<-done

	// Clear the batch after execution
	b.mutex.Lock()
	b.requests = []*Request{}
	b.mutex.Unlock()

	// Reset state and update the lastUsed time
	b.mutexBatcher.Lock()
	b.state = 0
	b.lastUsed = time.Now()
	b.mutexBatcher.Unlock()

	return true
}

// WaitAndExecute waits for a specified duration before executing the batch.
func (b *Batch) WaitAndExecute() {
	time.Sleep(b.maxWait)

	b.mutex.Lock()
	if len(b.requests) > 0 {
		b.mutex.Unlock()
		b.Execute()
		return
	}
	b.mutex.Unlock()
}
