. "$PSScriptRoot/env.ps1"
New-Item -ItemType Directory -Force -Path dist | Out-Null
go build -trimpath -tags 'desktop,production,wv2runtime.error' -ldflags '-H windowsgui -s -w' -o dist/termiaShell.exe .
if ($LASTEXITCODE -ne 0) { throw 'Falha no build' }
Copy-Item README.md,SDD.md -Destination dist
if (Test-Path VALIDATION.md) { Copy-Item VALIDATION.md -Destination dist }
if (Test-Path SECURITY.md) { Copy-Item SECURITY.md -Destination dist }
if (Test-Path security/dependencies-osv.json) {
    New-Item -ItemType Directory -Force -Path dist/security | Out-Null
    Copy-Item security/dependencies-osv.json -Destination dist/security
}
Copy-Item -Path third_party -Destination dist -Recurse -Force
Copy-Item frontend/termiaShell.svg,frontend/termiaShell.png,frontend/termiaShell.ico -Destination dist
Write-Host 'Pronto: dist/termiaShell.exe'
