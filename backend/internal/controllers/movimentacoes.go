package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"almoxarifados-a2g/internal/database"
)

const (
	tipoEntrada = "entrada"
	tipoSaida   = "saida"
	tipoPerda   = "perda"
	tipoAjuste  = "ajuste"
	tipoEstorno = "estorno"
)

var motivosPerda = []string{"quebra", "vencimento", "intemperie", "extravio", "variacao_medida", "outro"}

var errSaldoInsuficiente = errors.New("saldo insuficiente")

func aplicarSaldo(ctx context.Context, q *database.Queries, empID, itemID int64, delta decimal.Decimal) (decimal.Decimal, error) {
	if delta.IsPositive() {
		return q.CreditarSaldo(ctx, database.CreditarSaldoParams{
			EmpreendimentoID: empID, ItemID: itemID, Quantidade: delta,
		})
	}

	saldo, err := q.DebitarSaldo(ctx, database.DebitarSaldoParams{
		EmpreendimentoID: empID, ItemID: itemID, Quantidade: delta.Neg(),
	})
	var pgErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &pgErr) && pgErr.ConstraintName == "saldos_nao_negativo") {
		return decimal.Zero, errSaldoInsuficiente
	}
	return saldo, err
}

func registrar(ctx context.Context, q *database.Queries, p database.CreateMovimentacaoParams, msgSemSaldo string) (database.Movimentacao, error) {
	saldo, err := aplicarSaldo(ctx, q, p.EmpreendimentoID, p.ItemID, p.Quantidade)
	if errors.Is(err, errSaldoInsuficiente) {
		return database.Movimentacao{}, fiber.NewError(fiber.StatusUnprocessableEntity, msgSemSaldo)
	}
	if err != nil {
		return database.Movimentacao{}, err
	}
	p.SaldoApos = saldo
	return q.CreateMovimentacao(ctx, p)
}

func (h *Controller) itemParaMovimentar(ctx context.Context, v validacao, prefixo string, id int64, qtd decimal.Decimal) (database.Item, error) {
	item, err := h.db.GetItem(ctx, id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return item, err
	}
	if err != nil || !item.Ativo {
		v.add(prefixo+"item_id", "Escolha um item ativo do catálogo")
		return item, nil
	}
	validarQuantidade(v, prefixo+"quantidade", qtd, item.Unidade, false)
	return item, nil
}

func msgSaldoInsuficiente(item database.Item) string {
	return fmt.Sprintf("Saldo insuficiente de %s", item.Nome)
}

type LinhaEntrada struct {
	ItemID        int64               `json:"item_id" required:"true"`
	Quantidade    decimal.Decimal     `json:"quantidade" required:"true" description:"Maior que zero; inteiro para un, até 3 casas para as demais unidades"`
	ValorUnitario decimal.NullDecimal `json:"valor_unitario" description:"Só para compra"`
}

type EntradaReq struct {
	Tipo         string         `json:"tipo" required:"true" enum:"compra,devolucao"`
	FornecedorID *int64         `json:"fornecedor_id" description:"Obrigatório na compra"`
	OrigemCompra *string        `json:"origem_compra" enum:"obra,administracao" description:"Obrigatório na compra"`
	NotaFiscal   *string        `json:"nota_fiscal"`
	Servico      *string        `json:"servico" description:"Obrigatório na devolução: de qual serviço o item voltou"`
	Data         pgtype.Date    `json:"data" description:"Padrão: hoje. Não pode ser futura nem anterior ao início do empreendimento"`
	Observacoes  *string        `json:"observacoes"`
	Itens        []LinhaEntrada `json:"itens" required:"true" minItems:"1"`
}

