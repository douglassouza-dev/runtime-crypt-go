# Roteiro de testes em tela

Leia antes o [README](README.md): ele diz como compilar, preparar a pasta de teste e mandar uma falha.

- Commit testado: `________`
- Data: `________`
- Sistema: `________` (Windows 10/11, versão do WinFsp, versão do rclone)

Os textos entre crases são os da `master`. A letra da unidade pode ser outra que não `V:`. Onde o roteiro diz `V:\`, use a letra que aparecer.

Nomes usados no roteiro:

| Nome | O que é |
|---|---|
| `teste-a` | Cofre criado no T-022, na raiz do Google Drive de teste. Senha: `senha-teste-a` |
| `teste-b` | Cofre importado no T-032, na pasta `-rascunho`. Senha: `senha-teste-b` |
| `abrir.ps1` | `powershell -ExecutionPolicy Bypass -File .\abrir.ps1`, rodado dentro de `docs\testes\scripts` |

Para fechar o app, use sempre a bandeja → `Sair` ou o botão `✕  Sair` da janela. O `X` da janela só esconde a janela.

---

## 0. Antes de começar

- [ ] **P-1.** Compile a `master` (README, "Compilar a master"). Anote o commit no topo.
- [ ] **P-2.** Na conta de teste do Google Drive, crie as pastas da raiz: `alfa`, `minha pasta`, `ação`, `2024 fotos`, `-rascunho` e `vazia`.
- [ ] **P-3.** Feche o RuntimeCrypto de verdade (bandeja → `Sair`). No Gerenciador de Tarefas, aba Detalhes, não pode haver `rclone.exe`.
- [ ] **P-4.** Prepare a pasta de teste. Em `-PastaAntiga`, ponha a pasta onde o RuntimeCrypto antigo guardava o `vaults.json`:

  ```powershell
  cd docs\testes\scripts
  powershell -ExecutionPolicy Bypass -File .\preparar.ps1 -Exe ..\..\..\runtime-crypt-go.exe -PastaAntiga "C:\caminho\do\RuntimeCrypto\antigo"
  ```

  Anote o SHA256 do `vaults.json` antigo que o script mostra. O script também faz o primeiro backup.

---

## 1. Primeira abertura e pasta de configuração

### T-001 · O vaults.json antigo é copiado uma vez

- Demandas: 031, 007
- Preparação: P-4 feito. `RuntimeCrypto-teste\app\vaults.json` existe e `RuntimeCrypto-teste\appdata\` está vazia. Sem `vaults.json` antigo, pule o teste e escreva isso em `Resultado:`.
- Passos:
  1. Anote a data de modificação de `RuntimeCrypto-teste\app\vaults.json`.
  2. Rode `abrir.ps1`.
  3. Abra `RuntimeCrypto-teste\appdata\RuntimeCrypto\`.
  4. Rode `Get-FileHash` nos dois `vaults.json` (o de `app\` e o de `appdata\RuntimeCrypto\`).
- Esperado:
  - A janela abre com o título `RuntimeCrypto` e o subtítulo `Cofre Criptografado na Nuvem  •  v3.0`. Não há faixa amarela no topo.
  - Todos os cofres do `vaults.json` antigo aparecem, cada um com `Trancado` e o botão `Destrancar`.
  - `appdata\RuntimeCrypto\` tem `vaults.json` e `runtimecrypto.log`.
  - Os dois `vaults.json` têm o mesmo SHA256, igual ao do P-4.
  - O `vaults.json` de `app\` tem a mesma data do passo 1.
  - O log tem uma linha `pasta de configuração: vaults.json copiado de …\app para …\appdata\RuntimeCrypto (o antigo fica como está)`.
- [ ] Passou
- Resultado:

### T-002 · A segunda abertura não copia de novo

- Demandas: 031
- Preparação: T-001 feito.
- Passos:
  1. Anote a data de modificação de `appdata\RuntimeCrypto\vaults.json`.
  2. Feche o app pela bandeja (`Sair`) e rode `abrir.ps1` de novo.
- Esperado:
  - A data do passo 1 não mudou.
  - O log não ganhou outra linha `vaults.json copiado`.
  - Os cofres continuam na lista.
- [ ] Passou
- Resultado:

### T-003 · Janela sem cofres no modo normal

- Demandas: 031
- Preparação: feche o app.
- Passos:
  1. Rode `abrir.ps1 -Modo Vazio`.
- Esperado:
  - Sem faixa no topo.
  - No meio da janela, logo ao abrir, sem esperar: `Nenhum cofre configurado.` e, embaixo, `Clique em '＋ Adicionar Cofre' para começar.`
  - Os botões `＋  Adicionar Cofre`, `📥  Importar Cofre Existente` e `⚙  Configurações` estão habilitados.
- Depois: feche o app e rode `abrir.ps1` (modo normal).
- [ ] Passou
- Resultado:

---

## 2. Janela e bandeja

### T-010 · Menu e dica da bandeja

- Demandas: 031, 018
- Preparação: app aberto no modo normal.
- Passos:
  1. Passe o mouse no ícone da bandeja.
  2. Clique com o botão direito no ícone. Abra o submenu `Configurações`.
- Esperado:
  - Dica: `RuntimeCrypto` e, embaixo, `Trancado (N)`, com N igual ao número de cofres. Sem cofres, a dica é `RuntimeCrypto — nenhum cofre`.
  - Menu, nesta ordem: `Abrir RuntimeCrypto`, `Novo Cofre…`, `Configurações` (submenu), `Sobre`, `Sair`.
  - Submenu: `Abrir ao ligar o computador` (com marca quando ligado), `Configurações VFS…`, `Verificar WinFsp/FUSE`.
  - `…` é um caractere só (não três pontos). `Configurações` tem cedilha e til.
  - `Abrir ao ligar o computador` só é testado no T-087, porque mexe no registro.
- [ ] Passou
- Resultado:

### T-011 · Verificar WinFsp/FUSE com o WinFsp instalado

- Demandas: 027
- Passos:
  1. Bandeja → `Configurações` → `Verificar WinFsp/FUSE`.
- Esperado: diálogo `WinFsp/FUSE` com `WinFsp/FUSE está instalado e funcionando.` e o botão `OK`.
- [ ] Passou
- Resultado:

### T-012 · Sobre

- Demandas: —
- Passos:
  1. Clique em `ℹ  Sobre`.
- Esperado: diálogo `RuntimeCrypto`, com `Versão 3.0` e `Licença AGPL-3.0 — Copyright (c) Douglas Eufrauzino de Souza`.
- [ ] Passou
- Resultado:

---

## 3. Criar cofre

### T-020 · O wizard recusa dados incompletos

- Demandas: 017
- Passos:
  1. Clique em `＋  Adicionar Cofre`. Abre `Criar Novo Cofre`, com as abas `1. Provedor` e `2. Senha`.
  2. Na aba `1. Provedor`, confira o texto `Escolha o provedor de nuvem:` e os botões.
  3. Vá direto para a aba `2. Senha`, sem escolher provedor, e clique em `Criar Cofre`.
  4. Volte, escolha `Google Drive`. Deixe o nome vazio e clique em `Criar Cofre`.
  5. Nome `teste-x`, senha `curta`, confirmação `curta`. Clique em `Criar Cofre`.
  6. Senha `senha-teste-x`, confirmação `outra-senha`. Clique em `Criar Cofre`.
  7. Clique em `Cancelar`.
- Esperado:
  - Passo 2: botões `Google Drive`, `Microsoft OneDrive` e `Dropbox`, só esses. Não há `Amazon S3 / MinIO` nem `Pasta Local`.
  - Aba `2. Senha`: rótulos `Senha:`, `Confirmar senha:` e `Nome do cofre:`, com os textos de exemplo `Mínimo 8 caracteres`, `Repita a senha` e `Ex: MeusDocumentos`.
  - Passos 3 a 6: um diálogo `Erro` com, nesta ordem:
    - `Selecione um provedor primeiro.`
    - `Informe um nome para o cofre.`
    - `A senha deve ter pelo menos 8 caracteres.`
    - `As senhas não coincidem.`
  - Depois de cada `OK`, o wizard continua aberto com o que foi digitado.
  - Passo 7: o wizard fecha e nada muda na lista.
- [ ] Passou
- Resultado:

### T-021 · OAuth cancelado no navegador

- Demandas: 020, 018, 006
- Preparação: Gerenciador de Tarefas aberto na aba Detalhes.
- Passos:
  1. `＋  Adicionar Cofre` → `Google Drive`. Senha `senha-teste-x` duas vezes, nome `teste-cancelado`. Clique em `Criar Cofre`.
  2. Clique em `OK` no diálogo `Autorização`.
  3. Na página do Google, clique em `Cancelar`. Se a página não tiver esse botão, feche a aba e espere 2 minutos.
  4. Depois do erro, olhe o Gerenciador de Tarefas.
  5. Num PowerShell com o `RCLONE_CONFIG` de teste (README), rode `rclone listremotes`.
- Esperado:
  - Passo 1: abre o diálogo `Novo cofre` com `Autorizando no navegador…`. Depois aparece `Autorização`, com `O navegador foi aberto para autorização.` e `Após concluir, volte para esta janela.`. O navegador abre o Google.
  - Passo 3: um diálogo `Erro` com uma frase que começa por `Não deu para autorizar no navegador: `. Se você fechou a aba, a frase é `Não deu para autorizar no navegador: Autorização não concluída em 2m0s.`. Não aparece texto em inglês.
  - `teste-cancelado` não entra na lista.
  - Passo 4: nenhum `rclone.exe`.
  - Passo 5: não há `teste-cancelado:` nem `teste-cancelado_base:`.
- [ ] Passou
- Resultado:

### T-022 · Criar o cofre teste-a

- Demandas: 018, 029, 006, 017
- Passos:
  1. `＋  Adicionar Cofre` → `Google Drive`. Senha `senha-teste-a` duas vezes, nome `teste-a`. `Criar Cofre`.
  2. `OK` em `Autorização`. Autorize no navegador com a conta de teste.
  3. Volte para o app.
- Esperado:
  - O diálogo `Novo cofre` passa por `Autorizando no navegador…`, `Configurando o Google Drive…` e `Gravando o cofre…`.
  - Diálogo `Sucesso`:
    - `Cofre criado com sucesso!`
    - `Nome do cofre: teste-a`
    - `Provedor: Google Drive`
    - `Pasta: /`
    - `Use o botão 'Destrancar' para montar.`
  - A palavra "remoto" não aparece em nenhum diálogo.
  - O card `teste-a` aparece com `Google Drive`, `Trancado` e o botão `Destrancar`.
  - `rclone listremotes` (com o `RCLONE_CONFIG` de teste) mostra `teste-a:` e `teste-a_base:`.
- [ ] Passou
- Resultado:

### T-023 · Nomes recusados antes de autorizar

- Demandas: 006
- Preparação: T-022 feito.
- Passos: para cada nome abaixo, `＋  Adicionar Cofre` → `Google Drive` → senha `senha-teste-x` duas vezes → `Criar Cofre`:
  1. `teste-a`
  2. `teste-a_base`
  3. `a:b`
- Esperado:
  - O navegador não abre em nenhum dos três.
  - Diálogo `Erro` com:
    1. `Já existe um cofre com o nome 'teste-a'.`
    2. `O nome 'teste-a_base' já está em uso no rclone. Escolha outro nome para o cofre.`
    3. `Nome 'a:b' inválido: use letras, números, espaço e _ . + @ -, sem começar por hífen.`
  - `rclone listremotes` continua igual ao do T-022.
- [ ] Passou
- Resultado:

---

## 4. Importar cofre e seletor de pasta

### T-030 · O seletor mostra os nomes como estão no Drive

- Demandas: 021, 023, 027, 031
- Passos:
  1. `📥  Importar Cofre Existente`. Confira a aba `1. Provedor` e escolha `Google Drive`.
  2. Na aba `2. Senhas`: senha `senha-teste-b`, a segunda senha vazia, nome `teste-b`. `Avançar  →`.
  3. `OK` em `Autorização`. Autorize com a conta de teste.
  4. Olhe o seletor enquanto ele carrega e depois de carregar.
  5. Clique em `📁  vazia`.
  6. Clique em `⬆ Voltar`.
  7. Não feche o seletor: o T-031 continua nele.
- Esperado:
  - Passo 1: `Importar Cofre Existente` e `Onde está o cofre existente?`, com `Google Drive`, `Microsoft OneDrive` e `Dropbox`, só esses. Não há `Amazon S3 / MinIO` nem `Pasta Local`.
  - Passo 2: rótulos `Senha do cofre (password):`, `Senha 2 / salt (password2):` e `Nome para o cofre:`. Textos de exemplo `Senha usada na criação do cofre` e `Deixe vazio se igual à senha`.
  - Passo 4:
    - Título `Selecionar Pasta do Cofre` e `Navegando em: Google Drive`.
    - Topo: `Google Drive /`. Não aparece `teste-b_base:`.
    - Enquanto carrega: `Carregando pastas…`, sem emoji.
    - A lista tem `📁  alfa`, `📁  minha pasta`, `📁  ação`, `📁  2024 fotos` e `📁  -rascunho`, exatamente assim: sem `-1` na frente, com o espaço e o acento.
    - A janela não trava enquanto carrega.
  - Passo 5: topo `Google Drive /vazia` e uma linha só, `Nenhuma subpasta aqui.`, sem botão.
  - Passo 6: volta para `Google Drive /` com a lista do passo 4.
  - Botões `⬆ Voltar`, `Cancelar` e `✓  Selecionar esta pasta`.
- [ ] Passou
- Resultado:

### T-031 · Seletor sem rede

- Demandas: 009, 027
- Preparação: o seletor do T-030 aberto em `Google Drive /`.
- Passos:
  1. Desligue o Wi-Fi ou tire o cabo.
  2. Clique em `📁  alfa`. Espere até 1 minuto.
  3. Ligue a rede de novo. Espere a conexão voltar.
  4. Clique em `Tentar de novo`.
  5. Clique em `⬆ Voltar`.
- Esperado:
  - Passo 2: topo `Google Drive /alfa` e a linha `Não deu para listar as pastas: sem conexão com o Google Drive`, com o botão `Tentar de novo`. Nunca `Nenhuma subpasta aqui.` e nunca texto em inglês. Se vier `Não deu para listar as pastas: o rclone falhou, detalhes no log`, anote: é aceitável, mas mostra que a frase de conexão não pegou esse caso.
  - Passo 4: a linha de erro some e aparece o conteúdo de `alfa`, que é `Nenhuma subpasta aqui.`.
- Como desfazer: a rede ligada (passo 3).
- [ ] Passou
- Resultado:

### T-032 · Importar teste-b numa pasta que começa com hífen

- Demandas: 021, 029
- Preparação: o seletor em `Google Drive /`.
- Passos:
  1. Clique em `📁  -rascunho` e depois em `✓  Selecionar esta pasta`.
  2. Abra `appdata\RuntimeCrypto\vaults.json` no Bloco de Notas, sem salvar.
- Esperado:
  - O diálogo `Conectar cofre` mostra `Configurando o Google Drive…` e `Gravando o cofre…`.
  - Diálogo `Sucesso`:
    - `Cofre importado com sucesso!`
    - `Nome do cofre: teste-b`
    - `Provedor: Google Drive`
    - `Pasta: /-rascunho`
    - `Use o botão 'Destrancar' para montar.`
  - O card `teste-b` aparece com `Trancado`.
  - No `vaults.json`, `teste-b` tem `"remoto_base": "teste-b_base:-rascunho"`.
- [ ] Passou
- Resultado:

---

## 5. Destrancar e trancar

Antes desta seção, feche o app pela bandeja e rode `abrir.ps1`. Criar e importar deixam a senha na sessão, e esta seção precisa que o app peça a senha.

### T-040 · Senha errada

- Demandas: 030
- Preparação: Gerenciador de Tarefas aberto na aba Detalhes.
- Passos:
  1. No card `teste-a`, clique em `Destrancar`.
  2. Digite `errada-1` e clique em `Desbloquear`.
  3. Sem clicar no campo, digite `errada-2` e aperte Enter.
  4. Repita com `errada-3`.
- Esperado:
  - Passo 1: o diálogo tem o título `Desbloquear cofre`, o nome `teste-a`, o rótulo `Senha do cofre:`, o campo com o texto de exemplo `Senha` e os botões `Cancelar` e `Desbloquear`.
  - Passo 2: `Senha errada.` aparece embaixo do campo, em vermelho, dentro do mesmo diálogo. Não abre outro diálogo. O diálogo continua aberto, com o cursor no campo e o texto todo selecionado.
  - Passo 3: o que você digitou substitui o texto anterior, porque ele estava selecionado. Enter confirma como o botão. Enquanto confere, `Cancelar` e `Desbloquear` ficam desabilitados por um instante.
  - Passo 4: a terceira senha errada também só mostra `Senha errada.`. Não há bloqueio nem espera.
  - Em nenhum momento aparece `rclone.exe` no Gerenciador de Tarefas, e o card continua `Trancado`.
- [ ] Passou
- Resultado:

### T-041 · Cancelar o diálogo de senha

- Demandas: 030, 018
- Passos:
  1. Com o diálogo do T-040 aberto, clique em `Cancelar`.
- Esperado: o diálogo fecha. O card continua `Trancado`, sem linha de erro. Não aparece outro diálogo.
- [ ] Passou
- Resultado:

### T-042 · Senha certa

- Demandas: 030, 018, 001
- Passos:
  1. `teste-a` → `Destrancar`. Digite `senha-teste-a` e aperte Enter.
  2. Logo depois, clique de novo no botão do card.
  3. Olhe o Gerenciador de Tarefas.
  4. `OK` no diálogo. Passe o mouse no ícone da bandeja.
- Esperado:
  - Durante a montagem, o card mostra `Destrancando…` e o botão fica desabilitado. O clique do passo 2 não faz nada.
  - Passo 3: um `rclone.exe` só.
  - Diálogo `Cofre Destrancado`: `'teste-a' montado em V:\` e, embaixo, `O Explorador de Arquivos foi aberto.`. O Explorador abre em `V:\`.
  - O card mostra `Destrancado • V:\` e o botão `Trancar`.
  - A dica da bandeja tem `Destrancado • V:\ (1)` e `Trancado (N)` para os outros cofres.
- [ ] Passou
- Resultado:

### T-043 · O cofre continua destrancado depois de 10 s

- Demandas: 001, 010
- Preparação: T-042 feito.
- Passos:
  1. Espere 30 segundos sem mexer em nada.
  2. Abra `V:\` no Explorador.
- Esperado: o card continua `Destrancado • V:\`, com o botão `Trancar`. A unidade abre. O `rclone.exe` continua no Gerenciador de Tarefas.
- [ ] Passou
- Resultado:

### T-044 · Gravar, trancar e conferir na nuvem

- Demandas: 002, 022
- Passos:
  1. Num PowerShell, crie um arquivo de 1 MB no cofre:

     ```powershell
     $f = [IO.File]::Create('V:\teste-1mb.bin'); $f.SetLength(1MB); $f.Close()
     ```

  2. Feche o Explorador que está em `V:\`. Clique em `Trancar`.
  3. Num PowerShell com o `RCLONE_CONFIG` de teste, rode `rclone ls teste-a:`.
  4. Clique em `Destrancar` de novo.
- Esperado:
  - Passo 2: diálogo `Cofre Trancado` com `'teste-a' trancado.`. A letra `V:` some do Explorador. O card volta a `Trancado`. Não sobra `rclone.exe`.
  - Passo 3: aparece `teste-1mb.bin` com 1048576 bytes.
  - Passo 4: o app pede a senha de novo. Trancar tirou a senha da sessão.
- Depois: destranque com `senha-teste-a` e deixe destrancado para o T-045.
- [ ] Passou
- Resultado:

### T-045 · Trancar espera o envio

- Demandas: 025
- Preparação: `teste-a` destrancado, VFS no padrão (ainda não mexeu no T-050).
- Passos:
  1. Crie um arquivo grande e clique em `Trancar` logo em seguida, em menos de 5 segundos:

     ```powershell
     $f = [IO.File]::Create('V:\grande.bin'); $f.SetLength(300MB); $f.Close()
     ```

  2. Destranque de novo (`senha-teste-a`). Crie três arquivos e tranque logo em seguida:

     ```powershell
     1..3 | ForEach-Object { $f = [IO.File]::Create("V:\varios-$_.bin"); $f.SetLength(100MB); $f.Close() }
     ```

  3. Rode `rclone ls teste-a:`.
- Esperado:
  - Passo 1: o card mostra `Enviando 1 arquivo…`, no singular, com o botão desabilitado. Quando o envio acaba: `Cofre Trancado` e o card `Trancado`.
  - Passo 2: `Enviando 3 arquivos…`. O número cai conforme os arquivos sobem. O card nunca mostra `Enviando 1 arquivos…`.
  - Passo 3: `grande.bin`, `varios-1.bin`, `varios-2.bin` e `varios-3.bin`, com os tamanhos certos.
- [ ] Passou
- Resultado:

### T-046 · Trancar que não termina (opcional)

- Demandas: 022
- Este teste pode não ter como falhar no Windows: o WinFsp costuma soltar a unidade mesmo com um arquivo aberto. Se trancar normalmente, escreva "trancou" em `Resultado:` e siga.
- Passos:
  1. Destranque `teste-a`. Num PowerShell, rode `cd V:\` e deixe a janela aberta.
  2. Clique em `Trancar`.
  3. Se o card mostrar `Não trancou: …`, clique no botão do card.
- Esperado, se falhar:
  - O card continua `Destrancado • V:\` com uma linha vermelha `Não trancou: {motivo}`.
  - Passo 3: não pede a senha. A senha só sai quando o cofre tranca de verdade.
- Depois: feche o PowerShell e tranque `teste-a`.
- [ ] Passou / não se aplica
- Resultado:

---

## 6. Configurações VFS

### T-050 · Valor inválido não é salvo

- Demandas: 011
- Passos:
  1. Clique em `⚙  Configurações`. Abre `Configurações VFS` com `Ajustes de cache e streaming do RClone`.
  2. No campo `Modo de cache (off, minimal, writes, full)`, troque `full` por `tudo`. Clique em `Salvar`.
  3. Clique em `Cancelar`. Abra `⚙  Configurações` de novo.
- Esperado:
  - Passo 2: o painel continua aberto com `Nada foi salvo. Corrija:` e `Modo de cache (off, minimal, writes, full): "tudo" não é modo de cache; use off, minimal, writes ou full`.
  - Passo 3: o campo mostra `full`.
- [ ] Passou
- Resultado:

### T-051 · Restaurar Padrões e Cancelar não mudam nada

- Demandas: 011
- Passos:
  1. Mude `Tempo máximo de vida do cache` para `2h`. `Salvar`.
  2. Abra `⚙  Configurações`. Clique em `Restaurar Padrões`. O campo volta a `1h`. Clique em `Cancelar`.
  3. Abra `⚙  Configurações` de novo.
- Esperado:
  - Passo 1: diálogo `VFS` com `Configurações VFS salvas com sucesso.`.
  - Passo 3: o campo continua `2h`.
- [ ] Passou
- Resultado:

### T-052 · A configuração VFS sobrevive ao reinício

- Demandas: 012, 031
- Preparação: T-051 feito, com `2h` salvo.
- Passos:
  1. Feche o app pela bandeja. Rode `abrir.ps1`.
  2. Abra `⚙  Configurações`.
  3. Abra `appdata\RuntimeCrypto\`.
- Esperado:
  - Passo 2: `Tempo máximo de vida do cache` = `2h`.
  - Passo 3: existe `vfs.json`, ao lado do `vaults.json`.
- [ ] Passou
- Resultado:

---

## 7. Cofre que caiu e sair do app

Esta seção mata o `rclone.exe` de propósito. O envio padrão começa 5 s depois de o arquivo ser fechado. Para dar tempo de matar o rclone antes, o envio vai esperar 1 minuto até o T-067.

### T-060 · Atraso de envio de 1 minuto

- Demandas: 011
- Preparação: `teste-a` e `teste-b` trancados.
- Passos:
  1. `⚙  Configurações` → `Atraso antes de enviar arquivo à nuvem` = `1m`. `Salvar`.
- Esperado: `Configurações VFS salvas com sucesso.`.
- [ ] Passou
- Resultado:

### T-061 · O card mostra que o cofre caiu

- Demandas: 018, 010, 026
- Passos:
  1. Destranque `teste-a`.
  2. Rode `Set-Content V:\caiu-1.txt 'teste'`.
  3. Em menos de 1 minuto, no Gerenciador de Tarefas, aba Detalhes, finalize `rclone.exe`. Ou rode `Stop-Process -Name rclone`.
  4. Espere até 10 s. Passe o mouse no ícone da bandeja.
- Esperado:
  - O card mostra a linha vermelha `Caiu: o rclone parou` e dois botões: `Destrancar de novo` e `Trancar`.
  - A dica da bandeja tem `Caiu: o rclone parou (1)`.
- [ ] Passou
- Resultado:

### T-062 · Trancar recusa enquanto há arquivo que não subiu

- Demandas: 026
- Preparação: T-061 feito.
- Passos:
  1. Clique em `Trancar` no card.
- Esperado:
  - O card continua com `Caiu: o rclone parou` e ganha a linha `Não trancou: 1 arquivo ainda não subiu`.
  - Continuam os botões `Destrancar de novo` e `Trancar`.
- [ ] Passou
- Resultado:

### T-063 · Destrancar de novo retoma o envio

- Demandas: 026, 022, 025
- Passos:
  1. Clique em `Destrancar de novo`.
  2. Espere 1 minuto. Clique em `Trancar`.
  3. Rode `rclone ls teste-a:`.
- Esperado:
  - Passo 1: não pede a senha. Aparece `Cofre Destrancado`. O card volta a `Destrancado • V:\`, sem as linhas vermelhas.
  - Passo 2: se o arquivo ainda não subiu, o card mostra `Enviando 1 arquivo…`. Depois, `Cofre Trancado` e `Trancado`.
  - Passo 3: aparece `caiu-1.txt`.
- [ ] Passou
- Resultado:

### T-064 · Sair com um arquivo que não subiu: Enviar agora

- Demandas: 028, 026, 001
- Passos:
  1. Destranque `teste-a` (`senha-teste-a`). Rode `Set-Content V:\sair-1.txt 'teste'`.
  2. Em menos de 1 minuto, finalize o `rclone.exe`. Espere `Caiu: o rclone parou`.
  3. Bandeja → `Sair`.
  4. Clique em `Enviar agora`.
  5. Depois que o app fechar, olhe o Gerenciador de Tarefas. Rode `rclone ls teste-a:`.
- Esperado:
  - Passo 3: a janela aparece com o diálogo `Arquivos que ainda não subiram`:
    - `1 arquivo de teste-a`
    - `Ele sobe quando você destrancar de novo.` (no singular)
    - botões `Enviar agora` e `Sair`, sem botão de fechar;
    - nenhuma frase diz que os arquivos vão se perder.
  - Passo 4: não pede a senha. O card passa por `Enviando 1 arquivo…` (pode levar até 1 minuto) e o app fecha sozinho.
  - Passo 5: nenhum `rclone.exe`. Aparece `sair-1.txt`.
- [ ] Passou
- Resultado:

### T-065 · Sair com dois cofres: Sair

- Demandas: 028, 026
- Passos:
  1. Rode `abrir.ps1`. Destranque `teste-a` e `teste-b`. Anote a letra de cada um (por exemplo, `V:` e `W:`).
  2. Rode, trocando as letras se preciso:

     ```powershell
     Set-Content V:\dois-a.txt 'a'
     Set-Content W:\dois-b1.txt 'b1'; Set-Content W:\dois-b2.txt 'b2'
     Stop-Process -Name rclone
     ```

  3. Espere os dois cards mostrarem `Caiu: o rclone parou`. Bandeja → `Sair`.
  4. Clique em `Sair`.
  5. Rode `abrir.ps1`. Destranque `teste-a` e depois `teste-b`. Espere 1 minuto e tranque os dois.
  6. Rode `rclone ls teste-a:` e `rclone ls teste-b:`.
- Esperado:
  - Passo 3: um diálogo só, `Arquivos que ainda não subiram`:
    - `1 arquivo de teste-a`
    - `2 arquivos de teste-b`
    - `Eles sobem quando você destrancar de novo.` (no plural)
  - Passo 4: o app fecha sem enviar e sem mais perguntas. Nenhum `rclone.exe`.
  - Passo 5: os dois pedem a senha e destrancam. Trancar mostra `Enviando …` se ainda houver envio e termina em `Trancado`.
  - Passo 6: `dois-a.txt` em `teste-a`; `dois-b1.txt` e `dois-b2.txt` em `teste-b`.
- [ ] Passou
- Resultado:

### T-066 · Sair com o cofre destrancado e saudável

- Demandas: 001, 002, 025
- Passos:
  1. Destranque `teste-a`. Rode `Set-Content V:\saudavel.txt 'ok'`.
  2. Bandeja → `Sair`.
  3. Olhe o Explorador e o Gerenciador de Tarefas. Rode `rclone ls teste-a:`.
- Esperado:
  - Nenhum diálogo `Arquivos que ainda não subiram`.
  - O app espera o envio (pode levar até 1 minuto) e fecha.
  - `V:` some. Nenhum `rclone.exe`.
  - Aparece `saudavel.txt`.
- [ ] Passou
- Resultado:

### T-067 · Voltar a VFS ao padrão

- Demandas: 011, 012
- Passos:
  1. Rode `abrir.ps1`. `⚙  Configurações` → `Restaurar Padrões` → `Salvar`.
  2. Abra `⚙  Configurações` de novo.
- Esperado: `Configurações VFS salvas com sucesso.`. `Atraso antes de enviar arquivo à nuvem` = `5s` e `Tempo máximo de vida do cache` = `1h`.
- [ ] Passou
- Resultado:

---

## 8. Erros do rclone em português e log

### T-070 · Destrancar sem rede

- Demandas: 027, 018
- Preparação: `teste-a` trancado.
- Passos:
  1. Desligue o Wi-Fi ou tire o cabo.
  2. `teste-a` → `Destrancar` → `senha-teste-a`.
  3. Ligue a rede. Se o cofre não destrancou, clique em `Tentar de novo`.
- Esperado:
  - Passo 2, um destes dois:
    - **Não destrancou:** diálogo `Erro ao Destrancar` com `Falha ao montar 'teste-a':` e `sem conexão com o Google Drive`. O card mostra a linha vermelha `Não destrancou: sem conexão com o Google Drive` e o botão `Tentar de novo`.
    - **Destrancou mesmo sem rede:** o rclone pode montar e só falhar ao listar. Anote isso em `Resultado:`. O teste passa se nenhum texto em inglês aparecer na tela.
  - Em nenhum caso aparece texto em inglês na tela.
  - Passo 3: `Tentar de novo` pede a senha de novo (a falha tira a senha da sessão) e destranca.
- Como desfazer: a rede ligada. Tranque `teste-a`.
- [ ] Passou
- Resultado:

### T-071 · O log tem o texto do rclone e não tem senha

- Demandas: 027, 031
- Passos:
  1. Abra `appdata\RuntimeCrypto\runtimecrypto.log` no Bloco de Notas.
  2. Procure (Ctrl+F) `senha-teste-a`, `senha-teste-b`, `errada-1` e `senha-teste-x`.
- Esperado:
  - Há linhas que começam por `rclone ` com o texto original em inglês dos erros do T-031 e do T-070.
  - Há linhas `conferir senha de teste-a: …` das senhas erradas do T-040.
  - Nenhuma das senhas do passo 2 aparece.
- [ ] Passou
- Resultado:

### T-075 · Migração de verdade (opcional, quando for passar a usar a versão nova)

- Demandas: 031
- Este teste usa o seu ambiente de verdade, sem a pasta de teste. Ele só copia; nada é apagado. O backup do P-4 já tem uma cópia do seu `%AppData%\RuntimeCrypto` (se existia) e do seu `rclone.conf`.
- Preparação: o app de teste fechado. Anote a data e o SHA256 do `vaults.json` da pasta antiga. Se `%AppData%\RuntimeCrypto\vaults.json` já existe, a cópia não vai acontecer: escreva isso em `Resultado:` e pule.
- Passos:
  1. Copie o `runtime-crypt-go.exe` novo para a pasta antiga, no lugar do executável de antes. Guarde o antigo com outro nome.
  2. Abra o app pelo executável, sem o `abrir.ps1`.
- Esperado:
  - Sem faixa no topo. Todos os seus cofres na lista.
  - `%AppData%\RuntimeCrypto\` tem `vaults.json` (mesmo SHA256 do antigo) e `runtimecrypto.log`.
  - O `vaults.json` da pasta antiga tem a mesma data e o mesmo SHA256 de antes.
  - Se a pasta antiga for `C:\Program Files\…`, criar um cofre e salvar a VFS agora funcionam, porque o app grava em `%AppData%`.
- [ ] Passou
- Resultado:

---

## 9. Testes que mexem no ambiente

Todos os testes desta seção usam a pasta de teste, menos o T-087 (registro) e o T-088 (WinFsp). Cada um termina em `Como desfazer` e numa conferência. Não comece o próximo antes de a conferência passar.

**Antes da seção, faça um backup.** Feche o app e rode:

```powershell
cd docs\testes\scripts
powershell -ExecutionPolicy Bypass -File .\backup.ps1
```

Ele copia para `RuntimeCrypto-teste\backup\<data-hora>`:

- `RuntimeCrypto-teste\appdata\RuntimeCrypto` e `RuntimeCrypto-teste\rclone\rclone.conf`;
- o seu `%AppData%\RuntimeCrypto` e o seu `%AppData%\rclone\rclone.conf`, se existirem.

Sem o script, o mesmo backup à mão:

```powershell
$b = "$env:USERPROFILE\RuntimeCrypto-teste\backup\manual-$(Get-Date -Format yyyyMMdd-HHmmss)"
New-Item -ItemType Directory $b | Out-Null
Copy-Item "$env:USERPROFILE\RuntimeCrypto-teste\appdata\RuntimeCrypto" "$b\teste-RuntimeCrypto" -Recurse
Copy-Item "$env:USERPROFILE\RuntimeCrypto-teste\rclone\rclone.conf" "$b\teste-rclone.conf"
if (Test-Path "$env:APPDATA\RuntimeCrypto") { Copy-Item "$env:APPDATA\RuntimeCrypto" "$b\real-RuntimeCrypto" -Recurse }
if (Test-Path "$env:APPDATA\rclone\rclone.conf") { Copy-Item "$env:APPDATA\rclone\rclone.conf" "$b\real-rclone.conf" }
```

- [ ] Backup feito em: `________`

### T-080 · Modo só leitura: pasta de configuração sem gravação

- Demandas: 031
- Preparação: app fechado, backup feito, `teste-a` trancado.
- Passos:
  1. Anote o conteúdo de `appdata\RuntimeCrypto`:

     ```powershell
     Get-ChildItem "$env:USERPROFILE\RuntimeCrypto-teste\appdata\RuntimeCrypto" | Select-Object Name, Length, LastWriteTime
     ```

  2. Rode `powershell -ExecutionPolicy Bypass -File .\somente-leitura.ps1`. Na saída, a pasta tem uma linha com `(DENY)`.
  3. Rode `abrir.ps1`.
  4. Abra o menu da bandeja e o submenu `Configurações`.
  5. Destranque `teste-a`, crie `Set-Content V:\so-leitura.txt 'x'`, espere 10 s e tranque.
  6. Feche o app. Rode de novo o comando do passo 1.
- Esperado:
  - Passo 3: faixa fixa no topo, sem título: `Mudanças não serão salvas: não deu para gravar em C:\Users\…\RuntimeCrypto-teste\appdata\RuntimeCrypto.`
  - Os cofres aparecem. `＋  Adicionar Cofre`, `📥  Importar Cofre Existente` e `⚙  Configurações` estão desabilitados. Clicar neles não faz nada.
  - Passo 4: `Novo Cofre…` e `Configurações VFS…` desabilitados. `Abrir ao ligar o computador`, `Verificar WinFsp/FUSE`, `Sobre` e `Sair` habilitados.
  - Passo 5: destrancar, gravar e trancar funcionam como no T-044.
  - Passo 6: a lista é igual à do passo 1: mesmos nomes, tamanhos e datas. O `runtimecrypto.log` não cresceu.
  - Nada novo apareceu em `RuntimeCrypto-teste\app\`.
- Como desfazer:
  1. Rode `powershell -ExecutionPolicy Bypass -File .\somente-leitura.ps1 -Desfazer`.
  2. Rode `abrir.ps1`.
- Conferência: a saída do `-Desfazer` não tem `(DENY)`. O app abre sem faixa e com os três botões habilitados. Feche o app.
- [ ] Passou
- [ ] Ambiente restaurado
- Resultado:

### T-081 · Modo só leitura: pasta de configuração desconhecida

- Demandas: 031
- Preparação: app fechado.
- Passos:
  1. Rode `abrir.ps1 -Modo SemCaminho`. Ele abre o app sem a variável `APPDATA`.
- Esperado:
  - Faixa: `Mudanças não serão salvas: não deu para gravar na pasta de configuração.`
  - No meio da janela, só `Nenhum cofre ainda.`, sem a dica de `Adicionar Cofre`.
  - Os três botões desabilitados.
  - Se o app nem abrir, anote: pode ser a biblioteca da interface, que também usa `APPDATA`.
- Como desfazer: feche o app. Nada foi gravado: este modo só muda variáveis do processo.
- Conferência: `abrir.ps1` abre sem faixa e com os cofres.
- [ ] Passou
- [ ] Ambiente restaurado
- Resultado:

### T-082 · rclone ausente

- Demandas: 030
- Preparação: app fechado.
- Passos:
  1. Rode `abrir.ps1 -Modo SemRclone`. O app abre de `app-limpo\` (sem `rclone.exe` ao lado) e com um `PATH` sem nenhuma pasta que tenha `rclone.exe`. Nenhum arquivo é renomeado.
  2. `teste-a` → `Destrancar` → `senha-teste-a`.
  3. `Cancelar`. Clique em `＋  Adicionar Cofre`, escolha `Google Drive`, senha `senha-teste-x` duas vezes, nome `teste-sem-rclone`, `Criar Cofre`.
- Esperado:
  - Passo 2: embaixo do campo, `Não destrancou: o rclone não está instalado.`. O diálogo continua aberto e o card continua `Trancado`.
  - Passo 3: diálogo `Erro` com `O rclone não está instalado.`. O navegador não abre.
- Como desfazer: feche o app. Nada foi renomeado nem gravado.
- Conferência: `abrir.ps1` e destrancar `teste-a` funcionam. Tranque e feche.
- [ ] Passou
- [ ] Ambiente restaurado
- Resultado:

### T-083 · rclone.conf sem o cofre

- Demandas: 030
- Preparação: app fechado.
- Passos:
  1. Rode `abrir.ps1 -Modo ConfigVazia`. O app usa `rclone\vazio.conf`, um arquivo vazio. O seu `rclone.conf` de teste não é tocado.
  2. `teste-a` → `Destrancar` → `senha-teste-a`.
- Esperado: embaixo do campo, `Não destrancou: a configuração deste cofre está incompleta. Conecte o cofre de novo.`. O diálogo continua aberto. O card continua `Trancado`.
- Como desfazer: `Cancelar` e feche o app.
- Conferência: `vazio.conf` continua com 0 bytes. `abrir.ps1` e destrancar `teste-a` funcionam. Tranque e feche.
- [ ] Passou
- [ ] Ambiente restaurado
- Resultado:

### T-084 · rclone.conf ilegível

- Demandas: 030
- Preparação: app fechado.
- Passos:
  1. Rode `abrir.ps1 -Modo ConfigIlegivel`. `RCLONE_CONFIG` aponta para uma pasta, que o rclone não consegue ler como arquivo.
  2. `teste-a` → `Destrancar` → `senha-teste-a`.
- Esperado: embaixo do campo, `Não destrancou: não deu para ler a configuração do rclone.`. O diálogo continua aberto. O card continua `Trancado`.
- Como desfazer: `Cancelar` e feche o app.
- Conferência: `abrir.ps1` e destrancar `teste-a` funcionam. Tranque e feche.
- [ ] Passou
- [ ] Ambiente restaurado
- Resultado:

### T-085 · vaults.json corrompido

- Demandas: 007, 024
- Preparação: app fechado e backup desta seção feito. O `vaults.json` de teste vai ficar estragado durante o teste e volta do backup no fim.
- Passos:
  1. Abra `appdata\RuntimeCrypto\vaults.json` no Bloco de Notas. Ponha `{quebrado` no começo da primeira linha e salve.
  2. Anote o SHA256 do arquivo (`Get-FileHash`).
  3. Rode `abrir.ps1`.
  4. Feche o app. Repita `abrir.ps1` e feche mais duas vezes.
  5. Liste `appdata\RuntimeCrypto\`. Rode `Get-FileHash` no `vaults.json` de novo.
- Esperado:
  - Passo 3: diálogo `Erro ao ler os cofres` com `…\vaults.json está corrompido (…); nada será gravado por cima. Cópia guardada em …\vaults.json.corrompido-…`.
  - Passo 5: um arquivo `vaults.json.corrompido-…` só, mesmo depois de três aberturas. O SHA256 do `vaults.json` é o do passo 2: ele não foi regravado.
- Como desfazer:
  1. Feche o app.
  2. Copie de volta o `vaults.json` do backup desta seção:

     ```powershell
     Copy-Item "<pasta do backup>\teste-RuntimeCrypto\vaults.json" "$env:USERPROFILE\RuntimeCrypto-teste\appdata\RuntimeCrypto\vaults.json"
     ```

  3. Mova o `vaults.json.corrompido-…` para a pasta do backup. Não precisa apagar.
- Conferência: `abrir.ps1` abre sem diálogo de erro, com `teste-a`, `teste-b` e os cofres antigos. Feche o app.
- [ ] Passou
- [ ] Ambiente restaurado
- Resultado:

### T-086 · vfs.json com valor inválido

- Demandas: 012
- Preparação: app fechado. `appdata\RuntimeCrypto\vfs.json` existe (T-052) e está no backup desta seção.
- Passos:
  1. Abra `vfs.json` no Bloco de Notas. Troque o valor de `vfs_cache_mode` por `tudo` e salve.
  2. Rode `abrir.ps1`.
  3. Abra `⚙  Configurações`.
- Esperado:
  - Passo 2: diálogo `Configurações VFS` com `vfs.json: vfs_cache_mode="tudo" descartado ("tudo" não é modo de cache; use off, minimal, writes ou full); usando "full".`
  - Passo 3: o modo de cache mostra `full`.
- Como desfazer:
  1. Feche o app.
  2. Copie de volta o `vfs.json` do backup: `Copy-Item "<pasta do backup>\teste-RuntimeCrypto\vfs.json" "$env:USERPROFILE\RuntimeCrypto-teste\appdata\RuntimeCrypto\vfs.json"`.
- Conferência: `abrir.ps1` abre sem o aviso. Feche o app.
- [ ] Passou
- [ ] Ambiente restaurado
- Resultado:

### T-087 · Abrir ao ligar o computador

- Demandas: 031
- Este teste mexe no registro do seu usuário (`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, valor `RuntimeCrypto`). O item do menu mostra uma marca quando está ligado. O teste confere a marca contra o registro.
- Preparação:
  1. Rode e guarde a saída:

     ```powershell
     reg query HKCU\Software\Microsoft\Windows\CurrentVersion\Run /v RuntimeCrypto
     ```

  2. Se o valor já existe (o seu RuntimeCrypto de verdade abre com o Windows), anote o caminho que aparece. O primeiro clique vai **tirar** esse valor.
