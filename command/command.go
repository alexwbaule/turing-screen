package command

type Command interface {
	GetBytes() []byte
	GetName() string
	ValidateCommand([]byte, int) error
	GetSize() int
}
