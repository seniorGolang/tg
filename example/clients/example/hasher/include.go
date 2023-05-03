package hasher

type Hasher interface {
	Hash() (hash uint64, err error)
}

type Inc interface {
	HashInclude(field string, v interface{}) (has bool, err error)
}

type IncMap interface {
	HashIncludeMap(field string, k, v interface{}) (has bool, err error)
}
