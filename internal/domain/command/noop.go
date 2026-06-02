package command

// NoOpCommand is a command that performs no I/O. The worker processes it
// instantly: GetBytes() returns nil (nothing to write) and ValidateWrite()
// returns Size=0 (nothing to read back).
func NewNoOp() Command {
	return &NoOpCommand{}
}

type NoOpCommand struct{}

func (c *NoOpCommand) GetBytes() [][]byte      { return nil }
func (c *NoOpCommand) GetName() string          { return "NOOP" }
func (c *NoOpCommand) SetCount(int64)            {}
func (c *NoOpCommand) ValidateWrite() WriteValidation {
	return WriteValidation{Size: 0}
}
func (c *NoOpCommand) ValidateCommand([]byte, int) error { return nil }
