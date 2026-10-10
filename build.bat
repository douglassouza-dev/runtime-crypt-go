@echo off
setlocal enabledelayedexpansion

chcp 65001 >nul

echo ===================================================
echo   Compilacao do RuntimeCrypto
echo ===================================================

set NOME_BINARIO=runtime-crypt-go.exe
set FLAGS_LINK=-H windowsgui -s -w

if "%1"=="debug" (
    echo [INFO] Modo depuracao ativado: console habilitado.
    set FLAGS_LINK=-s -w
)
if "%1"=="console" (
    echo [INFO] Modo console ativado: terminal habilitado.
    set FLAGS_LINK=-s -w
)

where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERRO] Compilador Go nao encontrado no PATH.
    echo Certifique-se de que o Go esta instalado e configurado.
    exit /b 1
)

where gcc >nul 2>&1
if %errorlevel% neq 0 (
    echo [AVISO] Compilador GCC nao encontrado. O Fyne v2 necessita de CGO e GCC.
)

echo [1/3] Sincronizando dependencias (go mod tidy)...
go mod tidy
if %errorlevel% neq 0 (
    echo [ERRO] Falha ao sincronizar as dependencias.
    exit /b %errorlevel%
)

set LIMPAR_SYSO=0
where windres >nul 2>&1
if %errorlevel% equ 0 (
    if exist "assets\icone.ico" (
        echo [2/3] Incorporando icone do aplicativo via windres...
        echo 1 ICON "assets/icone.ico" > icone_recurso.rc
        windres -i icone_recurso.rc -O coff -o icone_recurso.syso
        del icone_recurso.rc >nul 2>&1
        set LIMPAR_SYSO=1
    )
)
if "!LIMPAR_SYSO!"=="0" (
    echo [2/3] Pulando incorporacao de recurso.
)

echo [3/3] Compilando executavel !NOME_BINARIO!...
set CGO_ENABLED=1
go build -o "!NOME_BINARIO!" -ldflags="!FLAGS_LINK!" .
set CODIGO_SAIDA=%errorlevel%

if "!LIMPAR_SYSO!"=="1" (
    if exist "icone_recurso.syso" del icone_recurso.syso >nul 2>&1
)

if %CODIGO_SAIDA% neq 0 (
    echo [ERRO] Falha na compilacao do executavel.
    exit /b %CODIGO_SAIDA%
)

echo ===================================================
echo   Compilacao concluida com sucesso!
echo   Arquivo gerado: !NOME_BINARIO!
echo ===================================================

exit /b 0