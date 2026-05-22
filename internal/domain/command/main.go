package command

const chunk = 249

type WriteValidation struct {
	Size  int
	Bytes []byte
}

type Command interface {
	GetBytes() [][]byte
	GetName() string
	ValidateCommand([]byte, int) error
	ValidateWrite() WriteValidation
	SetCount(num int64)
}

// RegionKey identifies a display region by its position and dimensions.
type RegionKey struct {
	X, Y, Width, Height int
}

// RegionIdentifier is implemented by commands that target a specific display region.
// UPDATE_BITMAP commands implement this interface to enable per-region deduplication.
type RegionIdentifier interface {
	GetRegion() RegionKey
}
