CREATE TABLE usuarios (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nome              text        NOT NULL,
    email             text        NOT NULL CONSTRAINT usuarios_email_key UNIQUE,
    telefone          text,
    perfil            text        NOT NULL CHECK (perfil IN ('admin', 'obra')),
    senha_hash        text        NOT NULL,
    ativo             boolean     NOT NULL DEFAULT true,
    trocar_senha      boolean     NOT NULL DEFAULT true,
    tentativas_falhas integer     NOT NULL DEFAULT 0,
    bloqueado_ate     timestamptz,
    criado_em         timestamptz NOT NULL DEFAULT now(),
    atualizado_em     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessoes (
    token_hash bytea       PRIMARY KEY,
    usuario_id bigint      NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
    criado_em  timestamptz NOT NULL DEFAULT now(),
    expira_em  timestamptz NOT NULL
);
CREATE INDEX sessoes_usuario_id_idx ON sessoes (usuario_id);

CREATE TABLE login_tentativas (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email      text        NOT NULL,
    usuario_id bigint      REFERENCES usuarios (id),
    sucesso    boolean     NOT NULL,
    ip         text        NOT NULL,
    criado_em  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE auditoria (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    usuario_id  bigint      NOT NULL REFERENCES usuarios (id),
    acao        text        NOT NULL,
    entidade    text        NOT NULL,
    entidade_id bigint      NOT NULL,
    criado_em   timestamptz NOT NULL DEFAULT now()
);

---- create above / drop below ----

DROP TABLE IF EXISTS auditoria;
DROP TABLE IF EXISTS login_tentativas;
DROP TABLE IF EXISTS sessoes;
DROP TABLE IF EXISTS usuarios;
