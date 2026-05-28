[CmdletBinding()]
param(
    [string]$AiUrl = $env:MUSU_MARKETER_INTEGRATION_AI_URL,
    [string]$Model = $env:MUSU_MARKETER_INTEGRATION_MODEL,
    [switch]$Json,
    [switch]$ProbeOnly
)

$ErrorActionPreference = "Stop"

function Get-OllamaCandidates {
    $candidates = @()

    if ($env:OLLAMA_HOST) {
        $hostValue = $env:OLLAMA_HOST.Trim()
        if ($hostValue) {
            if ($hostValue -notmatch '^https?://') {
                $hostValue = 'http://' + $hostValue
            }
            $normalized = $hostValue.TrimEnd('/')
            if ($normalized -match '/v1$') {
                $candidates += $normalized
            } else {
                $candidates += ($normalized + '/v1')
            }
        }
    }

    $candidates += @(
        'http://127.0.0.1:11434/v1',
        'http://localhost:11434/v1'
    )

    $seen = @{}
    foreach ($candidate in $candidates) {
        if (-not $seen.ContainsKey($candidate)) {
            $seen[$candidate] = $true
            $candidate
        }
    }
}

function Test-ModelEndpoint([string]$BaseUrl) {
    try {
        $probe = Invoke-WebRequest -UseBasicParsing -Uri ($BaseUrl.TrimEnd('/') + '/models') -TimeoutSec 3
        $modelIds = @()
        try {
            $json = $probe.Content | ConvertFrom-Json
            if ($json.data) {
                foreach ($entry in $json.data) {
                    if ($entry.id) {
                        $modelIds += [string]$entry.id
                    }
                }
            }
        } catch {}
        return @{
            ok = $true
            detail = "HTTP $($probe.StatusCode)"
            base = $BaseUrl
            models = @($modelIds)
        }
    } catch {
        return @{
            ok = $false
            detail = $_.Exception.Message
            base = $BaseUrl
            models = @()
        }
    }
}

function Test-OllamaApi([string]$BaseUrlWithoutV1) {
    try {
        $probe = Invoke-WebRequest -UseBasicParsing -Uri ($BaseUrlWithoutV1.TrimEnd('/') + '/api/tags') -TimeoutSec 3
        return @{
            ok = $true
            detail = "HTTP $($probe.StatusCode)"
        }
    } catch {
        return @{
            ok = $false
            detail = $_.Exception.Message
        }
    }
}

function Get-ExecutableHints {
    $paths = @(
        "$env:LOCALAPPDATA\Programs\Ollama\ollama.exe",
        "$env:ProgramFiles\Ollama\ollama.exe",
        "${env:ProgramFiles(x86)}\Ollama\ollama.exe"
    )
    $existing = @()
    foreach ($path in $paths) {
        if ($path -and (Test-Path $path)) {
            $existing += $path
        }
    }
    return $existing
}

function Test-RequiredModelAvailable([string[]]$AvailableModels, [string]$RequiredModel) {
    if (-not $AvailableModels -or $AvailableModels.Count -eq 0) {
        return $true
    }
    if (-not $RequiredModel) {
        return $true
    }

    $normalizedRequired = $RequiredModel.Trim().ToLowerInvariant()
    $requiredBase = $normalizedRequired -replace ':latest$',''

    foreach ($available in $AvailableModels) {
        $normalizedAvailable = ([string]$available).Trim().ToLowerInvariant()
        $availableBase = $normalizedAvailable -replace ':latest$',''
        if ($normalizedAvailable -eq $normalizedRequired -or $availableBase -eq $requiredBase) {
            return $true
        }
    }

    return $false
}

function Resolve-AiUrl([string]$ExplicitUrl) {
    if ($ExplicitUrl) {
        return @{
            Resolved = $ExplicitUrl
            Source = 'explicit'
            Diagnostics = @()
        }
    }

    $diagnostics = @()
    foreach ($candidate in Get-OllamaCandidates) {
        $probe = Test-ModelEndpoint $candidate
        $diagnostics += "probe $($probe.base)/models -> $($probe.detail)"
        if ($probe.ok) {
            return @{
                Resolved = $candidate
                Source = 'autodiscovered-model-endpoint'
                Diagnostics = $diagnostics
            }
        }

        $base = $candidate -replace '/v1$',''
        $ollamaProbe = Test-OllamaApi $base
        $diagnostics += "probe $base/api/tags -> $($ollamaProbe.detail)"
        if ($ollamaProbe.ok) {
            return @{
                Resolved = $base + '/v1'
                Source = 'autodiscovered-ollama-api'
                Diagnostics = $diagnostics
            }
        }
    }

    return @{
        Resolved = $null
        Source = 'none'
        Diagnostics = $diagnostics
    }
}

