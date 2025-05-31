package linkBuffer

type Options struct {
	bufferSize int
	handler  func(*Buffer)
}