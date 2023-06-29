package router

import "github.com/gin-gonic/gin"

type RouteGroup struct {
	Prefix    string
	RouteList []Route
	Comment   string
}

func NewRoute(method, path string, handler gin.HandlerFunc, comment string) Route {
	return Route{
		Method:      method,
		Path:        path,
		HandlerFunc: handler,
		Comment:     comment,
	}
}

type Route struct {
	Method      string
	Path        string
	HandlerFunc gin.HandlerFunc
	Comment     string
}

func AddRouteGroup(engine *gin.Engine, group RouteGroup) {
	api := engine.Group(group.Prefix)
	for _, r := range group.RouteList {
		api.Handle(r.Method, r.Path, r.HandlerFunc)
	}
}
