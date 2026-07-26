#Requires -Version 5.1
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# ── Constants ────────────────────────────────────────────────────────────────
$GITHUB_REPO    = 'archcore-ai/cli'
$BINARY_NAME    = 'archcore'

# ── Color / formatting (TTY-aware) ──────────────────────────────────────────
# `e escape only exists on PowerShell 6+; build ANSI sequences via [char]27 so
# Windows PowerShell 5.1 emits real escape codes instead of literal "`e[…".
$script:ESC      = [char]27
$script:UseColor = -not [Console]::IsOutputRedirected

# ── Logging helpers ─────────────────────────────────────────────────────────
function Write-Info {
    param([string]$Message)
    if ($script:UseColor) {
        Write-Host "$($script:ESC)[34m==>$($script:ESC)[0m $($script:ESC)[1m${Message}$($script:ESC)[0m"
    } else {
        Write-Host "==> $Message"
    }
}

function Write-Success {
    param([string]$Message)
    if ($script:UseColor) {
        Write-Host "$($script:ESC)[32m==>$($script:ESC)[0m $($script:ESC)[1m${Message}$($script:ESC)[0m"
    } else {
        Write-Host "==> $Message"
    }
}

function Write-WarnMsg {
    param([string]$Message)
    if ($script:UseColor) {
        Write-Host "$($script:ESC)[33mWarning:$($script:ESC)[0m $Message"
    } else {
        Write-Host "Warning: $Message"
    }
}

function Write-ErrExit {
    param([string]$Message)
    if ($script:UseColor) {
        $line = "$($script:ESC)[31mError:$($script:ESC)[0m $Message"
    } else {
        $line = "Error: $Message"
    }
    [Console]::Error.WriteLine($line)
    throw $Message
}

# ── Architecture detection ───────────────────────────────────────────────────
function Get-Arch {
    # Use host OS architecture (not process arch) so x64 PowerShell under
    # ARM64 Prism emulation still installs the correct ARM64 binary.
    $raw = $null
    try {
        $raw = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
    } catch {
        # Fallback cascade for older .NET / PS5.1 environments
        if ($env:PROCESSOR_ARCHITEW6432) {
            $raw = $env:PROCESSOR_ARCHITEW6432
        } elseif ($env:PROCESSOR_ARCHITECTURE) {
            $raw = $env:PROCESSOR_ARCHITECTURE
        }
    }

    if (-not $raw) {
        Write-ErrExit 'Could not determine system architecture.'
    }

    switch ($raw.ToUpper()) {
        'X64'   { return 'amd64' }
        'AMD64' { return 'amd64' }
        'ARM64' { return 'arm64' }
        default { Write-ErrExit "Unsupported architecture: $raw" }
    }
}

# ── Version resolution ──────────────────────────────────────────────────────
# Deliberately resolved through the github.com web redirect rather than
# api.github.com/repos/.../releases/latest. The REST API allows only 60
# unauthenticated requests per hour *per IP*, so every user behind a shared
# egress address — corporate NAT, CGNAT, CI runners — draws from one tiny
# shared budget and installs start failing with a 403. This redirect is plain
# github.com: it carries no rate-limit budget and needs no token.
function Get-LatestVersion {
    $url = "https://github.com/${GITHUB_REPO}/releases/latest"
    $location = $null
    $diagnostic = ''

    # HttpWebRequest is the one redirect API that behaves identically on
    # Windows PowerShell 5.1 and PowerShell 7: with AllowAutoRedirect disabled
    # GetResponse() returns the 302 itself instead of throwing, and .Headers is
    # a WebHeaderCollection on both, so a single string index works everywhere.
    #
    # Invoke-WebRequest -MaximumRedirection 0 cannot do this portably and was
    # tried first: PS7 raises HttpResponseException whose HttpResponseHeaders
    # has no string indexer (only a typed .Location), while PS5.1 raises
    # InvalidOperationException that carries no .Response at all. Each stack
    # failed on the other's shape and silently produced an empty version, so
    # every unpinned install broke — caught only once CI ran both shells.
    try {
        $req = [System.Net.WebRequest]::Create($url)
        $req.Method = 'HEAD'
        $req.AllowAutoRedirect = $false
        $req.UserAgent = 'archcore-installer'
        $resp = $req.GetResponse()
        try {
            $location = $resp.Headers['Location']
        } finally {
            $resp.Close()
        }
    } catch {
        # Surfaced in the error below: without it a resolution failure is
        # indistinguishable from a network outage, which is what made the
        # previous breakage so hard to read from CI logs.
        $diagnostic = " ($($_.Exception.Message))"
    }

    if ($location -is [array]) { $location = $location[0] }

    if (-not $location -or $location -notmatch '/releases/tag/(.+)$') {
        Write-ErrExit "Could not resolve the latest version from $url$diagnostic. Check your internet connection or proxy settings, or pin a version to skip this lookup: `$env:ARCHCORE_VERSION='x.y.z'"
    }

    return $Matches[1]
}