func (h *Controller) RegistrarEntrada(c *fiber.Ctx) error {
	emp := empreendimentoAtual(c)
	if err := exigirEmAndamento(emp); err != nil {
		return err
	}
	var req EntradaReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}

	ctx := c.UserContext()
	v := validacao{}
	base := database.CreateMovimentacaoParams{
		EmpreendimentoID: emp.ID,
		Tipo:             tipoEntrada,
		Subtipo:          &req.Tipo,
		Data:             validarDataMovimentacao(v, req.Data, emp),
		UsuarioID:        usuarioLogado(c).ID,
		Descricao:        texto(req.Observacoes),
	}

	switch req.Tipo {
	case "compra":
		base.OrigemCompra = texto(req.OrigemCompra)
		base.NotaFiscal = texto(req.NotaFiscal)
		v.se(!umDe(valor(base.OrigemCompra), "obra", "administracao"), "origem_compra",
			"Informe se a compra foi feita pela obra ou pela administração")
		if req.FornecedorID == nil {
			v.add("fornecedor_id", "Campo obrigatório")
		} else {
			f, err := h.db.GetFornecedor(ctx, *req.FornecedorID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			v.se(err != nil || !f.Ativo, "fornecedor_id", "Escolha um fornecedor ativo")
			base.FornecedorID = req.FornecedorID
		}
	case "devolucao":
		base.Destino = texto(req.Servico)
		v.se(base.Destino == nil, "servico", "Informe de qual serviço o item voltou")
	default:
		v.add("tipo", "Escolha compra ou devolução")
	}

	v.se(len(req.Itens) == 0, "itens", "Inclua pelo menos um item")
	itens := make([]database.Item, len(req.Itens))
	for i, linha := range req.Itens {
		prefixo := fmt.Sprintf("itens.%d.", i)
		item, err := h.itemParaMovimentar(ctx, v, prefixo, linha.ItemID, linha.Quantidade)
		if err != nil {
			return err
		}
		itens[i] = item
		v.se(linha.ValorUnitario.Valid && linha.ValorUnitario.Decimal.IsNegative(),
			prefixo+"valor_unitario", "Valor inválido")
	}
	if err := v.erro(); err != nil {
		return err
	}

	criadas := make([]database.Movimentacao, 0, len(req.Itens))
	err := h.db.Tx(ctx, func(q *database.Queries) error {
		for i, linha := range req.Itens {
			p := base
			p.ItemID = itens[i].ID
			p.Quantidade = linha.Quantidade
			if req.Tipo == "compra" {
				p.ValorUnitario = linha.ValorUnitario
			}
			m, err := registrar(ctx, q, p, "")
			if err != nil {
				return err
			}
			criadas = append(criadas, m)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(criadas)
}

type SaidaReq struct {
	ItemID      int64           `json:"item_id" required:"true"`
	Quantidade  decimal.Decimal `json:"quantidade" required:"true"`
	Destino     string          `json:"destino" required:"true" example:"Alvenaria – Bloco B" description:"Serviço ou local de destino"`
	RetiradoPor string          `json:"retirado_por" required:"true" example:"Zé (pedreiro)" description:"Quem retirou; pode não ser usuário do sistema"`
	Data        pgtype.Date     `json:"data" description:"Padrão: hoje"`
	Observacoes *string         `json:"observacoes"`
}

func (h *Controller) RegistrarSaida(c *fiber.Ctx) error {
	emp := empreendimentoAtual(c)
	if err := exigirEmAndamento(emp); err != nil {
		return err
	}
	var req SaidaReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}

	ctx := c.UserContext()
	v := validacao{}
	item, err := h.itemParaMovimentar(ctx, v, "", req.ItemID, req.Quantidade)
	if err != nil {
		return err
	}
	destino := strings.TrimSpace(req.Destino)
	retiradoPor := strings.TrimSpace(req.RetiradoPor)
	v.obrigatorio("destino", destino)
	v.obrigatorio("retirado_por", retiradoPor)
	data := validarDataMovimentacao(v, req.Data, emp)
	if err := v.erro(); err != nil {
		return err
	}

	var m database.Movimentacao
	err = h.db.Tx(ctx, func(q *database.Queries) error {
		m, err = registrar(ctx, q, database.CreateMovimentacaoParams{
			EmpreendimentoID: emp.ID,
			ItemID:           item.ID,
			Tipo:             tipoSaida,
			Quantidade:       req.Quantidade.Neg(),
			Data:             data,
			UsuarioID:        usuarioLogado(c).ID,
			Destino:          &destino,
			RetiradoPor:      &retiradoPor,
			Descricao:        texto(req.Observacoes),
		}, msgSaldoInsuficiente(item))
		return err
	})
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(m)
}

type PerdaReq struct {
	ItemID     int64           `json:"item_id" required:"true"`
	Quantidade decimal.Decimal `json:"quantidade" required:"true"`
	Motivo     string          `json:"motivo" required:"true" enum:"quebra,vencimento,intemperie,extravio,variacao_medida,outro"`
	Descricao  *string         `json:"descricao" description:"Obrigatória quando o motivo é outro"`
	Data       pgtype.Date     `json:"data" description:"Padrão: hoje"`
}

func (h *Controller) RegistrarPerda(c *fiber.Ctx) error {
	emp := empreendimentoAtual(c)
	if err := exigirEmAndamento(emp); err != nil {
		return err
	}
	var req PerdaReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}

	ctx := c.UserContext()
	v := validacao{}
	item, err := h.itemParaMovimentar(ctx, v, "", req.ItemID, req.Quantidade)
	if err != nil {
		return err
	}
	descricao := texto(req.Descricao)
	validarMotivo(v, "", req.Motivo, descricao)
	data := validarDataMovimentacao(v, req.Data, emp)
	if err := v.erro(); err != nil {
		return err
	}

	var m database.Movimentacao
	err = h.db.Tx(ctx, func(q *database.Queries) error {
		m, err = registrar(ctx, q, database.CreateMovimentacaoParams{
			EmpreendimentoID: emp.ID,
			ItemID:           item.ID,
			Tipo:             tipoPerda,
			Quantidade:       req.Quantidade.Neg(),
			Data:             data,
			UsuarioID:        usuarioLogado(c).ID,
			Motivo:           &req.Motivo,
			Descricao:        descricao,
		}, msgSaldoInsuficiente(item))
		return err
	})
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(m)
}

func validarMotivo(v validacao, prefixo, motivo string, descricao *string) {
	v.se(!umDe(motivo, motivosPerda...), prefixo+"motivo", "Escolha o motivo")
	v.se(motivo == "outro" && descricao == nil, prefixo+"descricao", "Descreva o motivo")
}

type EstornoReq struct {
	Justificativa string `json:"justificativa" required:"true"`
}

func (h *Controller) Estornar(c *fiber.Ctx) error {
	emp := empreendimentoAtual(c)
	if err := exigirEmAndamento(emp); err != nil {
		return err
	}
	movID, err := paramID(c, "movId")
	if err != nil {
		return err
	}
	var req EstornoReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}
	justificativa := strings.TrimSpace(req.Justificativa)
	if justificativa == "" {
		return validacao{"justificativa": "Campo obrigatório"}
	}

	ctx := c.UserContext()
	orig, err := h.db.GetMovimentacao(ctx, movID)
	if err != nil {
		return err
	}
	usuario := usuarioLogado(c)
	switch {
	case orig.EmpreendimentoID != emp.ID:
		return errNaoEncontrado
	case orig.Tipo == tipoEstorno:
		return fiber.NewError(fiber.StatusUnprocessableEntity, "Um estorno não pode ser estornado")
	case orig.Estornada:
		return fiber.NewError(fiber.StatusConflict, "Esta movimentação já foi estornada")
	case !ehAdmin(c) && orig.UsuarioID != usuario.ID:
		return fiber.NewError(fiber.StatusForbidden, "Você só pode estornar movimentações registradas por você")
	}

	item, err := h.db.GetItem(ctx, orig.ItemID)
	if err != nil {
		return err
	}

	var m database.Movimentacao
	err = h.db.Tx(ctx, func(q *database.Queries) error {
		m, err = registrar(ctx, q, database.CreateMovimentacaoParams{
			EmpreendimentoID: emp.ID,
			ItemID:           orig.ItemID,
			Tipo:             tipoEstorno,
			Quantidade:       orig.Quantidade.Neg(),
			Data:             hoje(),
			UsuarioID:        usuario.ID,
			Descricao:        &justificativa,
			EstornoDe:        &orig.ID,
		}, fmt.Sprintf("Não é possível estornar: o saldo de %s ficaria negativo. Corrija as saídas antes.", item.Nome))
		return err
	})
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(m)
}

