package input

type Input interface {
	ReadLine() ([]byte, error)
	Close()
}