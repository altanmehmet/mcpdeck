[CmdletBinding()]
param(
 [string]$Prefix = (Join-Path $env:LOCALAPPDATA 'Programs\MCPDeck'),
 [switch]$NoLaunch,
 [switch]$NoShortcuts
)
$ErrorActionPreference = 'Stop'
$source = Join-Path $PSScriptRoot 'MCPDeck.exe'
$manifest = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'package.json') -Raw | ConvertFrom-Json
if ($manifest.platform -ne 'windows' -or $manifest.application -ne 'MCPDeck Desktop') { throw 'Invalid desktop package.' }
if (!(Test-Path -LiteralPath $source -PathType Leaf) -or (Get-FileHash -LiteralPath $source -Algorithm SHA256).Hash.ToLowerInvariant() -ne $manifest.sha256) { throw 'Package checksum mismatch. Download the package again.' }
$architecture = [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
if (($manifest.architecture -eq 'amd64' -and $architecture -ne 'x64') -or ($manifest.architecture -eq 'arm64' -and $architecture -ne 'arm64')) { throw 'Choose the desktop package matching your Windows architecture.' }
$Prefix = [IO.Path]::GetFullPath($Prefix)
$ancestor = $Prefix
while ($ancestor) {
 if (Test-Path -LiteralPath $ancestor) {
  if ((Get-Item -LiteralPath $ancestor -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Refusing an installation path containing a junction or symlink.' }
 }
 $ancestor = Split-Path $ancestor -Parent
}
$marker = Join-Path $Prefix 'desktop-install.json'
if (Test-Path -LiteralPath $Prefix) {
 $files = @(Get-ChildItem -LiteralPath $Prefix -Force)
 if ($files.Count -gt 0) {
  if (!(Test-Path -LiteralPath $marker -PathType Leaf)) { throw 'Choose an empty directory; this location is not a managed desktop installation.' }
  $previous = Get-Content -LiteralPath $marker -Raw | ConvertFrom-Json
  if ($previous.owner -ne 'mcpdeck-desktop') { throw 'This directory is not owned by the desktop installer.' }
 }
}
$null = New-Item -ItemType Directory -Force -Path $Prefix
$target = Join-Path $Prefix 'MCPDeck.exe'
foreach ($name in @('MCPDeck.exe','uninstall.ps1','desktop-install.json','LICENSE','THIRD_PARTY_NOTICES.txt')) {
 $entry = Join-Path $Prefix $name
 if ((Test-Path -LiteralPath $entry) -and ((Get-Item -LiteralPath $entry -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Refusing to overwrite a linked installation file.' }
}
$stage = Join-Path $Prefix ('.stage-' + [Guid]::NewGuid().ToString('N'))
try {
 Copy-Item -LiteralPath $source -Destination $stage
 if ((Get-FileHash -LiteralPath $stage -Algorithm SHA256).Hash.ToLowerInvariant() -ne $manifest.sha256) { throw 'Staged application checksum mismatch.' }
 # Windows refuses replacement while the application/bridge holds the executable.
 Move-Item -LiteralPath $stage -Destination $target -Force
 foreach ($name in @('uninstall.ps1','LICENSE','THIRD_PARTY_NOTICES.txt')) { Copy-Item -LiteralPath (Join-Path $PSScriptRoot $name) -Destination $Prefix -Force }
 $metadata = @{owner='mcpdeck-desktop';version=$manifest.version;sha256=$manifest.sha256;prefix=$Prefix} | ConvertTo-Json -Compress
 [IO.File]::WriteAllText($marker,$metadata,(New-Object Text.UTF8Encoding($false)))
 if (!$NoShortcuts) {
  $shell = New-Object -ComObject WScript.Shell
  $folder = Join-Path ([Environment]::GetFolderPath('Programs')) 'MCPDeck'
  $null = New-Item -ItemType Directory -Force -Path $folder
  $shortcut = $shell.CreateShortcut((Join-Path $folder 'MCPDeck.lnk'))
  $shortcut.TargetPath=$target; $shortcut.WorkingDirectory=$Prefix; $shortcut.Save()
 }
 $key='HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\MCPDeckDesktop'
 $null=New-Item -Path $key -Force
 $values=@{DisplayName='MCPDeck Desktop';DisplayVersion=$manifest.version;InstallLocation=$Prefix;DisplayIcon=$target;UninstallString=('powershell.exe -NoProfile -ExecutionPolicy Bypass -File "'+(Join-Path $Prefix 'uninstall.ps1')+'"');Publisher='MCPDeck contributors'}
 foreach ($name in $values.Keys) { $null=New-ItemProperty -Path $key -Name $name -Value $values[$name] -PropertyType String -Force }
 $null=New-ItemProperty -Path $key -Name EstimatedSize -Value ([int][Math]::Ceiling((Get-Item -LiteralPath $target).Length/1024)) -PropertyType DWord -Force
 Write-Output 'MCPDeck Desktop installed. Open it from the Start menu.'
 if (!$NoLaunch) { Start-Process -FilePath $target }
} catch { throw ('Installation could not finish. Close MCPDeck and any agents using its Bridge, then retry. '+$_.Exception.Message) }
finally { if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage -Force } }
