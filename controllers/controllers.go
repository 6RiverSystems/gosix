package controllers

import "go.6river.tech/gosix/registry"

func AddCommonControllers(r *registry.Registry) {
	r.AddController(&KillController{})
	r.AddController(&LogController{})
	r.AddController(&UptimeController{})
}