- Passos:
  1. Rode `abrir.ps1`. Bandeja → `Configurações`. Olhe o item `Abrir ao ligar o computador` sem clicar.
  2. Clique em `Abrir ao ligar o computador`. Rode o `reg query` de novo. Abra o submenu de novo e olhe o item.
  3. Clique no item de novo. Rode o `reg query` de novo. Abra o submenu de novo e olhe o item.
  4. Feche o app e rode `abrir.ps1` de novo. Abra o submenu e olhe o item.
- Esperado:
  - O nome do item é `Abrir ao ligar o computador`, e não `Auto-iniciar com Windows`.
  - Passo 1: o item tem marca se, e só se, o `reg query` da preparação achou o valor `RuntimeCrypto`.
  - Passos 2 e 3: cada clique troca o estado. O valor `RuntimeCrypto` aparece (com o caminho de `RuntimeCrypto-teste\app\runtime-crypt-go.exe`) ou some, e a marca acompanha: com o valor, marcado; sem o valor, sem marca. Nenhum diálogo de erro.
  - Passo 4: a marca é a do último `reg query`.
- Como desfazer: clique no item até o registro ficar como na preparação. Se o valor existia com o caminho do app de verdade, devolva-o com:

  ```powershell
  reg add HKCU\Software\Microsoft\Windows\CurrentVersion\Run /v RuntimeCrypto /t REG_SZ /d "<valor anotado na preparação>" /f
  ```

