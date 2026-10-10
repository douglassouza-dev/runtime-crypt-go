<#
.SYNOPSIS
  Monta a pasta de teste do roteiro (docs/testes). So cria pastas e copia arquivos.

.DESCRIPTION
  Cria <Base> (padrao: %USERPROFILE%\RuntimeCrypto-teste) com:
    app\            o executavel novo + rclone.exe + (se houver) o vaults.json antigo,
                    no mesmo lugar em que ele ficava antes da demanda 031
    app-limpo\      so o executavel (sem rclone.exe e sem vaults.json)
    appdata\        faz o papel de %APPDATA% para o app de teste
    appdata-vazio\  %APPDATA% de teste sem nada (janela vazia)
    rclone\         rclone.conf (copia do seu), vazio.conf e uma pasta no lugar de um .conf
    backup\         copias de seguranca (backup.ps1)
  Nao apaga nada. Nao mexe no %APPDATA% de verdade nem no seu rclone.conf: so le e copia.
  Se <Base> ja existe, para sem fazer nada.

.PARAMETER Exe
  Caminho do runtime-crypt-go.exe compilado da master.
.PARAMETER PastaAntiga
  Pasta da instalacao antiga (onde fica o vaults.json de antes da 031). Opcional.

.EXAMPLE
  .\preparar.ps1 -Exe C:\src\runtime-crypt-go\runtime-crypt-go.exe -PastaAntiga 'C:\Program Files\RuntimeCrypto'
#>
param(
    [Parameter(Mandatory = $true)][string]$Exe,
    [string]$PastaAntiga = '',
    [string]$Base = (Join-Path $env:USERPROFILE 'RuntimeCrypto-teste')
)
$ErrorActionPreference = 'Stop'

if (Test-Path -LiteralPath $Base) {
    throw "A pasta $Base ja existe. Escolha outra com -Base ou mova a antiga para outro lugar. Este script nao apaga nada."
}
if (-not (Test-Path -LiteralPath $Exe -PathType Leaf)) {
    throw "Nao achei o executavel: $Exe"
}

foreach ($p in 'app', 'app-limpo', 'appdata', 'appdata-vazio', 'rclone', 'rclone\pasta-no-lugar-do-conf', 'backup') {
    New-Item -ItemType Directory -Path (Join-Path $Base $p) | Out-Null
}
Copy-Item -LiteralPath $Exe -Destination (Join-Path $Base 'app\runtime-crypt-go.exe')
Copy-Item -LiteralPath $Exe -Destination (Join-Path $Base 'app-limpo\runtime-crypt-go.exe')

# rclone: o da pasta antiga; se nao houver, o do PATH. So vai para app\.
$rclone = $null
if ($PastaAntiga -and (Test-Path -LiteralPath (Join-Path $PastaAntiga 'rclone.exe'))) {
    $rclone = Join-Path $PastaAntiga 'rclone.exe'
} else {
    $cmd = Get-Command rclone -ErrorAction SilentlyContinue
    if ($cmd) { $rclone = $cmd.Source }
}
if (-not $rclone) { throw 'Nao achei o rclone.exe (nem na pasta antiga, nem no PATH).' }
Copy-Item -LiteralPath $rclone -Destination (Join-Path $Base 'app\rclone.exe')
Write-Host "rclone copiado de: $rclone"

# vaults.json antigo: vai para app\, onde o app antigo o deixava.
$antigo = if ($PastaAntiga) { Join-Path $PastaAntiga 'vaults.json' } else { '' }
if ($antigo -and (Test-Path -LiteralPath $antigo)) {
    Copy-Item -LiteralPath $antigo -Destination (Join-Path $Base 'app\vaults.json')
    $h = (Get-FileHash -LiteralPath (Join-Path $Base 'app\vaults.json') -Algorithm SHA256).Hash
    Write-Host "vaults.json antigo copiado para app\. SHA256 (anote para o T-001): $h"
} else {
    Write-Host 'Sem vaults.json antigo: o T-001 vai ser pulado (anote isso nele).'
}

# rclone.conf: copia do que o rclone usa hoje. Se nao existe, um arquivo vazio.
$saida = & $rclone config file
$confReal = ($saida | Select-Object -Last 1).Trim()
$confTeste = Join-Path $Base 'rclone\rclone.conf'
if ($confReal -and (Test-Path -LiteralPath $confReal -PathType Leaf)) {
    Copy-Item -LiteralPath $confReal -Destination $confTeste
    Write-Host "rclone.conf copiado de: $confReal"
} else {
    New-Item -ItemType File -Path $confTeste | Out-Null
    Write-Host 'Sem rclone.conf: criei um vazio para os testes.'
}
New-Item -ItemType File -Path (Join-Path $Base 'rclone\vazio.conf') | Out-Null

& (Join-Path $PSScriptRoot 'backup.ps1') -Base $Base
Write-Host ""
Write-Host "Pronto: $Base"
Write-Host 'Abra o app com: .\abrir.ps1'
