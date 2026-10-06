[CmdletBinding(SupportsShouldProcess = $true, ConfirmImpact = 'High')]
param([string]$Prefix = (Join-Path $env:LOCALAPPDATA 'MCPDeck'), [switch]$NoPathUpdate)
$ErrorActionPreference = 'Stop'
# Use the runtime directly: isolated PowerShell 5.1 environments may not
# auto-load Get-FileHash from their module search path.
function Get-MCPDeckChecksum([string]$Path) {
    $stream = [IO.File]::OpenRead($Path)
    $sha = [Security.Cryptography.SHA256]::Create()
    try { return [BitConverter]::ToString($sha.ComputeHash($stream)).Replace('-', '').ToLowerInvariant() }
    finally { $stream.Dispose(); $sha.Dispose() }
}
$Prefix = [IO.Path]::GetFullPath($Prefix)
$versions = Join-Path $Prefix 'versions'
# Reject junctions/symlinks anywhere in the installation path before deletion.
$ancestor = $versions
while ($ancestor) {
    if (Test-Path -LiteralPath $ancestor) {
        if ((Get-Item -LiteralPath $ancestor -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) {
            throw 'Refusing to uninstall through a reparse point.'
        }
    }
    $ancestor = Split-Path $ancestor -Parent
}
if (!(Test-Path -LiteralPath $versions -PathType Container)) { throw 'No managed installation was found.' }
$owned = @()
$expected = @('mcpdeck.exe', 'README.txt', 'AKILLI-KURULUM.md', 'INSTRUCTIONS.md', 'RECOVERY.md', 'LICENSE', 'THIRD_PARTY_NOTICES.txt', 'uninstall.ps1', 'mcpdeck-install.json')
# Preflight every version before removing anything. Legacy or unknown files are retained.
foreach ($directory in @(Get-ChildItem -LiteralPath $versions -Force)) {
    if (!$directory.PSIsContainer -or ($directory.Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Unknown installation entry; nothing was removed.' }
    $marker = Join-Path $directory.FullName 'mcpdeck-install.json'
    if (!(Test-Path -LiteralPath $marker -PathType Leaf)) { throw 'An older or unmanaged version exists; nothing was removed. Consult the Windows guide.' }
    $entries = @(Get-ChildItem -LiteralPath $directory.FullName -Force)
    foreach ($entry in $entries) {
        if ($entry.PSIsContainer -or ($entry.Attributes -band [IO.FileAttributes]::ReparsePoint) -or $entry.Name -notin $expected) { throw 'Unknown installation contents; nothing was removed.' }
    }
    $metadata = Get-Content -LiteralPath $marker -Raw | ConvertFrom-Json
    if ($metadata.owner -ne 'mcpdeck' -or $metadata.version -ne $directory.Name -or
        $metadata.sha256 -ne (Get-MCPDeckChecksum (Join-Path $directory.FullName 'mcpdeck.exe'))) {
        throw 'Installation ownership check failed; nothing was removed.'
    }
    $owned += $directory.FullName
}
if (!$owned.Count) { throw 'No managed versions were found.' }
if ($PSCmdlet.ShouldProcess($Prefix, 'Remove MCPDeck application versions and their PATH entries; preserve user data and agent configurations')) {
    foreach ($directory in $owned) {
        foreach ($name in $expected) {
            $file = Join-Path $directory $name
            if (Test-Path -LiteralPath $file) { Remove-Item -LiteralPath $file -Force }
        }
        Remove-Item -LiteralPath $directory
    }
    if (!$NoPathUpdate) {
        foreach ($scope in @('User', 'Process')) {
            $path = [Environment]::GetEnvironmentVariable('Path', $scope)
            $keep = @($path -split ';' | Where-Object { $_ -and $_.TrimEnd('\') -notin $owned })
            [Environment]::SetEnvironmentVariable('Path', ($keep -join ';'), $scope)
        }
    }
    Write-Output 'MCPDeck application removed. User data and agent configurations were preserved.'
}
