# 028 — Sair com cofre que caiu e arquivos que não subiram

- Estado: Aberta
- Risco: Alto · dados
- Onde: `main.go:sair`; `internal/core/gerenciador.go:Encerrar`; `internal/gui/acoes.go`
- Depende de: 022, 025, 026

## Contexto

Sair do app (bandeja ou menu) chama `Encerrar`, que desmonta tudo. Um cofre que caiu (026) pode ter arquivos no cache da VFS que ainda não subiram. Sair não avisa nada. Eles só sobem quando o cofre for destrancado de novo nesta máquina, e o usuário não fica sabendo.

## O que muda

- Antes de sair, o app conta os arquivos que não subiram em cada cofre que caiu com o rclone parado, usando a mesma contagem da 026 (`vfsMeta` com `"Dirty": true`). Se algum tiver arquivo pendente, abre um diálogo:
  - com um cofre: `{n} arquivos de {cofre} ainda não subiram. Eles sobem quando você destrancar de novo.`;
  - no singular: `1 arquivo de {cofre} ainda não subiu. Ele sobe quando você destrancar de novo.`;
  - com vários cofres (proposta): um diálogo só, com uma linha por cofre (`{n} arquivos de {cofre} ainda não subiram.` ou `1 arquivo de {cofre} ainda não subiu.`) e, no fim, `Eles sobem quando você destrancar de novo.` Com mais de um cofre há sempre mais de um arquivo, então o plural `Eles` vale.
- Botões:
  - `Enviar agora` (principal): destranca de novo cada cofre afetado, espera o envio (025), tranca e sai. Se algum falhar, o diálogo mostra a frase da 022/025 (`Não destrancou: …` ou `Não trancou: …`, com o nome do cofre) e o app não sai.
  - `Sair`: sai e deixa o cache como está.
- O diálogo não tem botão de fechar: as únicas saídas são os dois botões.
- O texto nunca diz que os arquivos serão perdidos.

## O que fica de fora

- Cofres montados e saudáveis: o `Encerrar` de hoje já tranca esses, com a espera da 025.
- Avisar no início do app sobre cache de sessões anteriores.

## Pronto quando

- [ ] Teste: com um cofre caído e 2 arquivos sujos, a lista de pendências diz `{cofre}: 2`; sem arquivos sujos, a lista é vazia e o app sai sem diálogo.
- [ ] Teste das frases: singular, plural e vários cofres.
- [ ] Teste: `Enviar agora` destranca, tranca e então sai. Se destrancar ou trancar falhar, mostra a frase com o motivo e não sai.
- [ ] Teste: `Sair` sai sem mexer no `vfsMeta`.
- [ ] Linux com FUSE3: gravar, matar o rclone, pedir para sair. O diálogo mostra `1 arquivo de {cofre} ainda não subiu…`; `Enviar agora` envia, o arquivo cifrado chega à nuvem e o app sai.
- [ ] Na tela (Windows): o mesmo roteiro.
