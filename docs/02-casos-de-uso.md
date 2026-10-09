# 02 — Casos de uso

Ator único: o usuário do computador. Atores externos: o binário `rclone`, o provedor de armazenamento (Google Drive, OneDrive, Dropbox, S3) e o driver de sistema de arquivos (WinFsp ou FUSE).

## Telas

| Tela | Código | Ações que existem hoje |
|---|---|---|
| Janela de cofres | `internal/gui/janela_principal.go:construirInterface`, `criarCardCofre` | Destrancar ou Trancar (um botão por cofre), "＋ Adicionar Cofre", "📥 Importar Cofre Existente", "⚙ Configurações", "ℹ Sobre", "✕ Sair". Fechar no X esconde a janela |
| Diálogo de senha | `internal/gui/dialogos.go:DialogoSenha` | Campo de senha, "Cancelar", "Desbloquear" |
| Wizard novo cofre | `internal/gui/wizards.go:DialogoNovoCofre` | Aba "1. Provedor": Google Drive, Microsoft OneDrive, Dropbox, Amazon S3 / MinIO. Aba "2. Senha": senha, confirmar senha, nome do cofre, "Cancelar", "Criar Cofre" |
| Wizard conectar existente | `internal/gui/wizards.go:DialogoImportarCofre` e `internal/gui/seletor_pasta.go:DialogoSeletorPastaRemota` | Aba "1. Provedor": os quatro acima mais Pasta Local. Aba "2. Senhas": password, password2, nome, "Cancelar", "Avançar →". Depois, o seletor de pasta: entrar em pasta, "⬆ Voltar", "Cancelar", "✓ Selecionar esta pasta" |
| Configurações VFS | `internal/gui/config_vfs.go:DialogoConfigVfs` | 13 campos de texto, "Restaurar Padrões", "Cancelar", "Salvar" |
| Ícone da bandeja | `internal/tray/tray.go:aoIniciar` | "Abrir RuntimeCrypto", "Novo Cofre...", "Configuracoes" > "Auto-iniciar com Windows", "Configuracoes VFS...", "Verificar WinFsp/FUSE"; "Sobre"; "Sair" |
| Mensagem | `internal/gui/dialogos.go:DialogoMensagem` | "OK" |

A bandeja não lista cofres. O tipo `AcaoCofre` existe e `main.go` sabe tratá-lo, mas nenhum item de menu o envia. O tooltip é fixo, porque `AtualizarTooltip` nunca é chamado.

## Estados do cofre

### Regra pedida

