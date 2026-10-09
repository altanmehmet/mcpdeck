[CmdletBinding()]
param([switch]$Yes)
$ErrorActionPreference = 'Stop'
$ancestor=$PSScriptRoot
while($ancestor){
 if((Get-Item -LiteralPath $ancestor -Force).Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'Refusing removal through a linked directory.'}
 $ancestor=Split-Path $ancestor -Parent
}
$marker=Join-Path $PSScriptRoot 'desktop-install.json'
$metadata=Get-Content -LiteralPath $marker -Raw | ConvertFrom-Json
if ($metadata.owner -ne 'mcpdeck-desktop' -or [IO.Path]::GetFullPath($metadata.prefix) -ne [IO.Path]::GetFullPath($PSScriptRoot)) { throw 'Not a managed desktop installation.' }
if (!$Yes -and (Read-Host 'Close MCPDeck and Bridge connections first. Remove the desktop app? [y/N]') -notmatch '^(y|yes)$') { return }
$known=@('MCPDeck.exe','uninstall.ps1','desktop-install.json','LICENSE','THIRD_PARTY_NOTICES.txt')
foreach ($name in $known) {
 $path=Join-Path $PSScriptRoot $name
 if ((Test-Path -LiteralPath $path) -and ((Get-Item -LiteralPath $path -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Refusing to remove a linked installation file.' }
}
$binary=Join-Path $PSScriptRoot 'MCPDeck.exe'
if ((Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash.ToLowerInvariant() -ne $metadata.sha256) { throw 'Application changed outside the installer; refusing removal.' }
Remove-Item -LiteralPath $binary -Force
$shortcut=Join-Path ([Environment]::GetFolderPath('Programs')) 'MCPDeck\MCPDeck.lnk'
if (Test-Path -LiteralPath $shortcut) {
 $shell=New-Object -ComObject WScript.Shell
 if ($shell.CreateShortcut($shortcut).TargetPath -eq $binary) {Remove-Item -LiteralPath $shortcut -Force}
}
$key='HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\MCPDeckDesktop'
if ((Get-ItemProperty -Path $key -ErrorAction SilentlyContinue).InstallLocation -eq $PSScriptRoot) {Remove-Item -Path $key -Force}
foreach ($name in $known | Where-Object {$_ -ne 'MCPDeck.exe'}) { $path=Join-Path $PSScriptRoot $name;if(Test-Path -LiteralPath $path){Remove-Item -LiteralPath $path -Force} }
if (@(Get-ChildItem -LiteralPath $PSScriptRoot -Force).Count -eq 0) {Remove-Item -LiteralPath $PSScriptRoot}
Write-Output 'Desktop application removed. MCPDeck settings, instructions and agent configurations were kept. Installed MCP servers were not removed.'
