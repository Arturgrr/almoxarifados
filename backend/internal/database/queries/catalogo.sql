-- name: ListCategorias :many
SELECT * FROM categorias
WHERE (sqlc.narg('ativo')::boolean IS NULL OR ativo = sqlc.narg('ativo'))
ORDER BY nome;

-- name: GetCategoria :one
SELECT * FROM categorias WHERE id = $1;

-- name: CreateCategoria :one
INSERT INTO categorias (nome) VALUES ($1) RETURNING *;

-- name: UpdateCategoria :one
UPDATE categorias SET nome = $2, ativo = $3 WHERE id = $1 RETURNING *;

-- name: CreateItem :one
INSERT INTO itens (nome, tipo, categoria_id, unidade, granel, tolerancia_pct, descricao)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetItem :one
SELECT * FROM itens WHERE id = $1;

-- name: UpdateItem :one
UPDATE itens
SET nome = $2, tipo = $3, categoria_id = $4, unidade = $5, granel = $6,
    tolerancia_pct = $7, descricao = $8, atualizado_em = now()
WHERE id = $1
RETURNING *;

-- name: SetItemAtivo :exec
UPDATE itens SET ativo = $2, atualizado_em = now() WHERE id = $1;

-- name: DeleteItem :exec
DELETE FROM itens WHERE id = $1;

-- name: ItemTemMovimentacao :one
SELECT EXISTS (SELECT 1 FROM movimentacoes WHERE item_id = $1);

-- name: ListItens :many
SELECT i.*, c.nome AS categoria, count(*) OVER () AS total
FROM itens i
JOIN categorias c ON c.id = i.categoria_id
WHERE (sqlc.narg('busca')::text IS NULL
       OR i.nome ILIKE '%' || sqlc.narg('busca') || '%'
       OR i.codigo ILIKE sqlc.narg('busca') || '%')
  AND (sqlc.narg('tipo')::text IS NULL OR i.tipo = sqlc.narg('tipo'))
  AND (sqlc.narg('categoria_id')::bigint IS NULL OR i.categoria_id = sqlc.narg('categoria_id'))
  AND (sqlc.narg('ativo')::boolean IS NULL OR i.ativo = sqlc.narg('ativo'))
ORDER BY i.nome
LIMIT sqlc.arg('limite') OFFSET sqlc.arg('deslocamento');

-- name: CreateFornecedor :one
INSERT INTO fornecedores (nome, nome_fantasia, documento, telefone, email, cidade, observacoes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetFornecedor :one
SELECT * FROM fornecedores WHERE id = $1;

-- name: UpdateFornecedor :one
UPDATE fornecedores
SET nome = $2, nome_fantasia = $3, documento = $4, telefone = $5, email = $6,
    cidade = $7, observacoes = $8, atualizado_em = now()
WHERE id = $1
RETURNING *;

-- name: SetFornecedorAtivo :exec
UPDATE fornecedores SET ativo = $2, atualizado_em = now() WHERE id = $1;

-- name: DeleteFornecedor :exec
DELETE FROM fornecedores WHERE id = $1;

-- name: FornecedorTemMovimentacao :one
SELECT EXISTS (SELECT 1 FROM movimentacoes WHERE fornecedor_id = $1);

-- name: ListFornecedores :many
SELECT f.*, count(*) OVER () AS total
FROM fornecedores f
WHERE (sqlc.narg('busca')::text IS NULL
       OR f.nome ILIKE '%' || sqlc.narg('busca') || '%'
       OR f.nome_fantasia ILIKE '%' || sqlc.narg('busca') || '%'
       OR f.documento LIKE sqlc.narg('busca') || '%')
  AND (sqlc.narg('ativo')::boolean IS NULL OR f.ativo = sqlc.narg('ativo'))
ORDER BY f.nome
LIMIT sqlc.arg('limite') OFFSET sqlc.arg('deslocamento');
