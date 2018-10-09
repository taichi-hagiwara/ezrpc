package ezrpc

import "reflect"

// Service は、リモート呼び出しを定義する。
type Service interface {
	Init(*ServiceRegistry) error
}

// ServiceRegistry は、リモート呼び出しを管理する。
type ServiceRegistry struct {
	endpoints map[string]*endpointType
}

type endpointType struct {
	Args   reflect.Type
	Result reflect.Type
}

// Register は、リモート呼び出し先の引数と返り値の型を登録する。
func (r *ServiceRegistry) Register(name string, args, result interface{}) {
	r.endpoints[name] = &endpointType{
		Args:   reflect.TypeOf(args).Elem(),
		Result: reflect.TypeOf(result).Elem(),
	}
}

func newServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		endpoints: make(map[string]*endpointType),
	}
}
