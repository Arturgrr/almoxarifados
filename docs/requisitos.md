# Documento de Requisitos – Sistema Web de Gestão de Almoxarifado de Obras

- **Empresa parceira:** Grupo A2G
- **Disciplina:** CSI606 – Sistemas Web I · Prof. Fernando Bernardes de Oliveira
- **Etapa:** 3 – Levantamento de requisitos
- **Versão:** 1.0 · outubro de 2026

---

## 1. Introdução

### 1.1 Propósito

Este documento descreve os requisitos do sistema web de gestão de almoxarifado de obras desenvolvido para o Grupo A2G. Ele serve de referência para o planejamento, a implementação e a validação da aplicação.

### 1.2 Contexto

- O Grupo A2G é uma construtora e incorporadora com empreendimentos principalmente em Porto Seguro (BA)
  - Executa condomínios, casas e afins
  - Mantém mais de 5 obras em paralelo, cada uma com seu próprio almoxarifado
- Hoje o controle de materiais é feito em planilhas e depende do chefe de obra
  - Não há saldo confiável por obra
  - Não há registro de quem retirou cada material
  - As perdas não são medidas
- A forma de compra varia por obra
  - Em algumas, a própria equipe da obra orça e compra, após aprovação da administração
  - Em outras, a administração compra e entrega o material no canteiro

### 1.3 Escopo

- **Dentro do escopo**
  - Cadastro de usuários, empreendimentos, itens (materiais e equipamentos) e fornecedores
  - Controle de acesso por empreendimento
  - Movimentações de estoque: entrada, saída, perda, inventário, estorno e transferência
  - Consultas de saldo e histórico, visão consolidada e relatórios
- **Fora do escopo**
  - Emissão de nota fiscal
  - Financeiro (contas a pagar, pagamento de fornecedores)
  - Fluxo de cotação e aprovação de compras
  - Integração com ERP ou outros sistemas
  - Aplicativo nativo de celular (o acesso móvel é feito pelo navegador)

### 1.4 Glossário

- **Empreendimento:** obra do Grupo A2G que possui almoxarifado próprio
- **Almoxarifado:** local da obra onde materiais e equipamentos são recebidos, guardados e retirados
- **Item:** qualquer coisa controlada no estoque
  - **Material de consumo:** é gasto na obra (cimento, areia, tinta, fio)
  - **Equipamento/ferramenta:** é usado, mas não consumido (betoneira, furadeira, andaime)
- **Material a granel:** material sem contagem exata, cujo volume varia naturalmente (areia, brita)
- **Movimentação:** qualquer registro que altera o saldo (entrada, saída, perda, ajuste, estorno, transferência)
- **Saldo:** quantidade de um item disponível em um empreendimento, calculada a partir das movimentações
- **Inventário:** contagem física do estoque para comparar com o saldo registrado
- **Ajuste:** movimentação gerada pelo inventário para corrigir a diferença entre o contado e o registrado
- **Estorno:** movimentação que anula outra lançada por engano, sem apagá-la
- **Tolerância de variação:** percentual de diferença aceito como natural para materiais a granel

### 1.5 Convenções

- Requisitos funcionais são identificados por **RF**, não funcionais por **RNF** e regras de negócio por **RN**
- Itens marcados como *(desejável)* serão implementados se houver tempo; os demais são essenciais

---

## 2. Objetivos do sistema

- Centralizar o controle de materiais e equipamentos de todos os empreendimentos
  - Substituir planilhas e controles informais
  - Ser o mesmo sistema para o canteiro e para a administração
- Manter um saldo de estoque confiável e atualizado por empreendimento
  - O saldo é calculado a partir das movimentações, nunca digitado à mão
  - Disponível a qualquer momento, no celular ou no computador
- Garantir rastreabilidade
  - Toda movimentação registra o que foi movimentado, quanto, quando, por quem e para qual serviço
- Tornar as perdas mensuráveis
  - Registro de perdas com motivo
  - Inventário periódico que separa a variação natural de materiais a granel da perda anormal
- Reduzir compras erradas e duplicadas
  - Visão consolidada do estoque de todas as obras
  - Consulta de onde está cada equipamento antes de comprar outro

---

## 3. Perfis de usuário

