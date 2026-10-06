[CmdletBinding()]
param(
    [ValidateSet('amd64', 'arm64')][string]$Architecture = 'amd64',
    [string]$Version = $(if ($env:MCPDECK_VERSION) { $env:MCPDECK_VERSION } else { '0.1.0' })
)
$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^[A-Za-z0-9.+_-]+$') { throw 'Invalid package version.' }
$root = Split-Path $PSScriptRoot -Parent
$stage = Join-Path ([IO.Path]::GetTempPath()) ('mcpdeck-package-' + [Guid]::NewGuid().ToString('N'))
$oldOS, $oldArch, $oldCGO = $env:GOOS, $env:GOARCH, $env:CGO_ENABLED
Push-Location $root
try {
    $env:GOOS = 'windows'; $env:GOARCH = $Architecture; $env:CGO_ENABLED = '0'
    $null = New-Item -ItemType Directory -Path $stage
    & go build -trimpath -ldflags "-X github.com/altanmehmet/mcpdeck/cmd.version=$Version" -o (Join-Path $stage 'mcpdeck.exe') .
    if ($LASTEXITCODE -ne 0) { throw 'Windows build failed.' }
    Copy-Item scripts/install.ps1 $stage
    Copy-Item scripts/uninstall.ps1 $stage
    Copy-Item docs/CLI-BASLANGIC.md (Join-Path $stage 'README.txt')
    foreach ($name in @('AKILLI-KURULUM.md', 'INSTRUCTIONS.md', 'RECOVERY.md')) { Copy-Item (Join-Path 'docs' $name) $stage }
    Copy-Item LICENSE $stage
    $notices = New-Object IO.StreamWriter((Join-Path $stage 'THIRD_PARTY_NOTICES.txt'), $false, (New-Object Text.UTF8Encoding($false)))
    try {
        $notices.WriteLine('MCPDeck third-party notices - Go runtime and linked Windows modules')
        $goRoot = & go env GOROOT
        $notices.WriteLine([IO.File]::ReadAllText((Join-Path $goRoot 'LICENSE')))
        $modules = & go list -deps -f '{{if .Module}}{{if not .Module.Main}}{{.Module.Path}}|{{.Module.Version}}|{{.Module.Dir}}{{end}}{{end}}' .
        if ($LASTEXITCODE -ne 0) { throw 'Cannot list module licenses.' }
        foreach ($module in ($modules | Where-Object { $_ } | Sort-Object -Unique)) {
            $fields = $module -split '\|', 3
            $notices.WriteLine("`n=== $($fields[0]) $($fields[1]) ===`n")
            $found = $false
            foreach ($name in @('LICENSE', 'LICENSE.txt', 'LICENSE.md', 'LICENSE-MIT', 'LICENCE', 'COPYING')) {
                $path = Join-Path $fields[2] $name
                if (Test-Path -LiteralPath $path -PathType Leaf) { $notices.WriteLine([IO.File]::ReadAllText($path)); $found = $true }
            }
            # This exact upstream version ships its MIT declaration and author
            # in README.md instead of a separate LICENSE. Preserve it verbatim.
            if (!$found -and $fields[0] -eq 'github.com/mattn/go-localereader' -and $fields[1] -eq 'v0.0.1') {
                $readme = Join-Path $fields[2] 'README.md'
                if ((Get-FileHash -LiteralPath $readme -Algorithm SHA256).Hash -ne '0c5c52517f13becd7a1e1234f1e8a5ba370f5a5078cee9ac058c2d534e267fcb') {
                    throw 'Upstream license declaration changed; review it before packaging.'
                }
                $notices.WriteLine([IO.File]::ReadAllText($readme))
                $found = $true
            }
            if (!$found) { throw "Missing license for $($fields[0])." }
            $notice = Join-Path $fields[2] 'NOTICE'
            if (Test-Path -LiteralPath $notice -PathType Leaf) { $notices.WriteLine([IO.File]::ReadAllText($notice)) }
        }
    } finally { $notices.Dispose() }
    $null = New-Item -ItemType Directory -Force -Path dist
    $archive = Join-Path $root "dist/mcpdeck-windows-$Architecture.zip"
    if (Test-Path -LiteralPath $archive) { Remove-Item -LiteralPath $archive }
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    [IO.Compression.ZipFile]::CreateFromDirectory($stage, $archive)
    $checksum = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    [IO.File]::WriteAllText("$archive.sha256", "$checksum  $([IO.Path]::GetFileName($archive))`n", (New-Object Text.UTF8Encoding($false)))
    Write-Output "Windows package: $archive"
} finally {
    $env:GOOS = $oldOS; $env:GOARCH = $oldArch; $env:CGO_ENABLED = $oldCGO
    Pop-Location
    if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage -Recurse -Force }
}
