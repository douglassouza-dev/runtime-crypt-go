# 006 — Criar cofre pode sobrescrever um remoto existente e deixa remotos órfãos

- Estado: Feita (a474807)
- Risco: Alto — dados
- Onde: `main.go:acaoNovoCofre`, `acaoImportarCofre`; `internal/core/cofres.go:Adicionar`; `internal/core/gerenciador.go:CriarRemoto`
- Depende de: —

## Contexto

O nome do cofre só é conferido em `Cofres.Adicionar`, que roda **depois** de `config create <nome>_base` e `config create <nome>`. O `rclone config create` em nome existente sai com código 0 e substitui a seção. Reproduzido com rclone v1.60.1: a senha do crypt foi trocada e os arquivos que existiam deixaram de ser listados.

Então, criar ou conectar um cofre com o nome de um cofre que já existe, ou de qualquer remoto já configurado no rclone do usuário, apaga a configuração anterior. O programa então mostra "Ja existe um cofre com o nome ..." depois do estrago.

Se `CriarCrypt` ou `Adicionar` falham, `<nome>_base` fica no `rclone.conf` sem cofre nenhum.

O nome não tem regra de caracteres. O comportamento do rclone com espaço, `:` ou `/` no nome é desconhecido.

## O que muda

- Antes de qualquer chamada ao rclone, o `core` confere que `<nome>` e `<nome>_base` não existem nem em `vaults.json` nem em `rclone listremotes`.
- O nome é validado no `core` contra as regras de nome de remoto do rclone.
- Se um passo falhar depois de criar algo, os remotos criados naquela tentativa são removidos.

## O que fica de fora

- Mudar a convenção `<nome>_base`.
- Tela para remover cofre.

## Pronto quando

- [ ] Teste em `internal/core` com `RCLONE_CONFIG` apontando para um arquivo temporário que já tem `[teste]`: criar cofre `teste` falha e o arquivo continua byte a byte igual.
- [ ] Teste: falha simulada em `CriarCrypt` deixa o `rclone.conf` sem `teste_base`.
- [ ] Teste: nomes com `:` ou `/` são recusados antes de chamar o rclone.
