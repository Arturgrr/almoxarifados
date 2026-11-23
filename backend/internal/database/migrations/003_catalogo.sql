CREATE TABLE categorias (
    id        bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nome      text        NOT NULL,
    ativo     boolean     NOT NULL DEFAULT true,
    criado_em timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX categorias_nome_key ON categorias (lower(nome));

INSERT INTO categorias (nome) VALUES
    ('Agregados'), ('Aglomerantes'), ('Elétrica'), ('Hidráulica'),
    ('Acabamento'), ('Ferramentas'), ('EPI'), ('Outros');

CREATE SEQUENCE itens_codigo_seq;

CREATE TABLE itens (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigo         text          NOT NULL DEFAULT 'IT' || lpad(nextval('itens_codigo_seq')::text, 5, '0'),
    nome           text          NOT NULL,
    tipo           text          NOT NULL CHECK (tipo IN ('consumo', 'equipamento')),
    categoria_id   bigint        NOT NULL REFERENCES categorias (id),
    unidade        text          NOT NULL CHECK (unidade IN ('kg', 'L', 'm', 'm2', 'm3', 'un')),
    granel         boolean       NOT NULL DEFAULT false,
    tolerancia_pct numeric(5, 2) CHECK (tolerancia_pct BETWEEN 0 AND 100),
    descricao      text,
    ativo          boolean       NOT NULL DEFAULT true,
    criado_em      timestamptz   NOT NULL DEFAULT now(),
    atualizado_em  timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT itens_codigo_key UNIQUE (codigo)
);
CREATE UNIQUE INDEX itens_nome_key ON itens (lower(nome));

CREATE TABLE fornecedores (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nome          text        NOT NULL,
    nome_fantasia text,
    documento     text        CONSTRAINT fornecedores_documento_key UNIQUE,
    telefone      text,
    email         text,
    cidade        text,
    observacoes   text,
    ativo         boolean     NOT NULL DEFAULT true,
    criado_em     timestamptz NOT NULL DEFAULT now(),
    atualizado_em timestamptz NOT NULL DEFAULT now()
);

---- create above / drop below ----

DROP TABLE IF EXISTS fornecedores;
DROP TABLE IF EXISTS itens;
DROP SEQUENCE IF EXISTS itens_codigo_seq;
DROP TABLE IF EXISTS categorias;
