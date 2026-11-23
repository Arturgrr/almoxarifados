package controllers

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"almoxarifados-a2g/internal/database"
)

type EmpreendimentoReq struct {
	Nome            string      `json:"nome" required:"true" example:"Condomínio Sol"`
	Tipo            string      `json:"tipo" required:"true" enum:"condominio,casas,edificio,outro"`
	Endereco        *string     `json:"endereco"`
	Cidade          *string     `json:"cidade" example:"Porto Seguro"`
	DataInicio      pgtype.Date `json:"data_inicio" required:"true"`
	PrevisaoTermino pgtype.Date `json:"previsao_termino"`
	ResponsavelID   *int64      `json:"responsavel_id" description:"Chefe de obra. Se for usuário de obra, é associado à equipe automaticamente"`
}

func (h *Controller) validarEmpreendimento(ctx context.Context, r *EmpreendimentoReq) (*database.Usuario, error) {
	r.Nome = strings.TrimSpace(r.Nome)
	r.Endereco = texto(r.Endereco)
	r.Cidade = texto(r.Cidade)

	v := validacao{}
	v.obrigatorio("nome", r.Nome)
	v.se(!umDe(r.Tipo, "condominio", "casas", "edificio", "outro"), "tipo", "Escolha o tipo do empreendimento")
	v.se(!r.DataInicio.Valid, "data_inicio", "Campo obrigatório")
	v.se(r.DataInicio.Valid && r.PrevisaoTermino.Valid && r.PrevisaoTermino.Time.Before(r.DataInicio.Time),
		"previsao_termino", "A previsão de término deve ser depois da data de início")

	var responsavel *database.Usuario
	if r.ResponsavelID != nil {
		u, err := h.db.GetUsuario(ctx, *r.ResponsavelID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if err != nil || !u.Ativo {
			v.add("responsavel_id", "Escolha um usuário ativo")
		}
		responsavel = &u
	}
	return responsavel, v.erro()
}

func (h *Controller) ListEmpreendimentos(c *fiber.Ctx) error {
	params := database.ListEmpreendimentosParams{Status: queryTexto(c, "status")}
	if !ehAdmin(c) {
		id := usuarioLogado(c).ID
		params.UsuarioID = &id
	}
	emps, err := h.db.ListEmpreendimentos(c.UserContext(), params)
	if err != nil {
		return err
	}
	return c.JSON(emps)
}

func (h *Controller) CreateEmpreendimento(c *fiber.Ctx) error {
	var req EmpreendimentoReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}
	ctx := c.UserContext()
	responsavel, err := h.validarEmpreendimento(ctx, &req)
	if err != nil {
		return err
	}

	var emp database.Empreendimento
	err = h.db.Tx(ctx, func(q *database.Queries) error {
		emp, err = q.CreateEmpreendimento(ctx, database.CreateEmpreendimentoParams{
			Nome:            req.Nome,
			Tipo:            req.Tipo,
			Endereco:        req.Endereco,
			Cidade:          req.Cidade,
			DataInicio:      req.DataInicio,
			PrevisaoTermino: req.PrevisaoTermino,
			ResponsavelID:   req.ResponsavelID,
		})
		if err != nil {
			return err
		}
		return associarResponsavel(ctx, q, emp.ID, responsavel)
	})
	if err != nil {
		return err
	}
	h.auditar(c, "criar", "empreendimento", emp.ID)
	return c.Status(fiber.StatusCreated).JSON(emp)
}

func (h *Controller) GetEmpreendimento(c *fiber.Ctx) error {
	return c.JSON(empreendimentoAtual(c))
}

func (h *Controller) UpdateEmpreendimento(c *fiber.Ctx) error {
	atual := empreendimentoAtual(c)
	var req EmpreendimentoReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}
	ctx := c.UserContext()
	responsavel, err := h.validarEmpreendimento(ctx, &req)
	if err != nil {
		return err
	}

	var emp database.Empreendimento
	err = h.db.Tx(ctx, func(q *database.Queries) error {
		emp, err = q.UpdateEmpreendimento(ctx, database.UpdateEmpreendimentoParams{
			ID:              atual.ID,
			Nome:            req.Nome,
			Tipo:            req.Tipo,
			Endereco:        req.Endereco,
			Cidade:          req.Cidade,
			DataInicio:      req.DataInicio,
			PrevisaoTermino: req.PrevisaoTermino,
			ResponsavelID:   req.ResponsavelID,
		})
		if err != nil {
			return err
		}
		return associarResponsavel(ctx, q, emp.ID, responsavel)
	})
	if err != nil {
		return err
	}
	h.auditar(c, "editar", "empreendimento", emp.ID)
	return c.JSON(emp)
}

func associarResponsavel(ctx context.Context, q *database.Queries, empID int64, u *database.Usuario) error {
	if u == nil || u.Perfil != perfilObra {
		return nil
	}
	return q.AddMembro(ctx, database.AddMembroParams{EmpreendimentoID: empID, UsuarioID: u.ID})
}

func (h *Controller) EncerrarEmpreendimento(c *fiber.Ctx) error {
	return h.mudarStatus(c, statusConcluido, "encerrar")
}

func (h *Controller) ReabrirEmpreendimento(c *fiber.Ctx) error {
	return h.mudarStatus(c, statusEmAndamento, "reabrir")
}

func (h *Controller) mudarStatus(c *fiber.Ctx, status, acao string) error {
	emp := empreendimentoAtual(c)
	if emp.Status == status {
		return c.SendStatus(fiber.StatusNoContent)
	}
	err := h.db.SetEmpreendimentoStatus(c.UserContext(), database.SetEmpreendimentoStatusParams{ID: emp.ID, Status: status})
	if err != nil {
		return err
	}
	h.auditar(c, acao, "empreendimento", emp.ID)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Controller) ListEquipe(c *fiber.Ctx) error {
	equipe, err := h.db.ListEquipe(c.UserContext(), empreendimentoAtual(c).ID)
	if err != nil {
		return err
	}
	return c.JSON(equipe)
}

type MembroReq struct {
	UsuarioID int64 `json:"usuario_id" required:"true" description:"Usuário ativo com perfil obra"`
}

func (h *Controller) AddMembro(c *fiber.Ctx) error {
	emp := empreendimentoAtual(c)
	var req MembroReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}

	ctx := c.UserContext()
	u, err := h.db.GetUsuario(ctx, req.UsuarioID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err != nil || !u.Ativo || u.Perfil != perfilObra {
		return validacao{"usuario_id": "Escolha um usuário de obra ativo"}
	}

	if err := h.db.AddMembro(ctx, database.AddMembroParams{EmpreendimentoID: emp.ID, UsuarioID: u.ID}); err != nil {
		return err
	}
	h.auditar(c, "associar_usuario", "empreendimento", emp.ID)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Controller) RemoveMembro(c *fiber.Ctx) error {
	emp := empreendimentoAtual(c)
	usuarioID, err := paramID(c, "usuarioId")
	if err != nil {
		return err
	}
	err = h.db.RemoveMembro(c.UserContext(), database.RemoveMembroParams{EmpreendimentoID: emp.ID, UsuarioID: usuarioID})
	if err != nil {
		return err
	}
	h.auditar(c, "remover_usuario", "empreendimento", emp.ID)
	return c.SendStatus(fiber.StatusNoContent)
}
