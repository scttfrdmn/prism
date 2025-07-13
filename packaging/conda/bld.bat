@echo off

:: Copy binaries to the conda environment Scripts directory
copy cws.exe %PREFIX%\Scripts\
copy cwsd.exe %PREFIX%\Scripts\

:: Copy GUI if it exists
if exist cws-gui.exe (
    copy cws-gui.exe %PREFIX%\Scripts\
)

:: Add a message for users
echo CloudWorkstation v0.4.1 has been installed.
echo To get started, run: cws test
echo.