package astra

import "github.com/seniorGolang/tg/v2/pkg/astra/types"

func mergeImports(bunch ...[]*types.Import) []*types.Import {

	set := make(map[string]*types.Import)
	for i := range bunch {
		for j := range bunch[i] {
			if imp, ok := set[bunch[i][j].Package]; ok {
				// import already exist, update pointer to types
				bunch[i][j] = imp
			} else {
				// add new import to set
				set[bunch[i][j].Package] = bunch[i][j]
			}
		}
	}
	var result = make([]*types.Import, 0, len(set))
	for _, v := range set {
		result = append(result, v)
	}
	return result
}
