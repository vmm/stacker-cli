package stacker

// StackParam represents a stack parameter
type StackParam interface {
	Key() string
	Value() string
	UsePrevious() bool
}

// StackParams represents an set of StackParam
type StackParams []StackParam

// Stack interface for creating and updating stacks
type Stack interface {
	Name() string
	Region() string
	Params() ([]StackParam, error)
	TemplateBody() string
	Capabilities() []string
}

// Sortable list of Stacks
type StackList []Stack

func (s StackList) Len() int {
	return len(s)
}

func (s StackList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s StackList) Less(i, j int) bool {
	return s[i].Name() < s[j].Name()
}
