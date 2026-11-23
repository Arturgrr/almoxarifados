package docs

import (
	"net/http"

	"almoxarifados-a2g/internal/controllers"
	"almoxarifados-a2g/internal/database"
)

type acesso int

const (
	publico acesso = iota
	logado
	admin
	obra
	adminObra
)

type Operacao struct {
	Metodo   string
	Caminho  string
	tag      string
	resumo   string
	acesso   acesso
	params   any
	corpo    any
	resposta any
	status   int
}

type PathID struct {
	ID int64 `path:"id"`
}

type pathMembro struct {
	PathID
	UsuarioID int64 `path:"usuarioId"`
}

type pathMovimentacao struct {
	PathID
	MovID int64 `path:"movId"`
}

type pathInventario struct {
	PathID
	InvID int64 `path:"invId"`
}

type Paginacao struct {
	Pagina    int `query:"pagina" default:"1" minimum:"1"`
	PorPagina int `query:"por_pagina" default:"50" minimum:"1" maximum:"200"`
}

type filtroUsuarios struct {
	Busca  string `query:"busca" description:"Nome ou e-mail"`
	Perfil string `query:"perfil" enum:"admin,obra"`
	Ativo  *bool  `query:"ativo"`
}

type filtroAtivo struct {
	Ativo *bool `query:"ativo"`
}

type filtroItens struct {
	Paginacao
	Busca       string `query:"busca" description:"Nome ou código"`
	Tipo        string `query:"tipo" enum:"consumo,equipamento"`
	CategoriaID int64  `query:"categoria_id"`
	Ativo       *bool  `query:"ativo"`
}

type filtroFornecedores struct {
	Paginacao
	Busca string `query:"busca" description:"Nome, nome fantasia ou CNPJ/CPF"`
	Ativo *bool  `query:"ativo"`
}

type filtroEmpreendimentos struct {
	Status string `query:"status" enum:"em_andamento,concluido"`
}

type filtroSaldos struct {
	PathID
	Busca          string `query:"busca"`
	CategoriaID    int64  `query:"categoria_id"`
	Tipo           string `query:"tipo" enum:"consumo,equipamento"`
	IncluirZerados bool   `query:"incluir_zerados" description:"Itens com saldo zero ficam ocultos por padrão"`
}

type filtroMovimentacoes struct {
	PathID
	Paginacao
	DataInicio   string `query:"data_inicio" format:"date"`
	DataFim      string `query:"data_fim" format:"date"`
	Tipo         string `query:"tipo" enum:"entrada,saida,perda,ajuste,estorno"`
	ItemID       int64  `query:"item_id" description:"Com este filtro a lista vira a ficha do item"`
	UsuarioID    int64  `query:"usuario_id"`
	FornecedorID int64  `query:"fornecedor_id"`
}

type filtroPerdas struct {
	Paginacao
	EmpreendimentoID int64  `query:"empreendimento_id"`
	DataInicio       string `query:"data_inicio" format:"date"`
	DataFim          string `query:"data_fim" format:"date"`
}

type Saude struct {
	Status string `json:"status" example:"ok"`
}

const (
	tagAuth            = "Autenticação"
	tagUsuarios        = "Usuários"
	tagEmpreendimentos = "Empreendimentos"
	tagCatalogo        = "Catálogo"
	tagFornecedores    = "Fornecedores"
	tagEstoque         = "Estoque"
	tagInventario      = "Inventário"
	tagAdmin           = "Administração"
)

const (
	get  = http.MethodGet
	post = http.MethodPost
	put  = http.MethodPut
	del  = http.MethodDelete
)

