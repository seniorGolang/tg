package host

import (
	"github.com/tetratelabs/wazero/api"
)

func (h *Host) GetModule() (module api.Module) {

	if h == nil {
		return nil
	}
	return h.Module
}

func (h *Host) GetMalloc() (malloc api.Function) {

	if h == nil {
		return nil
	}
	return h.Malloc
}

func (h *Host) GetFree() (free api.Function) {

	if h == nil {
		return nil
	}
	return h.Free
}
