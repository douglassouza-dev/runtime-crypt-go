# 019 — Migrar a interface de Fyne para Wails

- Estado: Aberta
- Risco: Alto — uso e estabilidade
- Onde: `internal/gui/*`, `internal/tray/tray.go`, `main.go`
- Depende de: 001, 002, 003, 004, 005, 006, 007, 008, 009, 010, 016, 017, 018 e ADR-0007 com status Aceita
- ADR: [0007 — Interface em Wails no lugar de Fyne](../adr/0007-interface-wails.md) (Proposta)

## Contexto

Douglas decidiu levar a interface para Wails. A migração vem depois das demandas de estabilidade e de segredos, para não migrar defeitos. Vem depois da 017, para que o frontend não carregue regra de negócio, e depois da 018, para que os quatro estados já existam no `core`.

O ADR-0007 deixa em aberto a escolha entre Wails v3 (beta, com bandeja nativa) e Wails v2 (estável, sem bandeja nativa). Esta demanda não começa sem essa escolha.

## Regra: paridade antes de melhoria

Nenhuma ação que existe hoje em Fyne pode sumir. Tudo que é novo fica fora, exceto os quatro estados (018), que já estarão no `core`.

### Ações que existem hoje (inventário do código)

| # | Tela | Ação | Código atual |
|---|---|---|---|
| 1 | Janela de cofres | Listar cofres com nome, provedor, cor do provedor e estado | `janela_principal.go:criarCardCofre` |
| 2 | Janela de cofres | Destrancar | `criarCardCofre` → `main.go:destravarCofre` |
| 3 | Janela de cofres | Trancar | `criarCardCofre` → `main.go:travarCofre` |
| 4 | Janela de cofres | "＋ Adicionar Cofre" | `construirInterface` → `acaoNovoCofre` |
| 5 | Janela de cofres | "📥 Importar Cofre Existente" | `construirInterface` → `acaoImportarCofre` |
| 6 | Janela de cofres | "⚙ Configurações" (VFS) | `construirInterface` → `DialogoConfigVfs` |
| 7 | Janela de cofres | "ℹ Sobre" | `construirInterface` → `CallbackSobre` |
| 8 | Janela de cofres | "✕ Sair" | `construirInterface` → `CallbackSair` |
| 9 | Janela de cofres | Fechar no X esconde a janela e o programa segue na bandeja | `NovaJanelaPrincipal` (`SetCloseIntercept`) |
| 10 | Janela de cofres | Mensagem de lista vazia | `construirInterface` (`lblVazio`) |
| 11 | Janela de cofres | Atualização a cada 3 s | `agendarAtualizacao` |
| 12 | Diálogo de senha | Digitar senha, Cancelar, Desbloquear | `dialogos.go:DialogoSenha` |
| 13 | Wizard novo cofre | Escolher Google Drive, OneDrive, Dropbox ou S3 | `wizards.go:DialogoNovoCofre` |
| 14 | Wizard novo cofre | Senha, confirmação, nome, Cancelar, Criar Cofre | `DialogoNovoCofre` |
| 15 | Wizard novo cofre | Abrir navegador para OAuth e avisar | `main.go:acaoNovoCofre` |
| 16 | Wizard conectar existente | Escolher os quatro provedores acima ou Pasta Local | `wizards.go:DialogoImportarCofre` |
| 17 | Wizard conectar existente | password, password2 opcional, nome, Cancelar, Avançar | `DialogoImportarCofre` |
| 18 | Seletor de pasta remota | Listar pastas, entrar, Voltar, Cancelar, Selecionar esta pasta, caminho atual, "Carregando" | `seletor_pasta.go:DialogoSeletorPastaRemota` |
| 19 | Configurações VFS | 13 campos com descrição, Restaurar Padrões, Cancelar, Salvar | `config_vfs.go:DialogoConfigVfs` |
| 20 | Mensagens | Info, erro e aviso com OK | `dialogos.go:DialogoMensagem` |
| 21 | Destrancar | Abrir o Explorer na unidade montada | `main.go:destravarCofre` |
| 22 | Bandeja | Ícone e tooltip | `tray.go:aoIniciar` |
| 23 | Bandeja | Abrir RuntimeCrypto | `aoIniciar` |
| 24 | Bandeja | Novo Cofre... | `aoIniciar` |
| 25 | Bandeja | Configuracoes > Auto-iniciar com o sistema (liga e desliga) | `aoIniciar` → `plataforma.*AutoIniciar` |
| 26 | Bandeja | Configuracoes > Configuracoes VFS... | `aoIniciar` |
| 27 | Bandeja | Configuracoes > Verificar WinFsp/FUSE | `aoIniciar` → `plataforma.VerificarWinfsp` |
| 28 | Bandeja | Sobre | `aoIniciar` |
| 29 | Bandeja | Sair | `aoIniciar` |
| 30 | Inicialização | Auto-montar cofres marcados | `main.go:autoMontarCofres` |

Se a 014 ou a 015 estiverem feitas antes, as ações novas delas também entram no inventário.

## O que muda

- `internal/gui` e `internal/tray` são **substituídos** por um frontend Wails e pelo código de bind. Não há camada que embrulhe Fyne.
- O frontend só chama métodos expostos pelo `core` (ou por um adaptador fino de bind) e mostra o resultado. Validação, ordem de passos e estado vêm do `core`.
- Fyne e `getlantern/systray` saem do `go.mod`.

## O que fica de fora

- Telas novas, redesenho de fluxo e itens novos no menu da bandeja.
- Mudanças no `core`, exceto o adaptador de bind.
- Instalador e assinatura do binário.

## Pronto quando

- [ ] O ADR-0007 está com status Aceita e diz a versão do Wails escolhida.
- [ ] As 30 ações da tabela existem no app Wails. O PR traz uma tabela com cada número e onde a ação está no frontend novo. Se a bandeja não existir na versão escolhida, a tabela mostra o substituto documentado no ADR-0007 para as ações 22 a 29.
- [ ] Janela de cofres, wizard novo cofre, wizard conectar existente e bandeja (ou o substituto documentado) existem.
- [ ] Cada cofre mostra uma das quatro frases da 018: `Trancado`, `Destrancando…`, `Destrancado • X:\` ou `Não destrancou: {motivo}`, as mesmas no card e na bandeja. `Destrancado` só aparece com o processo `rclone mount` vivo **e** a letra ou a pasta existente. Verificação: destrancar, matar o `rclone.exe` pelo Gerenciador de Tarefas, e em até 5 s o card mostra `Não destrancou: {motivo}`.
- [ ] Sem regra de negócio no frontend: o código do frontend não contém `rclone`, `_base`, `password2`, o número `8` como tamanho de senha, nem decide estado do cofre. Conferido com `rg` sobre a pasta do frontend.
- [ ] `go list -deps ./... | rg "fyne|getlantern/systray"` não encontra nada.
- [ ] No Windows sem WebView2: o programa mostra mensagem clara ou o instalador o instala. O caminho escolhido está no ADR-0007.
- [ ] Todos os itens de "Pronto quando" das demandas 001 a 018 continuam passando.
