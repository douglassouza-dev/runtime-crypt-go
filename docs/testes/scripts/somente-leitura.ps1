<#
.SYNOPSIS
  Tira (ou devolve) a permissao de gravar na pasta de configuracao do app de TESTE.

.DESCRIPTION
  Mexe so em <Base>\appdata\RuntimeCrypto (padrao: %USERPROFILE%\RuntimeCrypto-teste\...).
  Sem -Desfazer: acrescenta uma regra "negar gravacao" para o seu usuario nessa pasta
  (icacls /deny usuario:(W)). Os arquivos continuam la e continuam legiveis.
  Com -Desfazer: tira as regras de negar do seu usuario (icacls /remove:d usuario).
  No fim mostra as permissoes da pasta. Depois de desfazer, nao deve sobrar "(DENY)".
  Nao apaga nada. Nunca toca no %APPDATA%\RuntimeCrypto de verdade.

.EXAMPLE
  .\somente-leitura.ps1
  .\somente-leitura.ps1 -Desfazer
#>
param(
    [switch]$Desfazer,
    [string]$Base = (Join-Path $env:USERPROFILE 'RuntimeCrypto-teste')
)
$ErrorActionPreference = 'Stop'

$pasta = Join-Path $Base 'appdata\RuntimeCrypto'
if (-not (Test-Path -LiteralPath $pasta -PathType Container)) {
    throw "Nao achei $pasta. Abra o app de teste uma vez (abrir.ps1) antes."
}
$usuario = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name

if ($Desfazer) {
    & icacls $pasta /remove:d $usuario | Out-Host
} else {
    & icacls $pasta /deny "${usuario}:(W)" | Out-Host
}
if ($LASTEXITCODE -ne 0) { throw "icacls falhou (codigo $LASTEXITCODE)." }

Write-Host ''
Write-Host "Permissoes de ${pasta}:"
& icacls $pasta | Out-Host
