#!/usr/bin/env pwsh
#Requires -Version 5.1
<#
.SYNOPSIS
    Build Vigil (bbx) install packages (tar.gz, RPM, DEB) inside WSL.
.DESCRIPTION
    This script enters WSL, ensures Go and nfpm are installed,
    then runs build_all.sh and package.sh. The resulting tar.gz
    and RPM files will be in the release/ directory.
.EXAMPLE
    .\build-wsl.ps1
    .\build-wsl.ps1 -Version 2.1.0
    .\build-wsl.ps1 -SudoPassword 'your_wsl_password'
#>

param(
    [string]$Version = "",
    [string]$ProjectRoot = "$PSScriptRoot",
    [string]$SudoPassword = ""
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

# Ensure Go binary path is present before we check
export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH

# Use China-accessible module proxy and disable toolchain auto-download
# (the installed Go version must already satisfy go.mod).
# GOSUMDB is set to off because sum.golang.org is not reachable from this network.
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=off
export GOTOOLCHAIN=local

# ------------------ Install Go if missing or version mismatch ------------------
GO_VERSION="1.26.4"
GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
GO_MIRRORS=(
    "https://go.dev/dl"
    "https://mirrors.aliyun.com/golang"
)

CURRENT_GO_VERSION=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')

if [ -z "$CURRENT_GO_VERSION" ] || [ "$CURRENT_GO_VERSION" != "$GO_VERSION" ]; then
    echo "[WSL] Installing Go $GO_VERSION (current: ${CURRENT_GO_VERSION:-none})..."

    download_go() {
        for mirror in "${GO_MIRRORS[@]}"; do
            echo "[WSL] Trying mirror: $mirror"
            rm -f "/tmp/$GO_TARBALL"
            if wget -q --timeout=60 --tries=2 "$mirror/$GO_TARBALL" -O "/tmp/$GO_TARBALL"; then
                if tar -tzf "/tmp/$GO_TARBALL" >/dev/null 2>&1; then
                    echo "[WSL] Downloaded valid tarball from $mirror"
                    return 0
                fi
                echo "[WSL] Tarball from $mirror is invalid"
            else
                echo "[WSL] Download from $mirror failed"
            fi
        done
        return 1
    }

    if ! download_go; then
        echo "[WSL] Falling back to apt install..."
        __SUDO__apt-get update
        __SUDO__apt-get install -y golang-go
    fi

    if [ -f "/tmp/$GO_TARBALL" ]; then
        __SUDO__rm -rf /usr/local/go
        __SUDO__tar -C /usr/local -xzf "/tmp/$GO_TARBALL"
    else
        echo "[WSL] No valid Go tarball available; apt install should have provided Go"
    fi
fi

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

# Handle sudo password for non-interactive installs in WSL.
# When a password is supplied, pipe it to sudo -S for each privileged command.
if ($SudoPassword) {
    $bashScript = $bashScript.Replace('__SUDO__', 'echo "$SUDO_PASSWORD" | sudo -S ')
} else {
    $bashScript = $bashScript.Replace('__SUDO__', 'sudo ')
}
# Write bash script to a temp file and pass it to WSL
$TempFile = Join-Path $env:TEMP "build-vigil-wsl.sh"
# PowerShell 5.1's Out-File -Encoding utf8 emits a UTF-8 BOM, which breaks
# bash. Write UTF-8 without BOM explicitly.
[System.IO.File]::WriteAllText($TempFile, $bashScript, [System.Text.UTF8Encoding]::new($false))

# Convert temp file path to WSL path
$WslTempFile = ($TempFile -replace '^([A-Za-z]):', '/mnt/$1' -replace '\\', '/').ToLower()

Write-Host "[PowerShell] Entering WSL to build at: $WslRoot" -ForegroundColor Cyan

# Execute via temp file inside WSL
if ($SudoPassword) {
    # Escape single quotes for the bash command line.
    $escapedPassword = $SudoPassword -replace "'", "'\''"
    wsl -e bash -c "SUDO_PASSWORD='$escapedPassword' bash '$WslTempFile'"
    # Clear the password from memory as soon as possible.
    $SudoPassword = $null
    $escapedPassword = $null
} else {
    wsl bash "$WslTempFile"
}

$exitCode = $LASTEXITCODE
Remove-Item -Path $TempFile -ErrorAction SilentlyContinue

if ($exitCode -ne 0) {
    Write-Host "[PowerShell] Build failed with exit code $exitCode" -ForegroundColor Red
    exit $exitCode
}

Write-Host "[PowerShell] Done. Check $ProjectRoot\release for artifacts." -ForegroundColor Green
