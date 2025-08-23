package zapx

import "github.com/HoangAnhNguyen269/loggerkit/provider/zapx/corefactories"

var getRegistry = func() corefactories.Registry {
	return corefactories.DefaultRegistry()
}

// UseFactoryRegistry allows tests to inject a custom registry.
func UseFactoryRegistry(r corefactories.Registry) {
	getRegistry = func() corefactories.Registry { return r }
}
