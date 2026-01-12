@echo off
echo Building Tunnel Manager...
go build -ldflags="-s -w" -o tunnel-manager.exe .\tunnel_manager.go

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Build successful!
    echo.
    
    if exist "..\..\externals\cloudflared.exe" (
        echo Copying cloudflared.exe...
        copy /Y "..\..\externals\cloudflared.exe" .\cloudflared.exe
        echo.
        echo Done! You can now run tunnel-manager.exe
        echo Both files are in: %CD%
    ) else (
        echo WARNING: cloudflared.exe not found in externals folder
        echo Please download cloudflared.exe and place it in the same folder as tunnel-manager.exe
    )
) else (
    echo Build failed!
)
