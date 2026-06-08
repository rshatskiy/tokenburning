# tokenburning installer for Windows — downloads the latest release binary.
$ErrorActionPreference = "Stop"
$repo = "rshatskiy/tokenburning"
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

$tag = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
$ver = if ($tag.StartsWith("v")) { $tag.Substring(1) } else { $tag }
$fname = "tokenburning_${ver}_windows_${arch}.zip"
# Зеркало на своём домене (GitHub release-CDN нестабилен в РФ); GitHub — фолбэк.
$mirror = "https://tokenburning.ru/dl/$tag"
$ghbase = "https://github.com/$repo/releases/download/$tag"
function Fetch($name, $out) {
    try { Invoke-WebRequest -Uri "$mirror/$name" -OutFile $out -ErrorAction Stop }
    catch { Invoke-WebRequest -Uri "$ghbase/$name" -OutFile $out }
}

$dest = Join-Path $env:LOCALAPPDATA "tokenburning"
New-Item -ItemType Directory -Force -Path $dest | Out-Null
$zip = Join-Path $env:TEMP $fname
$sums = Join-Path $env:TEMP "tokenburning_checksums.txt"

Write-Host "tokenburning: downloading $fname ($tag)"
try {
    Fetch $fname $zip
    Fetch "checksums.txt" $sums

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

$exe = Join-Path $dest "tokenburning.exe"
Write-Host "tokenburning: installed to $exe ($tag, checksum verified)"

# Добавляем в ПОЛЬЗОВАТЕЛЬСКИЙ PATH (не весь раскрытый $env:PATH — иначе setx режет на 1024
# символах) и сразу обновляем текущую сессию (через iex скрипт исполняется в ней).
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if (-not $userPath) { $userPath = "" }
if ($userPath -notlike "*$dest*") {
    [Environment]::SetEnvironmentVariable("PATH", ($userPath.TrimEnd(';') + ";" + $dest), "User")
}
if ($env:PATH -notlike "*$dest*") { $env:PATH = "$env:PATH;$dest" }

Write-Host ""
Write-Host "Installed (v$ver)."
Write-Host ""
Write-Host "See your AI spend (local, nothing leaves your machine):"
Write-Host "    tokenburning dashboard     # visual dashboard in your browser"
Write-Host "    tokenburning scan          # quick numbers in the terminal"
Write-Host ""
Write-Host "Send your stats to a team dashboard (optional):"
Write-Host "    1) open  https://tokenburning.ru/install   ->  click 'Generate token'"
Write-Host "    2) run:  tokenburning connect --to https://tokenburning.ru --token <YOUR-TOKEN> --breadth"
Write-Host ""
Write-Host "Opening your dashboard..."
Start-Process -FilePath $exe -ArgumentList "dashboard"
