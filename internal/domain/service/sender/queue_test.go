package sender

import (
	"sync"
	"testing"

	"github.com/alexwbaule/turing-screen/internal/domain/command"
)

// mockCommand is a non-UPDATE_BITMAP command for testing.
type mockCommand struct {
	name string
}

func (m *mockCommand) GetBytes() [][]byte                { return nil }
func (m *mockCommand) GetName() string                   { return m.name }
func (m *mockCommand) ValidateCommand([]byte, int) error { return nil }
func (m *mockCommand) ValidateWrite() command.WriteValidation {
	return command.WriteValidation{}
}
func (m *mockCommand) SetCount(int64) {}

// mockRegionCommand is an UPDATE_BITMAP command that implements RegionIdentifier.
type mockRegionCommand struct {
	mockCommand
	region command.RegionKey
}

func (m *mockRegionCommand) GetRegion() command.RegionKey {
	return m.region
}

func TestNewRegionQueue(t *testing.T) {
	q := NewRegionQueue(128)
	if q == nil {
		t.Fatal("expected non-nil queue")
	}
	if q.maxSize != 128 {
		t.Errorf("maxSize: got %d, want 128", q.maxSize)
	}
	if q.Len() != 0 {
		t.Errorf("initial length: got %d, want 0", q.Len())
	}
}

func TestEnqueueDequeue_BasicFIFO(t *testing.T) {
	q := NewRegionQueue(128)

	cmd1 := &mockCommand{name: "HELLO"}
	cmd2 := &mockCommand{name: "RESTART"}
	cmd3 := &mockCommand{name: "TURNOFF"}

	q.Enqueue(cmd1)
	q.Enqueue(cmd2)
	q.Enqueue(cmd3)

	if q.Len() != 3 {
		t.Fatalf("length: got %d, want 3", q.Len())
	}

	got, ok := q.Dequeue()
	if !ok || got.GetName() != "HELLO" {
		t.Errorf("first dequeue: got %v, want HELLO", got)
	}

	got, ok = q.Dequeue()
	if !ok || got.GetName() != "RESTART" {
		t.Errorf("second dequeue: got %v, want RESTART", got)
	}

	got, ok = q.Dequeue()
	if !ok || got.GetName() != "TURNOFF" {
		t.Errorf("third dequeue: got %v, want TURNOFF", got)
	}

	_, ok = q.Dequeue()
	if ok {
		t.Error("expected false from empty queue dequeue")
	}
}

func TestEnqueue_RegionDeduplication(t *testing.T) {
	q := NewRegionQueue(128)

	region := command.RegionKey{X: 10, Y: 20, Width: 100, Height: 50}

	cmd1 := &mockRegionCommand{
		mockCommand: mockCommand{name: "UPDATE_BITMAP_1"},
		region:      region,
	}
	cmd2 := &mockRegionCommand{
		mockCommand: mockCommand{name: "UPDATE_BITMAP_2"},
		region:      region,
	}

	q.Enqueue(cmd1)
	q.Enqueue(cmd2)

	// Should only have 1 entry since same region
	if q.Len() != 1 {
		t.Fatalf("length after dedup: got %d, want 1", q.Len())
	}

	got, ok := q.Dequeue()
	if !ok {
		t.Fatal("expected command from dequeue")
	}
	if got.GetName() != "UPDATE_BITMAP_2" {
		t.Errorf("dequeued command: got %q, want %q", got.GetName(), "UPDATE_BITMAP_2")
	}
}

func TestEnqueue_DifferentRegionsNotDeduplicated(t *testing.T) {
	q := NewRegionQueue(128)

	cmd1 := &mockRegionCommand{
		mockCommand: mockCommand{name: "UPDATE_1"},
		region:      command.RegionKey{X: 0, Y: 0, Width: 100, Height: 50},
	}
	cmd2 := &mockRegionCommand{
		mockCommand: mockCommand{name: "UPDATE_2"},
		region:      command.RegionKey{X: 100, Y: 0, Width: 100, Height: 50},
	}

	q.Enqueue(cmd1)
	q.Enqueue(cmd2)

	if q.Len() != 2 {
		t.Fatalf("length: got %d, want 2", q.Len())
	}
}

func TestEnqueue_MixedCommandsPreserveOrder(t *testing.T) {
	q := NewRegionQueue(128)

	hello := &mockCommand{name: "HELLO"}
	update := &mockRegionCommand{
		mockCommand: mockCommand{name: "UPDATE_BITMAP"},
		region:      command.RegionKey{X: 10, Y: 20, Width: 50, Height: 30},
	}
	restart := &mockCommand{name: "RESTART"}

	q.Enqueue(hello)
	q.Enqueue(update)
	q.Enqueue(restart)

	got1, _ := q.Dequeue()
	got2, _ := q.Dequeue()
	got3, _ := q.Dequeue()

	if got1.GetName() != "HELLO" {
		t.Errorf("first: got %q, want HELLO", got1.GetName())
	}
	if got2.GetName() != "UPDATE_BITMAP" {
		t.Errorf("second: got %q, want UPDATE_BITMAP", got2.GetName())
	}
	if got3.GetName() != "RESTART" {
		t.Errorf("third: got %q, want RESTART", got3.GetName())
	}
}

