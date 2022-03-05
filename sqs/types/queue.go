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

func (queue *Queue) HasAttribute(key string) bool {
	_, ok := queue.Attributes[key]
	return ok
}

type QueueAttribute struct {
	Name  string
	Value string
}

type GetQueueAttributesResult struct {
	Attribute []QueueAttribute
}

type ResponseMetadata struct {
	RequestId string
}

type GetQueueAttributesResponse struct {
	GetQueueAttributesResult GetQueueAttributesResult
	ResponseMetadata         ResponseMetadata
}

func (obj *GetQueueAttributesResponse) AddAttributeIfNotExists(key, value string) {
	for _, attribute := range obj.GetQueueAttributesResult.Attribute {
		if attribute.Name == key {
			return
		}
	}

	obj.GetQueueAttributesResult.Attribute = append(obj.GetQueueAttributesResult.Attribute,
		QueueAttribute{Name: key, Value: value},
	)
}
