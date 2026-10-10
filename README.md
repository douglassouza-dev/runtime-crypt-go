# RuntimeCrypto

<p align="center">
  <strong>🔒 Cofre Criptografado na Nuvem — Binário Nativo Multiplataforma</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.22+">
  <img src="https://img.shields.io/badge/Plataforma-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey?style=flat-square" alt="Multiplataforma">
  <img src="https://img.shields.io/badge/Licen%C3%A7a-GPL--3.0-blue?style=flat-square" alt="GPL-3.0">
  <img src="https://img.shields.io/badge/RClone-Crypt-green?style=flat-square" alt="RClone Crypt">
</p>

---

RuntimeCrypto é um gerenciador de cofres criptografados na nuvem. Monta unidades virtuais criptografadas usando **RClone Crypt** + **WinFsp/FUSE**, com interface gráfica nativa e ícone no System Tray.

Seus arquivos são **criptografados localmente antes de enviados** à nuvem e **descriptografados instantaneamente** ao acessar a unidade virtual. A chave de criptografia nunca sai do seu computador.

📚 Documentação técnica, demandas e ADRs: [docs/](docs/README.md)

🧪 Roteiro de testes em tela: [docs/testes/](docs/testes/README.md)

## ✨ Funcionalidades

- 🔐 **Criptografia ponta-a-ponta** — AES-256 via RClone Crypt
- 🖥️ **Unidades virtuais** — Monta como drive normal (ex: `V:\`) via WinFsp/FUSE
- ☁️ **Multi-nuvem** — Google Drive, OneDrive, Dropbox, Amazon S3, Pasta Local
- 🔑 **OAuth nativo** — Autenticação segura sem copiar tokens manualmente
- 🎨 **Interface gráfica** — GUI escura e moderna com Fyne
- 📌 **System Tray** — Roda em segundo plano com menu no ícone da bandeja
- ⚡ **Binário único** — Sem Python, sem pip, sem virtualenv (~15 MB)
- 🖱️ **Um clique** — Destrancar cofre = montar + abrir Explorer
- 🔄 **Auto-montar** — Cofres favoritos montam ao iniciar o programa
- ⚙️ **VFS configurável** — Cache, chunking, polling, write-back

## 🏗️ Arquitetura

```
runtime-crypto/
├── main.go                          # Entry point
├── go.mod / go.sum                  # Dependências Go
├── internal/
│   ├── core/                        # Lógica de negócio (sem GUI)
│   │   ├── gerenciador.go           # Struct principal — coordena tudo
│   │   ├── cofres.go                # CRUD de cofres (vaults.json)
│   │   ├── montagem.go              # Montar/Desmontar unidades via RClone
│   │   ├── senhas.go                # Cache de senhas em memória (RWMutex)
│   │   ├── oauth.go                 # Fluxo OAuth (rclone authorize)
│   │   ├── vfs.go                   # Configurações VFS + flags CLI
│   │   ├── provedores.go            # Provedores de nuvem suportados
│   │   ├── constantes.go            # Defaults e constantes
│   │   ├── plataforma_windows.go    # CREATE_NO_WINDOW para subprocessos
│   │   └── plataforma_outros.go     # Stub Linux/macOS
│   │
│   ├── gui/                         # Interface gráfica (Fyne v2)
│   │   ├── janela_principal.go      # Janela principal com cards de cofres
│   │   ├── dialogos.go              # Diálogos de senha e mensagem
│   │   ├── wizards.go               # Wizard criar/importar cofre
│   │   ├── config_vfs.go            # Painel de configurações VFS
│   │   ├── seletor_pasta.go         # Navegador de pastas remotas
│   │   └── tema.go                  # Paleta de cores (tema escuro)
│   │
│   ├── tray/                        # System Tray
│   │   └── tray.go                  # Ícone + menu dinâmico
│   │
│   └── plataforma/                  # Código específico de SO
│       ├── windows.go               # WinFsp, registro, auto-iniciar
│       ├── linux.go                 # FUSE, XDG autostart
│       └── darwin.go                # macFUSE, LaunchAgent
│
└── assets/
    └── icone.ico                    # Ícone do System Tray
```

## 📋 Pré-requisitos

| Requisito | Versão | Notas |
|-----------|--------|-------|
| **Go** | 1.22+ | Compilação |
| **GCC/MinGW** | Qualquer | CGO para Fyne (OpenGL) |
| **RClone** | 1.60+ | No PATH ou ao lado do executável |
| **WinFsp** | 2.0+ | Windows — montagem de unidades virtuais |
| **FUSE** | 3.x | Linux — `sudo apt install fuse3` |
| **macFUSE** | 4.x | macOS — [osxfuse.github.io](https://osxfuse.github.io/) |

## 🗂️ Onde o cofre aparece em cada sistema

| Sistema | Ponto de montagem | Ao trancar |
|---|---|---|
| Windows | Letra livre (V:, W:, …) | A letra some com o rclone |
| Linux | Pasta `~/RuntimeCrypto/<nome do cofre>`, criada ao destrancar | O rclone solta a pasta; se ela ficar presa, `fusermount3 -u` (ou `fusermount -u`). A pasta é removida se estiver vazia |
| macOS | Pasta `~/RuntimeCrypto/<nome do cofre>`, criada ao destrancar | Igual ao Linux, com `umount` no lugar do `fusermount` |

A pasta não é configurável por cofre (demanda 013).

### O que foi testado em cada sistema

- **Linux (FUSE3, rclone 1.60):** testado de verdade. Conectar um cofre crypt que já existia numa pasta local, destrancar, gravar um arquivo, trancar: `mount | grep rclone` fica vazio e a pasta do ponto sai. Destrancar de novo lê o arquivo. Foi feito pelo `core`, sem a janela. A janela não foi aberta no Linux.
- **Windows (WinFsp):** a montagem em letra é a de antes. O CI roda os testes no Windows com um rclone falso; a montagem real no WinFsp não foi refeita nesta mudança.
- **macOS (macFUSE/FUSE-T):** **não testado**. Não houve máquina. O código usa `umount` para soltar uma pasta presa e compila (`GOOS=darwin go vet ./internal/core`).

## 🚀 Compilação

```bash
# Clone o repositório
git clone https://github.com/eufrauzino/runtime-crypt-go.git
cd runtime-crypt-go

# Baixar dependências
go mod tidy

# Compilar (Windows — sem janela de console)
go build -o runtime-crypt-go.exe -ldflags="-H windowsgui" .

# Compilar (Linux)
go build -o runtime-crypt-go .

# Compilar (macOS)
go build -o runtime-crypt-go .
```

### Cross-Compilation

```bash
# De Linux/macOS para Windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o runtime-crypt-go.exe -ldflags="-H windowsgui" .

# De Windows para Linux
set GOOS=linux&& set GOARCH=amd64&& go build -o runtime-crypt-go .
```

## 📖 Uso

```bash
# Executar
./runtime-crypt-go

# O programa inicia com:
# 1. Ícone no System Tray
# 2. Janela principal aberta
```

### Fluxo Básico

1. **Adicionar Cofre** → Escolher provedor → Autorizar (OAuth) → Definir senha
2. **Destrancar** → Digitar senha → Unidade montada automaticamente → Explorer abre
3. **Trancar** → Um clique → Unidade desmontada → Senha limpa da memória
4. **Sair** → Todas as unidades são desmontadas automaticamente

## 🔒 Segurança

- Senhas ficam **apenas em RAM** (`sync.RWMutex`) — nunca salvas em disco
- `RCLONE_CONFIG_PASS` é injetada via variável de ambiente do subprocesso
- Subprocessos RClone rodam com `CREATE_NO_WINDOW` (sem console visível)
- Ao trancar: senha é zerada do cache e unidade desmontada
- Ao sair: todas as montagens são encerradas e senhas limpas

## 📄 Licença

Copyright (C) 2026 Douglas Eufrauzino de Souza

O RuntimeCrypto é licenciado sob a GPLv3 ([GNU General Public License v3.0](LICENSE)), somente a versão 3 (SPDX: `GPL-3.0-only`). O texto completo está em [LICENSE](LICENSE).