- Conferência: o `reg query` dá a mesma saída da preparação.
- [ ] Passou
- [ ] Ambiente restaurado
- Resultado:

### T-088 · Sem o WinFsp

- Demandas: 027, 018
- Este é o teste mais invasivo: desinstala o WinFsp do Windows. Faça por último. Ele não mexe em arquivo de configuração.
- Preparação: todos os cofres trancados, app fechado. Anote a versão do WinFsp instalada (Configurações do Windows → Aplicativos).
- Passos:
  1. Desinstale o WinFsp em Configurações do Windows → Aplicativos. Reinicie se o Windows pedir.
  2. Rode `abrir.ps1`. Bandeja → `Configurações` → `Verificar WinFsp/FUSE`.
  3. `teste-a` → `Destrancar` → `senha-teste-a`.
  4. Clique no botão `Baixar WinFsp` do card.
- Esperado:
  - Passo 2: diálogo `WinFsp/FUSE Ausente` com `Não montou: falta instalar o WinFsp.` e dois botões, `Baixar WinFsp` e `OK`. `Baixar WinFsp` abre `https://winfsp.dev/rel/` no navegador e não fecha o diálogo.
  - Passo 3: diálogo `Erro ao Destrancar` com `Não montou: falta instalar o WinFsp.`. O card mostra a mesma frase em vermelho e dois botões: `Tentar de novo` e `Baixar WinFsp`. Nenhum `rclone.exe` fica aberto. Não aparece `cannot find winfsp`.
  - Passo 4: o navegador abre `https://winfsp.dev/rel/`.
