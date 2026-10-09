# 06 — API

Não existe API HTTP. O programa não abre porta nenhuma. Não usa `rclone rc` nem `rclone rcd`: nenhum comando passa `--rc`, e não há cliente HTTP no código.

Há duas superfícies: a API Go do `internal/core`, usada por `main.go`, `internal/gui` e `internal/tray`, e o contrato de linha de comando com o binário `rclone`.

## 1. API Go do `internal/core`

Os pacotes estão em `internal/`, então só este módulo pode importá-los. As funções devolvem `(bool, string)` em vez de `error` na maior parte dos casos. A `string` é uma mensagem para o usuário, em português, sem acento.

### `GerenciadorRClone` (`gerenciador.go`)

| Função | Retorno | Faz | Chama rclone | Tempo limite |
|---|---|---|---|---|
| `NovoGerenciador()` | `*GerenciadorRClone` | Descobre o diretório do app, localiza o rclone, carrega `vaults.json` | `--version` até 3 vezes | Não |
| `EstaDisponivel()` | `bool` | `Executavel != ""` | — | — |
| `ObscurecerSenha(senha)` | `(string, error)` | Ofusca a senha | `obscure -` | Não |
| `CriarRemoto(nome, tipo, params)` | `(bool, string)` | Cria um remoto. Ignora parâmetros vazios | `config create` | Não |
| `CriarCrypt(nome, remotoBase, senha, senha2, configCrypt)` | `(bool, string)` | Ofusca as senhas e cria o crypt | `obscure` ×2, `config create` | Não |
| `ImportarCrypt(...)` | `(bool, string)` | Igual a `CriarCrypt` | idem | Não |
| `RemoverRemoto(nome)` | `(bool, string)` | Apaga o remoto | `config delete` | Não |
| `ListarRemotos()` | `[]string` | Remotos `crypt`, com `:` no fim. Erro vira `nil` | `config dump` | Não |
| `ListarTodosRemotos()` | `[]string` | Todos os remotos. Erro vira `nil` | `listremotes` | Não |
| `ListarRemotosDetalhado()` | `[]RemotoDetalhado` | Remotos com tipo e estado de montagem | `config dump` | Não |
| `ObterConfigRemoto(nome)` | `map[string]interface{}` | Seção do remoto, segredos incluídos | `config dump` | Não |
| `ListarDiretoriosRemoto(remoto, caminho)` | `[]string` | Subpastas, em ordem alfabética. Erro e tempo esgotado viram `nil` | `lsd` | 30 s |
| `ListarCofres()` | `[]CofreStatus` | Cofres com estado | — | — |
| `Encerrar()` | — | Desmonta tudo, limpa as senhas, aborta o OAuth | — | Até 15 s por montagem |

`ListarRemotos`, `ListarTodosRemotos`, `ListarRemotosDetalhado` e `ObterConfigRemoto` não têm chamador no repositório.

### `GerenciadorCofres` (`cofres.go`)

| Função | Retorno | Observação |
|---|---|---|
| `Listar(montagens, senhas)` | `[]CofreStatus` | — |
| `Adicionar(nome, provedorId, provedorNome, remotoBase)` | `(bool, string)` | Recusa nome repetido. Grava em disco |
| `Remover(nome)` | `(bool, string)` | Sem chamador. Não remove os remotos do rclone |
| `Atualizar(nome, campos)` | `bool` | Aceita só `auto_montar` (bool) e `caminho_cripto` (string). Ignora o erro de gravação. Sem chamador |
| `Obter(nome)` | `*Cofre` | Devolve uma cópia ou `nil` |

### `GerenciadorMontagem` (`montagem.go`)

| Função | Retorno | Observação |
|---|---|---|
| `MontarUnidade(remoto, letra, senha, override)` | `(ok, msg, letra)` | Bloqueia até 45 s. Letra vazia: escolhe sozinho, só no Windows |
| `DesmontarUnidade(letra)` | `(bool, string)` | Bloqueia até 15 s. Devolve sucesso mesmo sem confirmar o fim do processo |
| `DesmontarTodas()` | — | Percorre o mapa em sequência |
| `Status()` | `[]StatusMontagem` | **Apaga do mapa** as montagens cujo processo parece morto. Hoje apaga todas |
| `ObterMontagens()` | `map[nomeRemoto]InfoMontagem` | Chama `Status()` |
| `ObterLetraPorRemoto(nome)` | `string` | Chama `Status()` |
| `ObterLetrasDisponiveis(extra)` | `[]string` | Função de pacote. Fora do Windows devolve `nil` |

### `CacheSenhas` (`senhas.go`)

`Armazenar`, `Obter`, `Limpar`, `LimparTodas`, `Existe`. Mapa protegido por `sync.RWMutex`. Coberto por `TestCacheSenhas`.

### `GerenciadorOAuth` (`oauth.go`)

| Função | Observação |
|---|---|
| `Iniciar(executavel, tipo) bool` | Aborta o anterior, roda `authorize`, lê a saída em goroutine. Não tem tempo limite próprio |
| `ObterStatus() StatusOAuth` | `Concluido` é `Token != ""` |
| `Abortar()` | `Kill` e `Wait` |

