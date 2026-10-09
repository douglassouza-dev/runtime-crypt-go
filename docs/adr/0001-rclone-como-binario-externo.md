# ADR-0001 — rclone como binário externo, chamado pela linha de comando

- Status: Aceita (registrada a posteriori)
- Data do registro: 2026-10-09
- Evidência: `internal/core/gerenciador.go:localizarRclone`, `testarRclone`; todas as chamadas `exec.Command(g.Executavel, ...)`

## Contexto

O programa precisa de rclone crypt, de OAuth com vários provedores e de `rclone mount`. O rclone é escrito em Go e pode ser importado como biblioteca (`github.com/rclone/rclone`). Também pode ser controlado por `rclone rc` (HTTP) ou chamado pela linha de comando.

## Decisão

O programa chama o binário `rclone` como processo filho, pela linha de comando. Procura primeiro ao lado do executável e depois no `PATH`. Não importa o rclone como biblioteca e não usa `rclone rc`.

## Consequências

- O binário do programa não carrega o rclone. O usuário instala ou atualiza o rclone em separado.
- A versão do rclone não é fixada nem conferida. Flags que mudarem entre versões quebram a montagem sem aviso claro (ver demanda 003).
- Cada operação é um processo: `config create`, `config dump`, `obscure`, `lsd`, `authorize`, `mount`. O ciclo de vida desses processos é responsabilidade do programa. Hoje ele é falho (demandas 001, 002, 008).
- Segredos passam por argumentos, ambiente e stdin (demandas 004 e 005).
- O `rclone.conf` é o do usuário. O programa compartilha o arquivo com qualquer outro uso do rclone na mesma conta (ver ADR-0004).
- O contrato está documentado em [06-api.md](../06-api.md#2-contrato-com-o-rclone).
