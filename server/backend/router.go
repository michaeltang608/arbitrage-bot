package backend

import (
	"net/http"
	"ws-quant/pkg/router"
)

func (bs *backendServer) tradeRouteGroup() router.RouteGroup {
	return router.RouteGroup{
		Prefix:  "trade",
		Comment: "交易类",
		RouteList: []router.Route{
			router.NewRoute(http.MethodPost, "/openLimit", bs.openLimit, "open"),
			router.NewRoute(http.MethodPost, "/openStrategy", bs.openLimit, "open"),
			router.NewRoute(http.MethodPost, "/queryLiveOrder", bs.queryLiveOrder, "query"),
			router.NewRoute(http.MethodPost, "/closeMarket", bs.closeMarket, "close"),
			router.NewRoute(http.MethodPost, "/cancel", bs.cancelOrder, "cancel"),
		},
	}
}

func (bs *backendServer) configRouteGroup() router.RouteGroup {
	return router.RouteGroup{
		Prefix:  "config",
		Comment: "配置类",
		RouteList: []router.Route{
			router.NewRoute(http.MethodGet, "/query", bs.getConfig, "查看配置"),
			router.NewRoute(http.MethodPut, "/change", bs.changeConfig, "修改配置"),
			router.NewRoute(http.MethodGet, "/queryExecState", bs.queryExecState, "查看执行state"),
			router.NewRoute(http.MethodGet, "/refreshStrategy", bs.refreshStrategy, "refreshStrategy"),
		},
	}
}

func (bs *backendServer) testRouteGroup() router.RouteGroup {
	return router.RouteGroup{
		Prefix:  "account",
		Comment: "测试类",
		RouteList: []router.Route{
			router.NewRoute(http.MethodPost, "/t1", bs.t1, "第一个测试接口"),
			router.NewRoute(http.MethodGet, "/persistBalances", bs.marginBalances, "第一个测试接口"),
		},
	}
}
