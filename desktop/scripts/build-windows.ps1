[CmdletBinding()]
param([ValidateSet('amd64','arm64')][string]$Architecture='amd64',[string]$Version='0.1.0')
$ErrorActionPreference='Stop'
if ($Version -notmatch '^[A-Za-z0-9.+_-]+$') { throw 'Invalid package version.' }
$desktop=Split-Path $PSScriptRoot -Parent
Push-Location (Join-Path $desktop 'frontend')
try {
 if (!(Test-Path 'node_modules/vite/bin/vite.js')) {npm ci;if($LASTEXITCODE -ne 0){throw 'Frontend dependencies failed'}}
 npm test;if($LASTEXITCODE -ne 0){throw 'Frontend tests failed'}
 npm run build;if($LASTEXITCODE -ne 0){throw 'Frontend build failed'}
}finally{Pop-Location}
Push-Location $desktop
$originalOS=$env:GOOS;$originalArch=$env:GOARCH;$originalCGO=$env:CGO_ENABLED
try {
 $env:GOOS='windows';$env:GOARCH=$Architecture;$env:CGO_ENABLED='0'
 $binary=Join-Path $desktop "build/bin/windows-$Architecture/MCPDeck.exe"
 $null=New-Item -ItemType Directory -Force -Path (Split-Path $binary -Parent)
 go build -trimpath -tags 'production,wv2runtime.embed' -ldflags "-s -w -H windowsgui -X github.com/altanmehmet/mcpdeck/cmd.version=$Version" -o $binary .
 if($LASTEXITCODE -ne 0){throw 'Windows build failed'}
 $env:GOOS=$originalOS;$env:GOARCH=$originalArch;$env:CGO_ENABLED=$originalCGO
 go run ./package -platform windows -arch $Architecture -version $Version -source $binary
 if($LASTEXITCODE -ne 0){throw 'Windows packaging failed'}
 & (Join-Path $PSScriptRoot 'build-setup.ps1') -Architecture $Architecture
}finally{$env:GOOS=$originalOS;$env:GOARCH=$originalArch;$env:CGO_ENABLED=$originalCGO;Pop-Location}
