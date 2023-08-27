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
