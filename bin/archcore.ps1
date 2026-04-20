# archcore shim (Windows) — resolves and execs the real archcore CLI.
#
# Resolution order (same as POSIX shim):
#   1. $env:ARCHCORE_BIN override (if set and exists)
#   2. `archcore` in PATH
#   3. bin\vendor\archcore-v<VERSION>.exe (local cache)
#   4. Download from GitHub Releases into vendor\, verify, cache, use
#
# Env:
#   ARCHCORE_BIN            — explicit path to the CLI binary
#   ARCHCORE_SKIP_DOWNLOAD  — if "1", skip step 4 and exit 1 instead

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$ScriptDir = $PSScriptRoot
$Self = Join-Path $ScriptDir 'archcore.ps1'

function Invoke-Archcore {
    param([string]$BinPath, [string[]]$ArgV)
    & $BinPath @ArgV
    exit $LASTEXITCODE
}

function Write-ShimError {
    param([string]$Message)
    [Console]::Error.WriteLine("[archcore shim] $Message")
}

# --- 1. $env:ARCHCORE_BIN override ---
$bin = $env:ARCHCORE_BIN
if ($bin -and (Test-Path -LiteralPath $bin -PathType Leaf)) {
    Invoke-Archcore -BinPath $bin -ArgV $args
}

# --- 2. archcore in PATH ---
$cmd = Get-Command archcore -ErrorAction SilentlyContinue
if ($cmd) {
    # Loop guard: if it resolves to ourselves (unlikely but possible with PATH shenanigans), fall through
    try {
        $resolved = (Resolve-Path -LiteralPath $cmd.Source).Path
        $selfResolved = (Resolve-Path -LiteralPath $Self).Path
        if ($resolved -ne $selfResolved) {
            Invoke-Archcore -BinPath $cmd.Source -ArgV $args
        }
    } catch {
        Invoke-Archcore -BinPath $cmd.Source -ArgV $args
    }
}

# --- 3. Vendor cache ---
$versionFile = Join-Path $ScriptDir 'CLI_VERSION'
if (-not (Test-Path -LiteralPath $versionFile)) {
    Write-ShimError 'bin\CLI_VERSION not found'
    exit 1
}
$version = (Get-Content -LiteralPath $versionFile -Raw).Trim()

$vendorDir = Join-Path $ScriptDir 'vendor'
$vendorBin = Join-Path $vendorDir "archcore-v$version.exe"

if (Test-Path -LiteralPath $vendorBin -PathType Leaf) {
    Invoke-Archcore -BinPath $vendorBin -ArgV $args
}

# --- 4. Download ---
if ($env:ARCHCORE_SKIP_DOWNLOAD -eq '1') {
    Write-ShimError "archcore v$version not cached and ARCHCORE_SKIP_DOWNLOAD=1. Set ARCHCORE_BIN or unset ARCHCORE_SKIP_DOWNLOAD."
    exit 1
}

# Arch detection — use OS architecture so x64 PowerShell under Prism emulation
# still installs the correct ARM64 binary.
$rawArch = $null
try {
    $rawArch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
} catch {
    if ($env:PROCESSOR_ARCHITEW6432) { $rawArch = $env:PROCESSOR_ARCHITEW6432 }
    elseif ($env:PROCESSOR_ARCHITECTURE) { $rawArch = $env:PROCESSOR_ARCHITECTURE }
}
$arch = switch ($rawArch.ToUpper()) {
    'X64'   { 'amd64'; break }
    'AMD64' { 'amd64'; break }
    'ARM64' { 'arm64'; break }
    default { $null }
}
if (-not $arch) {
    Write-ShimError "Unsupported architecture: $rawArch"
    exit 1
}

$archiveName = "archcore_windows_${arch}.zip"
$baseUrl = "https://github.com/archcore-ai/cli/releases/download/v$version"
$archiveUrl = "$baseUrl/$archiveName"
$checksumsUrl = "$baseUrl/checksums.txt"

$tmpDir = Join-Path $env:TEMP "archcore-shim-$([guid]::NewGuid())"
New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

try {
    [Console]::Error.WriteLine("[archcore shim] Downloading archcore v$version (windows/$arch)...")

    $archivePath = Join-Path $tmpDir $archiveName
    $checksumsPath = Join-Path $tmpDir 'checksums.txt'

    $headers = @{ 'User-Agent' = 'archcore-shim' }

    # Download archive with retry
    $attempts = 0
    $ok = $false
    while ($attempts -lt 3 -and -not $ok) {
        $attempts++
        try {
            Invoke-WebRequest -UseBasicParsing -Uri $archiveUrl -Headers $headers -OutFile $archivePath
            $ok = $true
        } catch {
            if ($attempts -ge 3) {
                Write-ShimError "Failed to download $archiveUrl — check network connectivity or set ARCHCORE_BIN."
                exit 1
            }
            Start-Sleep -Seconds 2
        }
    }

    # Download checksums
    try {
        Invoke-WebRequest -UseBasicParsing -Uri $checksumsUrl -Headers $headers -OutFile $checksumsPath
    } catch {
        Write-ShimError "Failed to download checksums.txt from $baseUrl"
        exit 1
    }

    # Verify checksum
    $expected = $null
    foreach ($line in Get-Content -LiteralPath $checksumsPath) {
        $parts = $line -split '\s+'
        if ($parts.Count -ge 2 -and $parts[1] -ieq $archiveName) {
            $expected = $parts[0]
            break
        }
    }
    if (-not $expected) {
        Write-ShimError "No checksum for $archiveName in checksums.txt"
        exit 1
    }

    $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash
    if ($actual.ToUpper() -ne $expected.ToUpper()) {
        Write-ShimError "Checksum mismatch for $archiveName (expected $expected, got $actual)"
        exit 1
    }

    # Extract
    Expand-Archive -LiteralPath $archivePath -DestinationPath $tmpDir -Force

    $extractedBinary = Join-Path $tmpDir 'archcore.exe'
    if (-not (Test-Path -LiteralPath $extractedBinary)) {
        # GoReleaser fallback: binary may be named "cli.exe"
        $fallback = Join-Path $tmpDir 'cli.exe'
        if (Test-Path -LiteralPath $fallback) {
            Move-Item -LiteralPath $fallback -Destination $extractedBinary -Force
        } else {
            Write-ShimError 'archcore.exe not found in archive'
            exit 1
        }
    }

    # Atomic install into vendor/
    New-Item -ItemType Directory -Path $vendorDir -Force | Out-Null
    $staged = "$vendorBin.tmp.$PID"
    Copy-Item -LiteralPath $extractedBinary -Destination $staged -Force
    # Strip MOTW ADS so SmartScreen doesn't block the binary
    Unblock-File -LiteralPath $staged
    Move-Item -LiteralPath $staged -Destination $vendorBin -Force
} finally {
    if (Test-Path -LiteralPath $tmpDir) {
        Remove-Item -Recurse -Force -LiteralPath $tmpDir -ErrorAction SilentlyContinue
    }
}

Invoke-Archcore -BinPath $vendorBin -ArgV $args