- **Administrador**
  - Quem: equipe administrativa e gestão do Grupo A2G
  - Acessa todos os empreendimentos sem precisar ser associado
  - Gerencia usuários, empreendimentos, catálogo e configurações
- **Usuário de obra**
  - Quem: chefe de obra, encarregado ou responsável pelo almoxarifado
  - Acessa apenas os empreendimentos aos quais foi associado
  - Opera o estoque desses empreendimentos no dia a dia

Permissões por perfil:

| Ação | Administrador | Usuário de obra |
| --- | :---: | :---: |
| Gerenciar usuários | Sim | Não |
| Criar, editar e encerrar empreendimentos | Sim | Não |
| Associar usuários a empreendimentos | Sim | Não |
| Visualizar empreendimentos | Todos | Somente associados |
| Cadastrar itens e fornecedores | Sim | Sim |
| Editar e desativar itens e fornecedores | Sim | Não |
| Registrar entrada, saída e perda | Sim | Somente associados |
| Realizar inventário | Sim | Somente associados |
| Estornar movimentação | Qualquer uma | Somente as próprias |
| Definir tolerância de granel e estoque mínimo | Sim | Não |
| Transferir itens entre empreendimentos | Qualquer par | Entre associados |
| Visão consolidada de todas as obras | Sim | Não |
| Relatórios | Todos | Somente associados |

---

## 4. Requisitos funcionais

### 4.1 Autenticação e usuários

#### RF01 – Login

- O sistema deve permitir acesso apenas a funcionários cadastrados
  - Não existe tela de cadastro público
  - Contas são criadas exclusivamente pelo administrador (RF03)
- Tela de login
  - Campos: e-mail e senha
  - Validações
    - E-mail obrigatório e em formato válido
    - Senha obrigatória
  - Em caso de erro, exibir mensagem genérica: “E-mail ou senha inválidos”
    - Não informar qual dos dois está errado, por segurança
- Após o login
  - Administrador: abre o painel geral com todos os empreendimentos
  - Usuário de obra associado a um único empreendimento: abre direto o estoque dele
  - Usuário de obra associado a vários: abre a lista dos seus empreendimentos
- Usuário desativado não consegue entrar
  - Mensagem: “Usuário desativado. Procure o administrador.”

#### RF02 – Logout e sessão

- O sistema deve exibir a opção “Sair” em todas as telas
- A sessão expira após um período de inatividade (RNF02)
- Ao expirar, o usuário volta para a tela de login

#### RF03 – Cadastro de usuários

- O sistema deve permitir que o administrador cadastre usuários
- Campos
  - Nome completo (obrigatório)
  - E-mail (obrigatório)
    - Usado como login
    - Deve ser único no sistema
  - Telefone (opcional)
  - Perfil: Administrador ou Usuário de obra (obrigatório)
  - Senha inicial (obrigatória)
    - Mínimo de 8 caracteres
    - Informada pelo administrador ao funcionário
- Após o cadastro, o administrador pode associar o usuário a empreendimentos (RF07)
- No primeiro acesso, o sistema solicita a troca da senha inicial *(desejável)*

#### RF04 – Gerenciamento de usuários

- O sistema deve listar os usuários cadastrados
  - Colunas: nome, e-mail, perfil, situação (ativo/inativo) e empreendimentos associados
  - Busca por nome ou e-mail
  - Filtros por perfil e situação
- O sistema deve permitir editar um usuário
  - Nome, telefone e perfil
  - E-mail, desde que continue único
- O sistema deve permitir desativar e reativar um usuário
  - Não há exclusão de usuários (RN13)
  - O administrador não pode desativar a si mesmo
    - Garante que sempre exista ao menos um administrador ativo (RN15)

#### RF05 – Alteração e redefinição de senha

- O sistema deve permitir que o usuário altere a própria senha
  - Campos: senha atual, nova senha e confirmação
  - Nova senha com no mínimo 8 caracteres
  - Nova senha e confirmação devem ser iguais
- O sistema deve permitir que o administrador redefina a senha de qualquer usuário
  - O administrador define uma nova senha provisória

### 4.2 Empreendimentos

#### RF06 – Cadastro de empreendimentos

