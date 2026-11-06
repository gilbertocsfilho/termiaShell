. "$PSScriptRoot/env.ps1"
. "$PSScriptRoot/modules.ps1"
$licenseRoot = Join-Path $projectRoot 'third_party/go'
New-Item -ItemType Directory -Force -Path $licenseRoot | Out-Null
$lines = Get-ProjectModules
$manifest = @()
foreach ($line in $lines) {
    $fields = $line.Split('|')
    if ($fields.Count -lt 3 -or -not $fields[2]) { continue }
    $moduleName = $fields[0].Replace('/','_')
    $target = Join-Path $licenseRoot $moduleName
    $licenses = Get-ChildItem -LiteralPath $fields[2] -File | Where-Object { $_.Name -match '^(LICENSE|COPYING|NOTICE|COPYRIGHT)' }
    if ($licenses) {
        New-Item -ItemType Directory -Force -Path $target | Out-Null
        foreach ($license in $licenses) { Copy-Item -LiteralPath $license.FullName -Destination (Join-Path $target $license.Name) -Force }
    }
    $manifest += [pscustomobject]@{Module=$fields[0];Version=$fields[1];LicenseFiles=@($licenses.Name)}
}
$manifest | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath third_party/go-modules.json -Encoding UTF8
$assets = Get-ChildItem frontend/vendor -File | ForEach-Object { [pscustomobject]@{File=$_.Name;SHA256=(Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash} }
$assets | ConvertTo-Json | Set-Content -LiteralPath third_party/assets-sha256.json -Encoding UTF8
