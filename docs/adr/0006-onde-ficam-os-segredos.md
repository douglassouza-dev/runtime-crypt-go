# ADR-0006 — Onde ficam os segredos dos cofres

- Status: Proposta
- Data: 2026-10-09
- Substitui, se aceita: [ADR-0005](0005-senha-em-memoria-e-obscure.md)
- Demanda: [004](../demanda/004-senha-do-cofre-nao-protege.md)

## Contexto

Ver ADR-0005 e [05-modelo-de-dados.md](../05-modelo-de-dados.md#onde-ficam-os-segredos). Hoje a senha do cofre fica reversível no `rclone.conf`, e a senha digitada ao destrancar não é conferida. O requisito é que só quem sabe a senha abra o cofre, e que copiar o `rclone.conf` não entregue a senha.

## Opções

| Opção | Como | A favor | Contra |
|---|---|---|---|
| A. Não gravar `password` e `password2` | O crypt é criado sem senha no arquivo. Ao montar, a senha vai por variável de ambiente do remoto (`RCLONE_CONFIG_<NOME>_PASSWORD`, ofuscada em memória) | A senha digitada é a que abre. Nada reversível em disco | Segredo no ambiente do processo enquanto estiver montado. Operações sem montagem também precisam da senha. É preciso validar o comportamento com cada versão do rclone |
| B. Cifrar o `rclone.conf` inteiro | `rclone config encryption set`, e a senha do cofre vira a senha do arquivo | Usa um recurso pronto do rclone | Uma senha para todos os cofres, inclusive remotos que não são do programa. Mexe no arquivo do usuário. Todo comando passa a precisar da senha |
| C. `rclone.conf` próprio e cifrado | O programa passa `--config <arquivo do app>` e cifra esse arquivo | Não toca no rclone do usuário. Uma senha mestra para o programa | Uma senha para todos os cofres. Precisa migrar cofres existentes |
| D. Cofre de senhas do SO | Windows Credential Manager, Keychain ou Secret Service guardam a senha. O `rclone.conf` não guarda | Permite auto-montar sem digitar (demanda 015) | Quem tem a sessão do SO abre o cofre. Três implementações. A senha continua precisando chegar ao rclone por ambiente |

Não há dado de uso para decidir pelo número de cofres por usuário. Se a maioria tiver um cofre só, B e C perdem a desvantagem de "uma senha para todos".

## Decisão

**Pendente.** Douglas escolhe. A demanda 004 não começa antes.

## Consequências esperadas, por opção

- A ou D: o README pode dizer, com verdade, que a senha não fica no disco. Para D, só se a opção "lembrar" ficar desligada.
- B ou C: o README passa a dizer "senha mestra do programa", não "senha do cofre".
- Em qualquer opção: testes de senha certa e senha errada na demanda 004.