- O sistema deve permitir que o administrador crie diferentes empreendimentos
- Campos
  - Nome (obrigatório)
    - Deve ser único no sistema
  - Tipo: condomínio, casas, edifício ou outro
  - Endereço e cidade
  - Data de início (obrigatória)
  - Previsão de término (opcional)
  - Responsável: chefe de obra, escolhido entre os usuários
  - Status: em andamento (padrão) ou concluído
- Ao criar o empreendimento, o sistema cria automaticamente seu estoque interno, vazio
  - Cada empreendimento tem estoque próprio e independente (RN04)
- O administrador pode editar os dados do empreendimento a qualquer momento

#### RF07 – Associação de usuários ao empreendimento

- O sistema deve permitir selecionar quais usuários são associados a cada empreendimento
- Na tela do empreendimento, aba “Equipe”
  - Lista os usuários associados
  - Adicionar usuário
    - Seleção a partir da lista de usuários ativos com perfil Usuário de obra
    - Um usuário pode estar associado a vários empreendimentos
  - Remover usuário
    - O usuário perde o acesso imediatamente
    - As movimentações feitas por ele naquele empreendimento continuam no histórico
- Administradores têm acesso a todos os empreendimentos e não aparecem nessa lista

#### RF08 – Encerramento de empreendimento

- O sistema deve permitir que o administrador encerre um empreendimento
  - Ação: alterar o status para “concluído”
  - Antes de confirmar, o sistema exibe o saldo restante
    - Sugere transferir as sobras para outro empreendimento (RF20)
  - Pede confirmação explícita
- Empreendimento concluído
  - Fica disponível apenas para consulta
  - Não aceita novas movimentações (RN12)
  - Pode ser reaberto pelo administrador

#### RF09 – Listagem de empreendimentos

- O sistema deve listar os empreendimentos que o usuário pode acessar
  - Administrador: todos
  - Usuário de obra: somente os associados
- Informações exibidas para cada empreendimento
  - Nome, cidade e status
  - Quantidade de itens em estoque
  - Data da última movimentação
  - Quantidade de itens abaixo do estoque mínimo *(desejável)*
- Filtro por status (em andamento ou concluído)

### 4.3 Catálogo de itens

#### RF10 – Cadastro de itens

- O sistema deve manter um catálogo único de itens, compartilhado entre todos os empreendimentos
  - Evita que o mesmo material seja cadastrado com nomes diferentes em cada obra
  - O saldo, porém, é separado por empreendimento
- Quem pode cadastrar
  - Administrador e usuário de obra podem cadastrar novos itens
  - Somente o administrador edita e desativa itens
- Campos
  - Nome (obrigatório)
    - Ex.: “Areia média”, “Cimento CP-II”, “Betoneira 400 L”
    - Deve ser único no catálogo
    - O sistema avisa se já existe item com nome parecido *(desejável)*
  - Código interno, gerado automaticamente
  - Tipo (obrigatório)
    - Material de consumo
    - Equipamento/ferramenta
  - Categoria (obrigatória)
    - Ex.: agregados, aglomerantes, elétrica, hidráulica, acabamento, ferramentas, EPI, outros
    - Lista de categorias gerenciada pelo administrador
  - Unidade de medida de controle (obrigatória)
    - kg, litro (L), metro (m), metro quadrado (m²), metro cúbico (m³) ou unidade (un)
    - Não pode ser alterada depois da primeira movimentação (RN09)
  - Material a granel (sim ou não)
    - Se sim, o administrador define a tolerância de variação em % (RN10)
  - Descrição ou observações (opcional)

#### RF11 – Unidade de compra e conversão *(desejável)*

- O sistema deve permitir cadastrar unidades de compra alternativas para um item, com fator de conversão
  - Ex.: cimento controlado em kg e comprado em “saco de 50 kg” (fator 50)
  - Ex.: fio controlado em metros e comprado em “rolo de 100 m” (fator 100)
  - Ex.: areia controlada em m³ e comprada por “caminhão de 6 m³” (fator 6)
- Na entrada, o usuário escolhe a unidade de compra e o sistema converte
  - Ex.: entrada de 10 sacos de cimento lança 500 kg no estoque

#### RF12 – Listagem do catálogo

