package types

type Queue struct {
	Name       string
	Attributes map[string]string
	Tags       map[string]string
}

func NewQueue(name string) *Queue {
	return &Queue{
		Name:       name,
		Attributes: make(map[string]string),
		Tags:       make(map[string]string),
	}
}