func Operacoes() []Operacao {
	return []Operacao{
		{Metodo: get, Caminho: "/api/health", tag: tagAuth, resumo: "Verifica se a API está no ar", resposta: Saude{}},

		{Metodo: post, Caminho: "/api/auth/login", tag: tagAuth, resumo: "Entra no sistema e recebe o cookie de sessão", corpo: controllers.LoginReq{}, resposta: controllers.SessaoResp{}},
		{Metodo: post, Caminho: "/api/auth/logout", tag: tagAuth, resumo: "Sai do sistema", acesso: logado},
		{Metodo: get, Caminho: "/api/auth/me", tag: tagAuth, resumo: "Usuário logado e seus empreendimentos", acesso: logado, resposta: controllers.SessaoResp{}},
		{Metodo: put, Caminho: "/api/auth/senha", tag: tagAuth, resumo: "Altera a própria senha", acesso: logado, corpo: controllers.AlterarSenhaReq{}},

		{Metodo: get, Caminho: "/api/usuarios", tag: tagUsuarios, resumo: "Lista usuários", acesso: admin, params: filtroUsuarios{}, resposta: []database.ListUsuariosRow{}},
		{Metodo: post, Caminho: "/api/usuarios", tag: tagUsuarios, resumo: "Cadastra usuário", acesso: admin, corpo: controllers.NovoUsuarioReq{}, resposta: database.Usuario{}, status: http.StatusCreated},
		{Metodo: get, Caminho: "/api/usuarios/:id", tag: tagUsuarios, resumo: "Busca usuário", acesso: admin, params: PathID{}, resposta: database.Usuario{}},
		{Metodo: put, Caminho: "/api/usuarios/:id", tag: tagUsuarios, resumo: "Edita usuário", acesso: admin, params: PathID{}, corpo: controllers.UsuarioReq{}, resposta: database.Usuario{}},
		{Metodo: post, Caminho: "/api/usuarios/:id/desativar", tag: tagUsuarios, resumo: "Desativa usuário e encerra suas sessões", acesso: admin, params: PathID{}},
		{Metodo: post, Caminho: "/api/usuarios/:id/reativar", tag: tagUsuarios, resumo: "Reativa usuário", acesso: admin, params: PathID{}},
		{Metodo: put, Caminho: "/api/usuarios/:id/senha", tag: tagUsuarios, resumo: "Redefine a senha de um usuário", acesso: admin, params: PathID{}, corpo: controllers.RedefinirSenhaReq{}},

		{Metodo: get, Caminho: "/api/categorias", tag: tagCatalogo, resumo: "Lista categorias", acesso: logado, params: filtroAtivo{}, resposta: []database.Categoria{}},
		{Metodo: post, Caminho: "/api/categorias", tag: tagCatalogo, resumo: "Cadastra categoria", acesso: admin, corpo: controllers.CategoriaReq{}, resposta: database.Categoria{}, status: http.StatusCreated},
		{Metodo: put, Caminho: "/api/categorias/:id", tag: tagCatalogo, resumo: "Edita categoria", acesso: admin, params: PathID{}, corpo: controllers.CategoriaReq{}, resposta: database.Categoria{}},

		{Metodo: get, Caminho: "/api/itens", tag: tagCatalogo, resumo: "Lista o catálogo de itens", acesso: logado, params: filtroItens{}, resposta: controllers.Pagina[database.ListItensRow]{}},
		{Metodo: post, Caminho: "/api/itens", tag: tagCatalogo, resumo: "Cadastra item no catálogo", acesso: logado, corpo: controllers.ItemReq{}, resposta: database.Item{}, status: http.StatusCreated},
		{Metodo: get, Caminho: "/api/itens/:id", tag: tagCatalogo, resumo: "Busca item", acesso: logado, params: PathID{}, resposta: database.Item{}},
		{Metodo: put, Caminho: "/api/itens/:id", tag: tagCatalogo, resumo: "Edita item", acesso: admin, params: PathID{}, corpo: controllers.ItemReq{}, resposta: database.Item{}},
		{Metodo: del, Caminho: "/api/itens/:id", tag: tagCatalogo, resumo: "Exclui item sem movimentação", acesso: admin, params: PathID{}},
		{Metodo: post, Caminho: "/api/itens/:id/desativar", tag: tagCatalogo, resumo: "Desativa item", acesso: admin, params: PathID{}},
		{Metodo: post, Caminho: "/api/itens/:id/reativar", tag: tagCatalogo, resumo: "Reativa item", acesso: admin, params: PathID{}},
		{Metodo: get, Caminho: "/api/itens/:id/saldos", tag: tagCatalogo, resumo: "Saldo do item em cada empreendimento", acesso: admin, params: PathID{}, resposta: controllers.SaldoConsolidadoResp{}},

		{Metodo: get, Caminho: "/api/fornecedores", tag: tagFornecedores, resumo: "Lista fornecedores", acesso: logado, params: filtroFornecedores{}, resposta: controllers.Pagina[database.ListFornecedoresRow]{}},
		{Metodo: post, Caminho: "/api/fornecedores", tag: tagFornecedores, resumo: "Cadastra fornecedor", acesso: logado, corpo: controllers.FornecedorReq{}, resposta: database.Fornecedor{}, status: http.StatusCreated},
		{Metodo: get, Caminho: "/api/fornecedores/:id", tag: tagFornecedores, resumo: "Busca fornecedor", acesso: logado, params: PathID{}, resposta: database.Fornecedor{}},
		{Metodo: put, Caminho: "/api/fornecedores/:id", tag: tagFornecedores, resumo: "Edita fornecedor", acesso: admin, params: PathID{}, corpo: controllers.FornecedorReq{}, resposta: database.Fornecedor{}},
		{Metodo: del, Caminho: "/api/fornecedores/:id", tag: tagFornecedores, resumo: "Exclui fornecedor sem movimentação", acesso: admin, params: PathID{}},
		{Metodo: post, Caminho: "/api/fornecedores/:id/desativar", tag: tagFornecedores, resumo: "Desativa fornecedor", acesso: admin, params: PathID{}},
		{Metodo: post, Caminho: "/api/fornecedores/:id/reativar", tag: tagFornecedores, resumo: "Reativa fornecedor", acesso: admin, params: PathID{}},

		{Metodo: get, Caminho: "/api/empreendimentos", tag: tagEmpreendimentos, resumo: "Lista os empreendimentos que o usuário pode acessar", acesso: logado, params: filtroEmpreendimentos{}, resposta: []database.ListEmpreendimentosRow{}},
		{Metodo: post, Caminho: "/api/empreendimentos", tag: tagEmpreendimentos, resumo: "Cadastra empreendimento", acesso: admin, corpo: controllers.EmpreendimentoReq{}, resposta: database.Empreendimento{}, status: http.StatusCreated},
		{Metodo: get, Caminho: "/api/empreendimentos/:id", tag: tagEmpreendimentos, resumo: "Busca empreendimento", acesso: obra, params: PathID{}, resposta: database.Empreendimento{}},
		{Metodo: put, Caminho: "/api/empreendimentos/:id", tag: tagEmpreendimentos, resumo: "Edita empreendimento", acesso: adminObra, params: PathID{}, corpo: controllers.EmpreendimentoReq{}, resposta: database.Empreendimento{}},
		{Metodo: post, Caminho: "/api/empreendimentos/:id/encerrar", tag: tagEmpreendimentos, resumo: "Encerra o empreendimento (fica só para consulta)", acesso: adminObra, params: PathID{}},
		{Metodo: post, Caminho: "/api/empreendimentos/:id/reabrir", tag: tagEmpreendimentos, resumo: "Reabre empreendimento concluído", acesso: adminObra, params: PathID{}},
		{Metodo: get, Caminho: "/api/empreendimentos/:id/equipe", tag: tagEmpreendimentos, resumo: "Lista a equipe", acesso: obra, params: PathID{}, resposta: []database.ListEquipeRow{}},
		{Metodo: post, Caminho: "/api/empreendimentos/:id/equipe", tag: tagEmpreendimentos, resumo: "Associa usuário de obra", acesso: adminObra, params: PathID{}, corpo: controllers.MembroReq{}},
		{Metodo: del, Caminho: "/api/empreendimentos/:id/equipe/:usuarioId", tag: tagEmpreendimentos, resumo: "Remove usuário da equipe", acesso: adminObra, params: pathMembro{}},

		{Metodo: get, Caminho: "/api/empreendimentos/:id/saldos", tag: tagEstoque, resumo: "Saldo atual de cada item", acesso: obra, params: filtroSaldos{}, resposta: []database.ListSaldosRow{}},
		{Metodo: get, Caminho: "/api/empreendimentos/:id/destinos", tag: tagEstoque, resumo: "Destinos já usados em saídas (sugestões)", acesso: obra, params: PathID{}, resposta: []string{}},
		{Metodo: get, Caminho: "/api/empreendimentos/:id/movimentacoes", tag: tagEstoque, resumo: "Histórico de movimentações", acesso: obra, params: filtroMovimentacoes{}, resposta: controllers.Pagina[database.ListMovimentacoesRow]{}},
		{Metodo: post, Caminho: "/api/empreendimentos/:id/entradas", tag: tagEstoque, resumo: "Registra entrada (compra ou devolução)", acesso: obra, params: PathID{}, corpo: controllers.EntradaReq{}, resposta: []database.Movimentacao{}, status: http.StatusCreated},
		{Metodo: post, Caminho: "/api/empreendimentos/:id/saidas", tag: tagEstoque, resumo: "Registra saída", acesso: obra, params: PathID{}, corpo: controllers.SaidaReq{}, resposta: database.Movimentacao{}, status: http.StatusCreated},
		{Metodo: post, Caminho: "/api/empreendimentos/:id/perdas", tag: tagEstoque, resumo: "Registra perda", acesso: obra, params: PathID{}, corpo: controllers.PerdaReq{}, resposta: database.Movimentacao{}, status: http.StatusCreated},
		{Metodo: post, Caminho: "/api/empreendimentos/:id/movimentacoes/:movId/estorno", tag: tagEstoque, resumo: "Estorna uma movimentação (usuário de obra: só as próprias)", acesso: obra, params: pathMovimentacao{}, corpo: controllers.EstornoReq{}, resposta: database.Movimentacao{}, status: http.StatusCreated},

		{Metodo: get, Caminho: "/api/empreendimentos/:id/inventarios", tag: tagInventario, resumo: "Histórico de inventários", acesso: obra, params: PathID{}, resposta: []database.ListInventariosRow{}},
		{Metodo: post, Caminho: "/api/empreendimentos/:id/inventarios", tag: tagInventario, resumo: "Abre inventário", acesso: obra, params: PathID{}, corpo: controllers.AbrirInventarioReq{}, resposta: controllers.InventarioResp{}, status: http.StatusCreated},
		{Metodo: get, Caminho: "/api/empreendimentos/:id/inventarios/:invId", tag: tagInventario, resumo: "Inventário com itens; aberto mostra a prévia das diferenças", acesso: obra, params: pathInventario{}, resposta: controllers.InventarioResp{}},
		{Metodo: put, Caminho: "/api/empreendimentos/:id/inventarios/:invId/contagem", tag: tagInventario, resumo: "Salva a contagem (parcial ou total)", acesso: obra, params: pathInventario{}, corpo: controllers.ContagemReq{}, resposta: controllers.InventarioResp{}},
		{Metodo: post, Caminho: "/api/empreendimentos/:id/inventarios/:invId/finalizar", tag: tagInventario, resumo: "Finaliza e gera os ajustes", acesso: obra, params: pathInventario{}, corpo: controllers.FinalizarReq{}, resposta: controllers.InventarioResp{}},
		{Metodo: post, Caminho: "/api/empreendimentos/:id/inventarios/:invId/cancelar", tag: tagInventario, resumo: "Cancela inventário aberto", acesso: obra, params: pathInventario{}},

		{Metodo: get, Caminho: "/api/admin/painel", tag: tagAdmin, resumo: "Visão geral dos empreendimentos em andamento", acesso: admin, resposta: []database.PainelAdminRow{}},
		{Metodo: get, Caminho: "/api/admin/perdas-anormais", tag: tagAdmin, resumo: "Perdas e ajustes de inventário fora da tolerância", acesso: admin, params: filtroPerdas{}, resposta: controllers.Pagina[database.ListPerdasAnormaisRow]{}},
	}
}
