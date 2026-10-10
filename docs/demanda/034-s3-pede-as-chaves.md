# 034 — S3 não pede as chaves: o que o assistente precisa pedir

- Estado: Aberta
- Risco: Médio · uso, segredos
- Onde: `internal/core/provedores.go:Provedores` (`s3`), `ProvedoresOcultos`; `internal/core/casos_de_uso.go:criarRemotoBase`, `CriarCofre`, `ConectarCofre`; `internal/gui/wizards.go:DialogoNovoCofre`, `DialogoImportarCofre`; `internal/core/criar_remoto.go` (`rclone config create`)
- Depende de: 005, 014

## Contexto

`Provedores` declara para `s3` os campos `access_key_id`, `secret_access_key`, `region` e `endpoint`, mas nenhum assistente mostra esses campos. `criarRemotoBase` só cria o remoto base para provedores com OAuth e para `local_path`. Para S3 ele não cria nada, e o crypt aponta para `<nome>_base:`, um remoto que não existe. O cofre aparece na lista e não destranca.

Até esta demanda ficar pronta, o PR "textos e assistente" esconde S3 dos dois assistentes (`core.ProvedoresOcultos`). Um cofre S3 que já está em `vaults.json` continua na lista e funciona como antes.

A 014 trata S3 e Pasta Local juntos. Esta demanda define o escopo do S3: o que pedir, como validar e o que mostrar quando dá errado. Se for aprovada, a parte S3 da 014 passa a ser esta demanda, e a 014 fica só com Pasta Local.

## O que muda

### Campos que o assistente pede (proposta de texto)

No passo do provedor, ao escolher S3, aparecem estes campos no lugar da autorização no navegador. Sem OAuth, não há passo `Autorizando no navegador…`.

| Campo | Rótulo (proposta) | Obrigatório | Chave do rclone |
|---|---|---|---|
| Access key | `Access Key ID` | sim | `access_key_id` |
| Secret | `Secret Access Key` (campo de senha) | sim | `secret_access_key` |
| Região | `Região (ex: us-east-1)` | não; vazio vira `us-east-1` | `region` |
| Endpoint | `Endpoint (deixe vazio para a AWS; ex: http://localhost:9000 no MinIO)` | não | `endpoint` |
| Bucket | `Bucket` | sim | vai no caminho: `<nome>_base:<bucket>` |

O remoto base é `type = s3` com `provider = AWS` quando o endpoint está vazio e `provider = Minio` ou `Other` quando há endpoint (a escolha fica no PR, com o motivo). O crypt aponta para `<nome>_base:<bucket>` ao criar e para `<nome>_base:<bucket>/<pasta>` ao conectar, com a pasta do seletor de pasta.

### Validação antes de gravar

Antes de criar o crypt, o app lista o bucket (`rclone lsjson --dirs-only <nome>_base:<bucket>`, com o tempo limite da 008). Se falhar, desfaz o remoto base (006) e mostra o erro no passo (proposta de texto):

| O rclone diz | Frase (proposta) |
|---|---|
| `InvalidAccessKeyId` | `Não deu para conectar ao S3: o Access Key ID não existe.` |
| `SignatureDoesNotMatch` | `Não deu para conectar ao S3: o Secret Access Key está errado.` |
| `NoSuchBucket` | `Não deu para conectar ao S3: o bucket {bucket} não existe.` |
| Endpoint que não responde | `Não deu para conectar ao S3: {endpoint} não respondeu.` |
| Outro | `Não deu para conectar ao S3: {última linha do rclone}` (como a 027) |

Ao criar um cofre com um bucket que não existe, o app não cria o bucket. **Proposta**: só criar se a UI aprovar um texto como `O bucket {bucket} não existe. Criar agora?`.

### Segredo fora dos argumentos

A `secret_access_key` não pode ir em argumento de `rclone config create`, que é onde o token OAuth vai hoje (005). Esta demanda usa o caminho que a 005 escolher. Por isso depende da 005.

### Mostrar de novo

Quando os testes do "Pronto quando" passarem, S3 sai de `core.ProvedoresOcultos`.

## O que fica de fora

- Pasta Local (014).
- Outros provedores compatíveis com S3 com campos próprios (Wasabi, Backblaze B2 nativo, R2): funcionam pelo endpoint, sem campo novo.
- Credenciais temporárias (`session_token`), perfis de `~/.aws` e IAM role.
- Testar contra a AWS de verdade. Basta MinIO local.
- Editar as chaves de um cofre que já existe (033).

## Pronto quando

- [ ] Com MinIO local (`docker run -p 9000:9000 minio/minio server /data`) e um bucket criado: criar um cofre S3 pelo assistente, destrancar, gravar um arquivo. O objeto com nome cifrado aparece no bucket (`mc ls`).
- [ ] Conectar esse mesmo cofre como existente, escolhendo a pasta pelo seletor: destranca e mostra o arquivo.
- [ ] Teste em `internal/core` com o rclone falso: com `SignatureDoesNotMatch` no stderr, o assistente para com `Não deu para conectar ao S3: o Secret Access Key está errado.`, e nem `<nome>_base` nem `<nome>` ficam no `rclone.conf`.
- [ ] Teste em `internal/core` com o executável falso da 005: o arquivo de argumentos não contém a `secret_access_key`.
- [ ] `go test ./internal/gui/ -run Assistentes` mostra S3 na lista dos dois assistentes.
