package controllers

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"almoxarifados-a2g/internal/auth"
	"almoxarifados-a2g/internal/database"
)

const msgUsuarioDesativado = "Usuário desativado. Procure o administrador."

var errLoginInvalido = fiber.NewError(fiber.StatusUnauthorized, "E-mail ou senha inválidos")

var hashFalso, _ = auth.HashSenha("senha-falsa-para-tempo-constante")

type LoginReq struct {
	Email string `json:"email" required:"true" format:"email" example:"admin@a2g.com.br"`
	Senha string `json:"senha" required:"true" example:"trocar123"`
}

func (h *Controller) Login(c *fiber.Ctx) error {
	var req LoginReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	v := validacao{}
	v.obrigatorio("email", email)
	v.se(email != "" && !emailValido(email), "email", "Informe um e-mail válido")
	v.se(req.Senha == "", "senha", "Campo obrigatório")
	if err := v.erro(); err != nil {
		return err
	}

	ctx := c.UserContext()
	u, err := h.db.GetUsuarioByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		auth.ConferirSenha(hashFalso, req.Senha)
		h.registrarTentativa(c, email, nil, false)
		return errLoginInvalido
	}
	if err != nil {
		return err
	}

	if u.BloqueadoAte != nil && u.BloqueadoAte.After(time.Now()) {
		h.registrarTentativa(c, email, &u.ID, false)
		minutos := int(math.Ceil(time.Until(*u.BloqueadoAte).Minutes()))
		return fiber.NewError(fiber.StatusTooManyRequests,
			fmt.Sprintf("Muitas tentativas incorretas. Tente novamente em %d minuto(s).", minutos))
	}

	if !auth.ConferirSenha(u.SenhaHash, req.Senha) {
		if err := h.db.RegistrarFalhaLogin(ctx, u.ID); err != nil {
			return err
		}
		h.registrarTentativa(c, email, &u.ID, false)
		return errLoginInvalido
	}

	if !u.Ativo {
		h.registrarTentativa(c, email, &u.ID, false)
		return fiber.NewError(fiber.StatusForbidden, msgUsuarioDesativado)
	}

	if err := h.db.ResetarFalhasLogin(ctx, u.ID); err != nil {
		return err
	}
	h.registrarTentativa(c, email, &u.ID, true)

	token, err := auth.NovoToken()
	if err != nil {
		return err
	}
	err = h.db.CreateSessao(ctx, database.CreateSessaoParams{
		TokenHash: auth.HashToken(token),
		UsuarioID: u.ID,
		ExpiraEm:  time.Now().Add(duracaoSessao),
	})
	if err != nil {
		return err
	}
	if err := h.db.DeleteSessoesExpiradas(ctx); err != nil {
		log.Printf("limpando sessões expiradas: %v", err)
	}

	h.definirCookie(c, token)
	return h.respostaSessao(c, u)
}

func (h *Controller) Logout(c *fiber.Ctx) error {
	if err := h.db.DeleteSessao(c.UserContext(), auth.HashToken(c.Cookies(cookieSessao))); err != nil {
		return err
	}
	h.limparCookie(c)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Controller) Me(c *fiber.Ctx) error {
	u, err := h.db.GetUsuario(c.UserContext(), usuarioLogado(c).ID)
	if err != nil {
		return err
	}
	return h.respostaSessao(c, u)
}

type AlterarSenhaReq struct {
	SenhaAtual  string `json:"senha_atual" required:"true"`
	NovaSenha   string `json:"nova_senha" required:"true" minLength:"8" maxLength:"72"`
	Confirmacao string `json:"confirmacao" required:"true"`
}

func (h *Controller) AlterarSenha(c *fiber.Ctx) error {
	var req AlterarSenhaReq
	if err := lerJSON(c, &req); err != nil {
		return err
	}

	v := validacao{}
	v.se(req.SenhaAtual == "", "senha_atual", "Campo obrigatório")
	validarSenha(v, "nova_senha", req.NovaSenha)
	v.se(req.Confirmacao != req.NovaSenha, "confirmacao", "A confirmação não confere com a nova senha")
	if err := v.erro(); err != nil {
		return err
	}

	ctx := c.UserContext()
	u, err := h.db.GetUsuario(ctx, usuarioLogado(c).ID)
	if err != nil {
		return err
	}
	if !auth.ConferirSenha(u.SenhaHash, req.SenhaAtual) {
		return validacao{"senha_atual": "Senha atual incorreta"}
	}

	hash, err := auth.HashSenha(req.NovaSenha)
	if err != nil {
		return err
	}
	err = h.db.SetUsuarioSenha(ctx, database.SetUsuarioSenhaParams{ID: u.ID, SenhaHash: hash, TrocarSenha: false})
	if err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type SessaoResp struct {
	Usuario         database.Usuario                           `json:"usuario"`
	Empreendimentos []database.ListEmpreendimentosDoUsuarioRow `json:"empreendimentos"`
}

func (h *Controller) respostaSessao(c *fiber.Ctx, u database.Usuario) error {
	emps := []database.ListEmpreendimentosDoUsuarioRow{}
	if u.Perfil == perfilObra {
		var err error
		emps, err = h.db.ListEmpreendimentosDoUsuario(c.UserContext(), u.ID)
		if err != nil {
			return err
		}
	}
	return c.JSON(SessaoResp{Usuario: u, Empreendimentos: emps})
}

func (h *Controller) registrarTentativa(c *fiber.Ctx, email string, usuarioID *int64, sucesso bool) {
	err := h.db.CreateLoginTentativa(c.UserContext(), database.CreateLoginTentativaParams{
		Email:     email,
		UsuarioID: usuarioID,
		Sucesso:   sucesso,
		Ip:        c.IP(),
	})
	if err != nil {
		log.Printf("registrando tentativa de login: %v", err)
	}
}
