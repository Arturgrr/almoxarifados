-- name: CreateInventario :one
INSERT INTO inventarios (empreendimento_id, categoria_id, aberto_por)
VALUES ($1, $2, $3)
RETURNING *;

-- name: PopularInventario :exec
INSERT INTO inventario_itens (inventario_id, item_id)
SELECT sqlc.arg('inventario_id'), s.item_id
FROM saldos s
JOIN itens i ON i.id = s.item_id
WHERE s.empreendimento_id = sqlc.arg('empreendimento_id')
  AND s.quantidade > 0
  AND (sqlc.narg('categoria_id')::bigint IS NULL OR i.categoria_id = sqlc.narg('categoria_id'));

-- name: GetInventario :one
SELECT * FROM inventarios WHERE id = $1;

-- name: GetInventarioParaAtualizar :one
SELECT * FROM inventarios WHERE id = $1 FOR UPDATE;

-- name: ListInventarios :many
SELECT inv.id, inv.status, inv.categoria_id, c.nome AS categoria,
       inv.aberto_em, inv.aberto_por, ua.nome AS aberto_por_nome,
       inv.finalizado_em, inv.finalizado_por,
       count(ii.item_id)                                                AS total_itens,
       count(ii.item_id) FILTER (WHERE ii.classificacao = 'variacao_natural') AS variacoes_naturais,
       count(ii.item_id) FILTER (WHERE ii.classificacao = 'anormal')          AS diferencas_anormais
FROM inventarios inv
JOIN usuarios ua ON ua.id = inv.aberto_por
LEFT JOIN categorias c ON c.id = inv.categoria_id
LEFT JOIN inventario_itens ii ON ii.inventario_id = inv.id
WHERE inv.empreendimento_id = $1
GROUP BY inv.id, c.nome, ua.nome
ORDER BY inv.id DESC;

-- name: ListInventarioItens :many
SELECT ii.item_id, i.codigo, i.nome AS item, i.unidade, i.granel, i.tolerancia_pct,
       ii.quantidade_contada, ii.saldo_registrado, ii.diferenca, ii.classificacao,
       ii.motivo, ii.descricao,
       coalesce(s.quantidade, 0)::numeric AS saldo_atual
FROM inventario_itens ii
JOIN inventarios inv ON inv.id = ii.inventario_id
JOIN itens i ON i.id = ii.item_id
LEFT JOIN saldos s ON s.empreendimento_id = inv.empreendimento_id AND s.item_id = ii.item_id
WHERE ii.inventario_id = $1
ORDER BY i.nome;

-- name: SetQuantidadeContada :execrows
UPDATE inventario_itens
SET quantidade_contada = $3
WHERE inventario_id = $1 AND item_id = $2;

-- name: FinalizarInventarioItem :exec
UPDATE inventario_itens
SET saldo_registrado = $3, diferenca = $4, classificacao = $5, motivo = $6, descricao = $7
WHERE inventario_id = $1 AND item_id = $2;

-- name: SetInventarioStatus :exec
UPDATE inventarios
SET status = $2, finalizado_por = $3, finalizado_em = now()
WHERE id = $1;
