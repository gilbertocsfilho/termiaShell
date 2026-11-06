. "$PSScriptRoot/build.ps1"
Get-FileHash -LiteralPath dist/termiaShell.exe -Algorithm SHA256 | ForEach-Object { $_.Hash + '  termiaShell.exe' } | Set-Content -LiteralPath dist/SHA256.txt -Encoding ASCII
Compress-Archive -LiteralPath dist/termiaShell.exe,dist/termiaShell.svg,dist/termiaShell.png,dist/termiaShell.ico,dist/README.md,dist/SDD.md,dist/VALIDATION.md,dist/SECURITY.md,dist/security,dist/SHA256.txt,dist/third_party -DestinationPath dist/termiaShell-Windows-x64.zip -Force
Write-Host 'Pacote: dist/termiaShell-Windows-x64.zip'
