CREATE TABLE saldos (
    empreendimento_id      bigint         NOT NULL REFERENCES empreendimentos (id),
    item_id                bigint         NOT NULL REFERENCES itens (id),
    quantidade             numeric(14, 3) NOT NULL DEFAULT 0 CONSTRAINT saldos_nao_negativo CHECK (quantidade >= 0),
    ultima_movimentacao_em timestamptz    NOT NULL DEFAULT now(),
    PRIMARY KEY (empreendimento_id, item_id)
);

CREATE TABLE inventarios (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    empreendimento_id bigint      NOT NULL REFERENCES empreendimentos (id),
    categoria_id      bigint      REFERENCES categorias (id),
    status            text        NOT NULL DEFAULT 'aberto' CHECK (status IN ('aberto', 'finalizado', 'cancelado')),
    aberto_por        bigint      NOT NULL REFERENCES usuarios (id),
    aberto_em         timestamptz NOT NULL DEFAULT now(),
    finalizado_por    bigint      REFERENCES usuarios (id),
    finalizado_em     timestamptz
);
CREATE UNIQUE INDEX inventarios_um_aberto ON inventarios (empreendimento_id) WHERE status = 'aberto';

CREATE TABLE inventario_itens (
    inventario_id      bigint         NOT NULL REFERENCES inventarios (id),
    item_id            bigint         NOT NULL REFERENCES itens (id),
    quantidade_contada numeric(14, 3) CHECK (quantidade_contada >= 0),
    saldo_registrado   numeric(14, 3),
    diferenca          numeric(14, 3),
    classificacao      text CHECK (classificacao IN ('sem_diferenca', 'variacao_natural', 'anormal')),
    motivo             text,
    descricao          text,
    PRIMARY KEY (inventario_id, item_id)
);

CREATE TABLE movimentacoes (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    empreendimento_id bigint         NOT NULL REFERENCES empreendimentos (id),
    item_id           bigint         NOT NULL REFERENCES itens (id),
    tipo              text           NOT NULL CHECK (tipo IN ('entrada', 'saida', 'perda', 'ajuste', 'estorno')),
    subtipo           text           CHECK (subtipo IN ('compra', 'devolucao', 'variacao_natural', 'anormal')),
    quantidade        numeric(14, 3) NOT NULL CHECK (quantidade <> 0),
    saldo_apos        numeric(14, 3) NOT NULL,
    data              date           NOT NULL,
    lancado_em        timestamptz    NOT NULL DEFAULT now(),
    usuario_id        bigint         NOT NULL REFERENCES usuarios (id),
    fornecedor_id     bigint         REFERENCES fornecedores (id),
    origem_compra     text           CHECK (origem_compra IN ('obra', 'administracao')),
    nota_fiscal       text,
    valor_unitario    numeric(14, 2),
    destino           text,
    retirado_por      text,
    motivo            text           CHECK (motivo IN ('quebra', 'vencimento', 'intemperie', 'extravio', 'variacao_medida', 'outro')),
    descricao         text,
    estorno_de        bigint         REFERENCES movimentacoes (id),
    inventario_id     bigint         REFERENCES inventarios (id),
    CONSTRAINT movimentacoes_estorno_de_key UNIQUE (estorno_de)
);
CREATE INDEX movimentacoes_empreendimento_idx ON movimentacoes (empreendimento_id, id DESC);
CREATE INDEX movimentacoes_empreendimento_item_idx ON movimentacoes (empreendimento_id, item_id, id DESC);
CREATE INDEX movimentacoes_empreendimento_data_idx ON movimentacoes (empreendimento_id, data);
CREATE INDEX movimentacoes_item_idx ON movimentacoes (item_id);
CREATE INDEX movimentacoes_fornecedor_idx ON movimentacoes (fornecedor_id) WHERE fornecedor_id IS NOT NULL;

---- create above / drop below ----

DROP TABLE IF EXISTS movimentacoes;
DROP TABLE IF EXISTS inventario_itens;
DROP TABLE IF EXISTS inventarios;
DROP TABLE IF EXISTS saldos;
