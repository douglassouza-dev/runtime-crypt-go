# Changelog

Todas as mudanças notáveis deste projeto serão documentadas neste arquivo.

O formato segue [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/),
e o projeto adere ao [Versionamento Semântico](https://semver.org/lang/pt-BR/).

---

## [3.1.0] — 2026-06-23

### Estabilidade — Cofre Robusto

#### Persistência
- **Escrita atômica do `vaults.json`** — Salva em arquivo temporário (`.tmp`) e renomeia atomicamente, prevenindo corrupção por crash, disco cheio ou queda de energia.
- **Backup automático (`vaults.json.bak`)** — Cópia de segurança mantida a cada carga bem-sucedida. Se o arquivo principal corromper, o backup é usado como fallback automático.

#### Montagem / Desmontagem
- **Corrigido TOCTOU race** — Dupla verificação sob lock antes de inserir montagem no mapa, evitando que duas goroutines montem na mesma letra simultaneamente.
- **Corrigido `ProcessState.Exited()`** — O código verificava `cmd.ProcessState` que era sempre `nil` após `Start()`. Substituído por goroutine com `cmd.Wait()` e canal `finalizado` para detecção real de queda do processo.
- **Corrigido `processoAtivo()` no Windows** — Substituído por canal `finalizado` que funciona cross-platform (o `Signal(nil)` no Windows não detecta processos mortos).
- **Zumbis não removidos do mapa** — `DesmontarUnidade` não remove a montagem se `Kill()` falhar após 3 tentativas, evitando perda de rastreamento do processo.
- **`os.Interrupt` no Windows** — Pulado (é no-op na plataforma), indo direto para `Kill()`, economizando 5 segundos de espera inútil.

#### Validação
- **Validação de valores VFS** — Regex por categoria (cache_mode enum, tamanhos com sufixo K/M/G/T/Ki, durações com s/m/h/ms). Valores inválidos são silenciosamente rejeitados.
- **Validação de nome de cofre** — Proíbe `:`, `/`, `\` e nomes com mais de 64 caracteres. Aplicada nos wizards de criação e importação.
- **Validação de caminho local** — `DialogoPastaLocal` verifica se o caminho existe e é um diretório antes de confirmar.

#### Vazamentos / Concorrência
- **Goroutine leak no OAuth** — `Abortar()` agora coordena shutdown com canal `finalizado`, eliminando double-`Wait()` e garantindo que a goroutine de leitura do stdout termine.
- **Goroutine `canalAcoes` fechada no shutdown** — `close(canalAcoes)` nos dois caminhos de saída (botão Sair e tray Sair), eliminando goroutine pendurada.
- **Nil pointer em `ListarDiretoriosRemoto`** — Guarda `cmd.Process != nil` antes de `Kill()`, evitando panic se o timeout disparar antes do `cmd.Output()`.
- **Race em `caminhoAtual` no seletor de pasta** — Adicionado `sync.Mutex` protegendo leituras e escritas entre callbacks Fyne e goroutines.
- **`os.Getwd()` erro tratado** — Fallback para `"."` se ambos `os.Executable()` e `os.Getwd()` falharem, em vez de string vazia.

#### Outras correções
- **Parser de diretórios com espaços** — `strings.Fields` substitui `SplitN` no parser do `rclone lsd`, suportando pastas como `"Meus Documentos"`.
- **`Atualizar()` propaga erro** — `salvar()` que falhava silenciosamente agora retorna `false`, impedindo divergência entre memória e disco.
- **Console no `AbrirNavegador`** — Adicionado `CREATE_NO_WINDOW` no Windows, consistente com `AbrirExplorador`.

### Segurança
- **Zeroização explícita de senhas** — `CacheSenhas` agora armazena `[]byte` (não `string` imutável). `Limpar()` e `LimparTodas()` zeroizam cada byte antes de remover do mapa.

### Adicionado — Presets VFS por Caso de Uso

- **5 presets pré-otimizados** com valores específicos por tipo de conteúdo:
  - **Streaming de Vídeo** — Cache 50G/24h, chunk 128M, read-ahead 256M, buffer 32M
  - **Documentos e Escritório** — Cache writes 2G, polling 10s, write-back 1s
  - **Livros e PDFs** — Cache full 5G/4h, read-ahead 32M para leitura sequencial
  - **Backup** — Cache writes-only, chunks 64M, sem read-ahead
  - **Fotos** — Cache minimal 2G, attr_timeout 10m, baixo consumo de disco
- **Barra de presets** no diálogo ⚙ Configurações VFS — um clique aplica todos os valores do preset nos campos.

### Adicionado — Configuração VFS por Cofre

- **Campo `vfs_override` no cofre** — Persistido no `vaults.json` com `omitempty`. Cada cofre pode ter seus próprios parâmetros VFS, sobrescrevendo a configuração global.
- **Montagem com override** — `destravarCofre`, `autoMontarCofres` e o handler do tray passam `VfsOverride` para `MontarUnidade`.

### Adicionado — Pasta Local

- **`DialogoPastaLocal`** — Diálogo modal para informar o caminho de uma pasta no sistema de arquivos local.
- **Fluxo `local_path` corrigido** — Criação e importação de cofres em pasta local agora funcionam corretamente (antes estavam quebrados com `remote: ""` ou TODO).

### Adicionado — Testes

- **7 novos testes** no pacote `core`:
  - `TestValidarValorVfs` — 42 casos de validação de valores VFS
  - `TestAtualizarRejeitaValoresInvalidos` — Verifica que valores inválidos não alteram config
  - `TestPresetsVfs` — Valida que todos os presets têm valores sintaticamente corretos
  - `TestCacheSenhasZeroizacao` — Verifica `Limpar()` individual
  - `TestCacheSenhasZeroizacaoMultipla` — Verifica `LimparTodas()`
  - `TestCacheSenhasReutilizacaoAposLimpeza` — Reuso do cache após limpeza
  - `TestCacheSenhasConcorrenciaBasica` — Acesso concorrente com RWMutex
- Total de testes: 2 → 9

### Alterado
- Versão do programa: `3.0` → `3.1.0`
- `InfoMontagem` ganhou campo interno `finalizado chan struct{}` para coordenação de ciclo de vida do processo.
- `GerenciadorOAuth` ganhou campo interno `finalizado chan struct{}` para coordenação de shutdown da goroutine de leitura.

## [3.0.0] — 2026-06-06

### 🔄 Reescrita Completa — Python → Go

O RuntimeCrypto foi completamente reescrito em **Go**, substituindo a versão anterior em Python.

### Adicionado
- **Binário único nativo** — Sem dependência de Python, pip ou virtualenv
- **Interface gráfica Fyne v2** — Widgets multiplataforma com tema escuro
- **System Tray** — Via `github.com/getlantern/systray` com menu dinâmico
- **Suporte multiplataforma** — Estrutura preparada para Windows, Linux e macOS
  - Windows: WinFsp + registro para auto-iniciar
  - Linux: FUSE3 + XDG autostart
  - macOS: macFUSE + LaunchAgent
- **Cache de senhas com RWMutex** — Acesso concorrente seguro
- **Contextos com timeout** — Montagem com polling de 45s e backoff progressivo
- **Goroutines** — Operações assíncronas (OAuth, montagem, atualização de UI)
- **Configurações VFS editáveis** — 13 parâmetros com painel dedicado
- **Navegador de pastas remotas** — Seletor visual para importação de cofres
- **Wizards** — Criar e importar cofres com fluxo guiado em abas

### Removido
- Dependência de Python 3.x, `customtkinter`, `pystray`, `Pillow`
- Arquivo `requirements.txt`
- `runtime_crypto.py` (entry point Python)
- Diretórios `core/` e `gui/` do Python
- `tema_runtime.json` (tema CustomTkinter)
- `.venv/` e `__pycache__/`

### Alterado
- Arquitetura reorganizada em pacotes Go (`internal/core`, `internal/gui`, `internal/tray`, `internal/plataforma`)
- Nomenclatura mantida em Português (Brasil) conforme regras do projeto
- `GerenciadorRClone` agora é uma struct Go com subsistemas separados (Cofres, Montagens, Senhas, OAuth, VFS)
- Comunicação tray↔GUI via canais Go ao invés de `queue.Queue`

### Segurança
- Senhas protegidas por `sync.RWMutex` (era `threading.Lock`)
- Subprocessos com `CREATE_NO_WINDOW` via `SysProcAttr`
- Variáveis de ambiente isoladas por subprocesso (`RCLONE_CONFIG_PASS`)
