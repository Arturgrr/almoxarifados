package controllers

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"almoxarifados-a2g/internal/database"
)

const (
	inventarioAberto     = "aberto"
	inventarioFinalizado = "finalizado"
	inventarioCancelado  = "cancelado"

	semDiferenca    = "sem_diferenca"
	variacaoNatural = "variacao_natural"
	anormal         = "anormal"
)

var cem = decimal.NewFromInt(100)

func classificar(granel bool, tolerancia decimal.NullDecimal, registrado, contado decimal.Decimal) (decimal.Decimal, string) {
	diff := contado.Sub(registrado)
	switch {
	case diff.IsZero():
		return diff, semDiferenca

	case granel && registrado.IsPositive() && diff.Abs().Mul(cem).LessThanOrEqual(tolerancia.Decimal.Mul(registrado)):
		return diff, variacaoNatural
	default:
		return diff, anormal
	}
}

func percentual(registrado, diff decimal.NullDecimal) decimal.NullDecimal {
	if !registrado.Valid || !diff.Valid || registrado.Decimal.IsZero() {
		return decimal.NullDecimal{}
	}
	return decimal.NewNullDecimal(diff.Decimal.Div(registrado.Decimal).Mul(cem).Round(2))
}

type ItemInventario struct {
	database.ListInventarioItensRow
	Percentual decimal.NullDecimal `json:"percentual"`
}

type InventarioResp struct {
	Inventario database.Inventario `json:"inventario"`
	Itens      []ItemInventario    `json:"itens"`
}

func (h *Controller) respostaInventario(c *fiber.Ctx, inv database.Inventario) error {
	rows, err := h.db.ListInventarioItens(c.UserContext(), inv.ID)
	if err != nil {
		return err
	}
	itens := make([]ItemInventario, len(rows))
	for i, r := range rows {
		if inv.Status == inventarioAberto && r.QuantidadeContada.Valid {
			diff, classe := classificar(r.Granel, r.ToleranciaPct, r.SaldoAtual, r.QuantidadeContada.Decimal)
			r.SaldoRegistrado = decimal.NewNullDecimal(r.SaldoAtual)
			r.Diferenca = decimal.NewNullDecimal(diff)
			r.Classificacao = &classe
		}
		itens[i] = ItemInventario{r, percentual(r.SaldoRegistrado, r.Diferenca)}
	}
	return c.JSON(InventarioResp{Inventario: inv, Itens: itens})
}

func (h *Controller) inventarioDaRota(c *fiber.Ctx) (database.Inventario, error) {
	id, err := paramID(c, "invId")
	if err != nil {
		return database.Inventario{}, err
	}
	inv, err := h.db.GetInventario(c.UserContext(), id)
	if err != nil {
		return inv, err
	}
	if inv.EmpreendimentoID != empreendimentoAtual(c).ID {
		return inv, errNaoEncontrado
	}
	return inv, nil
}

func inventarioAbertoParaAtualizar(c *fiber.Ctx, q *database.Queries, id int64) (database.Inventario, error) {
	inv, err := q.GetInventarioParaAtualizar(c.UserContext(), id)
	if err != nil {
		return inv, err
	}
	if inv.Status != inventarioAberto {
		return inv, fiber.NewError(fiber.StatusConflict, "Este inventário já foi encerrado")
	}
	return inv, nil
}

func (h *Controller) ListInventarios(c *fiber.Ctx) error {
	invs, err := h.db.ListInventarios(c.UserContext(), empreendimentoAtual(c).ID)
	if err != nil {
		return err
	}
	return c.JSON(invs)
}

func (h *Controller) GetInventario(c *fiber.Ctx) error {
	inv, err := h.inventarioDaRota(c)
	if err != nil {
		return err
	}
	return h.respostaInventario(c, inv)
}

type AbrirInventarioReq struct {
	CategoriaID *int64 `json:"categoria_id" description:"Vazio para inventariar todos os itens com saldo"`
}

