@echo off
echo Building Windows Disk Cleanup Tool...

REM Download dependencies
go mod tidy

REM Build the executable
go build -ldflags "-s -w" -o disk-cleanup.exe main.go

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ✅ Build successful!
    echo 📁 Executable created: disk-cleanup.exe
    echo.
    echo To run the tool:
    echo   .\disk-cleanup.exe
    echo.
    echo For Administrator mode (recommended):
    echo   Right-click on disk-cleanup.exe ^> "Run as administrator"
) else (
    echo ❌ Build failed!
)

pause
