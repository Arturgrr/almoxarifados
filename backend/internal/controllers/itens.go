package controllers

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"almoxarifados-a2g/internal/database"
)

func (h *Controller) ListCategorias(c *fiber.Ctx) error {
	ativo, err := queryBool(c, "ativo")
	if err != nil {
		return err
	}
	categorias, err := h.db.ListCategorias(c.UserContext(), ativo)
	if err != nil {
		return err
	}
	return c.JSON(categorias)
}

type CategoriaReq struct {
	Nome  string `json:"nome" required:"true" example:"Agregados"`
	Ativo *bool  `json:"ativo" description:"Só na edição; se omitido mantém o valor atual"`
}

func (h *Controller) CreateCategoria(c *fiber.Ctx) error {
	var req CategoriaReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}
	req.Nome = strings.TrimSpace(req.Nome)
	if req.Nome == "" {
		return validacao{"nome": "Campo obrigatório"}
	}

	cat, err := h.db.CreateCategoria(c.UserContext(), req.Nome)
	if err != nil {
		return err
	}
	h.auditar(c, "criar", "categoria", cat.ID)
	return c.Status(fiber.StatusCreated).JSON(cat)
}

func (h *Controller) UpdateCategoria(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req CategoriaReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}
	req.Nome = strings.TrimSpace(req.Nome)
	if req.Nome == "" {
		return validacao{"nome": "Campo obrigatório"}
	}

	ctx := c.UserContext()
	atual, err := h.db.GetCategoria(ctx, id)
	if err != nil {
		return err
	}
	ativo := atual.Ativo
	if req.Ativo != nil {
		ativo = *req.Ativo
	}

	cat, err := h.db.UpdateCategoria(ctx, database.UpdateCategoriaParams{ID: id, Nome: req.Nome, Ativo: ativo})
	if err != nil {
		return err
	}
	h.auditar(c, "editar", "categoria", cat.ID)
	return c.JSON(cat)
}

type ItemReq struct {
	Nome          string              `json:"nome" required:"true" example:"Areia média"`
	Tipo          string              `json:"tipo" required:"true" enum:"consumo,equipamento"`
	CategoriaID   int64               `json:"categoria_id" required:"true"`
	Unidade       string              `json:"unidade" required:"true" enum:"kg,L,m,m2,m3,un" description:"Não pode mudar depois da primeira movimentação"`
	Granel        bool                `json:"granel"`
	ToleranciaPct decimal.NullDecimal `json:"tolerancia_pct" description:"Percentual de variação natural (0 a 100). Só o administrador define, e só para granel"`
	Descricao     *string             `json:"descricao"`
}

var unidades = []string{"kg", "L", "m", "m2", "m3", "un"}

func (h *Controller) validarItem(ctx context.Context, v validacao, r *ItemReq, admin bool) error {
	r.Nome = strings.TrimSpace(r.Nome)
	r.Descricao = texto(r.Descricao)

	v.obrigatorio("nome", r.Nome)
	v.se(!umDe(r.Tipo, "consumo", "equipamento"), "tipo", "Escolha material de consumo ou equipamento/ferramenta")
	v.se(!umDe(r.Unidade, unidades...), "unidade", "Escolha a unidade de medida")

	cat, err := h.db.GetCategoria(ctx, r.CategoriaID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	v.se(err != nil || !cat.Ativo, "categoria_id", "Escolha uma categoria")

	if !admin || !r.Granel {
		r.ToleranciaPct = decimal.NullDecimal{}
	}
	if t := r.ToleranciaPct; t.Valid {
		v.se(t.Decimal.IsNegative() || t.Decimal.GreaterThan(decimal.NewFromInt(100)) || !t.Decimal.Equal(t.Decimal.Truncate(2)),
			"tolerancia_pct", "Informe um percentual entre 0 e 100, com até 2 casas decimais")
	}
	return nil
}

func (h *Controller) ListItens(c *fiber.Ctx) error {
	categoriaID, err := queryID(c, "categoria_id")
	if err != nil {
		return err
	}
	ativo, err := queryBool(c, "ativo")
	if err != nil {
		return err
	}
	p := lerPagina(c)

	itens, err := h.db.ListItens(c.UserContext(), database.ListItensParams{
		Busca:        queryTexto(c, "busca"),
		Tipo:         queryTexto(c, "tipo"),
		CategoriaID:  categoriaID,
		Ativo:        ativo,
		Limite:       p.tamanho,
		Deslocamento: p.deslocamento(),
	})
	if err != nil {
		return err
	}
	var total int64
	if len(itens) > 0 {
		total = itens[0].Total
	}
	return respostaPaginada(c, p, itens, total)
}

func (h *Controller) CreateItem(c *fiber.Ctx) error {
	var req ItemReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}
	ctx := c.UserContext()
	v := validacao{}
	if err := h.validarItem(ctx, v, &req, ehAdmin(c)); err != nil {
		return err
	}
	if err := v.erro(); err != nil {
		return err
	}

	item, err := h.db.CreateItem(ctx, database.CreateItemParams{
		Nome:          req.Nome,
		Tipo:          req.Tipo,
		CategoriaID:   req.CategoriaID,
		Unidade:       req.Unidade,
		Granel:        req.Granel,
		ToleranciaPct: req.ToleranciaPct,
		Descricao:     req.Descricao,
	})
	if err != nil {
		return err
	}
	h.auditar(c, "criar", "item", item.ID)
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *Controller) GetItem(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	item, err := h.db.GetItem(c.UserContext(), id)
	if err != nil {
		return err
	}
	return c.JSON(item)
}

