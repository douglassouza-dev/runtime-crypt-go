# 028 — Sair com cofre que caiu e arquivos que não subiram

- Estado: Aberta
- Risco: Alto · dados
- Onde: `main.go:sair`; `internal/core/gerenciador.go:Encerrar`; `internal/gui/acoes.go`
- Depende de: 022, 025, 026

## Contexto

Sair do app (bandeja ou menu) chama `Encerrar`, que desmonta tudo. Um cofre que caiu (026) pode ter arquivos no cache da VFS que ainda não subiram. Sair não avisa nada. Eles só sobem quando o cofre for destrancado de novo nesta máquina, e o usuário não fica sabendo.

## O que muda

- Antes de sair, o app conta os arquivos que não subiram em cada cofre que caiu com o rclone parado, usando a mesma contagem da 026 (`vfsMeta` com `"Dirty": true`). Se algum tiver arquivo pendente, abre um diálogo:
  - cópia aprovada pela UI: título `Arquivos que ainda não subiram`; um diálogo só para todos os cofres, com uma linha por cofre (`{n} arquivos de {cofre}`, no singular `1 arquivo de {cofre}`) e, no fim, `Eles sobem quando você destrancar de novo.`;
  - com um cofre, o mesmo formato (uma linha e a frase do fim);
  - singular (aprovado pela UI): só quando há um arquivo ao todo (um cofre com 1 arquivo) a frase do fim é `Ele sobe quando você destrancar de novo.`, pela regra de singular da 026. Com mais de um cofre há sempre mais de um arquivo, então `Eles` vale.
- Botões:
  - `Enviar agora` (principal, aprovado): vale para todos os cofres da lista; destranca de novo cada um, espera o envio (025), tranca e sai. Se algum falhar, o diálogo mostra a frase da 022/025 (`Não destrancou: …` ou `Não trancou: …`, com o nome do cofre) e o app não sai.
  - `Sair` (aprovado): sai e deixa o cache como está.
- O diálogo não tem botão de fechar: as únicas saídas são os dois botões.
- O texto nunca diz que os arquivos serão perdidos.

## O que fica de fora

- Cofres montados e saudáveis: o `Encerrar` de hoje já tranca esses, com a espera da 025.
- Avisar no início do app sobre cache de sessões anteriores.

## Pronto quando

- [ ] Teste: com um cofre caído e 2 arquivos sujos, a lista de pendências diz `{cofre}: 2`; sem arquivos sujos, a lista é vazia e o app sai sem diálogo.
- [ ] Teste das frases: título, singular, plural e vários cofres.
- [ ] Teste: `Enviar agora` destranca, tranca e então sai. Se destrancar ou trancar falhar, mostra a frase com o motivo e não sai.
- [ ] Teste: `Sair` sai sem mexer no `vfsMeta`.
- [ ] Linux com FUSE3: gravar, matar o rclone, pedir para sair. O diálogo mostra `Arquivos que ainda não subiram`, `1 arquivo de {cofre}` e `Ele sobe quando você destrancar de novo.`; `Enviar agora` envia, o arquivo cifrado chega à nuvem e o app sai.
- [ ] Na tela (Windows): o mesmo roteiro.
