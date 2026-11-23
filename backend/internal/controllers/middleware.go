package controllers

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"almoxarifados-a2g/internal/auth"
	"almoxarifados-a2g/internal/database"
)

const (
	cookieSessao  = "sessao"
	duracaoSessao = 8 * time.Hour
)

var errSessao = fiber.NewError(fiber.StatusUnauthorized, "Sessão expirada. Faça login novamente.")

func (h *Controller) Autenticado(c *fiber.Ctx) error {
	token := c.Cookies(cookieSessao)
	if token == "" {
		return errSessao
	}

	ctx := c.UserContext()
	hash := auth.HashToken(token)
	u, err := h.db.TocarSessao(ctx, database.TocarSessaoParams{
		TokenHash: hash,
		ExpiraEm:  time.Now().Add(duracaoSessao),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		h.limparCookie(c)
		return errSessao
	}
	if err != nil {
		return err
	}
	if !u.Ativo {
		_ = h.db.DeleteSessao(ctx, hash)
		h.limparCookie(c)
		return fiber.NewError(fiber.StatusUnauthorized, msgUsuarioDesativado)
	}

	c.Locals(localUsuario, u)
	return c.Next()
}

func (h *Controller) SomenteAdmin(c *fiber.Ctx) error {
	if !ehAdmin(c) {
		return fiber.NewError(fiber.StatusForbidden, "Você não tem permissão para esta ação")
	}
	return c.Next()
}

func (h *Controller) AcessoEmpreendimento(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}

	ctx := c.UserContext()
	emp, err := h.db.GetEmpreendimento(ctx, id)
	if err != nil {
		return err
	}

	if !ehAdmin(c) {
		membro, err := h.db.IsMembro(ctx, database.IsMembroParams{
			EmpreendimentoID: emp.ID,
			UsuarioID:        usuarioLogado(c).ID,
		})
		if err != nil {
			return err
		}
		if !membro {
			return fiber.NewError(fiber.StatusForbidden, "Você não tem acesso a este empreendimento")
		}
	}

	c.Locals(localEmpreendimento, emp)
	return c.Next()
}

func (h *Controller) definirCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     cookieSessao,
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.cookieSecure,
		SameSite: fiber.CookieSameSiteStrictMode,
	})
}

func (h *Controller) limparCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     cookieSessao,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.cookieSecure,
		SameSite: fiber.CookieSameSiteStrictMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
