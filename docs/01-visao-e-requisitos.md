# 01 — Visão e requisitos

## Objetivo

O RuntimeCrypto guarda e simplifica o uso de cofres **rclone crypt**. Um cofre é um remoto `crypt` do rclone apontado para uma pasta em um armazenamento online. O programa monta esse remoto como unidade local com `rclone mount`, usando WinFsp no Windows e FUSE no Linux e no macOS.

O pedido de Douglas define três qualidades: integração com o armazenamento online o mais nativa possível, configurável e altamente estável. Este documento separa o que o código já faz do que ainda falta para chegar lá.

## Escopo atual (o que o código faz)

- Aplicação desktop em Go, com janela Fyne v2 e ícone na bandeja (`getlantern/systray`).
- Usa o binário `rclone` externo, procurado ao lado do executável e depois no `PATH` (`internal/core/gerenciador.go:localizarRclone`).
- Cria cofres novos no Google Drive, no OneDrive e no Dropbox, com OAuth feito por `rclone authorize`.
- Conecta um cofre crypt que já existe nesses provedores. O usuário escolhe a pasta em um navegador de pastas remotas.
- Monta e desmonta cofres em letras de unidade do Windows.
- Guarda a lista de cofres em `vaults.json`, ao lado do executável.
- Deixa editar 13 parâmetros VFS do `rclone mount`, só durante a sessão.
- Liga e desliga o início automático com o sistema: registro no Windows, `.desktop` no Linux, LaunchAgent no macOS.

## Fora do escopo atual

- API HTTP ou `rclone rc`. Não existe nenhuma das duas.
- Remover ou renomear cofre pela interface. `GerenciadorCofres.Remover` existe, mas nenhuma tela chama.
- Marcar cofre para montar automaticamente pela interface. `GerenciadorCofres.Atualizar` existe, mas nenhuma tela chama.
- Montagem em pasta no Linux e no macOS (ver [demanda 013](demanda/013-montagem-linux-macos.md)).

## Requisitos funcionais (derivados do código)

| ID | Requisito | Onde está | Situação |
|---|---|---|---|
| RF-01 | Criar cofre em provedor com OAuth | `main.go:acaoNovoCofre` | Funciona no caminho feliz |
| RF-02 | Criar cofre em S3 | `main.go:acaoNovoCofre` | Não funciona: as credenciais não são pedidas e o remoto base não é criado |
| RF-03 | Conectar cofre existente | `main.go:acaoImportarCofre` | Funciona para OAuth. Pasta Local para em um `TODO`. S3 não cria o remoto base |
| RF-04 | Destrancar (montar) | `main.go:destravarCofre`, `core/montagem.go:MontarUnidade` | Monta no Windows. No Linux e no macOS falha por falta de letra |
| RF-05 | Trancar (desmontar) | `main.go:travarCofre`, `core/montagem.go:DesmontarUnidade` | Na prática não chega a ser chamado (ver [demanda 001](demanda/001-rastreio-de-montagem.md)) |
| RF-06 | Auto-montar ao iniciar | `main.go:autoMontarCofres` | Nunca monta nada (ver [demanda 015](demanda/015-auto-montar.md)) |
| RF-07 | Editar VFS | `gui/config_vfs.go:DialogoConfigVfs` | Funciona, só em memória e sem validação |
| RF-08 | Verificar WinFsp/FUSE | `plataforma/*:VerificarWinfsp` | Só pelo menu da bandeja |
| RF-09 | Auto-iniciar com o sistema | `plataforma/*:AdicionarAutoIniciar` | Funciona. Erros são ignorados em `main.go` |
| RF-10 | Sair desmontando tudo | `core/gerenciador.go:Encerrar` | Não desmonta, pelo mesmo motivo do RF-05 |

## Requisitos não funcionais

Não há número de carga, latência ou disponibilidade no repositório. Nada disso foi inventado aqui.

