# Almoxarifado A2G

Sistema web de gestão de almoxarifado de obras, desenvolvido para o **Grupo A2G** na disciplina
CSI606 – Sistemas Web I.

O Grupo A2G mantém várias obras ao mesmo tempo, cada uma com seu próprio almoxarifado, e hoje controla
os materiais em planilhas. O sistema substitui esse controle: cada obra tem um saldo confiável,
toda retirada fica registrada (o quê, quanto, quando, quem e para qual serviço), as perdas passam a ser
medidas e a administração enxerga o estoque de todas as obras num só lugar.

## Documentação

| | |
| --- | --- |
| [`docs/requisitos.md`](docs/requisitos.md) | Documento de requisitos (RF, RNF e regras de negócio) |
| [`docs/openapi.yaml`](docs/openapi.yaml) / [`openapi.json`](docs/openapi.json) | Especificação OpenAPI 3 da API |
| [`docs/swagger.html`](docs/swagger.html) | Swagger UI estático: abra no navegador para consultar as rotas |
| `http://localhost:3000/docs` | Swagger servido pela própria API, com "Try it out" funcionando |

Para testar as rotas pelo Swagger, use o da API (`/docs`): chame `POST /api/auth/login` e as
próximas requisições já vão com o cookie de sessão. O `swagger.html` estático serve só para leitura,
porque o navegador bloqueia chamadas feitas de um arquivo local.

## Estrutura do repositório

```
docs/        requisitos e documentação da API
backend/     API REST em Go
frontend/    interface web (em desenvolvimento)
```

## Backend

**Tecnologias:** Go, [Fiber](https://gofiber.io) (HTTP), PostgreSQL, [pgx](https://github.com/jackc/pgx)
(driver), [sqlc](https://sqlc.dev) (gera o código Go a partir do SQL), [tern](https://github.com/jackc/tern)
(migrações), [air](https://github.com/air-verse/air) (hot reload) e
[openapi-go](https://github.com/swaggest/openapi-go) (gera o OpenAPI a partir dos tipos Go).

### Rodando localmente

Pré-requisitos:

- [Go](https://go.dev/dl/) 1.24 ou mais recente
- [Docker](https://docs.docker.com/get-docker/) (para o PostgreSQL)
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html), só se for alterar as queries (`brew install sqlc`)

O `tern` e o `air` estão declarados como ferramentas no `go.mod` (`go tool`), então não precisam ser instalados.

```bash
cd backend
cp .env.example .env    # variáveis do banco, porta e administrador inicial
make db                 # sobe o PostgreSQL no Docker (porta 5433)
make migrate            # cria as tabelas
make dev                # sobe a API com hot reload em http://localhost:3000
```

Na primeira execução, como não existe cadastro público, a API cria um administrador com os dados de
`ADMIN_EMAIL` e `ADMIN_PASSWORD` do `.env` (padrão: `admin@a2g.com.br` / `trocar123`).

Para conferir: `curl http://localhost:3000/api/health` ou abra `http://localhost:3000/docs`.

### Variáveis de ambiente

| Variável | Para quê |
| --- | --- |
| `PORT` | Porta da API (padrão `3000`) |
| `FRONTEND_URL` | Origem liberada no CORS (o frontend) |
| `COOKIE_SECURE` | `true` em produção (HTTPS); `false` para rodar em `http://localhost` |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | Conexão com o PostgreSQL; também usadas pelo `docker-compose` e pelo `tern` |
| `ADMIN_NAME`, `ADMIN_EMAIL`, `ADMIN_PASSWORD` | Administrador criado quando o banco não tem usuários |

### Comandos

| Comando | O que faz |
| --- | --- |
| `make db` | Sobe o PostgreSQL (`docker compose up -d`) |
| `make migrate` | Aplica as migrações pendentes |
| `make migrate ARGS="--destination -1"` | Desfaz a última migração |
| `make sqlc` | Regera `internal/database` a partir das queries SQL |
| `make openapi` | Exporta `docs/openapi.yaml`, `docs/openapi.json` e `docs/swagger.html` |
| `make dev` | Roda a API com hot reload |
| `make test` | Roda os testes |

Para criar uma migração: `go tool tern new nome --migrations ./internal/database/migrations`.
Para apagar o banco local e começar do zero: `docker compose down -v`.

### Organização do código

O backend segue um MVC enxuto, sem a camada de view (que fica no frontend):

```
backend/
├── cmd/
│   ├── api/           ponto de entrada: lê o .env, abre o banco e sobe o servidor
│   ├── terndotenv/    roda as migrações do tern com as variáveis do .env
│   └── openapi/       exporta a especificação OpenAPI para docs/
└── internal/
    ├── routes/        todas as rotas e os middlewares de cada uma
    ├── controllers/   handlers HTTP: leem a requisição, validam, aplicam as regras e respondem
    ├── database/      model: migrações, queries SQL e o código gerado pelo sqlc
    ├── docs/          lista das rotas documentadas e geração do OpenAPI/Swagger
    └── auth/          hash de senha e token de sessão
```

Fluxo de uma requisição: `routes` → middlewares (`Autenticado`, `SomenteAdmin`,
`AcessoEmpreendimento`) → `controllers` → `database` (queries do sqlc).

O OpenAPI é gerado por reflexão a partir das structs de request/response, usando a lista de
`internal/docs/operacoes.go`. Ao criar uma rota nova, adicione-a nessa lista: o teste
`internal/routes/routes_test.go` falha se alguma rota da API não estiver documentada.

### Principais decisões

- **Sessão em cookie HttpOnly** (`SameSite=Strict`), guardada na tabela `sessoes` e renovada a cada
  requisição; expira após 8 h sem uso. Desativar um usuário ou removê-lo da equipe corta o acesso na hora.
  Após 5 senhas erradas seguidas, o login fica bloqueado por 15 minutos.
- **Permissões verificadas no backend**: toda rota confere login, perfil e associação ao empreendimento
  (403 quando não pode).
- **Saldo nunca é digitado**: a tabela `saldos` só muda junto com uma movimentação, na mesma transação.
  O `CHECK (quantidade >= 0)` do banco impede saldo negativo, inclusive com duas saídas simultâneas.
- **Movimentações não são editadas nem apagadas**: a correção é feita por estorno, que gera um
  lançamento inverso vinculado ao original. A quantidade é gravada com sinal (+ entra, − sai).
- **Inventário**: compara o contado com o saldo. Material a granel dentro da tolerância vira
  variação natural; qualquer outra diferença exige justificativa e aparece em "perdas anormais".
- **Quantidades** em `NUMERIC` (nunca ponto flutuante), trafegadas no JSON como string (`"18.5"`).
  Datas no formato `aaaa-mm-dd`.
- **Erros** sempre em JSON: `{"erro": "..."}`; em validação (422) também `{"campos": {"campo": "mensagem"}}`.

### Pendências

Itens marcados como desejáveis nos requisitos e ainda não implementados: unidade de compra com
conversão (RF11), histórico do fornecedor (RF14), transferência entre empreendimentos (RF20),
relatórios e exportação (RF24), estoque mínimo, contagem cega e bloqueio de itens durante o inventário,
e troca obrigatória de senha no primeiro acesso (a API já devolve o campo `trocar_senha`).
