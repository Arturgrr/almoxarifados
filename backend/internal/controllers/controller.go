package controllers

import (
	"encoding/json"
	"errors"
	"log"
	"net/mail"
	"slices"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"almoxarifados-a2g/internal/database"
)

const (
	perfilAdmin = "admin"
	perfilObra  = "obra"

	statusEmAndamento = "em_andamento"
	statusConcluido   = "concluido"

	localUsuario        = "usuario"
	localEmpreendimento = "empreendimento"

	porPaginaPadrao = 50
	porPaginaMax    = 200
)

var fusoBrasilia = mustLoadLocation("America/Sao_Paulo")

type Controller struct {
	db           *database.Store
	cookieSecure bool
}

func New(db *database.Store, cookieSecure bool) *Controller {
	return &Controller{db: db, cookieSecure: cookieSecure}
}

type validacao map[string]string

func (v validacao) Error() string { return "dados inválidos" }

func (v validacao) add(campo, msg string) {
	if _, existe := v[campo]; !existe {
		v[campo] = msg
	}
}

func (v validacao) se(cond bool, campo, msg string) {
	if cond {
		v.add(campo, msg)
	}
}

func (v validacao) obrigatorio(campo, valor string) {
	v.se(strings.TrimSpace(valor) == "", campo, "Campo obrigatório")
}

func (v validacao) erro() error {
	if len(v) == 0 {
		return nil
	}
	return v
}

type Erro struct {
	Erro string `json:"erro" example:"Registro não encontrado"`
}

type ErroValidacao struct {
	Erro   string            `json:"erro" example:"Verifique os campos informados"`
	Campos map[string]string `json:"campos" example:"{\"nome\":\"Campo obrigatório\"}"`
}

type Pagina[T any] struct {
	Dados     []T   `json:"dados"`
	Pagina    int32 `json:"pagina"`
	PorPagina int32 `json:"por_pagina"`
	Total     int64 `json:"total"`
}

type msgUnico struct{ campo, msg string }

var unicos = map[string]msgUnico{
	"usuarios_email_key":           {"email", "Já existe um usuário com este e-mail"},
	"empreendimentos_nome_key":     {"nome", "Já existe um empreendimento com este nome"},
	"categorias_nome_key":          {"nome", "Já existe uma categoria com este nome"},
	"itens_nome_key":               {"nome", "Já existe um item com este nome no catálogo"},
	"fornecedores_documento_key":   {"documento", "Já existe um fornecedor com este CNPJ/CPF"},
	"movimentacoes_estorno_de_key": {"", "Esta movimentação já foi estornada"},
	"inventarios_um_aberto":        {"", "Já existe um inventário aberto neste empreendimento"},
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	var v validacao
	if errors.As(err, &v) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErroValidacao{"Verifique os campos informados", v})
	}

	var fe *fiber.Error
	if errors.As(err, &fe) {
		return c.Status(fe.Code).JSON(Erro{fe.Message})
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return c.Status(fiber.StatusNotFound).JSON(Erro{"Registro não encontrado"})
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			u, ok := unicos[pgErr.ConstraintName]
			if !ok {
				u = msgUnico{"", "Registro duplicado"}
			}
			if u.campo == "" {
				return c.Status(fiber.StatusConflict).JSON(Erro{u.msg})
			}
			return c.Status(fiber.StatusConflict).JSON(ErroValidacao{u.msg, map[string]string{u.campo: u.msg}})
		case "23503":
			return c.Status(fiber.StatusConflict).JSON(Erro{"Registro vinculado a outros dados"})
		}
	}

	log.Printf("erro em %s %s: %v", c.Method(), c.Path(), err)
	return c.Status(fiber.StatusInternalServerError).JSON(Erro{"Erro interno. Tente novamente."})
}

var errNaoEncontrado = fiber.NewError(fiber.StatusNotFound, "Registro não encontrado")

func lerJSON(c *fiber.Ctx, dst any) error {
	if len(c.Body()) == 0 {
		return nil
	}
	if err := json.Unmarshal(c.Body(), dst); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "JSON inválido")
	}
	return nil
}

func paramID(c *fiber.Ctx, nome string) (int64, error) {
	id, err := strconv.ParseInt(c.Params(nome), 10, 64)
	if err != nil || id <= 0 {
		return 0, errNaoEncontrado
	}
	return id, nil
}

func queryTexto(c *fiber.Ctx, nome string) *string {
	s := strings.TrimSpace(c.Query(nome))
	if s == "" {
		return nil
	}
	s = strings.Clone(s)
	return &s
}

