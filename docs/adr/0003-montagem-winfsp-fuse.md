# ADR-0003 — Montagem por `rclone mount` sobre WinFsp, FUSE ou macFUSE

- Status: Aceita (registrada a posteriori)
- Data do registro: 2026-10-09
- Evidência: `internal/core/montagem.go:MontarUnidade`; `internal/plataforma/*:VerificarWinfsp`

## Contexto

O usuário quer usar o cofre como uma unidade comum, com arquivos decifrados na hora.

## Decisão

Cada cofre destrancado é um processo `rclone mount <nome>: <LETRA>:`. No Windows, o driver é o WinFsp, com `--network-mode`. Em Linux e macOS, a intenção registrada no README é FUSE e macFUSE. O programa só verifica se o driver existe (`VerificarWinfsp`). Não instala nada.

## Consequências

- O WinFsp ou o FUSE são dependências que o usuário instala.
- Um processo por cofre. Se o processo morre, a unidade some. O programa precisa vigiar o processo e o ponto de montagem (demandas 001 e 010).
- O ponto de montagem é sempre uma letra. Em Linux e macOS isso não funciona (demanda 013).
- O cache VFS (`--vfs-cache-mode full` por padrão) guarda conteúdo decifrado em disco local.
