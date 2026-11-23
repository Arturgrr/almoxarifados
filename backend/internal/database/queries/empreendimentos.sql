-- name: CreateEmpreendimento :one
INSERT INTO empreendimentos (nome, tipo, endereco, cidade, data_inicio, previsao_termino, responsavel_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateEmpreendimento :one
UPDATE empreendimentos
SET nome = $2, tipo = $3, endereco = $4, cidade = $5, data_inicio = $6,
    previsao_termino = $7, responsavel_id = $8, atualizado_em = now()
WHERE id = $1
RETURNING *;

-- name: SetEmpreendimentoStatus :exec
UPDATE empreendimentos SET status = $2, atualizado_em = now() WHERE id = $1;

-- name: GetEmpreendimento :one
SELECT * FROM empreendimentos WHERE id = $1;

-- name: ListEmpreendimentos :many
SELECT e.id, e.nome, e.tipo, e.cidade, e.status, e.data_inicio,
       count(s.item_id) FILTER (WHERE s.quantidade > 0) AS itens_em_estoque,
       max(s.ultima_movimentacao_em)                      AS ultima_movimentacao_em
FROM empreendimentos e
LEFT JOIN saldos s ON s.empreendimento_id = e.id
WHERE (sqlc.narg('status')::text IS NULL OR e.status = sqlc.narg('status'))
  AND (sqlc.narg('usuario_id')::bigint IS NULL OR EXISTS (
        SELECT 1 FROM empreendimento_usuarios eu
        WHERE eu.empreendimento_id = e.id AND eu.usuario_id = sqlc.narg('usuario_id')))
GROUP BY e.id
ORDER BY e.status, e.nome;

-- name: ListEmpreendimentosDoUsuario :many
SELECT e.id, e.nome, e.status
FROM empreendimento_usuarios eu
JOIN empreendimentos e ON e.id = eu.empreendimento_id
WHERE eu.usuario_id = $1
ORDER BY e.nome;

-- name: IsMembro :one
SELECT EXISTS (
    SELECT 1 FROM empreendimento_usuarios WHERE empreendimento_id = $1 AND usuario_id = $2
);

-- name: ListEquipe :many
SELECT u.id, u.nome, u.email, u.telefone, u.ativo, eu.criado_em AS associado_em
FROM empreendimento_usuarios eu
JOIN usuarios u ON u.id = eu.usuario_id
WHERE eu.empreendimento_id = $1
ORDER BY u.nome;

-- name: AddMembro :exec
INSERT INTO empreendimento_usuarios (empreendimento_id, usuario_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveMembro :exec
DELETE FROM empreendimento_usuarios WHERE empreendimento_id = $1 AND usuario_id = $2;