`AbrirNavegador(url)` e `AbrirExplorador(caminho)` também estão em `oauth.go`. Usam `cmd /c start`, `explorer`, `open` ou `xdg-open`.

### `ConfigVfs` (`vfs.go`)

`Obter`, `Atualizar`, `Restaurar`, `ConstruirArgs(override)`. Coberto por `TestConfigVfs`. A ordem das flags em `ConstruirArgs` muda a cada chamada, porque vem de um `range` em mapa.

### `internal/plataforma`

Mesmo conjunto de funções nos três SOs, escolhido por build tag: `VerificarAutoIniciar() bool`, `AdicionarAutoIniciar() error`, `RemoverAutoIniciar() error`, `VerificarWinfsp() InfoWinfsp`. No Linux, `VerificarWinfsp` só testa se `/dev/fuse` existe. No macOS, procura `macfuse.fs` ou `fuse-t.fs`. No Windows, procura a DLL, depois a chave de registro, depois a pasta em Program Files.

## 2. Contrato com o rclone

Todos os comandos usam `exec.Command(Executavel, ...)`. Nenhum passa `--config`, `--log-file`, `--log-level` nem `--rc`.

| Função | Linha de comando | Ambiente | stdin | Saída usada | Esconde janela (Windows) | Tempo limite |
|---|---|---|---|---|---|---|
| `testarRclone` | `rclone --version` | herdado | — | só o código de saída | Não | Não |
| `ObscurecerSenha` | `rclone obscure -` | herdado | senha em texto puro | stdout | Sim | Não |
| `CriarRemoto` | `rclone config create <nome> <tipo> [<chave> <valor>]...` | herdado | — | stdout+stderr como mensagem de erro | Não | Não |
| `RemoverRemoto` | `rclone config delete <nome>` | herdado | — | stdout+stderr como mensagem de erro | Sim | Não |
| `ListarRemotos`, `ListarRemotosDetalhado`, `ObterConfigRemoto` | `rclone config dump` | herdado | — | JSON no stdout | Não | Não |
| `ListarTodosRemotos` | `rclone listremotes` | herdado | — | uma linha por remoto | Não | Não |
| `ListarDiretoriosRemoto` | `rclone lsd <remoto>:<caminho>` | herdado | — | 5ª coluna de cada linha | Sim | 30 s, depois `Kill` |
| `GerenciadorOAuth.Iniciar` | `rclone authorize <tipo>` | herdado | — | stdout+stderr: URL com `127.0.0.1` ou `localhost`; JSON entre `Paste the following` e `End paste` | Sim | Não. O chamador espera 120 s |
| `MontarUnidade` | ver abaixo | herdado + `RCLONE_CONFIG_PASS=<senha>` | — | descartada (`Stdout = Stderr = nil`) | Sim | 45 s até o ponto aparecer |

### Parâmetros de `config create` por fluxo

| Fluxo | Comando |
|---|---|
| Base OAuth (`drive`, `onedrive`, `dropbox`) | `rclone config create <nome>_base <tipo> token '<json>'` |
| Base local (ramo morto) | `rclone config create <nome>_base local` (o parâmetro `remote` vazio é descartado) |
| Crypt | `rclone config create <nome> crypt remote <base> password <obs> password2 <obs> filename_encryption standard directory_name_encryption true [no_data_encryption true]` |

As senhas chegam já ofuscadas. O código não passa `--obscure` nem `--no-obscure`, então o rclone decide sozinho se o valor já está ofuscado. Com rclone v1.60.1 a senha guardada voltou certa com `rclone reveal` (reproduzido). Em outras versões, desconhecido.

`config create` em um nome que já existe sai com código 0 e substitui a seção. Reproduzido com rclone v1.60.1.

### Linha de `rclone mount`

```text
rclone mount <nome>: <LETRA>:
  --vfs-cache-mode full
  --vfs-cache-max-size 10G
  --vfs-cache-max-age 1h
  --vfs-read-chunk-size 8M
  --vfs-read-chunk-size-limit 512M
  --vfs-read-ahead 16M
  --buffer-size 16M
  --dir-cache-time 30m
  --poll-interval 30s
  --attr-timeout 1m
  --vfs-write-back 5s
  --vfs-disk-space-total-size 1T
  [--cache-dir <pasta>]
  --volname "RuntimeCrypto (<nome>)"
  --no-checksum
  --no-modtime
  [--network-mode --no-console]   # só Windows
```

Os valores acima são os padrões de `core/constantes.go:ConfiguracoesVfsPadrao`. A ordem real das flags VFS varia. O código não confere se a versão do rclone instalada aceita cada flag. Se uma flag não existir, o rclone sai com erro, o stderr é descartado e o usuário só vê o aviso de 45 s.

### Encerramento

| Ação | Como |
|---|---|
| Desmontar | `Process.Signal(os.Interrupt)`; depois `Process.Kill()`. No Windows, o Go não envia `os.Interrupt` e devolve erro na hora. Sobram os `Kill` |
| Abortar OAuth | `Process.Kill()` + `Wait()` |
| Tempo esgotado em `lsd` ou na montagem | `Process.Kill()` sem `Wait()` |
