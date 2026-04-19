#requires -version 5.1
<#
.SYNOPSIS
  Fetches the prebuilt slop-cop binary for the current host into
  $PluginRoot\bin\slop-cop.exe. Called by the slop-cop-prose skill on
  first use from Claude Code or Cursor on Windows; safe to re-run.
#>

$ErrorActionPreference = 'Stop'

function Resolve-PluginRoot {
    if ($env:CLAUDE_PLUGIN_ROOT) { return $env:CLAUDE_PLUGIN_ROOT }
    if ($env:CURSOR_PLUGIN_ROOT) { return $env:CURSOR_PLUGIN_ROOT }
    return (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
}

$pluginRoot = Resolve-PluginRoot
$binDir = Join-Path $pluginRoot 'bin'
$binPath = Join-Path $binDir 'slop-cop.exe'

# Fast path: binary already works.
if (Test-Path $binPath) {
    try { & $binPath version | Out-Null; exit 0 } catch { }
}

switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { $arch = 'amd64' }
    'ARM64' { $arch = 'arm64' }
    default { throw "install-binary.ps1: unsupported arch: $env:PROCESSOR_ARCHITECTURE" }
}

$zip = "slop-cop_windows_${arch}.zip"
# /releases/latest/download/<asset> is GitHub's native redirect to the
# newest release's asset. Invoke-WebRequest follows the 302 by default.
$url = "https://github.com/yasyf/slop-cop/releases/latest/download/$zip"

New-Item -ItemType Directory -Force -Path $binDir | Out-Null
$tmp = New-Item -ItemType Directory -Path ([IO.Path]::GetTempPath()) -Name ([Guid]::NewGuid())
try {
    $archive = Join-Path $tmp.FullName 'slop-cop.zip'
    Write-Host "install-binary.ps1: downloading $url"
    Invoke-WebRequest -Uri $url -OutFile $archive -UseBasicParsing
    Expand-Archive -LiteralPath $archive -DestinationPath $tmp.FullName -Force
    $src = Join-Path $tmp.FullName "slop-cop_windows_${arch}\slop-cop.exe"
    Move-Item -Force -Path $src -Destination $binPath
    & $binPath version | Out-Null
    Write-Host "install-binary.ps1: installed $binPath"
}
finally {
    Remove-Item -Recurse -Force $tmp.FullName -ErrorAction SilentlyContinue
}
