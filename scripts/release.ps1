param(
    [Parameter(Mandatory=$true)]
    [string]$Version,
    [switch]$DryRun
)

if ($Version -notmatch '^v\d+\.\d+\.\d+$') {
    Write-Error "Version must be in format 'v1.0.16' (with 'v' prefix)"
    exit 1
}

Write-Host "Preparing release $Version" -ForegroundColor Green


if (-not (Test-Path ".git")) {
    Write-Error "This script must be run from the git repository root"
    exit 1
}

$gitStatus = git status --porcelain
if ($gitStatus) {
    Write-Warning "Working directory has uncommitted changes:"
    Write-Host $gitStatus
    
    if (-not $DryRun) {
        $continue = Read-Host "Continue anyway? (y/N)"
        if ($continue -ne "y" -and $continue -ne "Y") {
            Write-Host "Aborted by user" -ForegroundColor Yellow
            exit 0
        }
    }
}

$constFile = "internal/types.go"
if (Test-Path $constFile) {
    Write-Host "Updating version in $constFile" -ForegroundColor Blue
    
    if (-not $DryRun) {
        $content = Get-Content $constFile -Raw
        $pattern = 'AppVersion\s*=\s*"v[0-9.]+"'
        $replacement = 'AppVersion   = "' + $Version + '"'
        $newContent = $content -replace $pattern, $replacement
        Set-Content $constFile -Value $newContent -NoNewline
        Write-Host "Updated $constFile" -ForegroundColor Green
    } else {
        Write-Host "DRY RUN: Would update $constFile" -ForegroundColor Yellow
    }
}

Write-Host "Testing build..." -ForegroundColor Blue
$buildResult = go build -o "test-build.exe" .
if ($LASTEXITCODE -ne 0) {
    Write-Error "Build failed! Fix errors before releasing."
    exit 1
}


if (Test-Path "test-build.exe") {
    Remove-Item "test-build.exe"
}

Write-Host "Build test successful" -ForegroundColor Green

if (-not $DryRun) {
    
    Write-Host "Committing version updates..." -ForegroundColor Blue
    git add $constFile
    git commit -m "Bump version to $Version"
    
    
    Write-Host "Creating tag $Version..." -ForegroundColor Blue
    git tag $Version
    
    Write-Host "Pushing tag to trigger release..." -ForegroundColor Blue
    git push origin $Version
    
    Write-Host "Release $Version initiated!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Yellow
    Write-Host "1. Monitor GitHub Actions: https://github.com/ur-wesley/modhelper/actions" -ForegroundColor White
    Write-Host "2. Check release: https://github.com/ur-wesley/modhelper/releases" -ForegroundColor White
    Write-Host "3. Download and test the built executable" -ForegroundColor White
} else {
    Write-Host "DRY RUN: Would create tag $Version and push to trigger release" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Release process complete!" -ForegroundColor Green 