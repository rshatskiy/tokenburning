# tokenburning installer for Windows — downloads the latest release binary.
$ErrorActionPreference = "Stop"
$repo = "rshatskiy/tokenburning"
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

$tag = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
$ver = $tag.TrimStart("v")
$url = "https://github.com/$repo/releases/download/$tag/tokenburning_${ver}_windows_${arch}.zip"

$dest = Join-Path $env:LOCALAPPDATA "tokenburning"
New-Item -ItemType Directory -Force -Path $dest | Out-Null
$zip = Join-Path $env:TEMP "tokenburning.zip"

Write-Host "tokenburning: downloading $url"
Invoke-WebRequest -Uri $url -OutFile $zip
Expand-Archive -Path $zip -DestinationPath $dest -Force
Remove-Item $zip

Write-Host "tokenburning: installed to $dest\tokenburning.exe ($tag)"
Write-Host "tokenburning: add to PATH:  setx PATH `"$dest;$env:PATH`""
Write-Host "tokenburning: run  tokenburning scan"