- O sistema deve listar os itens do catálogo
- Busca por nome ou código
- Filtros por tipo, categoria e situação (ativo/inativo)

### 4.4 Fornecedores

#### RF13 – Cadastro de fornecedores

- O sistema deve permitir o cadastro de fornecedores
  - Administrador e usuário de obra podem cadastrar
  - Somente o administrador edita e desativa
- Campos
  - Nome ou razão social (obrigatório)
  - Nome fantasia (opcional)
  - CNPJ ou CPF (opcional)
    - Se preenchido, deve ser válido e único
  - Telefone/WhatsApp e e-mail
  - Cidade
  - Observações
- Listagem de fornecedores
  - Busca por nome, nome fantasia ou CNPJ/CPF
  - Filtro por situação (ativo/inativo)

#### RF14 – Histórico do fornecedor *(desejável)*

- O sistema deve exibir as entradas feitas com cada fornecedor, em todos os empreendimentos
- Filtros por período e empreendimento

### 4.5 Movimentações de estoque

#### RF15 – Registro de entrada

- O sistema deve permitir registrar entradas de itens no estoque de um empreendimento
- Tipo de entrada (obrigatório)
  - **Compra**
    - Fornecedor (obrigatório)
    - Origem da compra (obrigatória – RN14)
      - Compra feita pela equipe da obra
      - Compra feita pela administração e entregue no canteiro
    - Número da nota fiscal (opcional)
    - Valor unitário ou total (opcional)
      - Permite estimar o valor das perdas nos relatórios
  - **Devolução**
    - Sobra de material que volta de um serviço
    - Retorno de equipamento que estava em uso
    - Informar de qual serviço o item voltou
- Itens da entrada
  - Uma entrada pode ter vários itens, como em uma nota fiscal
  - Para cada item
    - Item do catálogo, com busca por nome
    - Quantidade, maior que zero e com as casas decimais permitidas pela unidade (RN09)
    - Unidade de compra, quando houver conversão (RF11)
- Data da entrada
  - Padrão: data atual
  - Pode ser retroativa, mas não futura (RN16)
- Observações (opcional)
- Foto da nota ou do canhoto de entrega *(desejável)*
- Ao salvar, o saldo é atualizado imediatamente

#### RF16 – Registro de saída

- O sistema deve permitir registrar saídas de itens do estoque de um empreendimento
- Campos
  - Item e quantidade (obrigatórios)
  - Destino ou serviço (obrigatório)
    - Ex.: “Fundação”, “Alvenaria – Bloco B”, “Reboco – Casa 3”
    - O sistema sugere destinos já usados naquele empreendimento
  - Quem retirou (obrigatório)
    - Pode ser diferente do usuário que registra
    - Ex.: pedreiro sem acesso ao sistema retira e o almoxarife registra
  - Data (padrão: data atual)
  - Observações (opcional)
- Validações
  - Ao selecionar o item, o sistema mostra o saldo disponível
  - A quantidade não pode ser maior que o saldo (RN06)
- Saída de equipamento
  - Indica que o equipamento foi retirado para uso
  - O retorno é registrado como entrada do tipo devolução

#### RF17 – Registro de perda

- O sistema deve permitir registrar perdas de itens
- Campos
  - Item e quantidade (obrigatórios)
  - Motivo (obrigatório – RN11)
    - Quebra ou avaria
    - Vencimento ou deterioração (ex.: cimento empedrado)
    - Intempérie (chuva, vento)
    - Extravio ou furto
    - Variação de medida (material a granel)
    - Outro, com descrição obrigatória
  - Descrição (opcional, exceto para “Outro”)
  - Data (padrão: data atual)
  - Foto *(desejável)*
- A quantidade não pode ser maior que o saldo (RN06)

#### RF18 – Inventário (contagem física)

- O sistema deve permitir realizar inventário do estoque de um empreendimento
- Abertura
  - Escopo: todos os itens ou apenas uma categoria
  - O sistema gera a lista dos itens com saldo no empreendimento
  - Enquanto o inventário estiver aberto, os itens incluídos ficam bloqueados para novas movimentações *(desejável)*
- Contagem
  - O usuário informa a quantidade contada de cada item
  - É possível salvar a contagem e continuar depois
  - Contagem cega *(desejável)*
    - O saldo registrado fica oculto durante a contagem
    - Evita que quem conta apenas copie o número do sistema
