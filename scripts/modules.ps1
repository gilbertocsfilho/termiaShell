function Get-ProjectModules {
    # Apenas módulos importados pelo aplicativo ou pelos testes.
    $format='{{with .Module}}{{.Path}}|{{.Version}}|{{.Dir}}{{end}}'
    $lines=@(go list -deps -test -tags 'desktop,production,wv2runtime.error' -f $format ./...)
    if ($LASTEXITCODE -ne 0) { throw 'Falha ao listar módulos Windows' }
    $savedGOOS=$env:GOOS
    $savedCGO=$env:CGO_ENABLED
    try {
        $env:GOOS='linux'
        $env:CGO_ENABLED='0'
        $lines+=go list -deps -test -f $format ./internal/... ./tests/frontend/...
        if ($LASTEXITCODE -ne 0) { throw 'Falha ao listar módulos Linux' }
    } finally {
        $env:GOOS=$savedGOOS
        $env:CGO_ENABLED=$savedCGO
    }
    $lines | Where-Object {$_ -and $_.Split('|').Count -eq 3 -and $_.Split('|')[1]} | Sort-Object -Unique
}
