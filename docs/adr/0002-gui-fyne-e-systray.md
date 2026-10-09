# ADR-0002 — Interface em Fyne v2 e bandeja com getlantern/systray

- Status: Aceita (registrada a posteriori). Pode ser substituída pelo [ADR-0007](0007-interface-wails.md), que está como Proposta
- Data do registro: 2026-10-09
- Evidência: `go.mod`; `main.go:main`; `internal/gui/*`; `internal/tray/tray.go`

## Contexto

O programa precisa de uma janela com lista de cofres e wizards, e de um ícone na bandeja para rodar em segundo plano, em Windows, Linux e macOS.

## Decisão

A janela usa Fyne v2 (v2.7.4). A bandeja usa `github.com/getlantern/systray` v1.2.2, em uma goroutine separada. O laço do Fyne fica na goroutine principal. A comunicação bandeja → aplicação passa por um canal `chan tray.AcaoTray`.

## Consequências

- O build precisa de CGO e de bibliotecas de sistema: OpenGL, X11 e GTK/AppIndicator no Linux. Sem elas, `go build` falha (verificado nesta leitura).
- Dois laços de eventos de GUI convivem no processo. O `getlantern/systray` pede a thread principal no macOS. O comportamento real em cada SO é desconhecido.
- O Fyne já traz bandeja própria (`fyne.io/systray`, que aparece como dependência indireta). O programa carrega duas bibliotecas de bandeja.
- O Fyne 2.6+ exige `fyne.Do` para mexer em widgets fora da goroutine principal. O código não usa `fyne.Do` (ver [04-arquitetura.md](../04-arquitetura.md#concorrência)).
- Diálogos bloqueiam a goroutine chamadora. Por isso toda ação em `main.go` roda em `go ...`.
