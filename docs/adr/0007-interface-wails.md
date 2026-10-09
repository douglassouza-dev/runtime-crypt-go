# ADR-0007 — Interface em Wails no lugar de Fyne

- Status: Proposta
- Data: 2026-10-09
- Substitui, se aceita: [ADR-0002](0002-gui-fyne-e-systray.md)
- Demanda: [019](../demanda/019-migracao-wails.md)

## Contexto

Douglas decidiu levar a interface para [Wails](https://wails.io/pt/): backend em Go e frontend web, renderizado pelo WebView do SO. Os requisitos que este ADR precisa respeitar:

- **"Altamente estável"**, pedido por Douglas.
- **Bandeja do sistema:** o programa roda em segundo plano, e fechar a janela só a esconde (`janela_principal.go:NovaJanelaPrincipal`). O menu da bandeja tem 7 ações ([demanda 019](../demanda/019-migracao-wails.md), itens 22 a 29).
- **`internal/core` sem import de GUI**, para o Wails fazer bind direto nele. Hoje o `core` já não importa Fyne. A orquestração ainda está em `main.go` e algumas regras estão em `internal/gui` ([demanda 017](../demanda/017-extrair-logica-para-core.md)).

### Estado do Wails, conferido em 2026-10-09

| Linha | Situação | Bandeja | Fonte |
|---|---|---|---|
| v2 | Estável. Última release no GitHub: v2.14.0 (2026-08-10). A documentação de wails.io já mostra v2.15.0 | **Não tem bandeja nativa.** O mantenedor escreveu que "Systray will not be supported in v2" (wailsapp/wails, discussão #4514). A issue #1521 foi fechada como "intended for v3". Existe a gambiarra de rodar `getlantern/systray` em outra goroutine. No macOS ela tem conflito de linkagem relatado | wails.io/docs/gettingstarted/installation; github.com/wailsapp/wails/issues/1521; discussão #4514 |
| v3 | **Beta.** v3.0.0-beta.28 publicada em 2026-10-05, marcada como pre-release. O post do beta (2026-08-02) diz: "This is a beta release, not the final 3.0 release... Wails v2 remains the current stable release" | **Tem bandeja nativa** (`app.SystemTray.New()`, menus, janela anexada). O changelog do beta.28 ainda traz correções de bandeja no Linux e no Windows: clique direito no Linux, crashes de `SetMenu` e vazamento de GDI no Windows | v3.wails.io/status; v3.wails.io/changelog; v3.wails.io/features/menus/systray; GitHub releases |

Dependências de execução:

- **Windows:** runtime **WebView2** nas duas linhas. Algumas instalações do Windows já têm, outras não (`wails doctor` confere).
- **Linux:** v2 usa GTK3 + WebKit2GTK. v3 usa GTK4 + WebKitGTK 6.0 por padrão, com GTK3 como opção legada até a v3.0.x, removida na v3.1.
- **Go:** v3 pede Go 1.25+. O `go.mod` atual declara 1.26.4.

## Opções

| Opção | Bandeja | Estabilidade da base | Custo e risco |
|---|---|---|---|
| A. Wails v3 beta | Nativa | Beta. A API pode mudar antes da 3.0.0, e há correções recentes justamente em bandeja | Acompanhar betas e fixar a versão exata. Risco de regressão em bandeja, que é tela central do produto |
| B. Wails v2 estável, sem bandeja | Nenhuma. É preciso um substituto documentado, por exemplo a janela minimizada na barra de tarefas | Estável | Muda o uso: o programa deixa de "morar" na bandeja. Fechar a janela precisa de outro comportamento. Viola a paridade da demanda 019, a não ser que Douglas aceite o substituto |
| C. Wails v2 + `getlantern/systray` à parte | Gambiarra, sem suporte do Wails | v2 estável, bandeja sem suporte | Dois laços de GUI no processo, como hoje com Fyne. Conflito relatado no macOS |
| D. Manter Fyne e só corrigir | A atual | Fyne estável | Não atende à decisão de Douglas. Fica aqui como referência de custo |

O pedido "altamente estável" puxa para B ou D. A paridade de telas, com a bandeja incluída, puxa para A. Nenhuma opção cumpre as duas coisas hoje.

## Decisão

**Pendente: Douglas escolhe entre A, B e C.** Este ADR não escolhe por ele.

O que vale para qualquer opção aceita:

1. `internal/core` não importa nenhum pacote de GUI. Conferir com `go list -deps ./internal/core`.
2. O código Fyne de `internal/gui` e `internal/tray` é **substituído**, não embrulhado. Nenhum adaptador mantém widgets Fyne vivos.
3. O frontend não tem regra de negócio: não valida senha, não monta nome de remoto e não decide estado de cofre. Só chama o `core` (diretamente ou por um adaptador fino de bind) e mostra o resultado.
4. A migração só começa depois das demandas 001 a 010, 016, 017 e 018.
5. A versão do Wails fica fixada no `go.mod` com versão exata. Na opção A, a troca de beta só entra com a demanda 019 passando de novo.
6. A falta do WebView2 no Windows é tratada: o instalador inclui o bootstrapper ou o programa mostra mensagem clara. A escolha fica registrada aqui ao aceitar.

## Consequências

- Some a dependência de OpenGL, X11 e Fyne no build. Entra Node/npm no build do frontend e WebView2/WebKitGTK na execução.
- Os quatro estados do cofre (demanda 018) chegam ao frontend como dados do `core`. Não são calculados na tela.
- Se a opção for A: risco aceito de mudar de API até a v3.0.0. Recomenda-se rever este ADR quando a v3.0.0 for publicada.
- Se a opção for B: a demanda 019 precisa registrar o substituto da bandeja para as ações 22 a 29, e o README perde "roda em segundo plano na bandeja".
