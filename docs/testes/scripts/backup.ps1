<#
.SYNOPSIS
  Copia de seguranca antes dos testes (docs/testes). So copia; nao apaga nem muda nada.

.DESCRIPTION
  Cria a pasta <Base>\backup\<data-hora> e copia para ela, quando existirem:
    - <Base>\appdata\RuntimeCrypto     (configuracao do app de teste)
    - <Base>\rclone\rclone.conf         (rclone.conf de teste)
    - %APPDATA%\RuntimeCrypto           (configuracao do app de verdade)
    - %APPDATA%\rclone\rclone.conf      (rclone.conf de verdade, no lugar padrao)
  Se voce usa RCLONE_CONFIG com outro caminho no dia a dia, copie esse arquivo a mao.

.EXAMPLE
  .\backup.ps1
#>
param(
    [string]$Base = (Join-Path $env:USERPROFILE 'RuntimeCrypto-teste')
)
$ErrorActionPreference = 'Stop'

$destino = Join-Path $Base ('backup\' + (Get-Date -Format 'yyyyMMdd-HHmmss'))
New-Item -ItemType Directory -Path $destino -Force | Out-Null

# A pasta AppData de verdade vem do Windows, nao da variavel APPDATA (que o
# abrir.ps1 troca so no processo do app).
$appdataReal = [Environment]::GetFolderPath('ApplicationData')
$itens = @(
    @{ De = (Join-Path $Base 'appdata\RuntimeCrypto'); Para = 'teste-RuntimeCrypto' },
    @{ De = (Join-Path $Base 'rclone\rclone.conf');    Para = 'teste-rclone.conf' }
)
if ($appdataReal) {
    $itens += @{ De = (Join-Path $appdataReal 'RuntimeCrypto');      Para = 'real-RuntimeCrypto' }
    $itens += @{ De = (Join-Path $appdataReal 'rclone\rclone.conf'); Para = 'real-rclone.conf' }
} else {
    Write-Host 'Nao achei a pasta AppData de verdade: copie %AppData%\RuntimeCrypto e o rclone.conf a mao.'
}
foreach ($i in $itens) {
    if (Test-Path -LiteralPath $i.De) {
        Copy-Item -LiteralPath $i.De -Destination (Join-Path $destino $i.Para) -Recurse
        Write-Host "copiado: $($i.De)"
    } else {
        Write-Host "nao existe (nada a copiar): $($i.De)"
    }
}
Write-Host ""
Write-Host "Backup em: $destino"
