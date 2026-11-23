package docs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"

	"almoxarifados-a2g/internal/controllers"
)

const cookieSessao = "sessao"

var descricao = strings.TrimSpace(`
API do sistema de gestão de almoxarifado de obras do Grupo A2G.

**Autenticação:** chame ` + "`POST /api/auth/login`" + `; a resposta define o cookie HttpOnly ` + "`sessao`" + `,
enviado automaticamente nas próximas requisições. A sessão expira após 8 h sem uso.

**Formatos:** quantidades e valores são strings decimais (` + "`\"18.5\"`" + `), datas usam ` + "`aaaa-mm-dd`" + `.
Erros sempre vêm como ` + "`{\"erro\": \"...\"}`" + `; erros de validação (422) trazem também ` + "`campos`" + `.
`)

var paramRota = regexp.MustCompile(`:(\w+)`)

func Gerar(servidor string) (*openapi3.Spec, error) {
	r := openapi3.NewReflector()
	r.Spec.Info.WithTitle("API Almoxarifado A2G").WithVersion("1.0.0").WithDescription(descricao)
	r.Spec.WithServers(openapi3.Server{URL: servidor})
	r.Spec.SetAPIKeySecurity(cookieSessao, cookieSessao, openapi.InCookie, "Cookie HttpOnly definido pelo login")

	texto := jsonschema.String.Type()
	textoOuNulo := jsonschema.Type{SliceOfSimpleTypeValues: []jsonschema.SimpleType{jsonschema.String, jsonschema.Null}}
	r.AddTypeMapping(decimal.Decimal{}, *(&jsonschema.Schema{}).WithType(texto).WithExamples("18.5"))
	r.AddTypeMapping(decimal.NullDecimal{}, *(&jsonschema.Schema{}).WithType(textoOuNulo).WithExamples("18.5"))
	r.AddTypeMapping(pgtype.Date{}, *(&jsonschema.Schema{}).WithType(textoOuNulo).WithFormat("date").WithExamples("2026-01-10"))
	r.DefaultOptions = append(r.DefaultOptions, jsonschema.InterceptDefName(nomeDefinicao))

	for _, op := range Operacoes() {
		if err := adicionar(r, op); err != nil {
			return nil, fmt.Errorf("%s %s: %w", op.Metodo, op.Caminho, err)
		}
	}
	return r.Spec, nil
}

func adicionar(r *openapi3.Reflector, op Operacao) error {
	oc, err := r.NewOperationContext(op.Metodo, paramRota.ReplaceAllString(op.Caminho, "{$1}"))
	if err != nil {
		return err
	}
	oc.SetTags(op.tag)
	oc.SetSummary(op.resumo)
	if d := descricaoAcesso(op.acesso); d != "" {
		oc.SetDescription(d)
	}

	if op.params != nil {
		oc.AddReqStructure(op.params)
	}
	if op.corpo != nil {
		oc.AddReqStructure(op.corpo)
	}

	status := op.status
	switch {
	case op.resposta == nil:
		oc.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
	case status == 0:
		status = http.StatusOK
		fallthrough
	default:
		oc.AddRespStructure(op.resposta, openapi.WithHTTPStatus(status))
	}

	if op.corpo != nil {
		oc.AddRespStructure(controllers.ErroValidacao{}, openapi.WithHTTPStatus(http.StatusUnprocessableEntity))
	}
	if op.acesso != publico {
		oc.AddSecurity(cookieSessao)
		oc.AddRespStructure(controllers.Erro{}, openapi.WithHTTPStatus(http.StatusUnauthorized))
	}
	if op.acesso != publico && op.acesso != logado {
		oc.AddRespStructure(controllers.Erro{}, openapi.WithHTTPStatus(http.StatusForbidden))
	}
	if strings.Contains(op.Caminho, ":") {
		oc.AddRespStructure(controllers.Erro{}, openapi.WithHTTPStatus(http.StatusNotFound))
	}

	return r.AddOperation(oc)
}

func descricaoAcesso(a acesso) string {
	switch a {
	case logado:
		return "Requer login."
	case admin, adminObra:
		return "Somente administrador."
	case obra:
		return "Administrador ou usuário de obra associado ao empreendimento."
	}
	return ""
}

func nomeDefinicao(t reflect.Type, padrao string) string {
	if t.Name() == "" {
		return padrao
	}
	nome := t.Name()
	if i := strings.Index(nome, "["); i >= 0 {
		arg := nome[i+1 : len(nome)-1]
		arg = arg[strings.LastIndex(arg, ".")+1:]
		nome = nome[:i] + strings.TrimSuffix(strings.TrimPrefix(arg, "List"), "Row")
	}
	return nome
}

func JSON(spec *openapi3.Spec) ([]byte, error) {
	b, err := spec.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, b, "", "  "); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func PaginaSwagger(specJSON []byte) []byte {
	var spec bytes.Buffer
	json.HTMLEscape(&spec, specJSON)
	return fmt.Appendf(nil, paginaSwagger, spec.String())
}

const paginaSwagger = `<!doctype html>
<html lang="pt-BR">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>API Almoxarifado A2G</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      spec: %s,
      dom_id: "#swagger-ui",
      deepLinking: true,
      persistAuthorization: true,
    });
  </script>
</body>
</html>
`