func TestEnqueue_FullQueueDropsOldestBitmap(t *testing.T) {
	q := NewRegionQueue(3)

	// Fill with UPDATE_BITMAP commands for different regions
	cmd1 := &mockRegionCommand{
		mockCommand: mockCommand{name: "UPDATE_1"},
		region:      command.RegionKey{X: 0, Y: 0, Width: 10, Height: 10},
	}
	cmd2 := &mockRegionCommand{
		mockCommand: mockCommand{name: "UPDATE_2"},
		region:      command.RegionKey{X: 10, Y: 0, Width: 10, Height: 10},
	}
	cmd3 := &mockRegionCommand{
		mockCommand: mockCommand{name: "UPDATE_3"},
		region:      command.RegionKey{X: 20, Y: 0, Width: 10, Height: 10},
	}

	q.Enqueue(cmd1)
	q.Enqueue(cmd2)
	q.Enqueue(cmd3)

	if q.Len() != 3 {
		t.Fatalf("length: got %d, want 3", q.Len())
	}

	// Enqueue a non-replaceable command when full - should drop oldest bitmap
	hello := &mockCommand{name: "HELLO"}
	q.Enqueue(hello)

	if q.Len() != 3 {
		t.Fatalf("length after overflow: got %d, want 3", q.Len())
	}

	// The oldest UPDATE_BITMAP (UPDATE_1) should have been dropped
	got1, _ := q.Dequeue()
	got2, _ := q.Dequeue()
	got3, _ := q.Dequeue()

	if got1.GetName() != "UPDATE_2" {
		t.Errorf("first: got %q, want UPDATE_2", got1.GetName())
	}
	if got2.GetName() != "UPDATE_3" {
		t.Errorf("second: got %q, want UPDATE_3", got2.GetName())
	}
	if got3.GetName() != "HELLO" {
		t.Errorf("third: got %q, want HELLO", got3.GetName())
	}
}

func TestEnqueue_FullQueueNewBitmapReplaces(t *testing.T) {
	q := NewRegionQueue(2)

	region := command.RegionKey{X: 10, Y: 20, Width: 50, Height: 30}

	cmd1 := &mockRegionCommand{
		mockCommand: mockCommand{name: "UPDATE_OLD"},
		region:      region,
	}
	other := &mockCommand{name: "HELLO"}

	q.Enqueue(cmd1)
	q.Enqueue(other)

	// Queue is full (2/2). Enqueue same region - should replace in-place, not overflow
	cmd2 := &mockRegionCommand{
		mockCommand: mockCommand{name: "UPDATE_NEW"},
		region:      region,
	}
	q.Enqueue(cmd2)

	if q.Len() != 2 {
		t.Fatalf("length: got %d, want 2", q.Len())
	}

	got1, _ := q.Dequeue()
	got2, _ := q.Dequeue()

	if got1.GetName() != "UPDATE_NEW" {
		t.Errorf("first: got %q, want UPDATE_NEW", got1.GetName())
	}
	if got2.GetName() != "HELLO" {
		t.Errorf("second: got %q, want HELLO", got2.GetName())
	}
}

func TestDrain(t *testing.T) {
	q := NewRegionQueue(128)

	q.Enqueue(&mockCommand{name: "A"})
	q.Enqueue(&mockCommand{name: "B"})
	q.Enqueue(&mockCommand{name: "C"})

	if q.Len() != 3 {
		t.Fatalf("length before drain: got %d, want 3", q.Len())
	}

	q.Drain()

	if q.Len() != 0 {
		t.Errorf("length after drain: got %d, want 0", q.Len())
	}

	_, ok := q.Dequeue()
	if ok {
		t.Error("expected false from drained queue")
	}
}

func TestDequeue_EmptyQueue(t *testing.T) {
	q := NewRegionQueue(128)

	_, ok := q.Dequeue()
	if ok {
		t.Error("expected false from empty queue")
	}
}

func TestConcurrentAccess(t *testing.T) {
	q := NewRegionQueue(128)
	var wg sync.WaitGroup

	// Spawn multiple goroutines enqueuing concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cmd := &mockRegionCommand{
					mockCommand: mockCommand{name: "UPDATE"},
					region:      command.RegionKey{X: id, Y: j, Width: 10, Height: 10},
				}
				q.Enqueue(cmd)
			}
		}(i)
	}

	// Spawn goroutines dequeuing concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				q.Dequeue()
			}
		}()
	}

	wg.Wait()

	// Queue should not exceed maxSize
	if q.Len() > 128 {
		t.Errorf("queue exceeded maxSize: got %d", q.Len())
	}
}

func TestCapacityInvariant(t *testing.T) {
	q := NewRegionQueue(5)

	// Enqueue more than maxSize items
	for i := 0; i < 20; i++ {
		cmd := &mockRegionCommand{
			mockCommand: mockCommand{name: "UPDATE"},
			region:      command.RegionKey{X: i, Y: 0, Width: 10, Height: 10},
		}
		q.Enqueue(cmd)

		if q.Len() > 5 {
			t.Fatalf("queue exceeded maxSize at iteration %d: got %d", i, q.Len())
		}
	}
}
