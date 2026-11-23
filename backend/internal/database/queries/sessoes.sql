-- name: CreateSessao :exec
INSERT INTO sessoes (token_hash, usuario_id, expira_em) VALUES ($1, $2, $3);

-- name: TocarSessao :one
UPDATE sessoes s
SET expira_em = sqlc.arg('expira_em')
FROM usuarios u
WHERE s.token_hash = sqlc.arg('token_hash')
  AND s.expira_em > now()
  AND u.id = s.usuario_id
RETURNING u.id, u.nome, u.email, u.perfil, u.ativo, u.trocar_senha;

-- name: DeleteSessao :exec
DELETE FROM sessoes WHERE token_hash = $1;

-- name: DeleteSessoesUsuario :exec
DELETE FROM sessoes WHERE usuario_id = $1;

-- name: DeleteSessoesExpiradas :exec
DELETE FROM sessoes WHERE expira_em <= now();
