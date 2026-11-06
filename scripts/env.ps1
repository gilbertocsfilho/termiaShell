$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path $PSScriptRoot -Parent
Set-Location -LiteralPath $projectRoot
$env:GOPATH = Join-Path $projectRoot '.local/gopath'
$env:GOMODCACHE = Join-Path $projectRoot '.local/gomodcache'
$env:GOCACHE = Join-Path $projectRoot '.local/gocache'
$env:GOBIN = Join-Path $projectRoot '.local/bin'
$env:GOTMPDIR = Join-Path $projectRoot '.local/tmp'
$env:TEMP = $env:GOTMPDIR
$env:TMP = $env:GOTMPDIR
$env:GOTOOLCHAIN = 'local'
@($env:GOPATH,$env:GOMODCACHE,$env:GOCACHE,$env:GOBIN,$env:GOTMPDIR) | ForEach-Object { New-Item -ItemType Directory -Force -Path $_ | Out-Null }
