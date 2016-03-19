package policy

type ReaderWriter interface {
	Read() (string, error)
	Write(content string) error
}
