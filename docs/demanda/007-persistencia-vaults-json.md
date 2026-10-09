# 007 — `vaults.json` corrompido é sobrescrito em silêncio

- Estado: Aberta
- Risco: Alto — dados
- Onde: `internal/core/cofres.go:carregar`, `salvar`, `Atualizar`; `internal/core/gerenciador.go:obterDiretorioApp`
- Depende de: —

## Contexto

`carregar` trata "arquivo não existe", "sem permissão" e "JSON inválido" do mesmo jeito: lista vazia, sem aviso. Na próxima gravação, `salvar` grava a lista nova por cima, e os cofres antigos somem da tela.

`salvar` usa `os.WriteFile` direto no arquivo final. Uma queda no meio deixa o arquivo truncado. `Atualizar` ignora o erro de `salvar`.

O arquivo fica no diretório do executável. Se o programa for instalado em pasta sem permissão de escrita, nada é gravado.

## O que muda

- `carregar` distingue "não existe" (lista vazia) de qualquer outro erro. No segundo caso, o programa não grava por cima, mostra o erro e preserva o arquivo com outro nome.
- `salvar` grava em arquivo temporário na mesma pasta e depois renomeia.
- `Atualizar` devolve o erro.

## O que fica de fora

- Mudar o local do arquivo (ver ADR-0004). Se mudar, vira outra demanda.
- Validar se cada cofre ainda tem remoto no `rclone.conf`.

## Pronto quando

- [ ] Teste: `vaults.json` com conteúdo `{quebrado` → `NovoGerenciadorCofres` devolve erro, e o arquivo continua igual depois de uma tentativa de `Adicionar`.
- [ ] Teste: `vaults.json` ausente → lista vazia, sem erro.
- [ ] Teste: `Atualizar` em diretório somente leitura devolve erro.
- [ ] `rg -n "os.WriteFile\(g.caminhoArquivo" internal/core` não encontra nada.
