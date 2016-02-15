package input

type Input interface {
	ReadLine() (error, []byte)
	Close()
}
