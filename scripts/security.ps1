param([switch]$AuditDependencies)
. "$PSScriptRoot/test.ps1"
go test ./internal/core -run '^$' -fuzz FuzzFilePathValidation -fuzztime 5s -parallel 2
if ($LASTEXITCODE -ne 0) { throw 'Fuzzing falhou' }
if ($AuditDependencies) { . "$PSScriptRoot/audit-dependencies.ps1" }
