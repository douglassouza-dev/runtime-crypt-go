# 012 — Configuração VFS volta ao padrão a cada reinício

- Estado: Feita (63d23f7)
- Risco: Baixo — uso
- Onde: `internal/core/vfs.go:NovoConfigVfs`
- Depende de: 007, 011

## Contexto

`ConfigVfs` vive só em memória. A mensagem "Configurações VFS salvas com sucesso." dá a entender que foi para o disco.

## O que muda

- A configuração VFS é gravada em arquivo, com a mesma escrita atômica da 007, no mesmo diretório do `vaults.json`.
- Na leitura, valores inválidos são descartados com aviso, usando a validação da 011.

## O que fica de fora

- Configuração por cofre.
- Mudar o diretório de dados.

## Pronto quando

- [ ] Teste: `Atualizar` e depois um novo `NovoConfigVfs` no mesmo diretório devolvem os valores salvos.
- [ ] Teste: arquivo com valor inválido → valor padrão naquela chave e aviso.
- [ ] Na tela: mudar `vfs_cache_max_age`, fechar o programa, abrir de novo. O valor continua.
