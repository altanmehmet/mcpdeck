[CmdletBinding()]
param(
    [string]$Prefix = (Join-Path $env:LOCALAPPDATA 'MCPDeck'),
    [switch]$NoPathUpdate
)
$ErrorActionPreference = 'Stop'
function Get-MCPDeckChecksum([string]$Path) {
    $stream = [IO.File]::OpenRead($Path)
    $sha = [Security.Cryptography.SHA256]::Create()
    try { return [BitConverter]::ToString($sha.ComputeHash($stream)).Replace('-', '').ToLowerInvariant() }
    finally { $stream.Dispose(); $sha.Dispose() }
}
$source = Join-Path $PSScriptRoot 'mcpdeck.exe'
if (!(Test-Path -LiteralPath $source -PathType Leaf)) { throw 'mcpdeck.exe must be beside this installer.' }
$reported = & $source --version
if ($LASTEXITCODE -ne 0 -or $reported -notmatch '^mcpdeck version ([A-Za-z0-9.+_-]+)$') { throw 'Cannot verify package version.' }
$version = $Matches[1]
$Prefix = [IO.Path]::GetFullPath($Prefix)
# Refuse junctions/symlinks before creating or updating installation files.
$ancestor = $Prefix
while ($ancestor) {
    if (Test-Path -LiteralPath $ancestor) {
        if ((Get-Item -LiteralPath $ancestor -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Refusing to install through a reparse point.' }
    }
    $ancestor = Split-Path $ancestor -Parent
}
$versions = Join-Path $Prefix 'versions'
if ((Test-Path -LiteralPath $versions) -and ((Get-Item -LiteralPath $versions -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Refusing to install through a reparse point.' }
$destination = Join-Path $versions $version
$null = New-Item -ItemType Directory -Force -Path $versions
$stage = Join-Path $versions ([Guid]::NewGuid().ToString('N'))
try {
    if (Test-Path -LiteralPath $destination) {
        if ((Get-Item -LiteralPath $destination -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Refusing to install through a reparse point.' }
        foreach ($name in @('mcpdeck.exe', 'mcpdeck-install.json')) {
            $entry = Join-Path $destination $name
            if ((Test-Path -LiteralPath $entry) -and ((Get-Item -LiteralPath $entry -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Refusing to install through a reparse point.' }
        }
        $existing = Join-Path $destination 'mcpdeck.exe'
        if (!(Test-Path -LiteralPath $existing) -or
            (Get-MCPDeckChecksum $source) -ne (Get-MCPDeckChecksum $existing)) {
            throw 'This version already exists with different contents. Use a new version or a different prefix.'
        }
    } else {
        $null = New-Item -ItemType Directory -Path $stage
        foreach ($name in @('mcpdeck.exe', 'README.txt', 'AKILLI-KURULUM.md', 'INSTRUCTIONS.md', 'RECOVERY.md', 'LICENSE', 'THIRD_PARTY_NOTICES.txt', 'uninstall.ps1')) {
            Copy-Item -LiteralPath (Join-Path $PSScriptRoot $name) -Destination $stage
        }
        Move-Item -LiteralPath $stage -Destination $destination
    }
    $metadata = @{ owner = 'mcpdeck'; version = $version; sha256 = (Get-MCPDeckChecksum $source) } | ConvertTo-Json -Compress
    [IO.File]::WriteAllText((Join-Path $destination 'mcpdeck-install.json'), $metadata, (New-Object Text.UTF8Encoding($false)))
    if (!$NoPathUpdate) {
        $oldPath = [Environment]::GetEnvironmentVariable('Path', 'User')
        $ownedPrefix = $versions.TrimEnd('\') + '\'
        $keep = @($oldPath -split ';' | Where-Object {
            $_ -and !($_.StartsWith($ownedPrefix, [StringComparison]::OrdinalIgnoreCase))
        })
        [Environment]::SetEnvironmentVariable('Path', (@($destination) + $keep -join ';'), 'User')
        $processKeep = @($env:Path -split ';' | Where-Object {
            $_ -and !($_.StartsWith($ownedPrefix, [StringComparison]::OrdinalIgnoreCase))
        })
        $env:Path = (@($destination) + $processKeep -join ';')
    }
    Write-Output "Installed MCPDeck $version at $destination"
    Write-Output 'Run mcpdeck in PowerShell or Windows Terminal. Reopen existing terminals to reload PATH.'
} finally {
    if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage -Recurse -Force }
}
