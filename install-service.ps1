#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Builds gost from source and installs it as a Windows service.

.DESCRIPTION
    Runs "go build" against the local source tree, copies the resulting binary
    to a target directory, and registers it as a Windows service using the
    native Windows Service Control Manager.  gost is built with go-svc and
    runs as a proper Windows service without any wrapper.

.PARAMETER InstallDir
    Directory where gost.exe and gost.yml are placed.
    Default: C:\Program Files\gost

.PARAMETER ConfigFile
    Path to an existing gost config file to use.  If omitted and no config
    exists in InstallDir, a minimal placeholder is created.

.PARAMETER ServiceName
    Windows service name.  Default: gost

.PARAMETER DisplayName
    Windows service display name.  Default: GOST Tunnel

.PARAMETER ExtraArgs
    Additional arguments passed to gost.exe, e.g. "-L :8080 -D".
    The -C flag pointing to the config file is always added automatically.

.PARAMETER StartupType
    Service start type: Automatic, Manual, or Disabled.  Default: Automatic

.PARAMETER Start
    Start the service immediately after installation.

.EXAMPLE
    # Build, install with defaults, and start immediately
    .\install-service.ps1 -Start

.EXAMPLE
    # Install with a custom config
    .\install-service.ps1 -ConfigFile C:\etc\gost.yml -Start

.EXAMPLE
    # Install with inline service definition (no config file)
    .\install-service.ps1 -ExtraArgs "-L socks5://:1080 -L http://:8080" -Start
#>

[CmdletBinding(SupportsShouldProcess)]
param(
    [string]$InstallDir  = "C:\Program Files\gost",
    [string]$ConfigFile  = "",
    [string]$ServiceName = "gost",
    [string]$DisplayName = "GOST Tunnel",
    [string]$ExtraArgs   = "",
    [ValidateSet("Automatic","Manual","Disabled")]
    [string]$StartupType = "Automatic",
    [switch]$Start
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Directory containing this script == repo root
$RepoRoot = $PSScriptRoot

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

function Write-Step([string]$msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }
function Write-Ok([string]$msg)   { Write-Host "    OK  $msg" -ForegroundColor Green }
function Write-Warn([string]$msg) { Write-Host "    WARN $msg" -ForegroundColor Yellow }

function Build-Binary([string]$destDir) {
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        throw "go not found in PATH. Please install Go from https://go.dev/dl/"
    }

    Write-Step "Building gost from source ($RepoRoot)..."

    if (-not (Test-Path $destDir)) {
        New-Item -ItemType Directory -Path $destDir -Force | Out-Null
    }

    $exeDest = Join-Path $destDir "gost.exe"

    # Embed version: prefer git tag, fall back to version.go
    $ldflags = "-s -w"
    $version = $null
    if (Get-Command git -ErrorAction SilentlyContinue) {
        $version = git -C $RepoRoot describe --tags --abbrev=0 2>$null
    }
    if (-not $version) {
        $verFile = Join-Path $RepoRoot "cmd\gost\version.go"
        if (Test-Path $verFile) {
            $match = Select-String -Path $verFile -Pattern 'version\s*=\s*"([^"]+)"'
            if ($match) { $version = $match.Matches[0].Groups[1].Value }
        }
    }
    if ($version) { $ldflags = "-s -w -X 'main.version=$version'" }

    $goArgs = @("build", "-ldflags", $ldflags, "-o", $exeDest, "./cmd/gost")
    Write-Host "    go $($goArgs -join ' ')" -ForegroundColor DarkGray

    & go @goArgs 2>&1 | ForEach-Object { Write-Host "    $_" -ForegroundColor DarkGray }
    if ($LASTEXITCODE -ne 0) { throw "go build failed (exit $LASTEXITCODE)" }

    Write-Ok "Built: $exeDest"
    return $exeDest
}

