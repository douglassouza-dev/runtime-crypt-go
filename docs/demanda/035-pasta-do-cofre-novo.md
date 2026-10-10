# 035 — Criar cofre sempre grava na raiz da conta (`Pasta: /`)

- Estado: Aberta
- Risco: Médio · dados, uso
- Onde: `internal/core/casos_de_uso.go:CriarCofre` (`remotoBase = nomeBase + ":"`), `ConectarCofre`, `Interacao.EscolherPasta`; `internal/gui/seletor_pasta.go:DialogoSeletorPastaRemota`, `listaPastas`; `internal/gui/acoes.go:NovoCofre`, `EscolherPasta`, `TextoCofrePronto`; `internal/core/gerenciador.go:ListarDiretoriosRemoto`
- Depende de: 006, 021, 023

## Contexto

`CriarCofre` aponta o crypt para `<nome>_base:`, a raiz da conta. O diálogo de sucesso mostra `Pasta: /`. Os arquivos cifrados (nomes ilegíveis) ficam misturados com os arquivos do usuário na raiz do Google Drive, OneDrive ou Dropbox. Dois cofres criados na mesma conta usam a mesma pasta. Por isso a 033 precisa bloquear `Deste computador e do {provedor}` quando os cofres têm arquivos em comum.

O seletor de pasta já existe e só aparece em "Importar Cofre Existente" (`ConectarCofre` chama `ui.EscolherPasta`). Ele lista as pastas com `rclone lsjson --dirs-only` (021), atualiza a tela na thread certa (023) e tem `Selecionar esta pasta`. Ele não cria pasta.

## O que muda

### Passo de pasta no "Adicionar Cofre"

Depois de autorizar no navegador, `CriarCofre` chama `ui.EscolherPasta`, como `ConectarCofre` já faz, e o crypt aponta para `<nome>_base:<pasta>`. Cancelar o seletor desfaz o remoto base (006) e para sem erro, como no importar.

Ordem nova do assistente: provedor → senha, confirmar e nome → Criar Cofre → autorizar no navegador → **escolher ou criar a pasta** → sucesso com `Pasta: /<pasta>`.

### Criar pasta no seletor (proposta de texto)

- Botão novo `Nova pasta…` ao lado de `⬆ Voltar`, só no "Adicionar Cofre". No importar, o cofre já existe e criar pasta não faz sentido.
- Pede o nome: título `Nova pasta`, campo com a dica `Nome da pasta`, botões `Criar` e `Cancelar`.
- Cria com `rclone mkdir <nome>_base:<caminho atual>/<nome>`, com o tempo limite da 008, e entra na pasta criada.
- Falha: `Não deu para criar a pasta: {motivo}` (motivo como na 027).
- Nome vazio ou com `/`: `Use um nome sem barra.`

### Sugestão de pasta (proposta)

O seletor do "Adicionar Cofre" abre na raiz. A dica no topo diz `Escolha ou crie uma pasta para o cofre. Ela deve estar vazia.` **Proposta alternativa** para a UI: sugerir direto uma pasta `RuntimeCrypto/<nome do cofre>`, já preenchida, com `Nova pasta…` e o seletor para trocar.

### Pasta que já tem arquivos (proposta)

Se a pasta escolhida tiver arquivos, o app pergunta antes de criar: `A pasta {pasta} já tem arquivos. O cofre vai guardar os dele junto. Usar mesmo assim?` com `Usar esta pasta` e `Escolher outra`. Se a pasta já for um cofre (os nomes não se leem), a resposta certa é "Importar Cofre Existente". Detectar isso fica de fora.

### Cofres que já existem

Nenhum impacto. `remoto_base` em `vaults.json` e o crypt no `rclone.conf` não mudam, e um cofre criado na raiz continua na raiz (`Pasta: /`). Mover um cofre para outra pasta fica de fora.

## O que fica de fora

- Mover os arquivos de um cofre que já existe da raiz para uma pasta.
- Detectar que a pasta escolhida já é um cofre e oferecer importar.
- S3 (034), em que o bucket faz o papel da pasta, e Pasta Local (014).
- Renomear ou apagar pasta no seletor.

## Pronto quando

- [ ] Teste em `internal/core` com o rclone falso: `CriarCofre` com `EscolherPasta` devolvendo `Docs/cofre` grava `remoto_base` como `<nome>_base:Docs/cofre`, e o crypt no `rclone.conf` aponta para o mesmo caminho.
- [ ] Teste em `internal/core`: cancelar a escolha da pasta no `CriarCofre` devolve `ErrCancelado`, e nem `<nome>_base` nem `<nome>` ficam no `rclone.conf`.
- [ ] Teste em `internal/gui` com o driver de teste do Fyne: `Nova pasta…` com `cofre` chama `rclone mkdir <base>:<atual>/cofre` e a lista entra em `cofre`. Com falha, mostra `Não deu para criar a pasta: {motivo}`.
- [ ] Teste: um `vaults.json` com um cofre `remoto_base: "x_base:"` continua abrindo, listando e destrancando com o rclone falso.
- [ ] Na tela, com uma conta real: criar um cofre numa pasta nova `RuntimeCrypto/teste`. O diálogo de sucesso mostra `Pasta: /RuntimeCrypto/teste`, e os arquivos cifrados aparecem nessa pasta no site do provedor, não na raiz.
