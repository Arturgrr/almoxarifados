package controllers

import (
	"github.com/gofiber/fiber/v2"

	"almoxarifados-a2g/internal/database"
)

func (h *Controller) Painel(c *fiber.Ctx) error {
	painel, err := h.db.PainelAdmin(c.UserContext())
	if err != nil {
		return err
	}
	return c.JSON(painel)
}

func (h *Controller) PerdasAnormais(c *fiber.Ctx) error {
	var params database.ListPerdasAnormaisParams
	var err error
	if params.EmpreendimentoID, err = queryID(c, "empreendimento_id"); err != nil {
		return err
	}
	if params.DataInicio, err = queryData(c, "data_inicio"); err != nil {
		return err
	}
	if params.DataFim, err = queryData(c, "data_fim"); err != nil {
		return err
	}
	p := lerPagina(c)
	params.Limite = p.tamanho
	params.Deslocamento = p.deslocamento()

	perdas, err := h.db.ListPerdasAnormais(c.UserContext(), params)
	if err != nil {
		return err
	}
	var total int64
	if len(perdas) > 0 {
		total = perdas[0].Total
	}
	return respostaPaginada(c, p, perdas, total)
}