func queryID(c *fiber.Ctx, nome string) (*int64, error) {
	s := c.Query(nome)
	if s == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Parâmetro inválido: "+nome)
	}
	return &id, nil
}

func queryBool(c *fiber.Ctx, nome string) (*bool, error) {
	s := c.Query(nome)
	if s == "" {
		return nil, nil
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Parâmetro inválido: "+nome)
	}
	return &b, nil
}

func queryData(c *fiber.Ctx, nome string) (pgtype.Date, error) {
	s := c.Query(nome)
	if s == "" {
		return pgtype.Date{}, nil
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return pgtype.Date{}, fiber.NewError(fiber.StatusBadRequest, "Data inválida em "+nome+" (use aaaa-mm-dd)")
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

type pagina struct {
	numero, tamanho int32
}

func lerPagina(c *fiber.Ctx) pagina {
	n := max(c.QueryInt("pagina", 1), 1)
	t := min(max(c.QueryInt("por_pagina", porPaginaPadrao), 1), porPaginaMax)
	return pagina{numero: int32(n), tamanho: int32(t)}
}

func (p pagina) deslocamento() int32 { return (p.numero - 1) * p.tamanho }

func respostaPaginada[T any](c *fiber.Ctx, p pagina, dados []T, total int64) error {
	return c.JSON(Pagina[T]{Dados: dados, Pagina: p.numero, PorPagina: p.tamanho, Total: total})
}

func usuarioLogado(c *fiber.Ctx) database.TocarSessaoRow {
	return c.Locals(localUsuario).(database.TocarSessaoRow)
}

func ehAdmin(c *fiber.Ctx) bool {
	return usuarioLogado(c).Perfil == perfilAdmin
}

func empreendimentoAtual(c *fiber.Ctx) database.Empreendimento {
	return c.Locals(localEmpreendimento).(database.Empreendimento)
}

func exigirEmAndamento(emp database.Empreendimento) error {
	if emp.Status != statusEmAndamento {
		return fiber.NewError(fiber.StatusConflict, "Empreendimento concluído: disponível apenas para consulta")
	}
	return nil
}

func (h *Controller) auditar(c *fiber.Ctx, acao, entidade string, id int64) {
	err := h.db.CreateAuditoria(c.UserContext(), database.CreateAuditoriaParams{
		UsuarioID:  usuarioLogado(c).ID,
		Acao:       acao,
		Entidade:   entidade,
		EntidadeID: id,
	})
	if err != nil {
		log.Printf("auditoria (%s %s %d): %v", acao, entidade, id, err)
	}
}

func texto(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func valor(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func umDe(v string, opcoes ...string) bool {
	return slices.Contains(opcoes, v)
}

func emailValido(s string) bool {
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s
}

func validarSenha(v validacao, campo, senha string) {
	switch {
	case utf8.RuneCountInString(senha) < 8:
		v.add(campo, "A senha deve ter no mínimo 8 caracteres")
	case len(senha) > 72:
		v.add(campo, "A senha deve ter no máximo 72 caracteres")
	}
}

var quantidadeMaxima = decimal.New(1, 11)

func validarQuantidade(v validacao, campo string, q decimal.Decimal, unidade string, permiteZero bool) {
	switch {
	case q.IsNegative() || (!permiteZero && q.IsZero()):
		v.add(campo, "Informe uma quantidade maior que zero")
	case q.GreaterThanOrEqual(quantidadeMaxima):
		v.add(campo, "Quantidade muito alta")
	case unidade == "un" && !q.IsInteger():
		v.add(campo, "Este item só aceita quantidades inteiras")
	case !q.Equal(q.Truncate(3)):
		v.add(campo, "Use no máximo 3 casas decimais")
	}
}

func hoje() pgtype.Date {
	agora := time.Now().In(fusoBrasilia)
	return pgtype.Date{
		Time:  time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, time.UTC),
		Valid: true,
	}
}

func validarDataMovimentacao(v validacao, data pgtype.Date, emp database.Empreendimento) pgtype.Date {
	if !data.Valid {
		return hoje()
	}
	v.se(data.Time.After(hoje().Time), "data", "A data não pode ser futura")
	v.se(data.Time.Before(emp.DataInicio.Time), "data",
		"A data não pode ser anterior ao início do empreendimento ("+emp.DataInicio.Time.Format("02/01/2006")+")")
	return data
}

func mustLoadLocation(nome string) *time.Location {
	loc, err := time.LoadLocation(nome)
	if err != nil {
		panic(err)
	}
	return loc
}
