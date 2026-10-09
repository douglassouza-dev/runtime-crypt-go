# 014 — S3 e Pasta Local não funcionam nos wizards

- Estado: Aberta
- Risco: Médio — uso
- Onde: `main.go:acaoNovoCofre`, `acaoImportarCofre`; `internal/gui/wizards.go:DialogoNovoCofre`, `DialogoImportarCofre`; `internal/core/provedores.go:Provedores`
- Depende de: 005, 006

## Contexto

- **S3:** `Provedores` define os campos `access_key_id`, `secret_access_key`, `region` e `endpoint`, mas nenhum wizard os mostra. `acaoNovoCofre` só cria remoto base para OAuth e para `local_path`. O crypt de S3 aponta para `<nome>_base:`, que não existe.
- **Pasta Local:** "Novo cofre" esconde provedores `LocalOnly`, então o ramo `local_path` de `acaoNovoCofre` está morto. "Conectar existente" para em um `TODO` com a mensagem "Selecione a pasta no explorador.".

## O que muda

- Os wizards mostram os campos de `Provedor.Campos` quando o provedor não usa OAuth.
- S3 cria `<nome>_base` com tipo `s3`, provedor e credenciais, sem segredo em argumento (005).
- Pasta Local escolhe uma pasta do disco e cria `<nome>_base` do tipo `local`, ou aponta o crypt direto para a pasta. A escolha fica no PR.
- Pasta Local aparece também em "Novo cofre".

## O que fica de fora

- Outros provedores.
- Testar contra AWS de verdade. Basta MinIO local.

## Pronto quando

- [ ] Com MinIO local: criar cofre S3, destrancar, gravar arquivo. O objeto cifrado aparece no bucket.
- [ ] Criar cofre Pasta Local em uma pasta temporária: destrancar, gravar arquivo. A pasta passa a ter um arquivo com nome cifrado.
- [ ] `rg -n "TODO" main.go` não encontra o `TODO` de Pasta Local.
