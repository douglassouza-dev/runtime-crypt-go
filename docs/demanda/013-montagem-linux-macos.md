# 013 — Montagem impossível em Linux e macOS

- Estado: Aberta
- Risco: Médio — uso
- Onde: `internal/core/montagem.go:ObterLetrasDisponiveis`, `MontarUnidade`; `main.go:destravarCofre`
- Depende de: 001, 010

## Contexto

`ObterLetrasDisponiveis` devolve `nil` fora do Windows, e `MontarUnidade` então falha com "Nenhuma letra de unidade disponivel.". Mesmo com uma letra passada, o ponto seria `X:`, que não é caminho válido nesses SOs. `destravarCofre` abre `X:\` no Explorer.

O README e o CHANGELOG anunciam Linux e macOS.

## O que muda

- Em Linux e macOS, o ponto de montagem é uma pasta. O local padrão e se é configurável por cofre ficam definidos no PR.
- A pasta é criada antes de montar e removida depois de desmontar, se estiver vazia.
- `AbrirExplorador` recebe o ponto real.
- A desmontagem nesses SOs confere que o ponto deixou de ser montagem, por exemplo com `fusermount -u` ou `umount` se o processo não liberar. A escolha fica no PR.

## O que fica de fora

- Empacotamento para Linux e macOS.
- Testar macFUSE ou FUSE-T de verdade, se não houver máquina. Nesse caso, o PR diz que não testou.

## Pronto quando

- [ ] Em Linux com FUSE3: criar um cofre de Pasta Local (depende de 014) ou conectar um existente, destrancar, gravar um arquivo, trancar. `mount | grep rclone` fica vazio depois de trancar.
- [ ] Teste em `internal/core` com build tag `linux` cobre a escolha do ponto de montagem.
- [ ] README atualizado com o que foi e o que não foi testado em cada SO.
