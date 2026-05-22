package sender

import (
	"sync"

	"github.com/alexwbaule/turing-screen/internal/domain/command"
)

// RegionQueue is a thread-safe command queue that deduplicates UPDATE_BITMAP
// commands targeting the same display region. When a new UPDATE_BITMAP command
// is enqueued for a region that already has a pending command, the older command
// is replaced in-place. Non-UPDATE_BITMAP commands are appended normally.
type RegionQueue struct {
	mu      sync.Mutex
	items   []command.Command
	regions map[command.RegionKey]int // maps region to index in items
	maxSize int
	notify  chan struct{} // signaled on Enqueue to wake the Worker
}

// NewRegionQueue creates a new RegionQueue with the given maximum capacity.
func NewRegionQueue(maxSize int) *RegionQueue {
	return &RegionQueue{
		items:   make([]command.Command, 0, maxSize),
		regions: make(map[command.RegionKey]int),
		maxSize: maxSize,
		notify:  make(chan struct{}, 1),
	}
}

// Notify returns the notification channel that is signaled when items are enqueued.
// The Worker selects on this channel to wake up and process commands.
func (q *RegionQueue) Notify() <-chan struct{} {
	return q.notify
}

// Enqueue adds a command to the queue. For UPDATE_BITMAP commands (those
// implementing RegionIdentifier), if the same region already has a pending
// command, the older command is replaced in-place. For non-UPDATE_BITMAP
// commands, the command is appended to the end of the queue.
//
// When the queue is full and a non-replaceable command arrives, the oldest
// UPDATE_BITMAP command is dropped to make room.
func (q *RegionQueue) Enqueue(cmd command.Command) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if this is a region-identifiable command (UPDATE_BITMAP)
	if ri, ok := cmd.(command.RegionIdentifier); ok {
		key := ri.GetRegion()

		// If same region already queued, replace in-place
		if idx, exists := q.regions[key]; exists {
			q.items[idx] = cmd
			q.signal()
			return
		}

		// If queue is full, drop the oldest UPDATE_BITMAP to make room
		if len(q.items) >= q.maxSize {
			q.dropOldestBitmap()
		}

		// Append and track the region
		q.regions[key] = len(q.items)
		q.items = append(q.items, cmd)
		q.signal()
		return
	}

	// Non-UPDATE_BITMAP command: if queue is full, drop oldest UPDATE_BITMAP
	if len(q.items) >= q.maxSize {
		q.dropOldestBitmap()
	}

	q.items = append(q.items, cmd)
	q.signal()
}

// signal sends a non-blocking notification to wake the Worker.
// Must be called with the mutex held.
func (q *RegionQueue) signal() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

// Dequeue removes and returns the next command in FIFO order.
// Returns false if the queue is empty.
func (q *RegionQueue) Dequeue() (command.Command, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return nil, false
	}

	cmd := q.items[0]
	q.items = q.items[1:]

	// Update region map: remove dequeued entry and adjust indices
	q.rebuildRegionMap()

	return cmd, true
}

// Len returns the current number of commands in the queue.
func (q *RegionQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// Drain removes all commands from the queue.
func (q *RegionQueue) Drain() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = q.items[:0]
	q.regions = make(map[command.RegionKey]int)
}

// dropOldestBitmap removes the oldest UPDATE_BITMAP command from the queue.
// Must be called with the mutex held.
func (q *RegionQueue) dropOldestBitmap() {
	for i, item := range q.items {
		if ri, ok := item.(command.RegionIdentifier); ok {
			key := ri.GetRegion()
			delete(q.regions, key)
			q.items = append(q.items[:i], q.items[i+1:]...)
			q.rebuildRegionMap()
			return
		}
	}
	// If no UPDATE_BITMAP found, drop the oldest entry regardless
	if len(q.items) > 0 {
		q.items = q.items[1:]
		q.rebuildRegionMap()
	}
}

// rebuildRegionMap reconstructs the region index map after items shift.
// Must be called with the mutex held.
func (q *RegionQueue) rebuildRegionMap() {
	q.regions = make(map[command.RegionKey]int, len(q.regions))
	for i, item := range q.items {
		if ri, ok := item.(command.RegionIdentifier); ok {
			q.regions[ri.GetRegion()] = i
		}
	}
}
