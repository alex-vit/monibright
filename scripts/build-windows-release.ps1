param(
  [string]$Version,
  [string]$Arch = "amd64"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $PSScriptRoot

Push-Location $RepoRoot
try {
  if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = (git describe --tags --always --dirty 2>$null)
    if ([string]::IsNullOrWhiteSpace($Version)) {
      $Version = "dev"
    }
  }

  $outDir = Join-Path $RepoRoot "out"
  New-Item -ItemType Directory -Path $outDir -Force | Out-Null

  $env:GOOS = "windows"
  $env:GOARCH = $Arch
  $env:CGO_ENABLED = "0"
  $ldflags = "-X main.version=$Version -H=windowsgui"
  go build -ldflags $ldflags -o (Join-Path $outDir "monibright.exe") .
  if ($LASTEXITCODE -ne 0) {
    throw "go build failed"
  }

  $isccCmd = Get-Command iscc.exe -ErrorAction SilentlyContinue
  $isccExe = if ($isccCmd) { $isccCmd.Source } else { $null }
  if (-not $isccExe) {
    $candidate = Join-Path ${env:ProgramFiles(x86)} "Inno Setup 6\ISCC.exe"
    if (Test-Path $candidate) {
      $isccExe = $candidate
    }
  }
  if (-not $isccExe) {
    throw "Inno Setup compiler (iscc.exe) not found in PATH"
  }

  & $isccExe "/DAppVersion=$Version" "installer.iss"
  if ($LASTEXITCODE -ne 0) {
    throw "Installer build failed"
  }

  $exePath = Join-Path $outDir "monibright.exe"
  $setupPath = Join-Path $outDir "monibright-setup.exe"
  if (-not (Test-Path $exePath)) {
    throw "Missing executable: $exePath"
  }
  if (-not (Test-Path $setupPath)) {
    throw "Missing installer: $setupPath"
  }

  Write-Host "Built executable: $exePath"
  Write-Host "Built installer:  $setupPath"
}
finally {
  Pop-Location
}
