#Requires -Version 5.1
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# ── Constants ────────────────────────────────────────────────────────────────
$GITHUB_REPO    = 'archcore-ai/cli'
$BINARY_NAME    = 'archcore'

# ── Telemetry constants ──────────────────────────────────────────────────────
# The key is a placeholder in this repository on purpose. archcore.ai's deploy
# workflow substitutes vars.POSTHOG_KEY while it syncs this file into public/,
# so the key never lands in git and any copy run straight from the repo — a
# local test, install-smoke.yml, a fork — reports nothing at all.
#
# The guard in Test-TelemetryEnabled looks for the `phc_` prefix of a real
# PostHog project key rather than comparing against the placeholder text, so the
# substitution can never accidentally rewrite its own off-switch.
$POSTHOG_KEY  = '__POSTHOG_KEY__'
$POSTHOG_HOST = 'https://ph.archcore.ai'

# Coarse progress marker. Reported as `stage` on a failed install so a genuine
# drop-off can be told apart from a network outage without ever transmitting an
# error message. Advanced by main().
$script:STAGE = 'start'

# Platform facts, filled in by main() once detected. A failure before detection
# still reports, hence the defaults — Set-StrictMode rejects unset variables.
$script:TELEMETRY_ARCH    = 'unknown'
$script:TELEMETRY_VERSION = ''

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
    # Stage category only. The message itself is never transmitted.
    Send-TelemetryEvent -EventName 'cli_install_failed' -Extra @{ stage = $script:STAGE }
    throw $Message
}

# ── Telemetry ────────────────────────────────────────────────────────────────
# Anonymous install analytics, sent to archcore.ai's first-party PostHog proxy.
# Documented at https://archcore.ai/privacy. Three properties of this code are
# load-bearing and must survive any refactor:
#
#   1. It can never fail the install. Every path is wrapped in try/catch and
#      bounded by a short timeout; $ErrorActionPreference is 'Stop' script-wide,
#      so an unguarded web call here would abort a successful install the way
#      Test-Install once did.
#   2. It can never run without a key injected at deploy time (see above), so
#      the repository copy is inert.
#   3. Opting out leaves no trace on disk — the id file below is created only
#      after the opt-out check has already passed.
function Test-TelemetryEnabled {
    if ($POSTHOG_KEY -notlike 'phc_*') { return $false }

    # consoledonottrack.com, plus the tool-specific override. Any value other
    # than absent, empty or "0" opts out.
    foreach ($name in @('DO_NOT_TRACK', 'ARCHCORE_TELEMETRY_OPTOUT')) {
        $value = [Environment]::GetEnvironmentVariable($name)
        if ($value -and $value -ne '0') { return $false }
    }
    return $true
}

# A random, opaque, per-machine identifier — not derived from hostname, user
# name or any hardware id, so it carries nothing about the machine it names.
#
# The path deliberately mirrors updateCheckCachePath() in cmd/update.go, which
# uses Go's os.UserHomeDir() + ".local/state" on every platform including
# Windows rather than %LOCALAPPDATA%. Matching it — instead of using the more
# idiomatic Windows location — is what lets the CLI's own telemetry adopt this
# same id later and join "installed" to "used" without a second identifier.
function Get-InstallIdPath {
    $base = [Environment]::GetEnvironmentVariable('XDG_STATE_HOME')
    if (-not $base) {
        $base = Join-Path $env:USERPROFILE '.local\state'
    }
    return Join-Path $base 'archcore\install-id'
}

# Returns a hashtable @{ Id = <32 hex chars>; IsReinstall = $bool }, or $null
# when the id could not be resolved and the caller must skip reporting.
function Get-InstallId {
    try {
        $path = Get-InstallIdPath

        if (Test-Path -LiteralPath $path) {
            $existing = (Get-Content -LiteralPath $path -Raw -ErrorAction Stop) -replace '[^0-9a-f]', ''
            if ($existing) {
                return @{ Id = $existing; IsReinstall = $true }
            }
        }

        # 'N' formats the GUID as 32 lowercase hex digits with no braces or
        # dashes, matching the format install.sh writes so one machine that has
        # used both installers is not counted twice.
        $id = [guid]::NewGuid().ToString('N')

        $dir = Split-Path -Parent $path
        if (-not (Test-Path -LiteralPath $dir)) {
            New-Item -ItemType Directory -Path $dir -Force -ErrorAction Stop | Out-Null
        }
        Set-Content -LiteralPath $path -Value $id -NoNewline -ErrorAction Stop

        return @{ Id = $id; IsReinstall = $false }
    } catch {
        return $null
    }
}

