package command

const chunk = 249

type Command interface {
	GetBytes() [][]byte
	GetName() string
	ValidateCommand([]byte, int) error
	GetSize() int
}
