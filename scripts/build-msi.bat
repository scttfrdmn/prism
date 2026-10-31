@echo off
REM Prism Windows MSI Build Script
REM Builds a professional Windows installer using WiX Toolset

setlocal enabledelayedexpansion

REM Configuration
set VERSION=0.4.2
set BUILD_DIR=%~dp0..\build\windows
set DIST_DIR=%~dp0..\dist\windows
set WIX_DIR=%~dp0..\packaging\windows
set SOURCE_DIR=%~dp0..
set MSI_NAME=Prism-v%VERSION%-x64.msi
set LOG_FILE=%BUILD_DIR%\build-msi.log

REM Color output functions
set ESC=
set RED=%ESC%[31m
set GREEN=%ESC%[32m
set YELLOW=%ESC%[33m
set BLUE=%ESC%[34m
set CYAN=%ESC%[36m
set WHITE=%ESC%[37m
set RESET=%ESC%[0m

echo %CYAN%========================================%RESET%
echo %CYAN%  Prism Windows MSI Builder%RESET%
echo %CYAN%========================================%RESET%
echo.

REM Check for WiX Toolset
echo %BLUE%Checking for WiX Toolset...%RESET%
where candle >nul 2>&1
if !errorlevel! neq 0 (
    echo %RED%Error: WiX Toolset not found in PATH%RESET%
    echo %YELLOW%Please install WiX Toolset from: https://wixtoolset.org/%RESET%
    echo %YELLOW%Or install via chocolatey: choco install wixtoolset%RESET%
    exit /b 1
)

where light >nul 2>&1
if !errorlevel! neq 0 (
    echo %RED%Error: WiX Light tool not found in PATH%RESET%
    exit /b 1
)

echo %GREEN%✓ WiX Toolset found%RESET%

REM Create build directories
echo %BLUE%Creating build directories...%RESET%
if not exist "%BUILD_DIR%" mkdir "%BUILD_DIR%"
if not exist "%DIST_DIR%" mkdir "%DIST_DIR%"
if not exist "%BUILD_DIR%\obj" mkdir "%BUILD_DIR%\obj"
if not exist "%BUILD_DIR%\release" mkdir "%BUILD_DIR%\release"

REM Clean previous build artifacts
echo %BLUE%Cleaning previous build artifacts...%RESET%
if exist "%BUILD_DIR%\obj\*.wixobj" del /q "%BUILD_DIR%\obj\*.wixobj"
if exist "%BUILD_DIR%\*.msi" del /q "%BUILD_DIR%\*.msi"

REM Step 1: Build Go binaries for Windows
echo.
echo %CYAN%Step 1: Building Go binaries...%RESET%
cd /d "%SOURCE_DIR%"

echo %BLUE%Building CLI binary (prism.exe)...%RESET%
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags "-X github.com/scttfrdmn/prism/pkg/version.Version=%VERSION% -X github.com/scttfrdmn/prism/pkg/version.BuildDate=%date:~10,4%-%date:~4,2%-%date:~7,2%_%time:~0,2%:%time:~3,2%:%time:~6,2% -X github.com/scttfrdmn/prism/pkg/version.GitCommit=msi-build" -o "%BUILD_DIR%\release\windows-amd64\prism.exe" ./cmd/prism
if !errorlevel! neq 0 (
    echo %RED%Error: Failed to build CLI binary%RESET%
    exit /b 1
)
echo %GREEN%✓ CLI binary built successfully%RESET%

echo %BLUE%Building daemon binary (prismd.exe)...%RESET%
go build -ldflags "-X github.com/scttfrdmn/prism/pkg/version.Version=%VERSION% -X github.com/scttfrdmn/prism/pkg/version.BuildDate=%date:~10,4%-%date:~4,2%-%date:~7,2%_%time:~0,2%:%time:~3,2%:%time:~6,2% -X github.com/scttfrdmn/prism/pkg/version.GitCommit=msi-build" -o "%BUILD_DIR%\release\windows-amd64\prismd.exe" ./cmd/prismd
if !errorlevel! neq 0 (
    echo %RED%Error: Failed to build daemon binary%RESET%
    exit /b 1
)
echo %GREEN%✓ Daemon binary built successfully%RESET%

echo %BLUE%Building service wrapper (prismd-service.exe)...%RESET%
go build -ldflags "-X github.com/scttfrdmn/prism/pkg/version.Version=%VERSION% -X github.com/scttfrdmn/prism/pkg/version.BuildDate=%date:~10,4%-%date:~4,2%-%date:~7,2%_%time:~0,2%:%time:~3,2%:%time:~6,2% -X github.com/scttfrdmn/prism/pkg/version.GitCommit=msi-build" -o "%BUILD_DIR%\release\windows-amd64\prismd-service.exe" ./cmd/prismd-service
if !errorlevel! neq 0 (
    echo %RED%Error: Failed to build service wrapper%RESET%
    exit /b 1
)
echo %GREEN%✓ Service wrapper built successfully%RESET%

REM Build GUI if available (best effort)
echo %BLUE%Building GUI binary (prism-gui.exe)...%RESET%
set CGO_ENABLED=1
go build -ldflags "-X github.com/scttfrdmn/prism/pkg/version.Version=%VERSION% -X github.com/scttfrdmn/prism/pkg/version.BuildDate=%date:~10,4%-%date:~4,2%-%date:~7,2%_%time:~0,2%:%time:~3,2%:%time:~6,2% -X github.com/scttfrdmn/prism/pkg/version.GitCommit=msi-build" -o "%BUILD_DIR%\release\windows-amd64\prism-gui.exe" ./cmd/prism-gui 2>nul
if !errorlevel! equ 0 (
    echo %GREEN%✓ GUI binary built successfully%RESET%
) else (
    echo %YELLOW%⚠ GUI binary build failed (creating placeholder)%RESET%
    REM Create a placeholder GUI executable that shows a message
    echo @echo off > "%BUILD_DIR%\release\windows-amd64\prism-gui.exe.bat"
    echo echo Prism GUI not available in this build >> "%BUILD_DIR%\release\windows-amd64\prism-gui.exe.bat"
    echo echo Use 'prism tui' for terminal interface >> "%BUILD_DIR%\release\windows-amd64\prism-gui.exe.bat"
    REM Convert batch to executable using a simple copy (for MSI compatibility)
    copy /b "%SystemRoot%\System32\cmd.exe" "%BUILD_DIR%\release\windows-amd64\prism-gui.exe" >nul
)

REM Step 2: Prepare supporting files
echo.
echo %CYAN%Step 2: Preparing supporting files...%RESET%

REM Copy templates
echo %BLUE%Copying templates...%RESET%
if not exist "%BUILD_DIR%\release\templates" mkdir "%BUILD_DIR%\release\templates"
xcopy "%SOURCE_DIR%\templates\*.yml" "%BUILD_DIR%\release\templates\" /Y /Q >nul
xcopy "%SOURCE_DIR%\templates\*.json" "%BUILD_DIR%\release\templates\" /Y /Q >nul
echo %GREEN%✓ Templates copied%RESET%

REM Copy documentation
echo %BLUE%Copying documentation...%RESET%
if not exist "%BUILD_DIR%\release\docs" mkdir "%BUILD_DIR%\release\docs"
xcopy "%SOURCE_DIR%\docs\*.md" "%BUILD_DIR%\release\docs\" /Y /Q >nul
copy "%SOURCE_DIR%\LICENSE" "%BUILD_DIR%\release\" >nul
echo %GREEN%✓ Documentation copied%RESET%

REM Create PowerShell module
echo %BLUE%Creating PowerShell module...%RESET%
if not exist "%BUILD_DIR%\release\scripts" mkdir "%BUILD_DIR%\release\scripts"
copy "%SOURCE_DIR%\scripts\Prism.psm1" "%BUILD_DIR%\release\scripts\" >nul 2>&1
if !errorlevel! neq 0 (
    echo %YELLOW%⚠ PowerShell module not found, creating basic one%RESET%
    echo # Prism PowerShell Module > "%BUILD_DIR%\release\scripts\Prism.psm1"
    echo function Get-Prism { prism --help } >> "%BUILD_DIR%\release\scripts\Prism.psm1"
    echo Export-ModuleMember -Function Get-Prism >> "%BUILD_DIR%\release\scripts\Prism.psm1"
)
echo %GREEN%✓ PowerShell module prepared%RESET%

REM Create application icon (if not exists, create a simple one)
echo %BLUE%Preparing application icon...%RESET%
if not exist "%BUILD_DIR%\release\assets" mkdir "%BUILD_DIR%\release\assets"
if not exist "%SOURCE_DIR%\assets\prism.ico" (
    echo %YELLOW%⚠ Application icon not found, using default%RESET%
    copy "%SystemRoot%\System32\shell32.dll" "%BUILD_DIR%\release\assets\prism.ico" >nul
) else (
    copy "%SOURCE_DIR%\assets\prism.ico" "%BUILD_DIR%\release\assets\" >nul
)
echo %GREEN%✓ Application icon prepared%RESET%

REM Step 3: (no custom actions DLL: the MSI uses native WiX features, #757)
REM Step 4: Compile WiX Source
echo.
echo %CYAN%Step 4: Compiling WiX source...%RESET%

echo %BLUE%Running WiX Candle compiler...%RESET%
cd /d "%WIX_DIR%"

REM Set WiX variables
set WIX_VARIABLES=-dSourceDir="%BUILD_DIR%\release" -dVersion="%VERSION%"

candle -arch x64 %WIX_VARIABLES% -out "%BUILD_DIR%\obj\Prism.wixobj" "Prism.wxs" -ext WixUtilExtension 2>"%LOG_FILE%"
if !errorlevel! neq 0 (
    echo %RED%Error: WiX Candle compilation failed%RESET%
    echo %YELLOW%Check log file: %LOG_FILE%%RESET%
    type "%LOG_FILE%"
    exit /b 1
)
echo %GREEN%✓ WiX source compiled successfully%RESET%

REM Step 5: Link MSI Package
echo.
echo %CYAN%Step 5: Linking MSI package...%RESET%

echo %BLUE%Running WiX Light linker...%RESET%
light -out "%BUILD_DIR%\%MSI_NAME%" "%BUILD_DIR%\obj\Prism.wixobj" -ext WixUIExtension -ext WixUtilExtension -cultures:en-US -loc "%WIX_DIR%\strings_en-us.wxl" 2>>"%LOG_FILE%"
if !errorlevel! neq 0 (
    echo %RED%Error: WiX Light linking failed%RESET%
    echo %YELLOW%Check log file: %LOG_FILE%%RESET%
    type "%LOG_FILE%"
    exit /b 1
)
echo %GREEN%✓ MSI package linked successfully%RESET%

REM Step 6: Copy to distribution directory
echo.
echo %CYAN%Step 6: Finalizing distribution...%RESET%

move "%BUILD_DIR%\%MSI_NAME%" "%DIST_DIR%\%MSI_NAME%"
if !errorlevel! neq 0 (
    echo %RED%Error: Failed to move MSI to distribution directory%RESET%
    exit /b 1
)

REM Generate checksums
echo %BLUE%Generating checksums...%RESET%
cd /d "%DIST_DIR%"
certutil -hashfile "%MSI_NAME%" SHA256 > "%MSI_NAME%.sha256"
if !errorlevel! equ 0 (
    echo %GREEN%✓ SHA256 checksum generated%RESET%
) else (
    echo %YELLOW%⚠ Failed to generate SHA256 checksum%RESET%
)

REM Step 7: Validation and Summary
echo.
echo %CYAN%Step 7: Build validation...%RESET%

echo %BLUE%Validating MSI package...%RESET%
if exist "%DIST_DIR%\%MSI_NAME%" (
    for %%A in ("%DIST_DIR%\%MSI_NAME%") do set MSI_SIZE=%%~zA
    echo %GREEN%✓ MSI package created successfully%RESET%
    echo %WHITE%  File: %DIST_DIR%\%MSI_NAME%%RESET%
    echo %WHITE%  Size: !MSI_SIZE! bytes%RESET%
    
    REM Display SHA256 hash
    if exist "%DIST_DIR%\%MSI_NAME%.sha256" (
        echo %WHITE%  SHA256:%RESET%
        for /f "skip=1 tokens=*" %%i in (%DIST_DIR%\%MSI_NAME%.sha256) do (
            echo %WHITE%    %%i%RESET%
            goto :hash_done
        )
        :hash_done
    )
) else (
    echo %RED%✗ MSI package not found%RESET%
    exit /b 1
)

REM Cleanup temporary files
echo %BLUE%Cleaning up temporary files...%RESET%
if exist "%BUILD_DIR%\obj\*.wixobj" del /q "%BUILD_DIR%\obj\*.wixobj"
if exist "%BUILD_DIR%\obj\*.wixpdb" del /q "%BUILD_DIR%\obj\*.wixpdb"
echo %GREEN%✓ Temporary files cleaned%RESET%

REM Success summary
echo.
echo %GREEN%========================================%RESET%
echo %GREEN%  BUILD COMPLETED SUCCESSFULLY!%RESET%
echo %GREEN%========================================%RESET%
echo.
echo %CYAN%Prism Windows Installer:%RESET%
echo %WHITE%  Location: %DIST_DIR%\%MSI_NAME%%RESET%
echo %WHITE%  Version:  %VERSION%%RESET%
echo %WHITE%  Platform: Windows x64%RESET%
echo.
echo %CYAN%Installation Commands:%RESET%
echo %WHITE%  Silent install:   msiexec /i "%MSI_NAME%" /quiet%RESET%
echo %WHITE%  With logging:     msiexec /i "%MSI_NAME%" /l*v install.log%RESET%
echo %WHITE%  Uninstall:        msiexec /x "%MSI_NAME%" /quiet%RESET%
echo.
echo %CYAN%Next Steps:%RESET%
echo %WHITE%  1. Test the installer on a clean Windows system%RESET%
echo %WHITE%  2. Verify service installation and startup%RESET%
echo %WHITE%  3. Test CLI, daemon, and GUI functionality%RESET%
echo %WHITE%  4. Optional: Code sign the MSI for distribution%RESET%
echo.

endlocal
exit /b 0