- Finalização
  - O sistema calcula a diferença de cada item
    - Diferença = quantidade contada − saldo registrado
    - Exibida em quantidade e em percentual
  - Classificação da diferença (RN10)
    - Diferença zero: nenhum ajuste
    - Material a granel dentro da tolerância: ajuste registrado como variação natural
    - Fora da tolerância, ou item que não é a granel: ajuste registrado como perda ou sobra, com justificativa obrigatória
  - Ao confirmar, o sistema gera as movimentações de ajuste
- Histórico de inventários
  - Data, responsável, escopo e resumo das diferenças encontradas

#### RF19 – Estorno de movimentação

- O sistema deve permitir cancelar uma movimentação lançada por engano, por meio de estorno
- Quem pode estornar
  - Usuário de obra: somente as próprias movimentações
  - Administrador: qualquer movimentação
- Justificativa obrigatória
- Efeito
  - Cria uma movimentação inversa, vinculada à original
  - A original continua no histórico, marcada como “estornada”
- Restrições
  - Uma movimentação já estornada não pode ser estornada de novo
  - Não é possível estornar uma entrada se o saldo ficar negativo
    - Ex.: entrada de 10 sacos que já foram consumidos; as saídas precisam ser corrigidas antes
- Para corrigir um lançamento, o usuário estorna e lança novamente; não existe edição de movimentação (RN08)

#### RF20 – Transferência entre empreendimentos *(desejável)*

- O sistema deve permitir transferir itens de um empreendimento para outro
- Campos
  - Empreendimento de origem (o atual) e de destino (ativo)
  - Itens e quantidades
  - Data e observações
- Permissões
  - Administrador: entre quaisquer empreendimentos
  - Usuário de obra: somente entre empreendimentos aos quais está associado
- Efeito
  - Gera uma saída na origem e uma entrada no destino, vinculadas
  - A quantidade não pode ser maior que o saldo da origem (RN06)

### 4.6 Consultas e relatórios

#### RF21 – Saldo do empreendimento

- O sistema deve exibir o saldo atual de cada item por empreendimento (tela principal da obra)
- Lista de itens
  - Colunas: item, categoria, unidade, saldo e data da última movimentação
  - Busca por nome e filtros por categoria e tipo
  - Itens com saldo zero ficam ocultos por padrão, com opção para exibi-los
- Estoque mínimo *(desejável)*
  - O administrador define uma quantidade mínima por item em cada empreendimento
  - Itens abaixo do mínimo aparecem destacados, com ícone e texto, e em um aviso no topo da tela
- Ficha do item
  - Ao clicar em um item, o sistema exibe todas as suas movimentações naquele empreendimento
  - Para cada movimentação, mostra o saldo resultante
- Atalhos na tela
  - Botões para registrar entrada, saída e perda
  - No celular, esses botões ficam sempre visíveis

#### RF22 – Histórico de movimentações

- O sistema deve exibir o histórico de movimentações de um empreendimento
- Filtros
  - Período
  - Tipo: entrada, saída, perda, ajuste, estorno e transferência
  - Item
  - Usuário que registrou
  - Fornecedor
- Colunas: data e hora, tipo, item, quantidade, unidade, usuário e detalhe (fornecedor, destino ou motivo)
- Movimentações estornadas aparecem marcadas
- Lista paginada

#### RF23 – Visão geral do administrador

- O sistema deve oferecer ao administrador um painel com todos os empreendimentos ativos
  - Para cada empreendimento
    - Quantidade de itens em estoque
    - Itens abaixo do estoque mínimo *(desejável)*
    - Perdas registradas no mês
- Consulta consolidada de item
  - O administrador busca um item e vê o saldo em cada empreendimento
  - Ex.: “Betoneira 400 L” → Obra A: 1, Obra B: 0, Obra C: 2
  - Objetivo: evitar a compra de um equipamento ou material que já existe em outra obra
- Perdas anormais
  - Lista de perdas e ajustes de inventário fora da tolerância
  - Filtros por empreendimento e período

#### RF24 – Relatórios *(desejável)*

