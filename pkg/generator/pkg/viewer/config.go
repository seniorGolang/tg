package viewer

type ConfigState struct {
	Indent   string
	MaxDepth int
}

var Config = ConfigState{Indent: " "}
