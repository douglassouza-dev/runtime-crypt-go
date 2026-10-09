# Análise de melhorias

Base: leitura completa do commit `c301180` (todos os arquivos `.go`, README, CHANGELOG, GEMINI.md e go.mod), `go build`, `go vet` e `go test` em Linux, e reprodução de quatro comportamentos: `Signal(nil)` no Go, `RCLONE_CONFIG_PASS` errada, `config create` em nome existente e `rclone reveal`.

O objetivo de Douglas é integração nativa com o armazenamento online, configurável e altamente estável. A análise mede o código contra esse objetivo. Cada item aponta para a demanda que o resolve.

## Resumo

O núcleo tem uma divisão boa: `internal/core` não importa GUI, os subsistemas têm mutex e os nomes seguem a linguagem do domínio (cofre, trancar, destrancar). O que impede o "altamente estável" está em três pontos:

1. **O programa não sabe o que está montado.** Um defeito de uma linha (`processoAtivo`) faz o programa esquecer cada montagem segundos depois de criá-la. Trancar e sair não desmontam nada.
2. **A senha não protege.** A senha do cofre fica reversível no `rclone.conf`, e a senha digitada ao destrancar não é conferida.
3. **Falhas não aparecem.** O stderr do rclone é descartado, a maioria das chamadas não tem tempo limite e erros viram lista vazia.

## Prioridade 1 — perda de controle e de segredos

| # | Achado | Onde | Demanda |
|---|---|---|---|
| 1 | `processoAtivo` sempre devolve falso, porque `Signal(os.Signal(nil))` dá erro. `Status()` apaga todas as montagens. A tela mostra "Trancado" com a unidade montada, trancar e sair não desmontam e um novo clique monta o mesmo cofre de novo | `core/montagem.go:processoAtivo`, `Status` | [001](demanda/001-rastreio-de-montagem.md) |
| 2 | A senha do crypt fica no `rclone.conf` ofuscada (reversível). `RCLONE_CONFIG_PASS` não tem efeito com o arquivo em claro, então qualquer senha destranca | `core/gerenciador.go:CriarCrypt`; `core/montagem.go:MontarUnidade` | [004](demanda/004-senha-do-cofre-nao-protege.md), [ADR-0006](adr/0006-onde-ficam-os-segredos.md) |
| 3 | Criar um cofre com nome repetido sobrescreve os remotos antes de a checagem de nome falhar. Reproduzido: o crypt antigo perde a senha | `main.go:acaoNovoCofre`, `acaoImportarCofre`; `core/cofres.go:Adicionar` | [006](demanda/006-criacao-sobrescreve-remoto.md) |
| 4 | `DesmontarUnidade` informa sucesso mesmo se o processo não terminou. No Windows, gasta 5 s com um `Interrupt` que o Go não envia e termina com `Kill`, o que pode cortar o write-back | `core/montagem.go:DesmontarUnidade` | [002](demanda/002-desmontagem-verificavel.md) |
| 5 | `vaults.json` com JSON inválido é tratado como vazio e sobrescrito na próxima gravação. A escrita não é atômica | `core/cofres.go:carregar`, `salvar` | [007](demanda/007-persistencia-vaults-json.md) |

## Prioridade 2 — falhas silenciosas

| # | Achado | Onde | Demanda |
|---|---|---|---|
| 6 | Se o rclone sai na hora, a montagem espera 45 s, porque `ProcessState` só existe depois de `Wait`. O stderr é descartado e o motivo se perde | `core/montagem.go:MontarUnidade` | [003](demanda/003-falha-de-montagem-visivel.md) |
| 7 | `config create`, `config dump`, `listremotes`, `obscure`, `--version` e `config delete` não têm tempo limite. `--version` roda antes de a janela abrir | `core/gerenciador.go` | [008](demanda/008-tempo-limite-chamadas-externas.md) |
| 8 | Erro de `lsd` aparece como "Nenhuma subpasta encontrada". Erros de listagem, auto-montar, auto-iniciar e de abrir o Explorer são descartados | `core/gerenciador.go:ListarDiretoriosRemoto`; `main.go` | [009](demanda/009-erros-engolidos.md) |
| 9 | Depois de montar, ninguém confere se a letra ainda existe | `core/montagem.go:Status` | [010](demanda/010-saude-da-montagem.md) |
| 10 | Token OAuth e senha ofuscada vão como argumentos de `rclone config create` e ficam visíveis na lista de processos | `core/gerenciador.go:CriarRemoto` | [005](demanda/005-segredos-em-argumentos.md) |

## Prioridade 3 — configurável e multiplataforma