| NFR | O que o código faz hoje | Lacuna |
|---|---|---|
| Estabilidade da montagem | Espera até 45 s a unidade aparecer | Perde o processo logo depois de montar. Não detecta queda do rclone durante a sessão |
| Tempo limite em chamadas externas | Só em `rclone lsd` (30 s) e na montagem (45 s) | `config create`, `config dump`, `listremotes`, `obscure`, `--version` e `config delete` não têm limite |
| Segredos | Senha do cofre na memória da sessão | A senha do crypt fica ofuscada (reversível) no `rclone.conf`. O token OAuth fica em texto puro no `rclone.conf` |
| Observabilidade | Nenhum log | O stderr do rclone é descartado |
| Portabilidade | Compila em Linux. O pacote `core` compila para Windows | A montagem só funciona com letra de unidade, ou seja, só no Windows |
| Testabilidade | 2 testes de unidade | Nada testa rclone, montagem ou persistência |

## O texto diz | O código

Comparação entre README.md, CHANGELOG.md e GEMINI.md e o código do commit `c301180`.

| O texto diz | O código |
|---|---|
| "Criptografia ponta-a-ponta — AES-256 via RClone Crypt" (README) | O rclone crypt usa XSalsa20-Poly1305 nos dados e AES-256 (EME) nos nomes. O programa não escolhe o algoritmo. Só repassa `filename_encryption`, `directory_name_encryption` e `no_data_encryption` (`core/gerenciador.go:CriarCrypt`) |
| "Senhas ficam apenas em RAM — nunca salvas em disco" (README) | `CriarCrypt` grava `password` e `password2` no `rclone.conf`, ofuscadas com `rclone obscure`. Ofuscação é reversível com `rclone reveal`. Reproduzido |
| "`RCLONE_CONFIG_PASS` é injetada via variável de ambiente" (README) | É verdade, mas essa variável só serve para abrir um `rclone.conf` cifrado. O programa nunca cifra o `rclone.conf`. Com o arquivo em claro, qualquer senha digitada destranca. Reproduzido: `rclone cat` com `RCLONE_CONFIG_PASS` errada leu o arquivo |
| "Subprocessos RClone rodam com `CREATE_NO_WINDOW`" (README) | Só `obscure`, `config delete`, `lsd`, `mount` e `authorize`. `--version`, `config create`, `config dump` e `listremotes` não recebem a flag |
| "Ao trancar: senha é zerada do cache e unidade desmontada" (README) | A senha sai do cache. A desmontagem depende de o processo estar no mapa de montagens, e `processoAtivo` sempre devolve falso (`core/montagem.go:processoAtivo`). Reproduzido |
| "Ao sair: todas as montagens são encerradas" (README) | `DesmontarTodas` percorre um mapa que `Status()` já esvaziou. O rclone continua rodando |
| "Auto-montar — Cofres favoritos montam ao iniciar o programa" (README) | `autoMontarCofres` exige senha no cache, e o cache está vazio ao iniciar. Nenhuma tela liga `auto_montar` |
| "Multi-nuvem — Google Drive, OneDrive, Dropbox, Amazon S3, Pasta Local" (README) | S3 não pede credenciais nem cria o remoto base. Pasta Local não aparece em "novo cofre" e para em um `TODO` em "conectar existente" |
| "Plataforma Windows / Linux / macOS" (README) e "Suporte multiplataforma" (CHANGELOG) | Compila nos três. A montagem precisa de letra, e `ObterLetrasDisponiveis` devolve `nil` fora do Windows |
| "Binário único (~15 MB)" (README) | Desconhecido. O tamanho não foi medido no Windows. O binário ainda depende do `rclone` externo e do WinFsp/FUSE |
| "Go 1.22+" (README) | `go.mod` declara `go 1.26.4` |
| "RClone 1.60+" (README) | O código não confere a versão. Só roda `rclone --version` para saber se o binário existe |
| "System Tray — via getlantern/systray com menu dinâmico" (CHANGELOG) | O menu é fixo. Existe `AcaoCofre`, mas nenhum item de cofre é criado. `AtualizarTooltip` nunca é chamado |
| "Configurações VFS editáveis — 13 parâmetros" (CHANGELOG) | São 13 mesmo. Ficam só em memória e voltam ao padrão quando o programa reinicia |
| `git clone https://github.com/eufrauzino/runtime-crypt-go.git` (README) | O repositório está em `douglassouza-dev/runtime-crypt-go`. O módulo Go continua `github.com/eufrauzino/runtime-crypt-go` |
| "Nunca salvar a chave mestra em texto puro. Sempre usar `cofre.bin`" e "chave de 256 bits na RAM do servidor de streaming" (GEMINI.md) | Não existe `cofre.bin`, chave mestra nem servidor de streaming no código Go |
