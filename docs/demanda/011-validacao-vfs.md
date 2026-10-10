# 011 — Configuração VFS aceita qualquer texto

- Estado: Feita (579a81f)
- Risco: Médio — estabilidade
- Onde: `internal/core/vfs.go:Atualizar`; `internal/gui/config_vfs.go:DialogoConfigVfs`
- Depende de: 003

## Contexto

`Atualizar` aceita qualquer texto não vazio nas 13 chaves. Um valor inválido (por exemplo, `vfs_cache_mode = tudo`) só aparece na hora de montar. Hoje isso dá o aviso de 45 s sem motivo (ver 003).

Não dá para limpar um valor: texto vazio mantém o anterior. "Restaurar Padrões" aplica na hora, mesmo que o usuário cancele depois.

## O que muda

- O `core` valida cada chave: lista fechada para `vfs_cache_mode` (`off`, `minimal`, `writes`, `full`), formato de tamanho para os campos de tamanho, formato de duração para os de tempo, e pasta existente e gravável para `cache_dir`.
- `Atualizar` devolve erro por chave e não aplica nada se houver erro.
- `cache_dir` vazio passa a significar "usar o padrão do rclone".
- "Restaurar Padrões" só aplica ao clicar em "Salvar".

## O que fica de fora

- Gravar em disco (012).
- Configuração VFS por cofre.

## Pronto quando

- [ ] Teste tabelado em `vfs_test.go` com pelo menos um valor válido e um inválido para cada uma das 13 chaves.
- [ ] Na tela: digitar `tudo` em modo de cache e clicar em "Salvar" mostra erro e não muda o valor.
- [ ] Na tela: "Restaurar Padrões" seguido de "Cancelar" mantém os valores anteriores.