- Como desfazer:
  1. Feche o app.
  2. Baixe o instalador em https://winfsp.dev/rel/ e instale o WinFsp. Reinicie se pedir.
- Conferência: `abrir.ps1` → `Verificar WinFsp/FUSE` mostra `WinFsp/FUSE está instalado e funcionando.`. Destrancar `teste-a` monta a unidade. Tranque e feche.
- [ ] Passou
- [ ] Ambiente restaurado
- Resultado:

---

## 10. Linux (opcional)

### T-090 · Montagem em pasta no Linux

- Demandas: 013, 001
- Preparação: Linux com `fuse3`, o app compilado da `master` e um cofre de teste (crie pelo wizard como no T-022). Os scripts não rodam no Linux: use o app direto. A configuração fica em `~/.config/RuntimeCrypto`.
- Passos:
  1. Destranque o cofre.
  2. Grave um arquivo na pasta do cofre e tranque.
  3. Rode `mount | grep rclone`.
- Esperado:
  - Passo 1: o card mostra `Destrancado • ~/RuntimeCrypto/<nome>`. O diálogo diz `'<nome>' montado em ~/RuntimeCrypto/<nome>` (ou o caminho completo).
  - Passo 2: `Cofre Trancado`. A pasta `~/RuntimeCrypto/<nome>` some se ficou vazia.
  - Passo 3: nenhuma linha.