# ── Download helper ─────────────────────────────────────────────────────────
function Invoke-Download {
    param(
        [string]$Url,
        [string]$OutFile
    )
    $headers = @{ 'User-Agent' = 'archcore-installer' }
    if ($env:GITHUB_TOKEN) {
        $headers['Authorization'] = "Bearer $env:GITHUB_TOKEN"
    }

    $attempts = 0
    $maxAttempts = 3
    while ($attempts -lt $maxAttempts) {
        $attempts++
        try {
            Invoke-WebRequest -UseBasicParsing -Uri $Url -Headers $headers -OutFile $OutFile
            return
        } catch {
            if ($attempts -ge $maxAttempts) {
                Write-ErrExit "Download failed after $maxAttempts attempts: $Url"
            }
            Start-Sleep -Seconds 2
        }
    }
}

# ── Checksum verification ───────────────────────────────────────────────────
function Test-Checksum {
    param(
        [string]$FilePath,
        [string]$ChecksumsPath,
        [string]$ArchiveName
    )

    $lines = Get-Content -Path $ChecksumsPath
    $expectedHash = $null
    foreach ($line in $lines) {
        $parts = $line -split '\s+'
        if ($parts.Count -ge 2 -and $parts[1] -ieq $ArchiveName) {
            $expectedHash = $parts[0]
            break
        }
    }

    if (-not $expectedHash) {
        Write-ErrExit "Checksum for ${ArchiveName} not found in checksums.txt"
    }

    $actualHash = (Get-FileHash -Algorithm SHA256 -Path $FilePath).Hash

    if ($actualHash.ToUpper() -ne $expectedHash.ToUpper()) {
        Write-ErrExit "Checksum verification failed! Expected: ${expectedHash}, actual: ${actualHash}"
    }
}

# ── Archive extraction ──────────────────────────────────────────────────────
function Expand-Release {
    param(
        [string]$ZipPath,
        [string]$TmpDir
    )

    Expand-Archive -Path $ZipPath -DestinationPath $TmpDir -Force

    $primary = Join-Path $TmpDir 'archcore.exe'
    if (Test-Path $primary) {
        return $primary
    }

    # GoReleaser may name binary after repo ("cli.exe")
    $fallback = Join-Path $TmpDir 'cli.exe'
    if (Test-Path $fallback) {
        Move-Item -Path $fallback -Destination $primary -Force
        return $primary
    }

    Write-ErrExit "Binary 'archcore.exe' not found in archive."
}

# ── Atomic install ──────────────────────────────────────────────────────────
function Install-Binary {
    param(
        [string]$SrcExe,
        [string]$DestDir
    )

    New-Item -ItemType Directory -Path $DestDir -Force | Out-Null

    $dest   = Join-Path $DestDir 'archcore.exe'
    $staged = Join-Path $DestDir "archcore.exe.tmp.$PID"

    Copy-Item -Path $SrcExe -Destination $staged -Force
    # Strip MOTW ADS so SmartScreen doesn't block the binary
    Unblock-File -Path $staged
    Move-Item -Path $staged -Destination $dest -Force

    return $dest
}

