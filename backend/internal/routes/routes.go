package routes

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"almoxarifados-a2g/internal/controllers"
	"almoxarifados-a2g/internal/database"
	"almoxarifados-a2g/internal/docs"
)

type Config struct {
	FrontendURL  string
	CookieSecure bool
}

func New(db *database.Store, cfg Config) (*fiber.App, error) {
	spec, err := docs.Gerar("/")
	if err != nil {
		return nil, err
	}
	specJSON, err := docs.JSON(spec)
	if err != nil {
		return nil, err
	}
	paginaSwagger := docs.PaginaSwagger(specJSON)

	app := fiber.New(fiber.Config{
		AppName:      "Almoxarifado A2G",
		ErrorHandler: controllers.ErrorHandler,
		BodyLimit:    1 << 20,
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(helmet.New(helmet.Config{
		Next: func(c *fiber.Ctx) bool { return strings.HasPrefix(c.Path(), "/docs") },
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.FrontendURL,
		AllowCredentials: true,
	}))

	app.Get("/docs", func(c *fiber.Ctx) error {
		c.Type("html", "utf-8")
		return c.Send(paginaSwagger)
	})
	app.Get("/docs/openapi.json", func(c *fiber.Ctx) error {
		c.Type("json", "utf-8")
		return c.Send(specJSON)
	})

	c := controllers.New(db, cfg.CookieSecure)
	api := app.Group("/api")

	api.Get("/health", func(ctx *fiber.Ctx) error { return ctx.JSON(fiber.Map{"status": "ok"}) })

	api.Post("/auth/login", c.Login)
	api.Post("/auth/logout", c.Autenticado, c.Logout)
	api.Get("/auth/me", c.Autenticado, c.Me)
	api.Put("/auth/senha", c.Autenticado, c.AlterarSenha)

	usuarios := api.Group("/usuarios", c.Autenticado, c.SomenteAdmin)
	usuarios.Get("", c.ListUsuarios)
	usuarios.Post("", c.CreateUsuario)
	usuarios.Get("/:id", c.GetUsuario)
	usuarios.Put("/:id", c.UpdateUsuario)
	usuarios.Post("/:id/desativar", c.DesativarUsuario)
	usuarios.Post("/:id/reativar", c.ReativarUsuario)
	usuarios.Put("/:id/senha", c.RedefinirSenha)

	categorias := api.Group("/categorias", c.Autenticado)
	categorias.Get("", c.ListCategorias)
	categorias.Post("", c.SomenteAdmin, c.CreateCategoria)
	categorias.Put("/:id", c.SomenteAdmin, c.UpdateCategoria)

	itens := api.Group("/itens", c.Autenticado)
	itens.Get("", c.ListItens)
	itens.Post("", c.CreateItem)
	itens.Get("/:id", c.GetItem)
	itens.Put("/:id", c.SomenteAdmin, c.UpdateItem)
	itens.Delete("/:id", c.SomenteAdmin, c.DeleteItem)
	itens.Post("/:id/desativar", c.SomenteAdmin, c.DesativarItem)
	itens.Post("/:id/reativar", c.SomenteAdmin, c.ReativarItem)
	itens.Get("/:id/saldos", c.SomenteAdmin, c.SaldoConsolidadoItem)

	fornecedores := api.Group("/fornecedores", c.Autenticado)
	fornecedores.Get("", c.ListFornecedores)
	fornecedores.Post("", c.CreateFornecedor)
	fornecedores.Get("/:id", c.GetFornecedor)
	fornecedores.Put("/:id", c.SomenteAdmin, c.UpdateFornecedor)
	fornecedores.Delete("/:id", c.SomenteAdmin, c.DeleteFornecedor)
	fornecedores.Post("/:id/desativar", c.SomenteAdmin, c.DesativarFornecedor)
	fornecedores.Post("/:id/reativar", c.SomenteAdmin, c.ReativarFornecedor)

	emp := api.Group("/empreendimentos", c.Autenticado)
	acesso := c.AcessoEmpreendimento
	emp.Get("", c.ListEmpreendimentos)
	emp.Post("", c.SomenteAdmin, c.CreateEmpreendimento)
	emp.Get("/:id", acesso, c.GetEmpreendimento)
	emp.Put("/:id", c.SomenteAdmin, acesso, c.UpdateEmpreendimento)
	emp.Post("/:id/encerrar", c.SomenteAdmin, acesso, c.EncerrarEmpreendimento)
	emp.Post("/:id/reabrir", c.SomenteAdmin, acesso, c.ReabrirEmpreendimento)

	emp.Get("/:id/equipe", acesso, c.ListEquipe)
	emp.Post("/:id/equipe", c.SomenteAdmin, acesso, c.AddMembro)
	emp.Delete("/:id/equipe/:usuarioId", c.SomenteAdmin, acesso, c.RemoveMembro)

	emp.Get("/:id/saldos", acesso, c.ListSaldos)
	emp.Get("/:id/destinos", acesso, c.ListDestinos)
	emp.Get("/:id/movimentacoes", acesso, c.ListMovimentacoes)
	emp.Post("/:id/entradas", acesso, c.RegistrarEntrada)
	emp.Post("/:id/saidas", acesso, c.RegistrarSaida)
	emp.Post("/:id/perdas", acesso, c.RegistrarPerda)
	emp.Post("/:id/movimentacoes/:movId/estorno", acesso, c.Estornar)

	emp.Get("/:id/inventarios", acesso, c.ListInventarios)
	emp.Post("/:id/inventarios", acesso, c.AbrirInventario)
	emp.Get("/:id/inventarios/:invId", acesso, c.GetInventario)
	emp.Put("/:id/inventarios/:invId/contagem", acesso, c.SalvarContagem)
	emp.Post("/:id/inventarios/:invId/finalizar", acesso, c.FinalizarInventario)
	emp.Post("/:id/inventarios/:invId/cancelar", acesso, c.CancelarInventario)

	admin := api.Group("/admin", c.Autenticado, c.SomenteAdmin)
	admin.Get("/painel", c.Painel)
	admin.Get("/perdas-anormais", c.PerdasAnormais)

	app.Use(func(ctx *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "Rota não encontrada")
	})

	return app, nil
}