- [ ] Passou
- Resultado:

---

## 11. Demandas 032 e 033 (ainda não implementadas)

> **PENDENTE.** As demandas [032](../demanda/032-app-proprio-do-google-drive.md) e 033 (editar cofre, PR #47) só têm a documentação. Na `master` de hoje, todos os testes desta seção falham, porque as telas não existem. Rode quando os PRs de código delas entrarem.
>
> Os textos aqui são os aprovados pela UI. Onde a doc da demanda marca um texto como proposta, o teste diz isso. Nesse caso, anote o texto que aparecer.

### 032 · App próprio do Google

Preparação para os testes da 032: um projeto no Google Cloud com a Drive API ligada e um OAuth client do tipo "Desktop app", seguindo o passo a passo da doc da 032. Publique o app, menos no T-105.

#### T-101 · PENDENTE · Com o link fechado, o fluxo é o de hoje

- Demandas: 032
- Passos:
  1. `＋  Adicionar Cofre` → `Google Drive`. Não abra `Usar meu próprio app do Google`.
  2. Crie o cofre `teste-032-a` como no T-022.
- Esperado:
  - O passo do Google Drive mostra um link discreto `Usar meu próprio app do Google`, fechado.
  - Tudo igual ao T-022. No `rclone.conf` de teste, a seção `[teste-032-a_base]` não tem `client_id` nem `client_secret`.
- [ ] Passou
- Resultado:

#### T-102 · PENDENTE · O link abre os campos

- Demandas: 032
- Passos:
  1. `＋  Adicionar Cofre` → `Google Drive` → clique em `Usar meu próprio app do Google`.
  2. Repita em `📥  Importar Cofre Existente` → `Google Drive`.
  3. Nos dois wizards, escolha `Microsoft OneDrive` e `Dropbox`.
- Esperado:
  - Passos 1 e 2: aparecem os campos `Client ID` e `Client secret` e um link para o guia do rclone (https://rclone.org/drive/#making-your-own-client-id). O `Client secret` esconde o texto, como um campo de senha.
  - Texto de ajuda: `Recomendado se você envia muitos arquivos. Evita os limites do app compartilhado.`. Nenhum número de velocidade.
  - Passo 3: o link não aparece em nenhum outro provedor.
- [ ] Passou
- Resultado:

#### T-103 · PENDENTE · Criar com o app próprio

- Demandas: 032
- Passos:
  1. Crie `teste-032-b` com o link aberto, o `Client ID` e o `Client secret` do seu projeto.
  2. No navegador, confira que a tela de consentimento mostra o nome do **seu** app, não o do rclone.
  3. Destranque, grave um arquivo e tranque.
- Esperado: o cofre é criado e funciona como o `teste-a`. No `rclone.conf` de teste, `[teste-032-b_base]` tem `client_id` e `client_secret`.
- [ ] Passou
- Resultado:

#### T-104 · PENDENTE · ID ou secret recusados pelo Google

- Demandas: 032
- Passos:
  1. Crie `teste-032-c` com o `Client ID` certo e um `Client secret` errado (troque um caractere).
  2. Tente de novo com os dois campos preenchidos e um `Client ID` que não existe, mas termina em `.apps.googleusercontent.com`.
- Esperado:
  - Passo 1: no passo do OAuth, `Não deu para autorizar: o Google recusou esse app.`. O cofre não é criado e não sobra `teste-032-c_base` no `rclone.conf`.
  - Passo 2: o Google mostra o erro no navegador. O que o app mostra depois ainda está em aberto na doc da 032 (pergunta 2): pode ser essa mesma frase ou `Autorização não concluída em 2m0s.` depois de 2 minutos. Anote qual apareceu.
  - Validação antes do rclone (frases ainda em **proposta**, anote o texto que aparecer):
    - só um campo preenchido: `Preencha o Client ID e o Client secret, ou nenhum dos dois.`;
    - ID sem `.apps.googleusercontent.com`: `O Client ID termina em .apps.googleusercontent.com.`.
- [ ] Passou
- Resultado:

#### T-105 · PENDENTE · Projeto em modo Testing: o acesso vence em 7 dias

- Demandas: 032, 027
- Preparação: um segundo OAuth client num projeto que continua em "Testing" (não publicado).
- Passos:
  1. Crie `teste-032-d` com esse app. Destranque e tranque uma vez.
  2. Espere 8 dias sem usar.
  3. Destranque `teste-032-d`.
- Esperado:
  - Passo 3: o cofre não destranca. O card mostra `Não destrancou: autorização expirou` e o botão `Tentar de novo`. Nenhum texto em inglês.
  - Sem a 033 não há como autorizar de novo. Com a 033, `Reconectar` resolve (T-113).
  - Se a tela da 032 tiver o aviso de modo de teste (proposta: `Publique o app no Google Cloud. Em modo de teste, o acesso vence em 7 dias.`), anote o texto.
- [ ] Passou
- Resultado:

### 033 · Editar cofre

#### T-111 · PENDENTE · Editar só com o cofre trancado

- Demandas: 033
- Passos:
  1. Olhe o card de `teste-a` trancado.
  2. Destranque `teste-a` e olhe o card. Passe o mouse em `Editar`.
  3. Repita com o cofre em `Destrancando…`, `Enviando …`, `Caiu: …` (como no T-061) e `Não destrancou: …`.
- Esperado:
  - Passo 1: o card tem o botão `Editar`, habilitado.
  - Passos 2 e 3: `Editar` aparece desabilitado com `Tranque o cofre para editar.`.
- [ ] Passou
- Resultado:

#### T-112 · PENDENTE · A tela de editar

- Demandas: 033, 032
- Passos:
  1. Com `teste-a` trancado, clique em `Editar`.
  2. Repita com um cofre do OneDrive ou do Dropbox, se tiver.
- Esperado:
  - A tela tem, nesta ordem: o campo com o nome do cofre; a parte do app do Google (`Usar meu próprio app do Google`, `Client ID`, `Client secret`, link do guia), só para Google Drive; o botão `Reconectar`.
  - Embaixo, separado e em vermelho: `Remover cofre`.
  - Num cofre que já usa app próprio, o `Client ID` vem preenchido e o `Client secret` vazio. O secret gravado nunca aparece.
  - Passo 2: sem a parte do app do Google; `Reconectar` aparece.
  - Botões `Cancelar` e `Salvar` (**proposta**). `Salvar` só habilita quando algo mudou.
- [ ] Passou
- Resultado:

#### T-113 · PENDENTE · Reconectar

- Demandas: 033, 032
- Passos:
  1. `teste-032-d` do T-105 (acesso vencido) → `Editar` → `Reconectar`. Autorize com um app publicado.
  2. Destranque `teste-032-d`.
- Esperado: a autorização abre no navegador e termina com a frase de sucesso (**proposta**: `Cofre reconectado.`). O cofre destranca. A senha do cofre não muda.
- [ ] Passou
- Resultado:

#### T-114 · PENDENTE · Remover cofre

- Demandas: 033
- Passos:
  1. `teste-032-a` trancado → `Editar` → `Remover cofre`.
  2. Digite um nome diferente de `teste-032-a`. Depois, digite `teste-032-a`.
  3. Confirme.
  4. Abra o Google Drive de teste no navegador.
- Esperado:
  - Passo 1: o diálogo de confirmação tem, logo abaixo da pergunta, `Os arquivos no Google Drive continuam lá.`. Título, corpo e rótulo do campo ainda são **proposta** (`Remover teste-032-a?`, `Digite teste-032-a para confirmar`): anote o que aparecer.
  - Passo 2: o botão de confirmar só habilita quando o texto é exatamente `teste-032-a`.
  - Passo 3: o card some da lista.
  - Passo 4: os arquivos cifrados continuam no Drive.
- [ ] Passou
- Resultado:

#### T-115 · PENDENTE · Remover bloqueado com arquivos que não subiram

- Demandas: 033, 026, 028
- Passos:
  1. Prepare como no T-065, com `teste-b` e o atraso de envio em `1m`: grave 2 arquivos, mate o rclone, `Sair` → `Sair`.
  2. Abra o app. `teste-b` está `Trancado`, com 2 arquivos no cache. `Editar` → `Remover cofre` → confirme com o nome.
  3. Repita com 1 arquivo só.
- Esperado:
  - Passo 2: nada é removido. A tela mostra `Não removeu: 2 arquivos ainda não subiram.`.
  - Passo 3: `Não removeu: 1 arquivo ainda não subiu.`.
- Depois: destranque `teste-b`, espere o envio e tranque. Volte a VFS ao padrão.
- [ ] Passou
- Resultado:

#### T-116 · PENDENTE · Editar e Remover no modo só leitura

- Demandas: 033, 031
- Passos:
  1. Prepare o modo só leitura como no T-080 (com backup e `Como desfazer`).
  2. Olhe `Editar` nos cards.
- Esperado: `Editar` e `Remover cofre` desabilitados, como `＋  Adicionar Cofre`. Nada no `rclone.conf` muda.
- Como desfazer: o mesmo do T-080.
- [ ] Passou
- [ ] Ambiente restaurado
- Resultado:

#### T-117 · PENDENTE · Renomear

- Demandas: 033
- Passos:
  1. `teste-a` trancado → `Editar` → nome `teste-a-novo` → `Salvar`.
  2. Destranque `teste-a-novo`.
  3. Crie um cofre novo chamado `teste-a`.
- Esperado:
  - Passo 1: o card passa a se chamar `teste-a-novo`. No `rclone.conf`, as seções continuam `[teste-a]` e `[teste-a_base]`.
  - Passo 2: destranca com a mesma senha e mostra os mesmos arquivos.
  - Passo 3: é recusado, porque o remoto `teste-a` ainda existe. Frase em **proposta**: `Esse nome ainda é usado pelo cofre teste-a-novo. Escolha outro.`
- [ ] Passou
- Resultado:

---

## 12. Cobertura por demanda

Só entram as demandas com PR de código na `master`. As que ainda não têm código (004, 005, 014, 015 e 019) não têm teste de tela.

| Demanda | Testes | O que fica fora da tela |
|---|---|---|
| 001 rastreio de montagem | T-042, T-043, T-064, T-066, T-090 | — |
| 002 desmontagem verificável | T-044, T-066 | O rclone que não morre com Interrupt e Kill só é testado no CI |
| 003 falha de montagem visível | T-070, T-088 (falhas que aparecem rápido, com motivo) | O passo do "pronto quando" (`vfs_cache_mode = invalido`) não dá mais para fazer na tela: desde a 011 o painel recusa o valor (T-050), e um `vfs.json` inválido volta ao padrão (T-086) |
| 006 criação não sobrescreve | T-021, T-022, T-023 | — |
| 007 vaults.json corrompido | T-001, T-085 | — |
| 008 tempo limite | — | Não há como fazer o rclone travar pela tela. Fica com os testes do CI |
| 009 erros engolidos | T-031 | — |
| 010 saúde da montagem | T-043, T-061 | `Caiu: a unidade V:\ sumiu` e `… não respondeu` não têm como ser provocados à mão no Windows |
| 011 validação VFS | T-050, T-051, T-060, T-067 | — |
| 012 persistência VFS | T-052, T-086 | — |
| 013 Linux e macOS | T-090 | macOS sem teste: não há máquina |
| 016 testes do core | — | Só CI |
| 017 lógica no core | T-020, T-022 | O resto é interno |
| 018 estados do cofre | T-010, T-021, T-022, T-041, T-042, T-061, T-070, T-088 | — |
| 020 OAuth `Wait` duplo | T-021 | `Iniciar` duas vezes seguidas só no CI |
| 021 nomes no seletor | T-030, T-032 | — |
| 022 senha só sai trancado | T-044, T-046, T-063 | T-046 pode não ter como falhar no Windows |
| 023 seletor na thread da tela | T-030 (a janela não trava) | O resto é interno, com `-race` no CI |
| 024 cópias `.corrompido` | T-085 | — |
| 025 trancar espera envio | T-045, T-063, T-064, T-066 | — |
| 026 cofre que caiu | T-061, T-062, T-063, T-065 | — |
| 027 erros do rclone em português | T-011, T-030, T-031, T-070, T-071, T-088 | `senha errada`, `autorização expirou` e `a pasta não existe no …` não têm roteiro seguro na tela. `autorização expirou` entra no T-105 quando a 032 existir |
| 028 sair com cofre que caiu | T-064, T-065 | Falha de `Enviar agora` (`Não deu para enviar`) não tem roteiro seguro |
| 029 hífen no `config create` | T-022, T-032 e toda criação ou importação da sessão | O hífen na senha ofuscada sai em 1 de cada 64 criações; a tela não mostra se saiu. A prova é o teste de 500 senhas do CI |
| 030 senha errada | T-040, T-041, T-042, T-082, T-083, T-084 | — |
| 031 pasta de configuração | T-001, T-002, T-003, T-010, T-052, T-071, T-075, T-080, T-081, T-087 | — |
| 032 app próprio do Google | T-101 a T-105 | Pendente de implementação |
| 033 editar cofre | T-111 a T-117 | Pendente de implementação |

## Diferenças de texto conhecidas

Desde o PR "textos e assistente" (#49), as mensagens da `master` têm acento. Este roteiro espera os textos novos. Mudaram além do acento:

| Onde | Antes | Agora |
|---|---|---|
| Criar sem rclone (T-082) | `RClone nao disponivel.` | `O rclone não está instalado.` |
| Verificar WinFsp sem WinFsp (T-088) | `WinFsp nao encontrado. Necessario para montar unidades virtuais.` e `Baixe em: …` | `Não montou: falta instalar o WinFsp.` com o botão `Baixar WinFsp` |
| Unidade que não fica pronta (o roteiro não provoca) | `Timeout: A unidade nao ficou pronta em …` | `Tempo esgotado: a unidade não ficou pronta em …` |

Se um texto aparecer sem acento, anote em `Resultado:`: é um erro.
