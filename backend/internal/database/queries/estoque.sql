-- name: CreditarSaldo :one
INSERT INTO saldos (empreendimento_id, item_id, quantidade)
VALUES (sqlc.arg('empreendimento_id'), sqlc.arg('item_id'), sqlc.arg('quantidade'))
ON CONFLICT (empreendimento_id, item_id) DO UPDATE
SET quantidade = saldos.quantidade + excluded.quantidade,
    ultima_movimentacao_em = now()
RETURNING quantidade;

-- name: DebitarSaldo :one
UPDATE saldos
SET quantidade = quantidade - sqlc.arg('quantidade'),
    ultima_movimentacao_em = now()
WHERE empreendimento_id = sqlc.arg('empreendimento_id') AND item_id = sqlc.arg('item_id')
RETURNING quantidade;

-- name: GetSaldoParaAtualizar :one
SELECT quantidade FROM saldos
WHERE empreendimento_id = $1 AND item_id = $2
FOR UPDATE;

-- name: ListSaldos :many
SELECT i.id AS item_id, i.codigo, i.nome, i.tipo, i.unidade, i.granel,
       c.id AS categoria_id, c.nome AS categoria,
       s.quantidade, s.ultima_movimentacao_em
FROM saldos s
JOIN itens i ON i.id = s.item_id
JOIN categorias c ON c.id = i.categoria_id
WHERE s.empreendimento_id = sqlc.arg('empreendimento_id')
  AND (sqlc.arg('incluir_zerados')::boolean OR s.quantidade > 0)
  AND (sqlc.narg('busca')::text IS NULL OR i.nome ILIKE '%' || sqlc.narg('busca') || '%')
  AND (sqlc.narg('categoria_id')::bigint IS NULL OR i.categoria_id = sqlc.narg('categoria_id'))
  AND (sqlc.narg('tipo')::text IS NULL OR i.tipo = sqlc.narg('tipo'))
ORDER BY i.nome;

-- name: SaldoConsolidadoItem :many
SELECT e.id AS empreendimento_id, e.nome AS empreendimento,
       coalesce(s.quantidade, 0)::numeric AS quantidade
FROM empreendimentos e
LEFT JOIN saldos s ON s.empreendimento_id = e.id AND s.item_id = $1
WHERE e.status = 'em_andamento'
ORDER BY e.nome;

-- name: CreateMovimentacao :one
INSERT INTO movimentacoes (
    empreendimento_id, item_id, tipo, subtipo, quantidade, saldo_apos, data, usuario_id,
    fornecedor_id, origem_compra, nota_fiscal, valor_unitario, destino, retirado_por,
    motivo, descricao, estorno_de, inventario_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
)
RETURNING *;

-- name: GetMovimentacao :one
SELECT m.*, EXISTS (SELECT 1 FROM movimentacoes e WHERE e.estorno_de = m.id) AS estornada
FROM movimentacoes m
WHERE m.id = $1;

-- name: ListMovimentacoes :many
SELECT m.id, m.tipo, m.subtipo, m.data, m.lancado_em,
       m.item_id, i.nome AS item, i.unidade, m.quantidade, m.saldo_apos,
       m.usuario_id, u.nome AS usuario,
       m.fornecedor_id, f.nome AS fornecedor, m.origem_compra, m.nota_fiscal, m.valor_unitario,
       m.destino, m.retirado_por, m.motivo, m.descricao, m.estorno_de, m.inventario_id,
       EXISTS (SELECT 1 FROM movimentacoes x WHERE x.estorno_de = m.id) AS estornada,
       count(*) OVER () AS total
FROM movimentacoes m
JOIN itens i ON i.id = m.item_id
JOIN usuarios u ON u.id = m.usuario_id
LEFT JOIN fornecedores f ON f.id = m.fornecedor_id
WHERE m.empreendimento_id = sqlc.arg('empreendimento_id')
  AND (sqlc.narg('data_inicio')::date IS NULL OR m.data >= sqlc.narg('data_inicio'))
  AND (sqlc.narg('data_fim')::date IS NULL OR m.data <= sqlc.narg('data_fim'))
  AND (sqlc.narg('tipo')::text IS NULL OR m.tipo = sqlc.narg('tipo'))
  AND (sqlc.narg('item_id')::bigint IS NULL OR m.item_id = sqlc.narg('item_id'))
  AND (sqlc.narg('usuario_id')::bigint IS NULL OR m.usuario_id = sqlc.narg('usuario_id'))
  AND (sqlc.narg('fornecedor_id')::bigint IS NULL OR m.fornecedor_id = sqlc.narg('fornecedor_id'))
ORDER BY m.id DESC
LIMIT sqlc.arg('limite') OFFSET sqlc.arg('deslocamento');

-- name: ListDestinos :many
SELECT destino::text
FROM movimentacoes
WHERE empreendimento_id = $1 AND tipo = 'saida' AND destino IS NOT NULL
GROUP BY destino
ORDER BY max(id) DESC
LIMIT 50;

-- name: PainelAdmin :many
SELECT e.id, e.nome, e.cidade,
       (SELECT count(*) FROM saldos s
        WHERE s.empreendimento_id = e.id AND s.quantidade > 0) AS itens_em_estoque,
       (SELECT count(*) FROM movimentacoes m
        WHERE m.empreendimento_id = e.id
          AND (m.tipo = 'perda' OR (m.tipo = 'ajuste' AND m.subtipo = 'anormal' AND m.quantidade < 0))
          AND m.data >= date_trunc('month', current_date)
          AND NOT EXISTS (SELECT 1 FROM movimentacoes x WHERE x.estorno_de = m.id)) AS perdas_no_mes
FROM empreendimentos e
WHERE e.status = 'em_andamento'
ORDER BY e.nome;

-- name: ListPerdasAnormais :many
SELECT m.id, m.empreendimento_id, e.nome AS empreendimento, m.tipo, m.data, m.lancado_em,
       m.item_id, i.nome AS item, i.unidade, m.quantidade, m.valor_unitario,
       m.motivo, m.descricao, u.nome AS usuario,
       count(*) OVER () AS total
FROM movimentacoes m
JOIN empreendimentos e ON e.id = m.empreendimento_id
JOIN itens i ON i.id = m.item_id
JOIN usuarios u ON u.id = m.usuario_id
WHERE (m.tipo = 'perda' OR (m.tipo = 'ajuste' AND m.subtipo = 'anormal'))
  AND NOT EXISTS (SELECT 1 FROM movimentacoes x WHERE x.estorno_de = m.id)
  AND (sqlc.narg('empreendimento_id')::bigint IS NULL OR m.empreendimento_id = sqlc.narg('empreendimento_id'))
  AND (sqlc.narg('data_inicio')::date IS NULL OR m.data >= sqlc.narg('data_inicio'))
  AND (sqlc.narg('data_fim')::date IS NULL OR m.data <= sqlc.narg('data_fim'))
ORDER BY m.id DESC
LIMIT sqlc.arg('limite') OFFSET sqlc.arg('deslocamento');
