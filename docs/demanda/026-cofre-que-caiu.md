# 026 — Cofre que caiu: destrancar de novo e trancar sem perder envio

- Estado: Aberta
- Risco: Alto · dados, segredos
- Onde: `internal/core/trancar.go:Trancar`; `internal/core/montagem.go:DesmontarUnidade`, `EstadosPorRemoto`; `internal/gui/frases/frases.go:Botao`; `internal/gui/janela_principal.go:criarCardCofre`; `internal/gui/acoes.go:Cofre`
- Depende de: 018, 022, 025

## Contexto

Quando a montagem cai (`Caiu: …`, por exemplo com o rclone morto), o card oferece só `Tentar de novo`. Nesse estado `Trancar` usa `ObterLetraPorRemoto`, que só enxerga montagens no estado montado. O cofre que caiu parece desmontado: `Trancar` apaga a senha da sessão e volta sem erro.

Com o rclone morto antes do envio, os arquivos gravados ficam só no cache da VFS (`{cache}/vfs/{cofre}/…`, com `"Dirty": true` em `{cache}/vfsMeta/{cofre}/…`). Eles só sobem quando o cofre é montado de novo. Se o usuário tranca nesse momento, a senha some e nada avisa que há arquivos que não chegaram à nuvem.

Reproduzido no Linux (FUSE3, rclone 1.60): gravei um arquivo e matei o rclone com `kill -9` antes do write-back. O arquivo ficou em `vfsMeta` com `"Dirty": true` e não chegou à nuvem. Montei de novo e o rclone registrou `queuing for upload`; o arquivo subiu.

## O que muda

- No estado `Caiu: …`, o botão principal é `Destrancar de novo` (no lugar de `Tentar de novo`, que fica para `Não destrancou: …`). Ele monta de novo com a senha da sessão, e o rclone retoma os envios que ficaram no cache.
- `Trancar` continua disponível no card que caiu e segue a 025: a senha só sai quando o card chega a `Trancado`.
- Se ainda há arquivos para subir, `Trancar` não tranca. O card mostra `Não trancou: {n} arquivos ainda não subiram`, ou `Não trancou: 1 arquivo ainda não subiu` com um só, e a senha fica.
- Com o rclone morto não há como perguntar a ele (a API `rc` da 025 morreu junto). A contagem vem do cache da VFS no disco: arquivos de `{cache}/vfsMeta/{cofre}/` com `"Dirty": true`, os mesmos que o rclone lê ao subir para reenviar. A pasta `{cache}` segue a regra do rclone: `--cache-dir` da configuração VFS, senão `RCLONE_CACHE_DIR`, senão a pasta de cache do usuário + `rclone`.
- Regra de singular: toda frase com contagem de arquivos usa `1 arquivo` / `{n} arquivos` (`core.Arquivos`), como na 025.

## O que fica de fora

- Enviar o cache sem montar o cofre (por exemplo, rodar o rclone só para enviar).
- Cache de sessões anteriores, de antes de o app abrir.
- Montagens que nunca subiram (`Não destrancou: …`): continuam com `Tentar de novo`.

## Pronto quando

- [ ] Teste: cofre que caiu com 2 arquivos sujos no `vfsMeta` → `Trancar` devolve `2 arquivos ainda não subiram`, a senha fica e o estado continua `Caiu`. Com 1 → `1 arquivo ainda não subiu`.
- [ ] Teste: cofre que caiu sem arquivos sujos → `Trancar` desmonta, o estado vira desmontado e só então a senha sai.
- [ ] Teste: `Trancar` não apaga a senha de um cofre que caiu e continua no mapa de montagens (o defeito antigo).
- [ ] Teste: o card em `Caiu: …` tem `Destrancar de novo` como botão principal e também `Trancar`; `Não destrancou: …` continua com `Tentar de novo`.
- [ ] Teste: a pasta de cache vem de `--cache-dir`, senão `RCLONE_CACHE_DIR`, senão a pasta de cache do usuário.
- [ ] Linux com FUSE3: gravar, matar o rclone antes do envio, ver `Caiu: o rclone parou`. `Trancar` recusa com a contagem; `Destrancar de novo` monta, e o arquivo cifrado chega à nuvem.
- [ ] Na tela (Windows): o mesmo roteiro, matando o `rclone.exe` pelo Gerenciador de Tarefas.
