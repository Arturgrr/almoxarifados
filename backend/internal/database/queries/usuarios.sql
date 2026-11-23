-- name: CountUsuarios :one
SELECT count(*) FROM usuarios;

-- name: CountAdminsAtivos :one
SELECT count(*) FROM usuarios WHERE perfil = 'admin' AND ativo;

-- name: CreateUsuario :one
INSERT INTO usuarios (nome, email, telefone, perfil, senha_hash)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUsuario :one
SELECT * FROM usuarios WHERE id = $1;

-- name: GetUsuarioByEmail :one
SELECT * FROM usuarios WHERE email = $1;

-- name: ListUsuarios :many
SELECT u.id, u.nome, u.email, u.telefone, u.perfil, u.ativo, u.criado_em,
       coalesce(array_agg(e.nome ORDER BY e.nome) FILTER (WHERE e.id IS NOT NULL), '{}')::text[] AS empreendimentos
FROM usuarios u
LEFT JOIN empreendimento_usuarios eu ON eu.usuario_id = u.id
LEFT JOIN empreendimentos e ON e.id = eu.empreendimento_id
WHERE (sqlc.narg('busca')::text IS NULL
       OR u.nome ILIKE '%' || sqlc.narg('busca') || '%'
       OR u.email ILIKE '%' || sqlc.narg('busca') || '%')
  AND (sqlc.narg('perfil')::text IS NULL OR u.perfil = sqlc.narg('perfil'))
  AND (sqlc.narg('ativo')::boolean IS NULL OR u.ativo = sqlc.narg('ativo'))
GROUP BY u.id
ORDER BY u.nome;

-- name: UpdateUsuario :one
UPDATE usuarios
SET nome = $2, email = $3, telefone = $4, perfil = $5, atualizado_em = now()
WHERE id = $1
RETURNING *;

-- name: SetUsuarioAtivo :exec
UPDATE usuarios SET ativo = $2, atualizado_em = now() WHERE id = $1;

-- name: SetUsuarioSenha :exec
UPDATE usuarios
SET senha_hash = $2, trocar_senha = $3, tentativas_falhas = 0, bloqueado_ate = NULL, atualizado_em = now()
WHERE id = $1;

-- name: RegistrarFalhaLogin :exec
UPDATE usuarios
SET tentativas_falhas = CASE WHEN tentativas_falhas + 1 >= 5 THEN 0 ELSE tentativas_falhas + 1 END,
    bloqueado_ate     = CASE WHEN tentativas_falhas + 1 >= 5 THEN now() + interval '15 minutes' ELSE bloqueado_ate END
WHERE id = $1;

-- name: ResetarFalhasLogin :exec
UPDATE usuarios SET tentativas_falhas = 0, bloqueado_ate = NULL WHERE id = $1;

-- name: CreateLoginTentativa :exec
INSERT INTO login_tentativas (email, usuario_id, sucesso, ip) VALUES ($1, $2, $3, $4);

-- name: CreateAuditoria :exec
INSERT INTO auditoria (usuario_id, acao, entidade, entidade_id) VALUES ($1, $2, $3, $4);
