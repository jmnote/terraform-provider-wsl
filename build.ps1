<#
.SYNOPSIS
    Development tasks for terraform-provider-wsl.

.DESCRIPTION
    A thin wrapper around plain `go`/`gofmt` commands -- see each task
    below for the command it runs. Requires nothing beyond PowerShell and
    a Go toolchain already on PATH: no `make`, no extra installer. This
    provider only ever builds/tests on native Windows (see
    docs/design/decisions/platform-support.md), so a native PowerShell
    script fits better here than a GNU Makefile would.

.PARAMETER Task
    fmt | vet | test | build | install | generate | testacc | all (default)

.EXAMPLE
    .\build.ps1
    .\build.ps1 test
    .\build.ps1 testacc
#>
param(
    [Parameter(Position = 0)]
    [ValidateSet('all', 'fmt', 'vet', 'test', 'build', 'install', 'generate', 'testacc')]
    [string]$Task = 'all'
)

$ErrorActionPreference = 'Stop'

function Invoke-Fmt { gofmt -l -w . }
function Invoke-Vet { go vet ./... }
function Invoke-Test { go test -v -count=1 ./... }
function Invoke-Build { go build -v ./... }
function Invoke-Install { go install -v ./... }
function Invoke-Generate { go generate ./... }

function Invoke-TestAcc {
    # Acceptance tests create and destroy real WSL instances; see
    # CONTRIBUTING.md, "Acceptance tests", for the WSL_ACC_TEST_DISTRIBUTION /
    # WSL_ACC_TEST_ROOTFS environment variables each one additionally
    # needs. Neither runs unless one of those is also set -- deliberately
    # not defaulted here, since a fixed distribution identifier could
    # collide with one a given host already has registered under.
    $env:TF_ACC = '1'
    if (-not $env:TF_LOG) {
        # These tests drive real, multi-minute wsl.exe install/import calls;
        # default to DEBUG so their progress is visible instead of a long
        # silence. Only applied when the caller hasn't already chosen a
        # level.
        $env:TF_LOG = 'DEBUG'
    }
    go test -v -count=1 ./internal/provider/...
}

switch ($Task) {
    'fmt' { Invoke-Fmt }
    'vet' { Invoke-Vet }
    'test' { Invoke-Test }
    'build' { Invoke-Build }
    'install' { Invoke-Install }
    'generate' { Invoke-Generate }
    'testacc' { Invoke-TestAcc }
    'all' {
        Invoke-Fmt
        Invoke-Vet
        Invoke-Test
        Invoke-Build
    }
}
