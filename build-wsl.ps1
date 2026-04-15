#!/usr/bin/env pwsh
#Requires -Version 5.1
<#
.SYNOPSIS
    Build Vigil (bbx) install packages and RPMs inside WSL.
.DESCRIPTION
    This script enters WSL, ensures Go and nfpm are installed,
    then runs build_all.sh and package.sh. The resulting tar.gz
    and RPM files will be in the release/ directory.
.EXAMPLE
    .\build-wsl.ps1
    .\build-wsl.ps1 -Version 2.1.0
#>

param(
    [string]$Version = "",
    [string]$ProjectRoot = "$PSScriptRoot"
)

# Resolve full path because /mnt/c/... needs an absolute path
$ProjectRoot = Resolve-Path $ProjectRoot | Select-Object -ExpandProperty Path

# Convert Windows path to WSL path (C:\Data\ancoo\vigil -> /mnt/c/Data/ancoo/vigil)
$WslRoot = ($ProjectRoot -replace '^([A-Za-z]):', '/mnt/$1' -replace '\\', '/').ToLower()

# Build the bash script content using a single-quoted here-string
# so PowerShell does not interpret any $ variables inside
$bashScriptTemplate = @'
set -e
cd "__WSLROOT__"

echo "=========================================="
echo " Building Vigil in WSL"
echo " Project root: __WSLROOT__"
echo "=========================================="

# ------------------ Install Go if missing ------------------
if ! command -v go >/dev/null 2>&1; then
    echo "[WSL] Go not found. Installing Go..."

    GO_VERSION="1.24.2"
    GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"

    if [ ! -f "/tmp/$GO_TARBALL" ]; then
        wget -q --show-progress "https://go.dev/dl/$GO_TARBALL" -O "/tmp/$GO_TARBALL" || true
    fi

    if [ -f "/tmp/$GO_TARBALL" ]; then
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf "/tmp/$GO_TARBALL"
    else
        echo "[WSL] Download failed, falling back to apt install..."
        sudo apt-get update
        sudo apt-get install -y golang-go
    fi
fi

# Ensure Go is on PATH for this session
export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH

echo "[WSL] Go version: $(go version)"

# ------------------ Install nfpm if missing ------------------
if ! command -v nfpm >/dev/null 2>&1; then
    echo "[WSL] nfpm not found. Installing nfpm..."
    go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
fi

export PATH=$PATH:$(go env GOPATH)/bin

echo "[WSL] nfpm version: $(nfpm --version)"

# ------------------ Clean previous builds ------------------
echo "[WSL] Cleaning previous builds..."
rm -rf pkg release

# ------------------ Build binaries ------------------
echo "[WSL] Running build_all.sh..."
bash ./build_all.sh

# ------------------ Package ------------------
echo "[WSL] Running package.sh..."
VERSION="__VERSION__" bash ./package.sh

echo ""
echo "=========================================="
echo " Build complete!"
echo " Output directory: __WSLROOT__/release"
echo "=========================================="
ls -lh "__WSLROOT__/release/"
'@

# Replace placeholders with actual values
$bashScript = $bashScriptTemplate.
    Replace('__WSLROOT__', $WslRoot).
    Replace('__VERSION__', $Version)

# Write bash script to a temp file and pass it to WSL
$TempFile = Join-Path $env:TEMP "build-vigil-wsl.sh"
$bashScript | Out-File -FilePath $TempFile -Encoding utf8 -NoNewline

# Convert temp file path to WSL path
$WslTempFile = ($TempFile -replace '^([A-Za-z]):', '/mnt/$1' -replace '\\', '/').ToLower()

Write-Host "[PowerShell] Entering WSL to build at: $WslRoot" -ForegroundColor Cyan

# Execute via temp file inside WSL
wsl bash "$WslTempFile"

$exitCode = $LASTEXITCODE
Remove-Item -Path $TempFile -ErrorAction SilentlyContinue

if ($exitCode -ne 0) {
    Write-Host "[PowerShell] Build failed with exit code $exitCode" -ForegroundColor Red
    exit $exitCode
}

Write-Host "[PowerShell] Done. Check $ProjectRoot\release for artifacts." -ForegroundColor Green