function Send-TelemetryEvent {
    param(
        [string]$EventName,
        [hashtable]$Extra = @{}
    )

    try {
        if (-not (Test-TelemetryEnabled)) { return }

        $identity = Get-InstallId
        if (-not $identity) { return }

        $ci = $false
        foreach ($name in @('CI', 'GITHUB_ACTIONS', 'GITLAB_CI', 'BUILDKITE', 'JENKINS_URL', 'TEAMCITY_VERSION')) {
            if ([Environment]::GetEnvironmentVariable($name)) { $ci = $true; break }
        }

        $properties = @{
            source              = 'installer'
            installer           = 'install.ps1'
            os                  = 'windows'
            arch                = $script:TELEMETRY_ARCH
            is_reinstall        = $identity.IsReinstall
            ci                  = $ci
            pinned_version      = [bool]$env:ARCHCORE_VERSION
            install_dir_default = -not [bool]$env:ARCHCORE_INSTALL_DIR
        }
        if ($script:TELEMETRY_VERSION) {
            # The one value with an external shape, so it is filtered to the
            # semver alphabet before it reaches the payload.
            $properties['archcore_version'] = $script:TELEMETRY_VERSION -replace '[^0-9A-Za-z.+\-]', ''
        }
        foreach ($key in $Extra.Keys) {
            $properties[$key] = $Extra[$key]
        }

        $payload = @{
            api_key     = $POSTHOG_KEY
            event       = $EventName
            distinct_id = $identity.Id
            properties  = $properties
        } | ConvertTo-Json -Depth 5 -Compress

        Invoke-WebRequest -UseBasicParsing -Method Post -TimeoutSec 3 `
            -Uri "$POSTHOG_HOST/i/v0/e/" `
            -ContentType 'application/json' `
            -Body $payload | Out-Null
    } catch {
        # Blocked host, offline, proxy interstitial, PostHog outage — all of it
        # is irrelevant to whether the CLI installed.
    }
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
    $script:STAGE = 'platform'
    $arch = Get-Arch
    $script:TELEMETRY_ARCH = $arch
    Write-Info "Detected platform: windows/$arch"

    # Version
    $script:STAGE = 'version'
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
    $script:TELEMETRY_VERSION = $version

    # Construct URLs
    $archiveName   = "archcore_windows_${arch}.zip"
    $downloadUrl   = "https://github.com/${GITHUB_REPO}/releases/download/v${version}/${archiveName}"
    $checksumsUrl  = "https://github.com/${GITHUB_REPO}/releases/download/v${version}/checksums.txt"

    # Temp directory
    $script:tmp_dir = Join-Path $env:TEMP "archcore-install-$([guid]::NewGuid())"
    New-Item -ItemType Directory -Path $script:tmp_dir -Force | Out-Null

    # Download archive
    $script:STAGE = 'download'
    $archivePath   = Join-Path $script:tmp_dir $archiveName
    Write-Info "Downloading ${archiveName}..."
    Invoke-Download -Url $downloadUrl -OutFile $archivePath

    # Download checksums
    $script:STAGE = 'checksum'
    Write-Info 'Verifying checksum...'
    $checksumsPath = Join-Path $script:tmp_dir 'checksums.txt'
    Invoke-Download -Url $checksumsUrl -OutFile $checksumsPath

    # Verify
    Test-Checksum -FilePath $archivePath -ChecksumsPath $checksumsPath -ArchiveName $archiveName
    Write-Success 'Checksum verified'

    # Extract
    $script:STAGE = 'extract'
    Write-Info 'Extracting...'
    $extractedExe = Expand-Release -ZipPath $archivePath -TmpDir $script:tmp_dir

    # Install
    $script:STAGE = 'install'
    Write-Info "Installing to ${InstallDir}..."
    $installPath = Install-Binary -SrcExe $extractedExe -DestDir $InstallDir

    # PATH
    Add-ToUserPath -InstallDir $InstallDir

    # Smoke test
    Test-Install -InstallPath $installPath

    Write-Success "Archcore CLI v${version} installed to ${installPath}"

    $script:STAGE = 'done'
    Send-TelemetryEvent -EventName 'cli_installed'
    # Disclosure belongs next to the thing being disclosed, not only in the
    # policy page. Printed after the success line so it never reads as an error,
    # and only when an event was actually sent.
    if (Test-TelemetryEnabled) {
        Write-Info 'Anonymous install ping sent (no personal data). Opt out with $env:DO_NOT_TRACK=1 — https://archcore.ai/privacy'
    }
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
