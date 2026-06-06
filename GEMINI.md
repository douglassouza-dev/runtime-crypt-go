# Diretrizes de Desenvolvimento — RuntimeCrypto

Este arquivo contém as instruções fundamentais para a evolução do projeto. O Gemini deve seguir estas regras em todas as sessões:

## 1. Idioma e Nomenclatura
- **Obrigatório:** Todo o código-fonte (variáveis, funções, classes, métodos) deve ser escrito em **Português (Brasil)**.
- Evitar termos técnicos em inglês no código, a menos que sejam palavras-chave da linguagem (ex: `func`, `struct`, `interface`, `package`).
- Comentários devem ser mínimos e estritamente técnicos.
- Em Go, seguir a convenção `PascalCase` para exports e `camelCase` para privados.

## 2. Arquitetura de Segurança
- **Cofre:** Nunca salvar a chave mestra em texto puro. Sempre usar o sistema de cofre (`cofre.bin`) com derivação de senha.
- **Memória:** A chave de 256-bits deve permanecer apenas na RAM do servidor de streaming.
- **Senhas:** O cache de senhas usa `sync.RWMutex` e é limpo ao trancar ou encerrar.

## 3. Gestão de Histórico (Git)
- Manter o histórico de commits limpo e sem rastros de refatorações de IA.
- Novos recursos devem ser comitados de forma atômica e organizada.
- Arquivos sensíveis (`cofre.bin`, `historico.bin`, `master.key`, `*.qnt`, `vaults.json`) estão no `.gitignore` e NUNCA devem ser comitados.

## 4. Integrações
- Manter o suporte ao **RClone** para streaming de nuvem.
- Utilizar o `GerenciadorRClone` (struct principal em `internal/core/gerenciador.go`) de forma unificada.
- Interface gráfica usa **Fyne v2** para widgets multiplataforma.
- System Tray usa `github.com/getlantern/systray`.

## 5. Arquitetura Go
- `internal/core/` — Lógica de negócio (sem dependência de GUI).
- `internal/gui/` — Interface gráfica Fyne.
- `internal/tray/` — System Tray.
- `internal/plataforma/` — Código específico de SO (build tags).
- `main.go` — Entry point e coordenação.
