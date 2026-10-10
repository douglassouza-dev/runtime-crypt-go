# Roteiro de testes em tela

Testes para fazer com o programa aberto, olhando a janela, a bandeja e o Explorador. Eles cobrem as demandas que já entraram na `master` e, numa seção à parte, as demandas 032 e 033, que ainda não foram implementadas.

| Arquivo | Conteúdo |
|---|---|
| [roteiro.md](roteiro.md) | Os testes, na ordem de uma sessão do começo ao fim, e a tabela demanda → testes |
| [scripts/](scripts/) | Scripts de PowerShell que preparam a pasta de teste, abrem o app em cada modo e fazem backup |

## Como usar

- Siga o [roteiro.md](roteiro.md) de cima para baixo. A ordem importa: um teste prepara o seguinte.
- Cada teste tem um ID (`T-001`), as demandas que ele confere, a preparação, os passos e o resultado esperado.
- O texto entre crases é o texto exato da tela na `master`. Se a tela mostrar outro texto, mesmo com uma letra de diferença, é falha.
- Marque `[x]` quando passar. Escreva em `Resultado:` o que viu, ou cole um print.
- Os testes que mexem no ambiente (pasta sem gravação, rclone ausente, WinFsp desinstalado, arquivos corrompidos) ficam todos no fim, na seção 9. Cada um tem `Como desfazer` e uma conferência de que o ambiente voltou.
- A seção 11 (032 e 033) é para depois. Hoje esses testes falham, porque as telas não existem.

## O que precisa antes

**Sistema.** O alvo principal é Windows 10 ou 11 com o [WinFsp](https://winfsp.dev/rel/) instalado. Os scripts são de Windows.

- **Linux:** precisa do FUSE 3 (`sudo apt install fuse3`). O cofre aparece em `~/RuntimeCrypto/<nome>` e não numa letra. A pasta de configuração é `~/.config/RuntimeCrypto`. Os scripts não rodam no Linux. O teste T-090 é o único só para Linux.
- **macOS:** precisa do macFUSE. A pasta de configuração é `~/Library/Application Support/RuntimeCrypto`. Ninguém testou o app no macOS ainda.

**rclone.** O rclone 1.60 ou mais novo, ao lado do executável ou no `PATH`.

**Conta de nuvem.** Use uma conta do Google Drive descartável, sem nada importante.

- `Criar Cofre` grava o cofre na raiz do Drive: a tela de sucesso diz `Pasta: /`. Não há como escolher a pasta ao criar.
- No Drive da conta de teste, crie estas pastas na raiz, vazias: `alfa`, `minha pasta`, `ação`, `2024 fotos`, `-rascunho` e `vazia`. Elas servem para o seletor de pasta.

**Pasta Local e S3 não servem para estes testes.** Os dois assistentes não mostram `Amazon S3 / MinIO` nem `Pasta Local` até funcionarem (demandas 014 e 034).

Por isso os testes usam só o Google Drive.

**Feche o RuntimeCrypto de verdade** (bandeja → `Sair`) antes de começar. Dois apps abertos ao mesmo tempo disputam as letras e o rclone.

## Compilar a master

No Windows, com Go (1.22 ou mais novo; o `go.mod` pede 1.26.4 e o Go baixa sozinho) e um GCC (MinGW-w64, por exemplo o do MSYS2), porque a interface usa CGO:

```powershell
git clone https://github.com/douglassouza-dev/runtime-crypt-go.git
cd runtime-crypt-go
git checkout master
git pull
git rev-parse --short HEAD   # anote o commit no topo do roteiro
go build -o runtime-crypt-go.exe -ldflags="-H windowsgui" .
```

## A pasta de teste

Os testes rodam numa pasta separada, `%USERPROFILE%\RuntimeCrypto-teste`. O seu `%AppData%\RuntimeCrypto` e o seu `rclone.conf` de verdade não são usados: o app de teste recebe cópias.

Os scripts ficam em [scripts/](scripts/). O Windows bloqueia scripts por padrão, então rode cada um com `powershell -ExecutionPolicy Bypass -File`:

| Script | O que faz | O que nunca faz |
|---|---|---|
| `preparar.ps1 -Exe <exe> -PastaAntiga <pasta>` | Cria `RuntimeCrypto-teste` e copia para lá o executável novo, o `rclone.exe`, o `vaults.json` antigo e o seu `rclone.conf`. No fim chama o `backup.ps1` | Apagar ou mudar qualquer arquivo fora da pasta de teste. Se a pasta já existe, para sem fazer nada |
| `abrir.ps1 [-Modo …]` | Abre o app de teste. Ele aponta `APPDATA` e `RCLONE_CONFIG` para a pasta de teste. Os outros modos simulam falhas só mudando variáveis de ambiente | Gravar ou apagar arquivos. O seu PowerShell volta como estava |
| `backup.ps1` | Copia a configuração de teste e a de verdade para `RuntimeCrypto-teste\backup\<data-hora>` | Apagar |
| `somente-leitura.ps1 [-Desfazer]` | Nega (ou devolve) a gravação na pasta de configuração de teste, com `icacls` | Tocar no `%AppData%\RuntimeCrypto` de verdade ou apagar |

Dentro da pasta de teste:

| Pasta | Papel |
|---|---|
| `app\` | Executável novo, `rclone.exe` e o `vaults.json` antigo, no lugar onde ele ficava antes da 031 |
| `app-limpo\` | Só o executável |
| `appdata\RuntimeCrypto\` | O que o app trata como `%AppData%\RuntimeCrypto`: `vaults.json`, `vfs.json` e `runtimecrypto.log` |
| `rclone\rclone.conf` | Cópia do seu `rclone.conf`; os cofres de teste entram aqui |
| `backup\` | Backups |

Para usar o `rclone` na linha de comando com a configuração de teste, abra um PowerShell e rode antes:

```powershell
$env:RCLONE_CONFIG = "$env:USERPROFILE\RuntimeCrypto-teste\rclone\rclone.conf"
```

## Onde fica o log

Depois da demanda 031, o log é `runtimecrypto.log`, na pasta de configuração do usuário. Ao passar de 1 MB, ele vira `runtimecrypto.log.1` e começa um novo.

- App de teste: `%USERPROFILE%\RuntimeCrypto-teste\appdata\RuntimeCrypto\runtimecrypto.log`
- App de verdade: `%AppData%\RuntimeCrypto\runtimecrypto.log`

No modo só leitura (seção 9), o app não grava log.

## Quando um teste falhar

Mande no grupo `runtime-crypt-go`:

1. o ID do teste e o passo (por exemplo, `T-030, passo 3`);
2. um print da tela;
3. o que você esperava e o que apareceu;
4. as últimas linhas do `runtimecrypto.log`. Antes de mandar, confira que não há senha nele.

Siga para o próximo teste, a menos que ele dependa do que falhou.
