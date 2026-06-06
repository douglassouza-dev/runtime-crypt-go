# Changelog

Todas as mudanças notáveis deste projeto serão documentadas neste arquivo.

O formato segue [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/),
e o projeto adere ao [Versionamento Semântico](https://semver.org/lang/pt-BR/).

---

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