func (h *Controller) ListSaldos(c *fiber.Ctx) error {
	categoriaID, err := queryID(c, "categoria_id")
	if err != nil {
		return err
	}
	incluirZerados, err := queryBool(c, "incluir_zerados")
	if err != nil {
		return err
	}
	saldos, err := h.db.ListSaldos(c.UserContext(), database.ListSaldosParams{
		EmpreendimentoID: empreendimentoAtual(c).ID,
		IncluirZerados:   incluirZerados != nil && *incluirZerados,
		Busca:            queryTexto(c, "busca"),
		CategoriaID:      categoriaID,
		Tipo:             queryTexto(c, "tipo"),
	})
	if err != nil {
		return err
	}
	return c.JSON(saldos)
}

func (h *Controller) ListMovimentacoes(c *fiber.Ctx) error {
	params := database.ListMovimentacoesParams{
		EmpreendimentoID: empreendimentoAtual(c).ID,
		Tipo:             queryTexto(c, "tipo"),
	}
	var err error
	if params.DataInicio, err = queryData(c, "data_inicio"); err != nil {
		return err
	}
	if params.DataFim, err = queryData(c, "data_fim"); err != nil {
		return err
	}
	if params.ItemID, err = queryID(c, "item_id"); err != nil {
		return err
	}
	if params.UsuarioID, err = queryID(c, "usuario_id"); err != nil {
		return err
	}
	if params.FornecedorID, err = queryID(c, "fornecedor_id"); err != nil {
		return err
	}
	p := lerPagina(c)
	params.Limite = p.tamanho
	params.Deslocamento = p.deslocamento()

	movs, err := h.db.ListMovimentacoes(c.UserContext(), params)
	if err != nil {
		return err
	}
	var total int64
	if len(movs) > 0 {
		total = movs[0].Total
	}
	return respostaPaginada(c, p, movs, total)
}

func (h *Controller) ListDestinos(c *fiber.Ctx) error {
	destinos, err := h.db.ListDestinos(c.UserContext(), empreendimentoAtual(c).ID)
	if err != nil {
		return err
	}
	return c.JSON(destinos)
}
