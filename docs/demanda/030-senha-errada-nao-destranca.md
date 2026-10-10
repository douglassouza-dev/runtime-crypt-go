# 030 — Senha errada não destranca

- Estado: Aberta
- Risco: Alto · segredos
- Onde: `internal/core/casos_de_uso.go:Destrancar`; `internal/gui/dialogos.go:DialogoSenha`; `internal/gui/acoes.go:destrancar`
- Depende de: 027
- Relação com a 004: parte da 004 que dá para fazer sem o ADR-0006. A 004 continua aberta.

## Contexto

`Destrancar` pega a senha digitada, guarda na sessão e chama `MontarUnidade`, que passa a senha em `RCLONE_CONFIG_PASS`. Essa variável só serve para um `rclone.conf` cifrado, e o programa nunca cifra o arquivo (demanda 004, ADR-0005). O `rclone mount` usa a senha do crypt que está gravada, ofuscada, no `rclone.conf`. Ninguém compara a senha digitada com ela, então qualquer senha não vazia destranca.

A 004 resolve isso junto com onde a senha fica guardada, e depende do ADR-0006, que ainda não foi decidido. Esta demanda só faz a senha errada parar de destrancar, sem mudar onde a senha fica.

Por que um número novo e não um pedaço da 004: o "Pronto quando" da 004 pede o ADR-0006 aceito e que `rclone reveal` não devolva a senha a partir do `rclone.conf`. Nada disso muda aqui. Juntar as duas deixaria a 004 meio feita sem dar para conferir qual metade.

## O que muda

- Novo `core.ConferirSenha(nome, senha)`: lê o remoto crypt `nome` com `rclone config dump`, revela o `password` ofuscado e compara com a senha digitada em tempo constante. Volta:
  - `nil` se a senha confere;
  - `core.ErrSenhaErrada` se não confere ou está vazia;
  - `*core.ErroConferirSenha` se o remoto não existe, não tem `password` ou o valor não se revela. O cofre não destranca (falha fechada).
- O valor ofuscado é revelado em Go (`core.revelarObscuro`), com o mesmo algoritmo do `rclone reveal` (AES-CTR com a chave pública do rclone, `fs/config/obscure`). O motivo para não chamar `rclone reveal`:
  - o valor ofuscado iria nos argumentos do processo, que no Linux outros usuários da máquina leem em `/proc` (demanda 005);
  - `rclone reveal` também quebra com valor que começa com `-` (demanda 029).
  - O custo é depender do formato do `obscure` do rclone. Um teste confere o resultado contra valores gerados pelo rclone real v1.60.1 e v1.75.2.
- `Destrancar` confere a senha digitada antes de tudo. Com `ErrSenhaErrada`, ela não entra na sessão, nenhum `rclone mount` é iniciado, nenhuma falha de montagem é gravada e o cofre continua `Trancado`. A senha que já está na sessão não é conferida de novo, porque veio de uma conferência que passou ou da criação/conexão do cofre.
- Na tela (aprovado pela UI):
  - a frase `Senha errada.` aparece embaixo do campo de senha, no próprio diálogo;
  - o diálogo continua aberto, com o campo focado e o texto selecionado;
  - o cofre continua `Trancado`;
  - não há bloqueio nem limite de tentativas.
  - Vale para o diálogo de `Destrancar` e para o de `Enviar agora` (028).
- Se a conferência não for possível (`ErroConferirSenha`), o diálogo fecha e o erro de destrancar diz `não deu para conferir a senha` (proposta, aguarda UI). O texto do rclone vai só para o log, como na 027.

## O que fica de fora (fica para a 004 e o ADR-0006)

- Cifrar o `rclone.conf` ou tirar dele `password` e `password2`. Quem copia o `rclone.conf` continua tendo a senha, com `rclone reveal`.
- O `rclone mount` continua lendo a senha do `rclone.conf`, não a digitada. A conferência é do programa: o `rclone` de linha de comando abre o cofre sem perguntar nada.
- `RCLONE_CONFIG_PASS` continua sendo passado ao mount como hoje.
- Corrigir o README ("senhas ficam apenas em RAM").
- `password2` (o sal) não é conferido: o diálogo pede uma senha só, e ela é comparada com `password`.
- `rclone.conf` já cifrado pelo usuário: o programa não lê esse arquivo hoje, e isso não muda.
- Bloqueio ou espera depois de tentativas erradas.

## Pronto quando

- [ ] Teste com o rclone falso (`NovoGerenciadorEm`): senha certa destranca (há `mount` em `chamadas.log`); senha errada volta `ErrSenhaErrada`, nenhum `mount` foi chamado, nenhum processo ficou vivo, o estado é `desmontado` sem motivo e a senha não está na sessão; senha vazia volta `ErrCancelado` sem chamar o rclone.
- [ ] Teste: remoto inexistente ou sem `password` volta `*ErroConferirSenha` e não monta.
- [ ] Teste: `revelarObscuro` devolve a senha certa para valores gerados pelo rclone real (incluindo um que começa com `-`).
- [ ] Teste da tela (driver de teste do Fyne): senha errada mostra `Senha errada.` embaixo do campo, o diálogo continua aberto, o campo tem o foco e o texto está todo selecionado; a senha certa fecha o diálogo e devolve a senha.
- [ ] No Windows: destrancar com senha errada mostra `Senha errada.` e não monta letra nenhuma; com a senha certa monta e lista os arquivos.
