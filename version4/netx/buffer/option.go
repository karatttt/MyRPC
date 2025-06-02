package buffer

type Options struct {
	bufferSize int
	handler  func(*Buffer)
}