# ── PATH management ─────────────────────────────────────────────────────────
function Add-ToUserPath {
    param([string]$InstallDir)

    $currentPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($null -eq $currentPath) { $currentPath = '' }

    $normalDir  = $InstallDir.TrimEnd('\')
    $inPath     = $false
    foreach ($segment in ($currentPath -split ';')) {
        if ($segment.TrimEnd('\') -ieq $normalDir) {
            $inPath = $true
            break
        }
    }

    if (-not $inPath) {
        $newPath = if ($currentPath) { "$currentPath;$InstallDir" } else { $InstallDir }
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        Write-Info "Added $InstallDir to your user PATH."
    } else {
        Write-Info "$InstallDir is already in your user PATH."
    }

    Write-WarnMsg 'Open a new terminal for the PATH change to take effect.'
}

# ── Post-install smoke test ─────────────────────────────────────────────────
# Advisory only. The binary is already installed and PATH is already updated by
# the time this runs, so nothing here may abort the script.
#
# Windows PowerShell 5.1 converts a native command's stderr writes into error
# records, and $ErrorActionPreference = 'Stop' promotes the first one into a
# terminating error. `archcore --help` prints ~1 KB of banner to stderr while
# still exiting 0, so this call used to kill the installer *after* a completely
# successful install — and silently, because `*> $null` discarded the error text
# along with everything else. PowerShell 7 does not perform that conversion,
# which is why the failure only appeared once CI started running both shells.
function Test-Install {
    param([string]$InstallPath)

    $exitCode = $null
    $previous = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        & $InstallPath --help *> $null
        $exitCode = $LASTEXITCODE
    } catch {
        $exitCode = -1
    } finally {
        $ErrorActionPreference = $previous
    }

    if ($exitCode -eq 0) {
        Write-Success 'Binary executes OK'
    } else {
        Write-WarnMsg 'Binary installed but --help did not exit cleanly. It may still work.'
    }
}

# ── Main ────────────────────────────────────────────────────────────────────
function main {
    # Env var overrides
    $InstallDir = $env:ARCHCORE_INSTALL_DIR
    if (-not $InstallDir) {
        $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\archcore'
    }
    $PinnedVersion = $env:ARCHCORE_VERSION

    Write-Info 'Installing Archcore CLI...'

    # Architecture
    $arch = Get-Arch
    Write-Info "Detected platform: windows/$arch"

    # Version
    $version = $null
    if ($PinnedVersion) {
        $version = $PinnedVersion.TrimStart('v')
        Write-Info "Using pinned version: $version"
    } else {
        Write-Info 'Fetching latest version...'
        $tag = Get-LatestVersion
        $version = $tag.TrimStart('v')
        Write-Info "Latest version: $version"
    }

    # Construct URLs
    $archiveName   = "archcore_windows_${arch}.zip"
    $downloadUrl   = "https://github.com/${GITHUB_REPO}/releases/download/v${version}/${archiveName}"
    $checksumsUrl  = "https://github.com/${GITHUB_REPO}/releases/download/v${version}/checksums.txt"

    # Temp directory
    $script:tmp_dir = Join-Path $env:TEMP "archcore-install-$([guid]::NewGuid())"
    New-Item -ItemType Directory -Path $script:tmp_dir -Force | Out-Null

    # Download archive
    $archivePath   = Join-Path $script:tmp_dir $archiveName
    Write-Info "Downloading ${archiveName}..."
    Invoke-Download -Url $downloadUrl -OutFile $archivePath

    # Download checksums
    Write-Info 'Verifying checksum...'
    $checksumsPath = Join-Path $script:tmp_dir 'checksums.txt'
    Invoke-Download -Url $checksumsUrl -OutFile $checksumsPath

    # Verify
    Test-Checksum -FilePath $archivePath -ChecksumsPath $checksumsPath -ArchiveName $archiveName
    Write-Success 'Checksum verified'

    # Extract
    Write-Info 'Extracting...'
    $extractedExe = Expand-Release -ZipPath $archivePath -TmpDir $script:tmp_dir

    # Install
    Write-Info "Installing to ${InstallDir}..."
    $installPath = Install-Binary -SrcExe $extractedExe -DestDir $InstallDir

    # PATH
    Add-ToUserPath -InstallDir $InstallDir

    # Smoke test
    Test-Install -InstallPath $installPath

    Write-Success "Archcore CLI v${version} installed to ${installPath}"
}

$script:tmp_dir = $null
try {
    main
} catch {
    # Write-ErrExit inside main already printed "Error: …" and re-threw.
    # Don't re-print; just propagate a clean exit code.
    if ($script:UseColor) {
        [Console]::Error.WriteLine("")
    }
    exit 1
} finally {
    if ($script:tmp_dir -and (Test-Path $script:tmp_dir)) {
        Remove-Item -Recurse -Force $script:tmp_dir -ErrorAction SilentlyContinue
    }
}
