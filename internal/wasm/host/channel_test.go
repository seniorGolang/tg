package host

import "testing"

func TestCallChannelCloseIsIdempotent(t *testing.T) {
	cc := NewCallChannel(t.Context(), nil)

	cc.Close()
	cc.Close()
}

func TestCallChannelCallAfterCloseReturnsError(t *testing.T) {
	cc := NewCallChannel(t.Context(), nil)
	cc.Close()

	result := <-cc.Call("handler", nil)
	if result.Error == nil {
		t.Fatal("expected call after close to return error")
	}
}
