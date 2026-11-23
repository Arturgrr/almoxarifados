package controllers

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"almoxarifados-a2g/internal/database"
)

type FornecedorReq struct {
	Nome         string  `json:"nome" required:"true" example:"Depósito Porto Seguro LTDA"`
	NomeFantasia *string `json:"nome_fantasia"`
	Documento    *string `json:"documento" description:"CNPJ (numérico ou alfanumérico) ou CPF, com ou sem máscara" example:"11.222.333/0001-81"`
	Telefone     *string `json:"telefone"`
	Email        *string `json:"email" format:"email"`
	Cidade       *string `json:"cidade"`
	Observacoes  *string `json:"observacoes"`
}

func (r *FornecedorReq) validar() error {
	r.Nome = strings.TrimSpace(r.Nome)
	r.NomeFantasia = texto(r.NomeFantasia)
	r.Telefone = texto(r.Telefone)
	r.Cidade = texto(r.Cidade)
	r.Observacoes = texto(r.Observacoes)
	r.Email = texto(r.Email)
	if r.Email != nil {
		e := strings.ToLower(*r.Email)
		r.Email = &e
	}

	v := validacao{}
	v.obrigatorio("nome", r.Nome)
	v.se(r.Email != nil && !emailValido(*r.Email), "email", "Informe um e-mail válido")
	if doc := texto(r.Documento); doc != nil {
		d := normalizarDocumento(*doc)
		v.se(!cpfValido(d) && !cnpjValido(d), "documento", "CNPJ ou CPF inválido")
		r.Documento = &d
	} else {
		r.Documento = nil
	}
	return v.erro()
}

func (h *Controller) ListFornecedores(c *fiber.Ctx) error {
	ativo, err := queryBool(c, "ativo")
	if err != nil {
		return err
	}
	p := lerPagina(c)
	fornecedores, err := h.db.ListFornecedores(c.UserContext(), database.ListFornecedoresParams{
		Busca:        queryTexto(c, "busca"),
		Ativo:        ativo,
		Limite:       p.tamanho,
		Deslocamento: p.deslocamento(),
	})
	if err != nil {
		return err
	}
	var total int64
	if len(fornecedores) > 0 {
		total = fornecedores[0].Total
	}
	return respostaPaginada(c, p, fornecedores, total)
}

func (h *Controller) CreateFornecedor(c *fiber.Ctx) error {
	var req FornecedorReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}
	if err := req.validar(); err != nil {
		return err
	}
	f, err := h.db.CreateFornecedor(c.UserContext(), database.CreateFornecedorParams{
		Nome:         req.Nome,
		NomeFantasia: req.NomeFantasia,
		Documento:    req.Documento,
		Telefone:     req.Telefone,
		Email:        req.Email,
		Cidade:       req.Cidade,
		Observacoes:  req.Observacoes,
	})
	if err != nil {
		return err
	}
	h.auditar(c, "criar", "fornecedor", f.ID)
	return c.Status(fiber.StatusCreated).JSON(f)
}

func (h *Controller) GetFornecedor(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	f, err := h.db.GetFornecedor(c.UserContext(), id)
	if err != nil {
		return err
	}
	return c.JSON(f)
}

func (h *Controller) UpdateFornecedor(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req FornecedorReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}
	if err := req.validar(); err != nil {
		return err
	}
	f, err := h.db.UpdateFornecedor(c.UserContext(), database.UpdateFornecedorParams{
		ID:           id,
		Nome:         req.Nome,
		NomeFantasia: req.NomeFantasia,
		Documento:    req.Documento,
		Telefone:     req.Telefone,
		Email:        req.Email,
		Cidade:       req.Cidade,
		Observacoes:  req.Observacoes,
	})
	if err != nil {
		return err
	}
	h.auditar(c, "editar", "fornecedor", f.ID)
	return c.JSON(f)
}

func (h *Controller) DesativarFornecedor(c *fiber.Ctx) error { return h.setFornecedorAtivo(c, false) }
func (h *Controller) ReativarFornecedor(c *fiber.Ctx) error  { return h.setFornecedorAtivo(c, true) }

func (h *Controller) setFornecedorAtivo(c *fiber.Ctx, ativo bool) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	ctx := c.UserContext()
	if _, err := h.db.GetFornecedor(ctx, id); err != nil {
		return err
	}
	if err := h.db.SetFornecedorAtivo(ctx, database.SetFornecedorAtivoParams{ID: id, Ativo: ativo}); err != nil {
		return err
	}
	acao := "desativar"
	if ativo {
		acao = "reativar"
	}
	h.auditar(c, acao, "fornecedor", id)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Controller) DeleteFornecedor(c *fiber.Ctx) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	ctx := c.UserContext()
	if _, err := h.db.GetFornecedor(ctx, id); err != nil {
		return err
	}
	movimentado, err := h.db.FornecedorTemMovimentacao(ctx, &id)
	if err != nil {
		return err
	}
	if movimentado {
		return fiber.NewError(fiber.StatusConflict, "Este fornecedor tem entradas registradas. Desative-o em vez de excluir.")
	}
	if err := h.db.DeleteFornecedor(ctx, id); err != nil {
		return err
	}
	h.auditar(c, "excluir", "fornecedor", id)
	return c.SendStatus(fiber.StatusNoContent)
}

func normalizarDocumento(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func somenteDigitos(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func todosIguais(s string) bool {
	return strings.Count(s, s[:1]) == len(s)
}

func cpfValido(d string) bool {
	if len(d) != 11 || !somenteDigitos(d) || todosIguais(d) {
		return false
	}
	for _, n := range []int{9, 10} {
		soma := 0
		for i := range n {
			soma += int(d[i]-'0') * (n + 1 - i)
		}
		if soma*10%11%10 != int(d[n]-'0') {
			return false
		}
	}
	return true
}

func cnpjValido(d string) bool {
	if len(d) != 14 || !somenteDigitos(d[12:]) || todosIguais(d) {
		return false
	}
	pesos := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	for _, n := range []int{12, 13} {
		soma := 0
		for i := range n {
			soma += int(d[i]-'0') * pesos[i+13-n]
		}
		dv := soma % 11
		if dv < 2 {
			dv = 0
		} else {
			dv = 11 - dv
		}
		if dv != int(d[n]-'0') {
			return false
		}
	}
	return true
}