Cada cofre mostra um de quatro estados: **desmontado**, **montando**, **montado**, **falhou**. São nomes do `core`. A tela escreve `Trancado`, `Destrancando…`, `Destrancado • X:\` e `Não destrancou: {motivo}`.

**montado** só vale quando as duas condições são verdadeiras ao mesmo tempo:

1. o processo `rclone mount` daquele cofre está vivo; e
2. o ponto de montagem existe (a letra `X:\` no Windows, ou a pasta em Linux e macOS).

### O que o código faz hoje

| Estado | Existe hoje? | Como o código decide |
|---|---|---|
| desmontado | Sim, como "● Trancado" | Qualquer cofre fora do mapa `GerenciadorMontagem.montagens` |
| montando | Não | `destravarCofre` roda em goroutine por até 45 s sem nenhum sinal na tela. O botão "Destrancar" continua ativo e um segundo clique dispara outra montagem |
| montado | Sim, como "● Destrancado • X:\" | O cofre está no mapa **e** `processoAtivo` devolve verdadeiro. O ponto de montagem só é testado durante a espera inicial. Depois disso, ninguém confere |
| falhou | Não | A falha aparece em um diálogo de erro. Depois de fechado, o card volta a "● Trancado" |

Há um defeito que derruba a regra inteira: `processoAtivo` (`internal/core/montagem.go`) chama `p.Signal(os.Signal(nil))`. O Go recusa esse sinal com `os: unsupported signal type`, então a função sempre devolve falso. Isso foi reproduzido no Linux. No Windows, o `Signal` do Go só aceita `Kill`, então também devolve erro. Na primeira chamada a `Status()`, o cofre recém-montado sai do mapa. A tela passa a mostrar "Trancado" com o rclone ainda montado. Ver [demanda 001](demanda/001-rastreio-de-montagem.md).

Diagrama de estados alvo e diagrama do comportamento atual: [07-fluxogramas.md](07-fluxogramas.md#estados-do-cofre).

## Casos de uso

### UC-01 Criar cofre novo

- **Tela:** wizard novo cofre, aberto pela janela ou pela bandeja.
- **Pré-condição:** o rclone foi encontrado. Se não foi, `CriarRemoto` devolve "RClone nao disponivel." só no fim do fluxo.
- **Fluxo principal (OAuth):**
  1. O usuário escolhe o provedor, digita a senha duas vezes e o nome.
  2. `GerenciadorOAuth.Iniciar` roda `rclone authorize <tipo>`.
  3. Depois de 1 s, se a URL local foi lida da saída, o navegador abre.
  4. O programa consulta o token uma vez por segundo, por até 120 s.
  5. Cria o remoto base `<nome>_base` com o token.
  6. Cria o remoto `crypt` `<nome>`, apontado para `<nome>_base:`, com a mesma senha em `password` e `password2`.
  7. Grava o cofre em `vaults.json` e guarda a senha no cache da sessão.
- **Alternativas e falhas:**
  - Token não chega em 120 s: aborta e mostra "Autorização não concluída em 2 minutos." O remoto base não foi criado.
  - S3: não há passo de credenciais. Nenhum remoto base é criado e o crypt aponta para `<nome>_base:`, que não existe.
  - Nome já usado por outro cofre: o rclone **sobrescreve** `<nome>_base` e `<nome>` antes de `Cofres.Adicionar` recusar o nome. Reproduzido com `rclone config create` em nome existente: sai com código 0 e troca a senha. Ver [demanda 006](demanda/006-criacao-sobrescreve-remoto.md).
  - Falha no passo 6 ou 7: o remoto base fica órfão no `rclone.conf`.

### UC-02 Conectar cofre existente

- **Tela:** wizard conectar existente e seletor de pasta.
- **Fluxo principal (OAuth):** igual aos passos 2 a 5 do UC-01. Depois, o usuário navega com `rclone lsd` (30 s de limite por pasta), escolhe a pasta, e o programa cria o crypt `<nome>` em `<nome>_base:<pasta>` com `password` e `password2`. Se `password2` ficar vazio, usa `password`.
- **Alternativas e falhas:**
  - Cancelar no seletor: remove `<nome>_base` com `rclone config delete`.
  - Erro ou tempo esgotado no `lsd`: a tela mostra "Nenhuma subpasta encontrada. Você pode selecionar esta pasta." O erro some.
  - Pasta Local: mostra "Selecione a pasta no explorador." e encerra. Nada é criado.
  - Senha errada: não há verificação. O cofre é criado e a montagem mostra a pasta vazia ou nomes ilegíveis. Desconhecido como o rclone se comporta em cada caso.

### UC-03 Destrancar (montar)

- **Tela:** janela de cofres (botão "Destrancar"). A bandeja não tem esse item.
- **Fluxo principal:**
  1. Se não há senha no cache, abre o diálogo de senha.
  2. Guarda a senha no cache.
  3. Escolhe a primeira letra livre de `LetrasPreferidas` (V, W, X, ...).
  4. Roda `rclone mount <nome>: X: <flags VFS> --volname ... --no-checksum --no-modtime [--network-mode --no-console]` com `RCLONE_CONFIG_PASS=<senha>` no ambiente.
  5. Consulta `X:\` com intervalo crescente de 200 ms até 1 s, por até 45 s.
  6. Abre o Explorer em `X:\` e mostra "Cofre Destrancado".
- **Falhas:** sem letra livre, tempo esgotado (o processo é morto) ou erro ao iniciar o rclone. Se o rclone sai logo no início, o código só percebe no fim dos 45 s, porque `cmd.ProcessState` nunca é preenchido sem `Wait`.
- **Linux e macOS:** falha sempre no passo 3 com "Nenhuma letra de unidade disponivel."

### UC-04 Trancar (desmontar)

- **Tela:** janela de cofres (botão "Trancar", que só aparece com estado montado).
- **Fluxo principal:** manda `os.Interrupt` ao processo, espera 5 s, depois `Kill` mais duas vezes com 5 s cada, remove do mapa e limpa a senha.
- **Hoje:** por causa do defeito de `processoAtivo`, o botão "Trancar" quase nunca aparece. Quando aparece, a função informa sucesso mesmo que o processo não tenha terminado.

### UC-05 Auto-montar ao iniciar

- **Gatilho:** 1,5 s depois de abrir o programa (`main.go:main`).
- **Hoje:** não monta nada. Exige `TemSenha`, e o cache está vazio ao iniciar.

### UC-06 Ajustar VFS

- **Tela:** Configurações VFS, aberta pela janela ou pela bandeja.
- **Efeito:** muda os valores em memória para as próximas montagens. Não há validação. "Restaurar Padrões" aplica na hora, mesmo que depois o usuário clique em "Cancelar".

### UC-07 Verificar WinFsp/FUSE

- **Tela:** só a bandeja. `CallbackVerificarFuse` existe na janela, mas não há botão ligado a ele.

### UC-08 Auto-iniciar com o sistema

- **Tela:** bandeja, "Auto-iniciar com Windows". O texto é o mesmo em Linux e macOS. O item não mostra se a opção está ligada. Erros são descartados.

### UC-09 Sair

- **Telas:** botão "✕ Sair" na janela ou "Sair" na bandeja.
- **Efeito pretendido:** desmontar tudo, limpar senhas, abortar OAuth e fechar.
- **Hoje:** as montagens continuam, porque o mapa já está vazio (ver UC-04). `systray.Quit` não é chamado.
