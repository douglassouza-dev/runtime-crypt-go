<#
.SYNOPSIS
  Abre o RuntimeCrypto de teste (docs/testes) com o ambiente de cada teste.

.DESCRIPTION
  So muda variaveis de ambiente do processo do app. Nada e gravado nem apagado,
  e o seu PowerShell volta como estava. Todos os modos usam
  RCLONE_CONFIG=<Base>\rclone\rclone.conf e APPDATA=<Base>\appdata, salvo:
    Normal         app\runtime-crypt-go.exe
    Vazio          app-limpo\, APPDATA=<Base>\appdata-vazio (janela sem cofres)
    SemCaminho     app-limpo\, sem APPDATA (o app nao sabe a pasta de configuracao)
    SemRclone      app-limpo\, PATH sem nenhuma pasta que tenha rclone.exe
    ConfigVazia    RCLONE_CONFIG=<Base>\rclone\vazio.conf
    ConfigIlegivel RCLONE_CONFIG=<Base>\rclone\pasta-no-lugar-do-conf (uma pasta)

.EXAMPLE
  .\abrir.ps1
  .\abrir.ps1 -Modo SemRclone
#>
param(
    [ValidateSet('Normal', 'Vazio', 'SemCaminho', 'SemRclone', 'ConfigVazia', 'ConfigIlegivel')]
    [string]$Modo = 'Normal',
    [string]$Base = (Join-Path $env:USERPROFILE 'RuntimeCrypto-teste')
)
$ErrorActionPreference = 'Stop'

if (-not (Test-Path -LiteralPath $Base)) { throw "Nao achei $Base. Rode antes o preparar.ps1." }
if (Get-Process -Name 'runtime-crypt-go' -ErrorAction SilentlyContinue) {
    throw 'O RuntimeCrypto ja esta aberto. Feche pela bandeja (Sair) e rode de novo.'
}

$nomes = 'APPDATA', 'RCLONE_CONFIG', 'PATH'
$antes = @{}
foreach ($n in $nomes) { $antes[$n] = [Environment]::GetEnvironmentVariable($n, 'Process') }

try {
    $exe = Join-Path $Base 'app\runtime-crypt-go.exe'
    $limpo = Join-Path $Base 'app-limpo\runtime-crypt-go.exe'
    [Environment]::SetEnvironmentVariable('APPDATA', (Join-Path $Base 'appdata'), 'Process')
    [Environment]::SetEnvironmentVariable('RCLONE_CONFIG', (Join-Path $Base 'rclone\rclone.conf'), 'Process')

    switch ($Modo) {
        'Vazio' {
            $exe = $limpo
            [Environment]::SetEnvironmentVariable('APPDATA', (Join-Path $Base 'appdata-vazio'), 'Process')
        }
        'SemCaminho' {
            $exe = $limpo
            [Environment]::SetEnvironmentVariable('APPDATA', $null, 'Process')
        }
        'SemRclone' {
            $exe = $limpo
            $semRclone = ($antes['PATH'] -split ';') | Where-Object {
                $_ -and -not (Test-Path -LiteralPath (Join-Path $_ 'rclone.exe') -ErrorAction SilentlyContinue)
            }
            [Environment]::SetEnvironmentVariable('PATH', ($semRclone -join ';'), 'Process')
        }
        'ConfigVazia' {
            [Environment]::SetEnvironmentVariable('RCLONE_CONFIG', (Join-Path $Base 'rclone\vazio.conf'), 'Process')
        }
        'ConfigIlegivel' {
            [Environment]::SetEnvironmentVariable('RCLONE_CONFIG', (Join-Path $Base 'rclone\pasta-no-lugar-do-conf'), 'Process')
        }
    }
    Start-Process -FilePath $exe -WorkingDirectory (Split-Path -Parent $exe)
    Write-Host "Aberto no modo $Modo ($exe)"
} finally {
    foreach ($n in $nomes) { [Environment]::SetEnvironmentVariable($n, $antes[$n], 'Process') }
}
