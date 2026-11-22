$ErrorActionPreference = 'Stop'
$serviceRoot = Split-Path $PSScriptRoot
$repoRoot = Split-Path $serviceRoot
$compiler = Get-Command protoc -CommandType Application -ErrorAction SilentlyContinue
if ($compiler) {
    $protoc = $compiler.Source
} else {
    $protoc = Join-Path $repoRoot 'protoc/bin/protoc.exe'
    if (!(Test-Path -LiteralPath $protoc)) {
        throw 'Install protoc and add it to PATH.'
    }
}

Push-Location $serviceRoot
try {
    & $protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/database.proto
    if ($LASTEXITCODE -ne 0) {
        throw "protoc exited with code $LASTEXITCODE"
    }

    foreach ($consumer in 'ai-service', 'api-gateway', 'auth-service') {
        $destination = Join-Path $repoRoot "$consumer/proto/db"
        foreach ($name in 'database.proto', 'database.pb.go', 'database_grpc.pb.go') {
            Copy-Item -LiteralPath (Join-Path $PSScriptRoot $name) -Destination $destination
        }
    }
} finally {
    Pop-Location
}