func (h *Controller) UpdateItem(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req ItemReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}

	ctx := c.UserContext()
	atual, err := h.db.GetItem(ctx, id)
	if err != nil {
		return err
	}

	v := validacao{}
	if err := h.validarItem(ctx, v, &req, true); err != nil {
		return err
	}
	if req.Unidade != atual.Unidade {
		movimentado, err := h.db.ItemTemMovimentacao(ctx, id)
		if err != nil {
			return err
		}
		v.se(movimentado, "unidade", "A unidade não pode ser alterada depois da primeira movimentação")
	}
	if err := v.erro(); err != nil {
		return err
	}

	item, err := h.db.UpdateItem(ctx, database.UpdateItemParams{
		ID:            id,
		Nome:          req.Nome,
		Tipo:          req.Tipo,
		CategoriaID:   req.CategoriaID,
		Unidade:       req.Unidade,
		Granel:        req.Granel,
		ToleranciaPct: req.ToleranciaPct,
		Descricao:     req.Descricao,
	})
	if err != nil {
		return err
	}
	h.auditar(c, "editar", "item", item.ID)
	return c.JSON(item)
}

func (h *Controller) DesativarItem(c *fiber.Ctx) error { return h.setItemAtivo(c, false) }
func (h *Controller) ReativarItem(c *fiber.Ctx) error  { return h.setItemAtivo(c, true) }

func (h *Controller) setItemAtivo(c *fiber.Ctx, ativo bool) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	ctx := c.UserContext()
	if _, err := h.db.GetItem(ctx, id); err != nil {
		return err
	}
	if err := h.db.SetItemAtivo(ctx, database.SetItemAtivoParams{ID: id, Ativo: ativo}); err != nil {
		return err
	}
	acao := "desativar"
	if ativo {
		acao = "reativar"
	}
	h.auditar(c, acao, "item", id)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Controller) DeleteItem(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	ctx := c.UserContext()
	if _, err := h.db.GetItem(ctx, id); err != nil {
		return err
	}
	movimentado, err := h.db.ItemTemMovimentacao(ctx, id)
	if err != nil {
		return err
	}
	if movimentado {
		return fiber.NewError(fiber.StatusConflict, "Este item já foi movimentado. Desative-o em vez de excluir.")
	}
	if err := h.db.DeleteItem(ctx, id); err != nil {
		return err
	}
	h.auditar(c, "excluir", "item", id)
	return c.SendStatus(fiber.StatusNoContent)
}

type SaldoConsolidadoResp struct {
	Item   database.Item                      `json:"item"`
	Saldos []database.SaldoConsolidadoItemRow `json:"saldos"`
}

func (h *Controller) SaldoConsolidadoItem(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	ctx := c.UserContext()
	item, err := h.db.GetItem(ctx, id)
	if err != nil {
		return err
	}
	saldos, err := h.db.SaldoConsolidadoItem(ctx, id)
	if err != nil {
		return err
	}
	return c.JSON(SaldoConsolidadoResp{Item: item, Saldos: saldos})
}
