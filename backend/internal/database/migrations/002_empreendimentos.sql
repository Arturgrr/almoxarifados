CREATE TABLE empreendimentos (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nome             text        NOT NULL,
    tipo             text        NOT NULL CHECK (tipo IN ('condominio', 'casas', 'edificio', 'outro')),
    endereco         text,
    cidade           text,
    data_inicio      date        NOT NULL,
    previsao_termino date,
    responsavel_id   bigint      REFERENCES usuarios (id),
    status           text        NOT NULL DEFAULT 'em_andamento' CHECK (status IN ('em_andamento', 'concluido')),
    criado_em        timestamptz NOT NULL DEFAULT now(),
    atualizado_em    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX empreendimentos_nome_key ON empreendimentos (lower(nome));

CREATE TABLE empreendimento_usuarios (
    empreendimento_id bigint      NOT NULL REFERENCES empreendimentos (id),
    usuario_id        bigint      NOT NULL REFERENCES usuarios (id),
    criado_em         timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (empreendimento_id, usuario_id)
);
CREATE INDEX empreendimento_usuarios_usuario_id_idx ON empreendimento_usuarios (usuario_id);

---- create above / drop below ----

DROP TABLE IF EXISTS empreendimento_usuarios;
DROP TABLE IF EXISTS empreendimentos;