- Consumo por empreendimento e período
  - Total de saídas por item
  - Agrupamento por destino ou serviço
- Perdas
  - Por item, motivo e empreendimento
  - Percentual de perda: perdas ÷ entradas no período
  - Variação natural de granel separada de perda anormal
  - Valor estimado, quando houver valor nas entradas
- Entradas por fornecedor e por origem da compra (obra ou administração)
- Exportação em CSV e PDF

---

## 5. Requisitos não funcionais

#### RNF01 – Responsividade

- A interface deve funcionar em celular, tablet e computador
  - Largura mínima suportada: 360 px
- No celular, priorizar as telas usadas no canteiro
  - Botões grandes, fáceis de tocar
  - Cartões em vez de tabelas largas
  - Formulários de entrada, saída e perda em uma coluna
- Navegadores suportados: versões atuais de Chrome, Edge, Firefox e Safari (inclusive no celular)

#### RNF02 – Segurança

- Senhas
  - Armazenadas com hash (bcrypt ou argon2), nunca em texto puro
  - Mínimo de 8 caracteres
- Toda comunicação via HTTPS
- Autenticação
  - Token (JWT) ou sessão em cookie HttpOnly
  - Expiração após 8 horas de inatividade
- Bloqueio temporário de 15 minutos após 5 tentativas de login incorretas seguidas
- Proteção contra ataques comuns
  - SQL injection: consultas parametrizadas
  - XSS: tratamento dos dados exibidos na tela
  - CSRF: proteção quando a sessão usar cookie

#### RNF03 – Autorização no backend

- Toda requisição à API deve verificar
  - Se o usuário está autenticado e ativo
  - Se o perfil permite a ação
  - Se o usuário está associado ao empreendimento acessado
- Esconder botões na interface não substitui essa verificação
- Requisições sem permissão retornam erro 403

#### RNF04 – Usabilidade

- Registrar uma entrada ou saída deve levar menos de 1 minuto
- Busca de itens com autocompletar
- Mensagens de erro claras, em português, exibidas junto ao campo
- Confirmação antes de ações irreversíveis (estorno, encerramento, desativação)
- Confirmação visual após salvar

#### RNF05 – Desempenho

- Telas principais devem carregar em até 2 segundos em conexão 4G
- Listas paginadas (ex.: 50 registros por página)
- Índices no banco para as consultas por empreendimento, item e data

#### RNF06 – Integridade dos dados

- Cada movimentação deve ser gravada em transação
  - Movimentação e saldo são atualizados juntos, ou nenhum dos dois
- Controle de concorrência
  - Duas saídas simultâneas do mesmo item não podem deixar o saldo negativo
  - Uso de bloqueio de linha no banco ou verificação atômica do saldo
- Quantidades armazenadas em tipo decimal exato (NUMERIC)
  - Nunca em ponto flutuante, para evitar erros de arredondamento

#### RNF07 – Auditoria

- Registrar usuário, data e hora de toda criação e alteração de cadastros
- Registrar tentativas de login, com sucesso e com falha

#### RNF08 – Acessibilidade

- Contraste mínimo conforme WCAG 2.1 nível AA
- Todos os campos de formulário com rótulo
- Navegação completa por teclado
- Não depender apenas de cor para indicar estado
  - Ex.: alerta de estoque mínimo com ícone e texto, além da cor

#### RNF09 – Disponibilidade e backup

- Sistema hospedado em VPS, acessível pela internet
- Backup diário automático do banco de dados
  - Retenção mínima de 7 dias
  - Restauração testada ao menos uma vez antes da entrega
- Processo de implantação documentado no repositório

#### RNF10 – Tecnologias e manutenibilidade

- Backend em Go com o framework Fiber, expondo uma API REST em JSON
- Banco de dados PostgreSQL
  - Estrutura do banco criada por migrações versionadas
- Frontend web responsivo (framework a definir)
- Código-fonte versionado em repositório Git
  - README com instruções para rodar o projeto
- Documentação da API em OpenAPI *(desejável)*

#### RNF11 – Localização

- Interface em português do Brasil
- Datas no formato dd/mm/aaaa e horas no formato 24 h
- Fuso horário de Brasília (UTC−3)
- Números com vírgula decimal e ponto como separador de milhar
- Valores monetários em reais (R$)