function Ensure-Config([string]$installDir, [string]$userConfig) {
    $dest = Join-Path $installDir "gost.yml"

    if ($userConfig -ne "") {
        if (-not (Test-Path $userConfig)) {
            throw "Config file not found: $userConfig"
        }
        $resolvedSrc  = (Resolve-Path $userConfig).Path
        $resolvedDest = if (Test-Path $dest) { (Resolve-Path $dest).Path } else { "" }
        if ($resolvedSrc -ne $resolvedDest) {
            Copy-Item $userConfig $dest -Force
            Write-Ok "Config copied from $userConfig"
        }
        return $dest
    }

    if (Test-Path $dest) {
        Write-Ok "Using existing config: $dest"
        return $dest
    }

    # Create a minimal placeholder config
    $placeholder = @"
# gost configuration file
# Documentation: https://gost.run/
#
# Example: HTTP proxy on port 8080
# services:
# - name: http-proxy
#   addr: ":8080"
#   handler:
#     type: http
#   listener:
#     type: tcp

log:
  level: info
  format: json
"@
    Set-Content -Path $dest -Value $placeholder -Encoding UTF8
    Write-Warn "A placeholder config was created at $dest"
    Write-Warn "Edit it before starting the service, or pass -ExtraArgs with -L/-F flags."
    return $dest
}

function Register-GostService([string]$exePath, [string]$cfgPath, [string]$extraArgs) {
    $binPath = "`"$exePath`" -C `"$cfgPath`""
    if ($extraArgs -ne "") { $binPath += " $extraArgs" }

    $existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue

    if ($existing) {
        Write-Step "Service '$ServiceName' already exists — updating..."
        if ($existing.Status -eq "Running") {
            Write-Step "Stopping existing service..."
            Stop-Service -Name $ServiceName -Force
            $existing.WaitForStatus("Stopped", [TimeSpan]::FromSeconds(30))
        }
        sc.exe config $ServiceName binPath= $binPath | Out-Null
        $startValue = switch ($StartupType) {
            "Automatic" { "auto"     }
            "Manual"    { "demand"   }
            "Disabled"  { "disabled" }
        }
        sc.exe config $ServiceName start= $startValue | Out-Null
        Write-Ok "Service updated."
    } else {
        Write-Step "Registering service '$ServiceName'..."
        $startValue = switch ($StartupType) {
            "Automatic" { "auto"     }
            "Manual"    { "demand"   }
            "Disabled"  { "disabled" }
        }
        sc.exe create $ServiceName `
            binPath= $binPath `
            DisplayName= $DisplayName `
            start= $startValue | Out-Null

        # Restart on failure: 5 s / 10 s / 30 s, reset counter after 1 day
        sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/10000/restart/30000 | Out-Null
        Write-Ok "Service registered."
    }

    sc.exe description $ServiceName "GOST (GO Simple Tunnel) - secure tunnel service" | Out-Null
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

Write-Host ""
Write-Host "  gost Windows Service Installer (build from source)" -ForegroundColor White
Write-Host "  ===================================================" -ForegroundColor White
Write-Host "  Repo       : $RepoRoot"
Write-Host "  Install dir: $InstallDir"
Write-Host "  Service    : $ServiceName ($StartupType)"
Write-Host ""

# 1. Build binary from source
$exePath = Build-Binary $InstallDir

# 2. Ensure config exists
$cfgPath = Ensure-Config $InstallDir $ConfigFile

# 3. Register Windows service
Register-GostService $exePath $cfgPath $ExtraArgs

# 4. Optionally start
if ($Start) {
    Write-Step "Starting service '$ServiceName'..."
    Start-Service -Name $ServiceName
    $svc = Get-Service -Name $ServiceName
    $svc.WaitForStatus("Running", [TimeSpan]::FromSeconds(15))
    Write-Ok "Service is running."
}

Write-Host ""
Write-Host "  Done!" -ForegroundColor Green
Write-Host ""
Write-Host "  Useful commands:" -ForegroundColor White
Write-Host "    Start    : Start-Service $ServiceName"
Write-Host "    Stop     : Stop-Service $ServiceName"
Write-Host "    Status   : Get-Service $ServiceName"
Write-Host "    Logs     : Get-EventLog -LogName Application -Source $ServiceName -Newest 20"
Write-Host "    Uninstall: sc.exe delete $ServiceName"
Write-Host ""
