# 023 — O seletor de pasta mexe na tela a partir de uma goroutine

- Estado: Aberta
- Risco: Médio · estabilidade
- Onde: `internal/gui/seletor_pasta.go:DialogoSeletorPastaRemota`
- Depende de: 021

## Contexto

A listagem roda numa goroutine e atualiza os widgets Fyne direto dela. Fyne pede que a tela seja atualizada pela thread da interface. Isso pode travar ou corromper a lista.

## O que muda

- A goroutine só busca a lista. A atualização dos widgets volta para a thread da interface (`fyne.Do` ou o equivalente da versão em uso).

## O que fica de fora

- Texto e layout do seletor (021 e 009).

## Pronto quando

- [ ] `rg -n "go func" internal/gui/seletor_pasta.go` mostra que nenhuma goroutine chama `SetText`, `Refresh`, `Objects =` ou parecido diretamente.
- [ ] `go test -race ./...` passa.
