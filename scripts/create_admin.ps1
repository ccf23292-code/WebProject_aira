param(
  [Parameter(Mandatory = $true)]
  [string]$Username,

  [Parameter(Mandatory = $true)]
  [string]$Password,

  [Parameter(Mandatory = $true)]
  [string]$Email,

  [string]$Nickname = "admin"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$backDir = Join-Path $repoRoot "back"

if (-not $env:DATABASE_URL) {
  throw "DATABASE_URL is required before running this script."
}

Push-Location $backDir
try {
  go run .\cmd\seed_admin --username $Username --password $Password --email $Email --nickname $Nickname
}
finally {
  Pop-Location
}
