# ADR-0004 — Onde fica a configuração

- Status: Aceita (registrada a posteriori)
- Data do registro: 2026-10-09
- Evidência: `internal/core/cofres.go:caminhoArquivo`; `internal/core/gerenciador.go:obterDiretorioApp`; `internal/core/vfs.go:NovoConfigVfs`; ausência de `--config` em todas as chamadas ao rclone

## Contexto

Há três tipos de configuração: a lista de cofres do programa, os remotos do rclone e os parâmetros VFS.

## Decisão

| Dado | Onde |
|---|---|
| Lista de cofres | `vaults.json` no diretório do executável |
| Remotos (`<nome>_base`, `<nome>`) | `rclone.conf` padrão do rclone para o usuário. O programa não passa `--config` |
| Parâmetros VFS | Só em memória, com padrões em `core/constantes.go` |

## Consequências

- Programa instalado em pasta sem permissão de escrita não consegue gravar `vaults.json`.
- Um programa portátil (pasta no pendrive) leva `vaults.json`, mas não leva os remotos, que ficam no perfil do usuário.
- O programa divide o `rclone.conf` com outros usos do rclone. Um nome de cofre igual a um remoto existente sobrescreve esse remoto (demanda 006).
- A configuração VFS se perde a cada reinício (demanda 012).
- `vaults.json` e `rclone.conf` podem divergir. Nada confere a consistência entre os dois.
