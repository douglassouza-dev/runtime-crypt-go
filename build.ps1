<#
.SYNOPSIS
    Script de compilação do RuntimeCrypto para Windows.
.DESCRIPTION
    Compila o projeto Go gerando o executável com ou sem console e incorporando o ícone do sistema.
.PARAMETER ModoConsole
    Se especificado, compila sem a flag -H windowsgui, permitindo ver saídas no terminal.
.PARAMETER Limpar
    Remove os binários compilados anteriormente antes de iniciar.
#>
[CmdletBinding()]
param(
    [switch]$ModoConsole,
    [switch]$Limpar
)

$ErrorActionPreference = "Stop"

$NomeBinario = "runtime-crypt-go.exe"
$CaminhoRaiz = $PSScriptRoot

Set-Location $CaminhoRaiz

Write-Host "===================================================" -ForegroundColor Cyan
Write-Host "  Compilação do RuntimeCrypto (PowerShell)" -ForegroundColor Cyan
Write-Host "===================================================" -ForegroundColor Cyan

if ($Limpar) {
    Write-Host "[INFO] Limpando arquivos anteriores..." -ForegroundColor Yellow
    if (Test-Path $NomeBinario) { Remove-Item $NomeBinario -Force }
    if (Test-Path "*.syso") { Remove-Item "*.syso" -Force }
}

# Verificar se Go está instalado
if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
    Write-Error "[ERRO] Go não encontrado no PATH."
    exit 1
}

# Verificar se GCC está instalado (necessário para CGO no Fyne)
if (-not (Get-Command "gcc" -ErrorAction SilentlyContinue)) {
    Write-Warning "[AVISO] Compilador GCC não encontrado no PATH. O Fyne necessita do GCC para CGO."
}

# 1. Sincronizar módulos
Write-Host "[1/3] Sincronizando dependências (go mod tidy)..." -ForegroundColor Gray
go mod tidy

# 2. Incorporação do ícone via windres se disponível
$GerouSyso = $false
$ArquivoSyso = Join-Path $CaminhoRaiz "recurso_icone.syso"
$ArquivoRc = Join-Path $CaminhoRaiz "recurso_icone.rc"

if ((Get-Command "windres" -ErrorAction SilentlyContinue) -and (Test-Path "assets\icone.ico")) {
    Write-Host "[2/3] Gerando recurso de ícone com windres..." -ForegroundColor Gray
    try {
        Set-Content -Path $ArquivoRc -Value '1 ICON "assets/icone.ico"' -Encoding ASCII
        & windres -i $ArquivoRc -O coff -o $ArquivoSyso
        $GerouSyso = $true
    } catch {
        Write-Warning "[AVISO] Não foi possível gerar o recurso de ícone: $_"
    } finally {
        if (Test-Path $ArquivoRc) { Remove-Item $ArquivoRc -Force }
    }
} else {
    Write-Host "[2/3] Pulando geração de ícone (windres não encontrado)..." -ForegroundColor Gray
}

# 3. Compilar
$FlagsLink = "-s -w"
if (-not $ModoConsole) {
    $FlagsLink += " -H windowsgui"
    Write-Host "[3/3] Compilando em modo GUI silencioso (sem console)..." -ForegroundColor Gray
} else {
    Write-Host "[3/3] Compilando em modo console (logs visíveis)..." -ForegroundColor Gray
}

$env:CGO_ENABLED = "1"

try {
    & go build -o $NomeBinario -ldflags="$FlagsLink" .
    if ($LASTEXITCODE -ne 0) {
        throw "Erro na execução do comando go build."
    }
} finally {
    if ($GerouSyso -and (Test-Path $ArquivoSyso)) {
        Remove-Item $ArquivoSyso -Force
    }
}

if (Test-Path $NomeBinario) {
    $Info = Get-Item $NomeBinario
    $TamanhoMB = [math]::Round($Info.Length / 1MB, 2)
    Write-Host "===================================================" -ForegroundColor Green
    Write-Host "  Compilação concluída com sucesso!" -ForegroundColor Green
    Write-Host "  Executável: $($Info.Name) ($TamanhoMB MB)" -ForegroundColor Green
    Write-Host "===================================================" -ForegroundColor Green
} else {
    Write-Error "[ERRO] O executável não foi gerado."
    exit 1
}
