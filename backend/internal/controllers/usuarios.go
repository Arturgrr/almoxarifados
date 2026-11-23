package controllers

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"almoxarifados-a2g/internal/auth"
	"almoxarifados-a2g/internal/database"
)

type UsuarioReq struct {
	Nome     string  `json:"nome" required:"true" example:"João da Silva"`
	Email    string  `json:"email" required:"true" format:"email" example:"joao@a2g.com.br"`
	Telefone *string `json:"telefone" example:"(73) 99999-0000"`
	Perfil   string  `json:"perfil" required:"true" enum:"admin,obra"`
}

type NovoUsuarioReq struct {
	UsuarioReq
	Senha string `json:"senha" required:"true" minLength:"8" maxLength:"72" description:"Senha inicial, informada pelo administrador ao funcionário"`
}

func (r *UsuarioReq) validar(v validacao) {
	r.Nome = strings.TrimSpace(r.Nome)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Telefone = texto(r.Telefone)

	v.obrigatorio("nome", r.Nome)
	v.obrigatorio("email", r.Email)
	v.se(r.Email != "" && !emailValido(r.Email), "email", "Informe um e-mail válido")
	v.se(!umDe(r.Perfil, perfilAdmin, perfilObra), "perfil", "Escolha Administrador ou Usuário de obra")
}

func (h *Controller) ListUsuarios(c *fiber.Ctx) error {
	ativo, err := queryBool(c, "ativo")
	if err != nil {
		return err
	}
	usuarios, err := h.db.ListUsuarios(c.UserContext(), database.ListUsuariosParams{
		Busca:  queryTexto(c, "busca"),
		Perfil: queryTexto(c, "perfil"),
		Ativo:  ativo,
	})
	if err != nil {
		return err
	}
	return c.JSON(usuarios)
}

func (h *Controller) CreateUsuario(c *fiber.Ctx) error {
	var req NovoUsuarioReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}
	v := validacao{}
	req.validar(v)
	validarSenha(v, "senha", req.Senha)
	if err := v.erro(); err != nil {
		return err
	}

	hash, err := auth.HashSenha(req.Senha)
	if err != nil {
		return err
	}
	u, err := h.db.CreateUsuario(c.UserContext(), database.CreateUsuarioParams{
		Nome:      req.Nome,
		Email:     req.Email,
		Telefone:  req.Telefone,
		Perfil:    req.Perfil,
		SenhaHash: hash,
	})
	if err != nil {
		return err
	}
	h.auditar(c, "criar", "usuario", u.ID)
	return c.Status(fiber.StatusCreated).JSON(u)
}

func (h *Controller) GetUsuario(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	u, err := h.db.GetUsuario(c.UserContext(), id)
	if err != nil {
		return err
	}
	return c.JSON(u)
}

func (h *Controller) UpdateUsuario(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req UsuarioReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}
	v := validacao{}
	req.validar(v)
	if err := v.erro(); err != nil {
		return err
	}

	ctx := c.UserContext()
	atual, err := h.db.GetUsuario(ctx, id)
	if err != nil {
		return err
	}
	if atual.Perfil == perfilAdmin && atual.Ativo && req.Perfil != perfilAdmin {
		if err := h.garantirOutroAdmin(c); err != nil {
			return err
		}
	}

	u, err := h.db.UpdateUsuario(ctx, database.UpdateUsuarioParams{
		ID:       id,
		Nome:     req.Nome,
		Email:    req.Email,
		Telefone: req.Telefone,
		Perfil:   req.Perfil,
	})
	if err != nil {
		return err
	}
	h.auditar(c, "editar", "usuario", u.ID)
	return c.JSON(u)
}

func (h *Controller) DesativarUsuario(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	if id == usuarioLogado(c).ID {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "Você não pode desativar o próprio usuário")
	}

	ctx := c.UserContext()
	atual, err := h.db.GetUsuario(ctx, id)
	if err != nil {
		return err
	}
	if atual.Perfil == perfilAdmin && atual.Ativo {
		if err := h.garantirOutroAdmin(c); err != nil {
			return err
		}
	}

	if err := h.db.SetUsuarioAtivo(ctx, database.SetUsuarioAtivoParams{ID: id, Ativo: false}); err != nil {
		return err
	}
	if err := h.db.DeleteSessoesUsuario(ctx, id); err != nil {
		return err
	}
	h.auditar(c, "desativar", "usuario", id)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Controller) ReativarUsuario(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	ctx := c.UserContext()
	if _, err := h.db.GetUsuario(ctx, id); err != nil {
		return err
	}
	if err := h.db.SetUsuarioAtivo(ctx, database.SetUsuarioAtivoParams{ID: id, Ativo: true}); err != nil {
		return err
	}
	h.auditar(c, "reativar", "usuario", id)
	return c.SendStatus(fiber.StatusNoContent)
}

type RedefinirSenhaReq struct {
	Senha string `json:"senha" required:"true" minLength:"8" maxLength:"72" description:"Nova senha provisória"`
}

func (h *Controller) RedefinirSenha(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req RedefinirSenhaReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}
	v := validacao{}
	validarSenha(v, "senha", req.Senha)
	if err := v.erro(); err != nil {
		return err
	}

	ctx := c.UserContext()
	if _, err := h.db.GetUsuario(ctx, id); err != nil {
		return err
	}
	hash, err := auth.HashSenha(req.Senha)
	if err != nil {
		return err
	}
	err = h.db.SetUsuarioSenha(ctx, database.SetUsuarioSenhaParams{ID: id, SenhaHash: hash, TrocarSenha: true})
	if err != nil {
		return err
	}
	if err := h.db.DeleteSessoesUsuario(ctx, id); err != nil {
		return err
	}
	h.auditar(c, "redefinir_senha", "usuario", id)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Controller) garantirOutroAdmin(c *fiber.Ctx) error {
	n, err := h.db.CountAdminsAtivos(c.UserContext())
	if err != nil {
		return err
	}
	if n <= 1 {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "O sistema precisa ter pelo menos um administrador ativo")
	}
	return nil
}
