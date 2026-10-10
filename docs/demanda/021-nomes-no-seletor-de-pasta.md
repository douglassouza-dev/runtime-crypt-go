# 021 — O seletor de pasta mostra "-1 alfa" no lugar de "alfa"

- Estado: Feita (c431ba8)
- Risco: Médio — uso, dados
- Onde: `internal/core/gerenciador.go:ListarDiretoriosRemoto`; `internal/gui/seletor_pasta.go:DialogoSeletorPastaRemota`
- Depende de: 016

## Contexto

`ListarDiretoriosRemoto` roda `rclone lsd` e corta cada linha com `strings.SplitN(linha, " ", 5)`, supondo um espaço entre colunas. O `rclone lsd` de verdade alinha as colunas com vários espaços (largura fixa), por exemplo:

```
          -1 2024-01-01 00:00:00        -1 alfa
```

Depois do `TrimSpace`, os espaços extras viram campos vazios e o quinto campo sai como `-1 alfa`. O seletor de "Conectar cofre existente" mostra esse texto, e ao clicar entra no caminho `-1 alfa`, que não existe. Se o usuário escolhe essa pasta, `-1 alfa` vai para `remoto_base` em `vaults.json` e para o `remote` do crypt.

Nomes com espaço no começo, com vários espaços seguidos ou que começam com número ou hífen também podem sair errados com qualquer leitura por colunas de texto.

O teste `TestListarDiretoriosRemotoFormatoRealDoRclone` em `internal/core/gerenciador_test.go` (criado na demanda 016) está com `t.Skip` por causa disso.

## O que muda

- A lista de pastas vem com o nome exato de cada pasta no remoto.
- Opção preferida, a confirmar no PR: trocar `lsd` por uma saída feita para máquina, `rclone lsjson --dirs-only` (campo `Name`) ou `rclone lsf --dirs-only` (uma pasta por linha, com `/` no fim). As duas existem no rclone desde antes da v1.50; o PR confere na versão usada e escreve qual escolheu e por quê. Se ficar com `lsd`, o nome é o que vem depois da quarta coluna, sem cortar espaços do próprio nome.
- O seletor mostra uma linha dizendo que não há subpastas quando a pasta está vazia.

## O que fica de fora

- Distinguir "pasta vazia" de "erro ao listar" (demanda 009).
- Tempo limite do `lsd` (demanda 008).
- Redesenho do seletor ou migração para Wails (demanda 019).
- Remotos do tipo `local` no assistente de conectar (demanda 014).

## Pronto quando

- [ ] `TestListarDiretoriosRemotoFormatoRealDoRclone` sem `t.Skip` e passando.
- [ ] Teste em `internal/core` com o rclone falso devolvendo pastas `alfa`, `minha pasta`, `ação`, `2024 fotos`, `-rascunho` e `  dois  espaços`: `ListarDiretoriosRemoto` devolve exatamente esses nomes, sem tirar nem pôr caractere.
- [ ] Na tela (revisão de UI): o seletor mostra o nome da pasta exatamente como está no remoto, inclusive com espaço, acento, ou começando por número ou hífen.
- [ ] Na tela: ao escolher uma pasta, o caminho mostrado no topo do seletor e o valor salvo no cofre (`remoto_base` em `vaults.json`) são esse mesmo nome.
- [ ] Na tela: uma pasta sem subpastas mostra uma linha só, exatamente `Nenhuma subpasta aqui.`, sem botão, e não uma lista em branco. A pasta continua escolhível pelo caminho do topo.