func (h *Controller) AbrirInventario(c *fiber.Ctx) error {
	emp := empreendimentoAtual(c)
	if err := exigirEmAndamento(emp); err != nil {
		return err
	}
	var req AbrirInventarioReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}

	ctx := c.UserContext()
	if req.CategoriaID != nil {
		if _, err := h.db.GetCategoria(ctx, *req.CategoriaID); errors.Is(err, pgx.ErrNoRows) {
			return validacao{"categoria_id": "Categoria não encontrada"}
		} else if err != nil {
			return err
		}
	}

	var inv database.Inventario
	err := h.db.Tx(ctx, func(q *database.Queries) error {
		var err error
		inv, err = q.CreateInventario(ctx, database.CreateInventarioParams{
			EmpreendimentoID: emp.ID,
			CategoriaID:      req.CategoriaID,
			AbertoPor:        usuarioLogado(c).ID,
		})
		if err != nil {
			return err
		}
		return q.PopularInventario(ctx, database.PopularInventarioParams{
			InventarioID:     inv.ID,
			EmpreendimentoID: emp.ID,
			CategoriaID:      req.CategoriaID,
		})
	})
	if err != nil {
		return err
	}
	c.Status(fiber.StatusCreated)
	return h.respostaInventario(c, inv)
}

type LinhaContagem struct {
	ItemID            int64               `json:"item_id" required:"true"`
	QuantidadeContada decimal.NullDecimal `json:"quantidade_contada" description:"null limpa a contagem do item"`
}

type ContagemReq struct {
	Itens []LinhaContagem `json:"itens" required:"true"`
}

