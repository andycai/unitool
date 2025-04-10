package user

import (
	"github.com/andycai/goapi/core"
	"github.com/gofiber/fiber/v2"
)

const (
	ModulePriorityUser = 1000 // 系统-用户管理
)

var app *core.App

type userModule struct {
	core.BaseModule
}

func init() {
	core.RegisterModule(&userModule{}, ModulePriorityUser)
}

func (m *userModule) Awake(a *core.App) error {
	app = a
	// 数据迁移
	if err := autoMigrate(); err != nil {
		return err
	}

	return nil
}

func (m *userModule) Start() error {
	return initData()
}

func (m *userModule) AddAuthRouters() error {
	// admin
	app.RouterAdmin.Get("/users", app.HasPermission("user:view"), func(c *fiber.Ctx) error {
		return c.Render("admin/users", fiber.Map{
			"Title": "用户管理",
			"Scripts": []string{
				"/static/js/admin/users.js",
			},
		}, "admin/layout")
	})

	// api
	app.RouterApi.Get("/users", app.HasPermission("user:view"), listUsersHandler)
	app.RouterApi.Post("/users", app.HasPermission("user:create"), createUserHandler)
	app.RouterApi.Put("/users/:id", app.HasPermission("user:update"), updateUserHandler)
	app.RouterApi.Delete("/users/:id", app.HasPermission("user:delete"), deleteUserHandler)

	return nil
}
