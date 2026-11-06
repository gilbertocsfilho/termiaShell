$ErrorActionPreference = 'Stop'
$projectRoot = [IO.Path]::GetFullPath((Split-Path $PSScriptRoot -Parent))
$localRoot = Join-Path $projectRoot '.local'
# Apenas artefatos regeneráveis conhecidos. Não toca em data, código ou módulos Go.
$targets = @('assets','bin','checks','gocache','gopath','tmp','app-stderr.txt','app-stdout.txt')
foreach ($name in $targets) {
    $target = [IO.Path]::GetFullPath((Join-Path $localRoot $name))
    if (-not $target.StartsWith($localRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Destino fora da pasta de artefatos locais' }
    if (Test-Path -LiteralPath $target) {
        $item = Get-Item -LiteralPath $target -Force
        if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) { throw "Limpeza recusada: link em $target" }
        Remove-Item -LiteralPath $target -Recurse -Force
    }
}
Write-Host 'Artefatos temporários removidos. Código, dados e dependências locais preservados.'
