#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Installs gost as a Windows service.

.DESCRIPTION
    Downloads the latest (or specified) gost release from GitHub, installs it to
    a target directory, and registers it as a Windows service using the native
    Windows Service Control Manager.  gost is built with go-svc and runs as a
    proper Windows service without any wrapper.

.PARAMETER Version
    The release tag to install, e.g. "v3.2.6".  Defaults to the latest release.

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
    # Install latest release with defaults and start immediately
    .\install-service.ps1 -Start

.EXAMPLE
    # Install a specific version with a custom config
    .\install-service.ps1 -Version v3.2.6 -ConfigFile C:\etc\gost.yml -Start

.EXAMPLE
    # Install with inline service definition (no config file)
    .\install-service.ps1 -ExtraArgs "-L socks5://:1080 -L http://:8080" -Start
#>

[CmdletBinding(SupportsShouldProcess)]
param(
    [string]$Version      = "",
    [string]$InstallDir   = "C:\Program Files\gost",
    [string]$ConfigFile   = "",
    [string]$ServiceName  = "gost",
    [string]$DisplayName  = "GOST Tunnel",
    [string]$ExtraArgs    = "",
    [ValidateSet("Automatic","Manual","Disabled")]
    [string]$StartupType  = "Automatic",
    [switch]$Start
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

function Write-Step([string]$msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }
function Write-Ok([string]$msg)   { Write-Host "    OK  $msg" -ForegroundColor Green }
function Write-Warn([string]$msg) { Write-Host "    WARN $msg" -ForegroundColor Yellow }

function Get-Architecture {
    switch ($env:PROCESSOR_ARCHITECTURE) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        "x86"   { return "386"   }
        default {
            # Also check PROCESSOR_ARCHITEW6432 for WoW64 processes
            if ($env:PROCESSOR_ARCHITEW6432 -eq "AMD64") { return "amd64" }
            throw "Unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)"
        }
    }
}

function Get-LatestVersion {
    Write-Step "Fetching latest gost release from GitHub..."
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/go-gost/gost/releases/latest" `
                                 -Headers @{ "User-Agent" = "gost-install-script" }
    return $release.tag_name
}

function Get-DownloadUrl([string]$tag, [string]$arch) {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/go-gost/gost/releases/tags/$tag" `
                                 -Headers @{ "User-Agent" = "gost-install-script" }
    $pattern = "windows.*$arch"
    $asset   = $release.assets | Where-Object { $_.name -match $pattern } | Select-Object -First 1
    if (-not $asset) {
        throw "No Windows/$arch asset found for release $tag.  Available: $($release.assets.name -join ', ')"
    }
    return $asset.browser_download_url
}

function Install-Binary([string]$url, [string]$destDir) {
    $zipPath = Join-Path $env:TEMP "gost-install.zip"
    Write-Step "Downloading $url ..."
    Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing

    Write-Step "Extracting to $destDir ..."
    if (-not (Test-Path $destDir)) {
        New-Item -ItemType Directory -Path $destDir -Force | Out-Null
    }

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [System.IO.Compression.ZipFile]::OpenRead($zipPath)
    try {
        foreach ($entry in $zip.Entries) {
            if ($entry.Name -eq "gost.exe") {
                $dest = Join-Path $destDir "gost.exe"
                [System.IO.Compression.ZipFileExtensions]::ExtractToFile($entry, $dest, $true)
                Write-Ok "gost.exe extracted to $dest"
                break
            }
        }
    } finally {
        $zip.Dispose()
        Remove-Item $zipPath -Force -ErrorAction SilentlyContinue
    }

    $exePath = Join-Path $destDir "gost.exe"
    if (-not (Test-Path $exePath)) {
        throw "gost.exe not found in the downloaded archive."
    }
    return $exePath
}

function Ensure-Config([string]$installDir, [string]$userConfig) {
    $dest = Join-Path $installDir "gost.yml"

    if ($userConfig -ne "") {
        if (-not (Test-Path $userConfig)) {
            throw "Config file not found: $userConfig"
        }
        if ((Resolve-Path $userConfig).Path -ne (Resolve-Path $dest -ErrorAction SilentlyContinue)?.Path) {
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
    if ($extraArgs -ne "") {
        $binPath += " $extraArgs"
    }

    $existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue

    if ($existing) {
        Write-Step "Service '$ServiceName' already exists — updating..."
        if ($existing.Status -eq "Running") {
            Write-Step "Stopping existing service..."
            Stop-Service -Name $ServiceName -Force
            $existing.WaitForStatus("Stopped", [TimeSpan]::FromSeconds(30))
        }
        # Update the binary path
        sc.exe config $ServiceName binPath= $binPath | Out-Null
        sc.exe config $ServiceName start= $(if ($StartupType -eq "Automatic") { "auto" } elseif ($StartupType -eq "Manual") { "demand" } else { "disabled" }) | Out-Null
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

        # Configure failure recovery: restart on first/second failure, reset after 1 day
        sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/10000/restart/30000 | Out-Null
        Write-Ok "Service registered."
    }

    # Set description
    sc.exe description $ServiceName "GOST (GO Simple Tunnel) - secure tunnel service" | Out-Null
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

$arch    = Get-Architecture
$version = if ($Version -ne "") { $Version } else { Get-LatestVersion }

Write-Host ""
Write-Host "  gost Windows Service Installer" -ForegroundColor White
Write-Host "  ================================" -ForegroundColor White
Write-Host "  Version    : $version"
Write-Host "  Arch       : $version / windows-$arch"
Write-Host "  Install dir: $InstallDir"
Write-Host "  Service    : $ServiceName ($StartupType)"
Write-Host ""

# 1. Download & install binary
$url    = Get-DownloadUrl $version $arch
$exePath = Install-Binary $url $InstallDir

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
Write-Host "    Start   : Start-Service $ServiceName"
Write-Host "    Stop    : Stop-Service $ServiceName"
Write-Host "    Status  : Get-Service $ServiceName"
Write-Host "    Logs    : Get-EventLog -LogName Application -Source $ServiceName -Newest 20"
Write-Host "    Uninstall: sc.exe delete $ServiceName"
Write-Host ""
