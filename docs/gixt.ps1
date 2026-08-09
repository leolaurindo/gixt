[CmdletBinding()]
param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\gixt\bin"
)

$ErrorActionPreference = "Stop"
$repo = "leolaurindo/gixt"
$architecture = if ($env:PROCESSOR_ARCHITEW6432) {
    $env:PROCESSOR_ARCHITEW6432
} else {
    $env:PROCESSOR_ARCHITECTURE
}

if ($architecture -ne "AMD64") {
    throw "Unsupported architecture: $architecture"
}

[Net.ServicePointManager]::SecurityProtocol = `
    [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

if ($Version -eq "latest") {
    $release = Invoke-RestMethod -Headers @{ "User-Agent" = "gixt-installer" } `
        -Uri "https://api.github.com/repos/$repo/releases/latest"
    $Version = $release.tag_name
} elseif (-not $Version.StartsWith("v")) {
    $Version = "v$Version"
}

if ($Version -notmatch '^v[0-9]') {
    throw "Could not determine a release version"
}

$asset = "gixt_${Version}_windows_amd64.zip"
$releaseBase = "https://github.com/$repo/releases/download/$Version"
$tempDir = Join-Path ([IO.Path]::GetTempPath()) "gixt-$([guid]::NewGuid())"

try {
    New-Item -ItemType Directory -Path $tempDir | Out-Null
    $archive = Join-Path $tempDir $asset
    $checksums = Join-Path $tempDir "checksums.txt"

    Write-Host "Downloading gixt $Version for windows/amd64..."
    Invoke-WebRequest -UseBasicParsing -Uri "$releaseBase/$asset" -OutFile $archive
    Invoke-WebRequest -UseBasicParsing -Uri "$releaseBase/checksums.txt" -OutFile $checksums

    $expected = $null
    foreach ($line in Get-Content $checksums) {
        $parts = $line.Trim() -split '\s+', 2
        if ($parts.Count -eq 2 -and ($parts[1] -replace '^\*', '') -eq $asset) {
            $expected = $parts[0]
            break
        }
    }
    if (-not $expected) {
        throw "Checksum not found for $asset"
    }

    $actual = (Get-FileHash -Algorithm SHA256 $archive).Hash
    if ($actual -ine $expected) {
        throw "Checksum verification failed"
    }

    Expand-Archive -Path $archive -DestinationPath $tempDir -Force
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item (Join-Path $tempDir "gixt.exe") (Join-Path $InstallDir "gixt.exe") -Force

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $hasInstallDir = @($userPath -split ';') | Where-Object {
        $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\')
    }
    if (-not $hasInstallDir) {
        $newPath = if ([string]::IsNullOrWhiteSpace($userPath)) {
            $InstallDir
        } else {
            "$userPath;$InstallDir"
        }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = "$env:Path;$InstallDir"
        Write-Host "Added $InstallDir to your user PATH. Open a new terminal to use it."
    }

    Write-Host "Installed gixt $Version to $InstallDir\gixt.exe"
} finally {
    if (Test-Path $tempDir) {
        Remove-Item -Recurse -Force $tempDir
    }
}
