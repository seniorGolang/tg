package header

const (
XRequestID  contextKey = "X-Request-Id"
)

type contextKey string

func (key contextKey) String() string {
return string(key)
}
