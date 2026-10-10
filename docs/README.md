# Documentação do RuntimeCrypto (runtime-crypt-go)

Leitura feita sobre o commit `c301180` da `master`. Tudo aqui descreve o que o código faz hoje. Onde o código não permite afirmar algo, o texto diz **desconhecido**.

| Arquivo | Conteúdo |
|---|---|
| [01-visao-e-requisitos.md](01-visao-e-requisitos.md) | Objetivo, escopo, requisitos e o que o README promete sem o código cumprir |
| [02-casos-de-uso.md](02-casos-de-uso.md) | Casos de uso, telas e estados do cofre |
| [03-regras-de-negocio.md](03-regras-de-negocio.md) | Regras que o código aplica, com arquivo e função |
| [04-arquitetura.md](04-arquitetura.md) | Contexto, contêineres e componentes (mermaid) |
| [05-modelo-de-dados.md](05-modelo-de-dados.md) | Structs, arquivos persistidos, rclone.conf e onde ficam os segredos |
| [06-api.md](06-api.md) | API interna dos pacotes Go e contrato com o rclone (comandos e flags) |
| [07-fluxogramas.md](07-fluxogramas.md) | Fluxogramas e diagrama de estados do cofre |
| [08-sequencia.md](08-sequencia.md) | Diagramas de sequência |
| [analise-de-melhorias.md](analise-de-melhorias.md) | Análise de melhorias, priorizada |
| [demanda/](demanda/index.md) | Demandas numeradas. Nenhum PR de código sem demanda |
| [adr/](adr/index.md) | Decisões de arquitetura |
| [testes/](testes/README.md) | Roteiro de testes em tela, para fazer com o programa aberto |

## Como a leitura foi verificada

- `go build ./...`, `go vet ./...` e `go test ./...` passam em Linux (Debian, toolchain Go 1.26.4 baixada pelo `GOTOOLCHAIN=auto`), depois de instalar `pkg-config`, `libgl1-mesa-dev`, `xorg-dev`, `libgtk-3-dev` e `libayatana-appindicator3-dev`. Sem esses pacotes o build falha em `go-gl`, `glfw` e `getlantern/systray`.
- `GOOS=windows CGO_ENABLED=0 go build ./internal/core ./internal/plataforma` passa. O binário completo para Windows não foi compilado.
- Testes existentes: `TestCacheSenhas` e `TestConfigVfs`. Nenhum teste cobre montagem, cofres, OAuth ou rclone.
- Três comportamentos do rclone foram reproduzidos com rclone v1.60.1 e um `rclone.conf` descartável. Eles aparecem nos documentos como "reproduzido".
- Nada foi montado de verdade. A caixa usada não tem FUSE nem WinFsp.