func (h *Controller) SalvarContagem(c *fiber.Ctx) error {
	inv, err := h.inventarioDaRota(c)
	if err != nil {
		return err
	}
	var req ContagemReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}

	ctx := c.UserContext()
	rows, err := h.db.ListInventarioItens(ctx, inv.ID)
	if err != nil {
		return err
	}
	unidade := make(map[int64]string, len(rows))
	for _, r := range rows {
		unidade[r.ItemID] = r.Unidade
	}

	v := validacao{}
	for i, linha := range req.Itens {
		campo := fmt.Sprintf("itens.%d.quantidade_contada", i)
		u, ok := unidade[linha.ItemID]
		if !ok {
			v.add(fmt.Sprintf("itens.%d.item_id", i), "Item não faz parte deste inventário")
			continue
		}
		if linha.QuantidadeContada.Valid {
			validarQuantidade(v, campo, linha.QuantidadeContada.Decimal, u, true)
		}
	}
	if err := v.erro(); err != nil {
		return err
	}

	err = h.db.Tx(ctx, func(q *database.Queries) error {
		if inv, err = inventarioAbertoParaAtualizar(c, q, inv.ID); err != nil {
			return err
		}
		for _, linha := range req.Itens {
			_, err := q.SetQuantidadeContada(ctx, database.SetQuantidadeContadaParams{
				InventarioID:      inv.ID,
				ItemID:            linha.ItemID,
				QuantidadeContada: linha.QuantidadeContada,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return h.respostaInventario(c, inv)
}

type Justificativa struct {
	ItemID    int64   `json:"item_id" required:"true"`
	Motivo    string  `json:"motivo" required:"true" enum:"quebra,vencimento,intemperie,extravio,variacao_medida,outro"`
	Descricao *string `json:"descricao" description:"Obrigatória quando o motivo é outro"`
}

type FinalizarReq struct {
	Justificativas []Justificativa `json:"justificativas" description:"Obrigatória para cada item com diferença anormal"`
}

func (h *Controller) FinalizarInventario(c *fiber.Ctx) error {
	emp := empreendimentoAtual(c)
	if err := exigirEmAndamento(emp); err != nil {
		return err
	}
	inv, err := h.inventarioDaRota(c)
	if err != nil {
		return err
	}
	var req FinalizarReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}
	type justificativa struct {
		motivo    string
		descricao *string
	}
	justificativas := make(map[int64]justificativa, len(req.Justificativas))
	for _, j := range req.Justificativas {
		justificativas[j.ItemID] = justificativa{j.Motivo, texto(j.Descricao)}
	}

	ctx := c.UserContext()
	usuarioID := usuarioLogado(c).ID
	err = h.db.Tx(ctx, func(q *database.Queries) error {
		if inv, err = inventarioAbertoParaAtualizar(c, q, inv.ID); err != nil {
			return err
		}
		rows, err := q.ListInventarioItens(ctx, inv.ID)
		if err != nil {
			return err
		}

		type ajuste struct {
			row               database.ListInventarioItensRow
			registrado, diff  decimal.Decimal
			classe            string
			motivo, descricao *string
		}
		ajustes := make([]ajuste, 0, len(rows))
		v := validacao{}

		for _, r := range rows {
			campo := fmt.Sprintf("itens.%d", r.ItemID)
			if !r.QuantidadeContada.Valid {
				v.add(campo, "Informe a quantidade contada de "+r.Item)
				continue
			}

			registrado, err := q.GetSaldoParaAtualizar(ctx, database.GetSaldoParaAtualizarParams{
				EmpreendimentoID: emp.ID, ItemID: r.ItemID,
			})
			if errors.Is(err, pgx.ErrNoRows) {
				registrado = decimal.Zero
			} else if err != nil {
				return err
			}

			a := ajuste{row: r, registrado: registrado}
			a.diff, a.classe = classificar(r.Granel, r.ToleranciaPct, registrado, r.QuantidadeContada.Decimal)
			if a.classe == anormal {
				j, ok := justificativas[r.ItemID]
				switch {
				case !ok || !umDe(j.motivo, motivosPerda...):
					v.add(campo, fmt.Sprintf("%s: diferença de %s %s precisa de justificativa", r.Item, a.diff, r.Unidade))
				case j.motivo == "outro" && j.descricao == nil:
					v.add(campo, r.Item+": descreva o motivo")
				}
				a.motivo, a.descricao = &j.motivo, j.descricao
			}
			ajustes = append(ajustes, a)
		}
		if err := v.erro(); err != nil {
			return err
		}

		for _, a := range ajustes {
			if !a.diff.IsZero() {
				_, err := registrar(ctx, q, database.CreateMovimentacaoParams{
					EmpreendimentoID: emp.ID,
					ItemID:           a.row.ItemID,
					Tipo:             tipoAjuste,
					Subtipo:          &a.classe,
					Quantidade:       a.diff,
					Data:             hoje(),
					UsuarioID:        usuarioID,
					Motivo:           a.motivo,
					Descricao:        a.descricao,
					InventarioID:     &inv.ID,
				}, msgSaldoInsuficiente(database.Item{Nome: a.row.Item}))
				if err != nil {
					return err
				}
			}
			err := q.FinalizarInventarioItem(ctx, database.FinalizarInventarioItemParams{
				InventarioID:    inv.ID,
				ItemID:          a.row.ItemID,
				SaldoRegistrado: decimal.NewNullDecimal(a.registrado),
				Diferenca:       decimal.NewNullDecimal(a.diff),
				Classificacao:   &a.classe,
				Motivo:          a.motivo,
				Descricao:       a.descricao,
			})
			if err != nil {
				return err
			}
		}

		return q.SetInventarioStatus(ctx, database.SetInventarioStatusParams{
			ID: inv.ID, Status: inventarioFinalizado, FinalizadoPor: &usuarioID,
		})
	})
	if err != nil {
		return err
	}

	inv, err = h.db.GetInventario(ctx, inv.ID)
	if err != nil {
		return err
	}
	return h.respostaInventario(c, inv)
}

func (h *Controller) CancelarInventario(c *fiber.Ctx) error {
	inv, err := h.inventarioDaRota(c)
	if err != nil {
		return err
	}
	usuarioID := usuarioLogado(c).ID
	err = h.db.Tx(c.UserContext(), func(q *database.Queries) error {
		if _, err := inventarioAbertoParaAtualizar(c, q, inv.ID); err != nil {
			return err
		}
		return q.SetInventarioStatus(c.UserContext(), database.SetInventarioStatusParams{
			ID: inv.ID, Status: inventarioCancelado, FinalizadoPor: &usuarioID,
		})
	})
	if err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}
