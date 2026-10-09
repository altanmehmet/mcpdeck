[CmdletBinding()]
param([ValidateSet('amd64','arm64')][string]$Architecture='amd64')
$ErrorActionPreference='Stop'
$desktop=Split-Path $PSScriptRoot -Parent
Push-Location $desktop
$previousOS=$env:GOOS;$previousArch=$env:GOARCH;$previousCGO=$env:CGO_ENABLED
try {
 $archive="build/bin/MCPDeck-windows-$Architecture.zip"
 if(!(Test-Path -LiteralPath $archive)){throw 'Build the desktop package first.'}
 $digest=(Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
 Copy-Item -LiteralPath $archive -Destination "setup/payload_$Architecture.zip" -Force
 $env:GOOS='windows';$env:GOARCH=$Architecture;$env:CGO_ENABLED='0'
 $output="build/bin/MCPDeck-Setup-$Architecture.exe"
 go build -trimpath -ldflags "-s -w -H windowsgui -X main.expectedPayloadHash=$digest" -o $output ./setup
 if($LASTEXITCODE -ne 0){throw 'Setup executable build failed'}
 if((Get-Item -LiteralPath $output).Length -gt 20MB){throw 'Setup exceeds the 20 MiB download budget'}
 $hash=(Get-FileHash -LiteralPath $output -Algorithm SHA256).Hash.ToLowerInvariant()
 [IO.File]::WriteAllText((Join-Path $desktop ($output+'.sha256')),$hash+'  '+[IO.Path]::GetFileName($output)+[Environment]::NewLine)
}finally{$env:GOOS=$previousOS;$env:GOARCH=$previousArch;$env:CGO_ENABLED=$previousCGO;Pop-Location}
