. "$PSScriptRoot/env.ps1"
. "$PSScriptRoot/modules.ps1"
New-Item -ItemType Directory -Force -Path security | Out-Null
$queries = @()
$modules=Get-ProjectModules
foreach ($line in $modules) {
    if (-not $line.Trim()) { continue }
    $fields = $line.Split('|')
    if ($fields.Count -lt 2 -or -not $fields[1]) { continue }
    $queries += @{package=@{name=[string]$fields[0];ecosystem='Go'};version=[string]$fields[1]}
}
$goVersion = (go env GOVERSION) -replace '^go',''
$queries += @{package=@{name='stdlib';ecosystem='Go'};version=$goVersion}
foreach ($frontendPackage in (Get-Content -LiteralPath third_party/frontend-packages.json -Raw -Encoding UTF8 | ConvertFrom-Json)) {
    $queries += @{package=@{name=[string]$frontendPackage.name;ecosystem='npm'};version=[string]$frontendPackage.version}
}
# Somente nomes e versões públicos são enviados. Nenhum código, perfil ou segredo.
$request = @{queries=$queries} | ConvertTo-Json -Depth 6 -Compress
$response = Invoke-RestMethod -Method Post -Uri 'https://api.osv.dev/v1/querybatch' -ContentType 'application/json' -Body $request -TimeoutSec 90
if (@($response.results).Count -ne $queries.Count) { throw 'Resposta OSV incompleta' }
$results = @()
for ($i=0; $i -lt $queries.Count; $i++) {
    $page = $response.results[$i]
    $vulns = @($page.vulns | Where-Object { $_ })
    while ($page.next_page_token) {
        $nextQuery=@{package=$queries[$i].package;version=$queries[$i].version;page_token=$page.next_page_token}
        $page = (Invoke-RestMethod -Method Post -Uri 'https://api.osv.dev/v1/querybatch' -ContentType 'application/json' -Body (@{queries=@($nextQuery)} | ConvertTo-Json -Depth 6 -Compress) -TimeoutSec 90).results[0]
        $vulns += @($page.vulns | Where-Object { $_ })
    }
    $results += [pscustomobject]@{Package=$queries[$i].package.name;Ecosystem=$queries[$i].package.ecosystem;Version=$queries[$i].version;Vulnerabilities=@($vulns | ForEach-Object {$_.id} | Where-Object {$_})}
}
$report=[pscustomobject]@{CheckedAtUTC=[DateTime]::UtcNow.ToString('o');Source='https://api.osv.dev/v1/querybatch';Scope='Imported Windows runtime/test modules, Linux core/adapters/tests, Go stdlib and vendored npm assets; package/version; no call-graph reachability analysis';Results=$results}
$report | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath security/dependencies-osv.json -Encoding UTF8
$affected=@($results | Where-Object {$_.Vulnerabilities.Count -gt 0})
Write-Host ($results.Count.ToString() + ' pacotes/versões consultados; ' + $affected.Count + ' com alertas conhecidos.')
if ($affected.Count) { $affected | Format-Table Package,Version,Vulnerabilities; throw 'Auditoria encontrou alertas; consulte security/dependencies-osv.json' }
