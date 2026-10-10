# 031 — vaults.json e log na pasta de configuração do usuário

- Estado: Aberta
- Risco: Alto · dados
- Onde: `internal/core/gerenciador.go:NovoGerenciador`; `internal/core/pasta_config.go` (novo); `internal/core/cofres.go`; `internal/core/vfs.go`; `main.go`; `internal/gui/acoes.go:AvisosDaAbertura`.
- Depende de: 007, 012, 027

## Contexto

`vaults.json`, `vfs.json` e `runtimecrypto.log` ficam na pasta do executável (`obterDiretorioApp`). No Windows, instalado em Program Files, essa pasta não aceita gravação: criar cofre, mudar a VFS e gravar o log falham.

## Onde fica cada coisa

| O quê | Hoje | Depois | Por quê |
|---|---|---|---|
| `vaults.json` | pasta do executável | `os.UserConfigDir()/RuntimeCrypto` | pedido da demanda |
| `runtimecrypto.log` (e `.log.1`) | pasta do executável | `os.UserConfigDir()/RuntimeCrypto` | pedido da demanda |
| `vfs.json` | ao lado do `vaults.json` (012) | junto com o `vaults.json` | é configuração do usuário gravada pelo app, sofre do mesmo problema em Program Files, e a 012 o põe ao lado do `vaults.json` |
| cópias `vaults.json.corrompido-*` (007/024) | ao lado do `vaults.json` | seguem o `vaults.json` (nascem na pasta nova) | a cópia fica ao lado do arquivo que ela guarda; as antigas ficam onde estão |
| `rclone.conf` | padrão do rclone (`~/.config/rclone/rclone.conf`, `%AppData%\rclone\rclone.conf`) ou `RCLONE_CONFIG` | fica | o app não escolhe o caminho; é do rclone |
| cache da VFS | `cache_dir` do `vfs.json`, senão o padrão do rclone (`os.UserCacheDir()/rclone`) | fica | já está fora da pasta do executável; mudar perderia o que não subiu (026/028) |
| `rclone`/`rclone.exe` | procurado na pasta do executável e no PATH | fica | é programa, não estado |

Nome da pasta: `RuntimeCrypto`, o nome do app na bandeja, em `~/RuntimeCrypto` e no `--volname`. Exemplos: `%AppData%\RuntimeCrypto` (Windows), `~/.config/RuntimeCrypto` (Linux, ou `$XDG_CONFIG_HOME/RuntimeCrypto`), `~/Library/Application Support/RuntimeCrypto` (macOS). Sem variável de ambiente nem flag para trocar.

## O que muda

- Na abertura, o app cria a pasta nova e confere que ela aceita gravação (cria e apaga um arquivo temporário).
- Cópia única, por arquivo (`vaults.json`, `vfs.json`), como a 007 do publisher-go:
  - se o arquivo novo existe, ele vale e o antigo é ignorado;
  - se só o antigo existe, ele é copiado para um temporário na pasta nova e renomeado no fim;
  - o antigo nunca é apagado nem regravado;
  - a cópia vai para o log.
- O log não é copiado: o novo começa na pasta nova; os antigos ficam onde estão.
- Ler nunca grava: abrir o app não muda a data do `vaults.json` nem do `vfs.json`.
- Se a pasta nova não pode ser criada ou gravada (ou a cópia falha), ou se não dá para saber a pasta do usuário:
  - a tela avisa na abertura (proposta, aguarda a UI): título `Pasta de configuração`, texto `Não deu para usar a pasta de configuração ({pasta}): mudanças nos cofres e nas configurações não serão salvas.` (sem o caminho quando ele não é conhecido);
  - cada arquivo é lido da pasta nova se existir lá, senão da pasta do executável, só para leitura;
  - nada é gravado em lugar nenhum: criar, remover ou editar cofre e mudar a VFS recusam com a mesma frase; nem a cópia `.corrompido` é gravada;
  - o log não vai para arquivo (nunca para a pasta do executável).
- `NovoGerenciadorEm` (testes) continua usando uma pasta só para tudo, sem cópia.

## O que fica de fora

- Apagar ou limpar os arquivos antigos da pasta do executável.
- Mover `rclone.conf` ou o cache da VFS.
- Tela para escolher a pasta.

## Pronto quando

- [ ] Teste: só o antigo → copiado para a pasta nova; o antigo igual, byte a byte e na data; sem temporário sobrando; a segunda abertura não copia de novo.
- [ ] Teste: os dois existem → vale o novo; nenhum é regravado.
- [ ] Teste: abrir com os arquivos na pasta nova não muda a data deles.
- [ ] Teste: pasta nova impossível de criar, pasta nova sem gravação (Unix), pasta do usuário desconhecida e cópia que falha no rename → frase, leitura do antigo, gravações recusadas, nada novo na pasta do executável.
- [ ] Teste: `vaults.json` antigo corrompido com a pasta nova inutilizável não deixa `.corrompido` na pasta do executável.
- [ ] Linux: executável com `vaults.json` ao lado, `XDG_CONFIG_HOME` vazio de app → o arquivo aparece em `RuntimeCrypto/`, o log também, o antigo fica igual.
- [ ] Na tela (Windows, instalado em Program Files): o mesmo roteiro.