---

## 6. Regras de negócio

#### RN01 – Acesso restrito

- Somente usuários cadastrados pelo administrador acessam o sistema
- Não existe autocadastro

#### RN02 – Permissões por perfil

- As ações permitidas a cada perfil seguem a tabela da seção 3
- Somente o administrador cria usuários e empreendimentos e associa usuários a empreendimentos

#### RN03 – Visibilidade por empreendimento

- O usuário de obra só visualiza e movimenta o estoque dos empreendimentos aos quais está associado
- O administrador visualiza todos
- Ao ser removido de um empreendimento, o usuário perde o acesso imediatamente

#### RN04 – Estoque independente por empreendimento

- Cada movimentação pertence a exatamente um empreendimento
- O mesmo item do catálogo tem saldos diferentes em cada empreendimento

#### RN05 – Cálculo do saldo

- O saldo de um item em um empreendimento é calculado pelas movimentações
  - Somam: entradas por compra, devoluções, ajustes positivos e transferências recebidas
  - Subtraem: saídas, perdas, ajustes negativos e transferências enviadas
- Um estorno anula o efeito da movimentação original
- O saldo nunca é digitado diretamente

#### RN06 – Saldo não negativo

- Saída, perda e transferência não podem ser maiores que o saldo disponível
- Estorno de entrada também não pode deixar o saldo negativo

#### RN07 – Registro automático de autoria

- Toda movimentação registra automaticamente o usuário e a data e hora do lançamento
- A data do lançamento (gerada pelo servidor) é guardada separada da data informada pelo usuário

#### RN08 – Movimentações não são editadas nem excluídas

- Uma movimentação registrada não pode ser alterada nem apagada
- A correção é feita por estorno seguido de novo lançamento
- O estorno exige justificativa

#### RN09 – Unidade de medida e casas decimais

- Cada item tem uma única unidade de controle
- A unidade não pode ser alterada após a primeira movimentação do item
- Casas decimais permitidas
  - Unidade (un): somente números inteiros
  - kg, L, m, m² e m³: até 3 casas decimais

#### RN10 – Materiais a granel e tolerância de variação

- O administrador define uma tolerância de variação, em %, para cada material a granel
  - Ex.: areia e brita com 10%
- No inventário, a diferença percentual é calculada sobre o saldo registrado
  - Diferença % = (contado − registrado) ÷ registrado × 100
- Dentro da tolerância
  - Registrada como variação natural, sem justificativa obrigatória
  - Ex.: saldo de 20 m³ de areia, contados 18,5 m³ → −1,5 m³ (−7,5%) → variação natural
- Fora da tolerância
  - Registrada como perda (ou sobra) anormal, com justificativa obrigatória
  - Aparece na lista de perdas anormais do administrador (RF23)
  - Ex.: saldo de 20 m³, contados 16 m³ → −4 m³ (−20%) → perda anormal
- Para itens que não são a granel, qualquer diferença exige justificativa

#### RN11 – Motivo obrigatório

- Registros de perda e ajustes de inventário que exigem justificativa precisam de motivo
- O motivo “Outro” exige descrição

#### RN12 – Empreendimento concluído

- Não aceita novas movimentações
- Continua disponível para consulta
- Só o administrador pode reabri-lo

#### RN13 – Desativação em vez de exclusão

- Usuários, itens e fornecedores com qualquer movimentação vinculada não podem ser excluídos, apenas desativados
- Itens e fornecedores desativados
  - Não aparecem para novas movimentações
  - Continuam aparecendo no histórico
- Cadastros sem nenhuma movimentação vinculada podem ser excluídos pelo administrador

#### RN14 – Origem da compra

- Toda entrada do tipo compra deve indicar a origem
  - Compra feita pela equipe da obra
  - Compra feita pela administração e entregue no canteiro

#### RN15 – Administrador ativo

- O sistema deve ter sempre pelo menos um administrador ativo
- O último administrador ativo não pode ser desativado nem ter o perfil alterado

#### RN16 – Datas das movimentações

- A data informada não pode ser futura
- Datas retroativas são permitidas; o sistema guarda também a data real do lançamento (RN07)
- A data não pode ser anterior à data de início do empreendimento