function Get-ActionableFix([hashtable]$Result) {
    $issueCodes = @($Result.issue_codes)
    if ($issueCodes -contains 'ollama_host_unspecified_bind_address') {
        return "OLLAMA_HOST is set to a bind address (0.0.0.0/::) that clients cannot target. Set MUSU_MARKETER_INTEGRATION_AI_URL to http://127.0.0.1:11434/v1 or set OLLAMA_HOST to a reachable host."
    }
    if ($issueCodes -contains 'ollama_not_installed') {
        return "No common ollama.exe install path was found. Install Ollama or set MUSU_MARKETER_INTEGRATION_AI_URL to another reachable OpenAI-compatible endpoint."
    }
    if ($issueCodes -contains 'localhost_probe_timeout') {
        return "A localhost candidate timed out. Confirm the AI runtime is actually serving /v1/models and not hung behind a stale OLLAMA_HOST setting."
    }
    if ($issueCodes -contains 'loopback_connection_refused') {
        return "Loopback probes were refused. Start a local Ollama-compatible server or set MUSU_MARKETER_INTEGRATION_AI_URL explicitly."
    }
    if ($issueCodes -contains 'missing_required_model') {
        return "The endpoint is reachable but the required marketer chat model is missing. Set MUSU_MARKETER_INTEGRATION_MODEL to an available chat model or pull the expected model."
    }
    return "Set MUSU_MARKETER_INTEGRATION_AI_URL or start a local Ollama-compatible server."
}

function Emit-Result([hashtable]$Result, [int]$ExitCode) {
    if ($Json) {
        $Result | ConvertTo-Json -Depth 6
    } else {
        foreach ($line in $Result.diagnostics) {
            Write-Host "DIAG: $line"
        }
        if ($Result.status -eq 'success') {
            Write-Host "==> status: success"
            if ($Result.resolved_ai_url) {
                Write-Host "==> resolved_ai_url: $($Result.resolved_ai_url)"
            }
            if ($Result.tests_ran) {
                Write-Host "==> integration tests passed"
            }
        } else {
            Write-Host "==> status: error"
            Write-Host "==> message: $($Result.message)"
            if ($Result.actionable_fix) {
                Write-Host "==> actionable_fix: $($Result.actionable_fix)"
            }
        }
    }
    exit $ExitCode
}

$resolution = Resolve-AiUrl $AiUrl
$result = @{
    status = "error"
    tool = "musu-marketer"
    resolved_ai_url = $null
    resolution_source = $resolution.Source
    diagnostics = $resolution.Diagnostics
    issue_codes = @()
    available_models = @()
    tests_ran = $false
    actionable_fix = $null
    message = $null
}

if (-not $resolution.Resolved) {
    $ollamaExe = Get-ExecutableHints
    if ($env:OLLAMA_HOST) {
        $result.diagnostics += "OLLAMA_HOST=$($env:OLLAMA_HOST)"
        if ($env:OLLAMA_HOST -match '^(0\.0\.0\.0|\[?::0?\]?)(:\d+)?$' -or $env:OLLAMA_HOST -match '^0\.0\.0\.0:\d+$') {
            $result.issue_codes += "ollama_host_unspecified_bind_address"
        }
    }
    if ($ollamaExe.Count -gt 0) {
        $result.diagnostics += "found ollama executable(s): $($ollamaExe -join ', ')"
    } else {
        $result.diagnostics += "no common ollama.exe install path found"
        $result.issue_codes += "ollama_not_installed"
    }

    foreach ($line in $result.diagnostics) {
        if ($line -like 'probe http://localhost:11434*timed out*') {
            $result.issue_codes += "localhost_probe_timeout"
        }
        if ($line -like 'probe http://127.0.0.1:11434*연결할 수 없습니다*') {
            $result.issue_codes += "loopback_connection_refused"
        }
    }
    $result.message = "No reachable OpenAI-compatible AI endpoint found."
    $result.issue_codes = @($result.issue_codes | Select-Object -Unique)
    $result.actionable_fix = Get-ActionableFix $result
    Emit-Result $result 1
}

$AiUrl = $resolution.Resolved
$result.status = "success"
$result.resolved_ai_url = $AiUrl
$result.message = "Reachable AI endpoint found."
$result.available_models = @()
$requiredModel = $Model
if (-not $requiredModel) {
    $requiredModel = "llama3"
}
foreach ($candidate in Get-OllamaCandidates) {
    if ($candidate -eq $AiUrl) {
        $probe = Test-ModelEndpoint $AiUrl
        $result.available_models = @($probe.models)
        break
    }
}

if (-not (Test-RequiredModelAvailable $result.available_models $requiredModel)) {
    $result.status = "error"
    $result.message = "Reachable AI endpoint found, but required model is missing."
    $result.issue_codes = @("missing_required_model")
    $result.actionable_fix = Get-ActionableFix $result
    Emit-Result $result 1
}

$env:MUSU_MARKETER_INTEGRATION_AI_URL = $AiUrl
if (-not $env:MUSU_MARKETER_INTEGRATION_MODEL) {
    $env:MUSU_MARKETER_INTEGRATION_MODEL = $requiredModel
}
if ($ProbeOnly) {
    Emit-Result $result 0
}

$testOutput = & go test -tags integration ./cmd 2>&1 | Out-String
if ($LASTEXITCODE -ne 0) {
    $result.status = "error"
    $result.message = "go test -tags integration ./cmd failed"
    $result.actionable_fix = "Inspect test output and verify the endpoint is OpenAI-compatible for marketer draft flows."
    $result.test_output = $testOutput.Trim()
    Emit-Result $result 1
}

$result.tests_ran = $true
$result.test_output = $testOutput.Trim()
$result.message = "marketer real integration passed"
Emit-Result $result 0
