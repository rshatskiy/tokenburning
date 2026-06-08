# tokenburning installer for Windows — downloads the latest release binary.
$ErrorActionPreference = "Stop"
$repo = "rshatskiy/tokenburning"
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

$tag = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
$ver = if ($tag.StartsWith("v")) { $tag.Substring(1) } else { $tag }
$fname = "tokenburning_${ver}_windows_${arch}.zip"
$base = "https://github.com/$repo/releases/download/$tag"

$dest = Join-Path $env:LOCALAPPDATA "tokenburning"
New-Item -ItemType Directory -Force -Path $dest | Out-Null
$zip = Join-Path $env:TEMP $fname
$sums = Join-Path $env:TEMP "tokenburning_checksums.txt"

Write-Host "tokenburning: downloading $fname ($tag)"
try {
    Invoke-WebRequest -Uri "$base/$fname" -OutFile $zip
    Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile $sums

    # verify SHA-256 against checksums.txt before installing
    $want = (Select-String -Path $sums -Pattern ([regex]::Escape($fname)) | Select-Object -First 1).Line -split '\s+' | Select-Object -First 1
    if (-not $want) { throw "no checksum for $fname in checksums.txt — refusing to install" }
    $got = (Get-FileHash -Algorithm SHA256 $zip).Hash.ToLower()
    if ($want.ToLower() -ne $got) {
        throw "CHECKSUM MISMATCH — refusing to install (expected $want, got $got)"
    }

    Expand-Archive -Path $zip -DestinationPath $dest -Force
} finally {
    if (Test-Path $zip)  { Remove-Item $zip }
    if (Test-Path $sums) { Remove-Item $sums }
}

Write-Host "tokenburning: installed to $dest\tokenburning.exe ($tag, checksum verified)"
Write-Host "tokenburning: add to PATH:  setx PATH `"$dest;$env:PATH`""
Write-Host ""
Write-Host "Installed. Run:"
Write-Host "    tokenburning scan"