| # | Achado | Onde | Demanda |
|---|---|---|---|
| 11 | Os 13 parâmetros VFS aceitam qualquer texto. Valor inválido só aparece como espera de 45 s | `core/vfs.go:Atualizar` | [011](demanda/011-validacao-vfs.md) |
| 12 | A configuração VFS não é gravada | `core/vfs.go:NovoConfigVfs` | [012](demanda/012-persistencia-vfs.md) |
| 13 | Linux e macOS não montam: não há letra, e não há pasta como alternativa | `core/montagem.go:ObterLetrasDisponiveis` | [013](demanda/013-montagem-linux-macos.md) |
| 14 | S3 não pede credenciais. Pasta Local para em `TODO` | `main.go`; `gui/wizards.go` | [014](demanda/014-provedores-s3-e-local.md) |
| 15 | Auto-montar exige senha no cache, que está vazio ao iniciar. Nenhuma tela liga a opção | `main.go:autoMontarCofres` | [015](demanda/015-auto-montar.md) |

## Prioridade 4 — base para evoluir

| # | Achado | Onde | Demanda |
|---|---|---|---|
| 16 | Só há 2 testes. Toda chamada ao rclone é `exec.Command` direto, sem ponto de troca para teste | `internal/core` | [016](demanda/016-testes-do-core.md) |
| 17 | A orquestração dos casos de uso está em `main.go`. Validações de senha e nome estão em `internal/gui`. O laço de OAuth está duplicado | `main.go`; `gui/wizards.go`; `gui/seletor_pasta.go` | [017](demanda/017-extrair-logica-para-core.md) |
| 18 | O estado do cofre é `bool`. Não há "montando" nem "falhou". A bandeja não mostra estado | `core/cofres.go:CofreStatus`; `gui/janela_principal.go`; `tray/tray.go` | [018](demanda/018-estados-do-cofre.md) |
| 19 | Migração para Wails, decidida por Douglas. A versão (v3 beta com bandeja ou v2 estável sem bandeja) está pendente | `internal/gui`, `internal/tray` | [019](demanda/019-migracao-wails.md), [ADR-0007](adr/0007-interface-wails.md) |

## Achados sem demanda própria

Estes itens ficam registrados aqui. Nenhum justifica uma demanda sozinho, ou eles somem com outra demanda.

| Achado | Onde | Destino |
|---|---|---|
| Widgets Fyne são alterados fora da goroutine principal, sem `fyne.Do` (exigido desde o Fyne 2.6) | `gui/janela_principal.go:agendarAtualizacao`, `gui/seletor_pasta.go`, `main.go` | Some com a 019. Se a 019 for adiada, vira demanda |
| `systray.Run` roda fora da goroutine principal, junto com o laço do Fyne. O `getlantern/systray` pede a thread principal no macOS | `main.go:main`, `tray/tray.go:Iniciar` | Some com a 019 |
| `--version`, `config create`, `config dump` e `listremotes` não recebem `CREATE_NO_WINDOW`. No Windows, pode piscar um console | `core/gerenciador.go` | Junto da 008, que mexe nas mesmas chamadas |
| A leitura do OAuth junta blocos de 4096 bytes sem guardar o resto da linha. Uma linha cortada perde a URL ou o token | `core/oauth.go:Iniciar` | Junto da 016 (teste) ou da 017 |
| O mesmo cofre pode ser montado em duas letras | `core/montagem.go:MontarUnidade` | Junto da 018 (`montando` bloqueia o segundo clique) e da 001 |
| `LetrasPreferidas` inclui A, B e C | `core/constantes.go` | Baixo risco: `ObterLetrasDisponiveis` já pula letras em uso |
| O item da bandeja diz "Auto-iniciar com Windows" em todos os SOs e não mostra se está ligado | `tray/tray.go:aoIniciar` | Junto da 019 |
| A ordem das flags VFS muda a cada chamada (`range` em mapa) | `core/vfs.go:ConstruirArgs` | Junto da 011 |
| O módulo Go é `github.com/eufrauzino/runtime-crypt-go` e o repositório é `douglassouza-dev/runtime-crypt-go` | `go.mod`, README | Decisão de Douglas. Não afeta o build |
| `GEMINI.md` descreve `cofre.bin`, chave mestra e servidor de streaming, que não existem no código | `GEMINI.md` | Corrigir o texto quando a 004 fechar |
| O README promete coisas que o código não faz | ver tabela em [01-visao-e-requisitos.md](01-visao-e-requisitos.md#o-texto-diz--o-código) | Cada PR de demanda corrige a linha do README que lhe cabe |

## O que não foi verificado

- A montagem real no Windows com WinFsp e em Linux com FUSE. A caixa de leitura não tem esses drivers.
- O build completo para Windows e macOS. Só `internal/core` e `internal/plataforma` foram compilados para Windows.
- O tamanho do binário.
- O comportamento do rclone em versões diferentes da v1.60.1.
- O comportamento real de dois laços de GUI (Fyne e systray) em cada SO.
