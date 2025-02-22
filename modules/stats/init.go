package stats

import (
	"github.com/andycai/unitool/core"
	"github.com/andycai/unitool/enum"
	"github.com/andycai/unitool/models"
	"github.com/gofiber/fiber/v2"
)

var app *core.App

type statsModule struct {
	core.BaseModule
}

func init() {
	core.RegisterModule(&statsModule{}, enum.ModulePriorityStats)
}

func (m *statsModule) Awake(a *core.App) error {
	app = a

	// 数据迁移
	return app.DB.AutoMigrate(&models.StatsRecord{}, &models.StatsInfo{})
}

func (m *statsModule) AddPublicRouters() error {
	app.RouterPublicApi.Post("/stats", CreateStats)

	return nil
}

func (m *statsModule) AddAuthRouters() error {

	app.RouterApi.Get("/stats", app.HasPermission("stats:list"), getStats)

	app.RouterApi.Delete("/stats/before", app.HasPermission("stats:delete"), deleteStatsBefore)

	app.RouterApi.Get("/stats/details", app.HasPermission("stats:list"), getStatDetails)

	app.RouterApi.Delete("/stats/:id", app.HasPermission("stats:delete"), deleteStat)

	// admin
	app.RouterAdmin.Get("/stats", app.HasPermission("stats:list"), func(c *fiber.Ctx) error {
		return c.Render("admin/stats", fiber.Map{
			"Title": "游戏统计",
			"Scripts": []string{
				"/static/js/chart-4.4.4.js",
				"/static/js/hammer-2.0.8.js",
				"/static/js/chartjs-plugin-zoom.min.js",
				"/static/js/chartjs-adapter-date-fns.bundle.min.js",
				"/static/js/admin/stats.js",
			},
		}, "admin/layout")
	})

	return nil
}
