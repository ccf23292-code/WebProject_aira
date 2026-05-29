@echo off
REM AIRA one-click launcher - double-click this file to start.
REM It just calls start-dev.ps1 next to it, bypassing the
REM execution-policy block so you don't need to set it manually.
REM
REM   -NoProfile               skip user's PS profile, faster startup
REM   -ExecutionPolicy Bypass  bypass any local policy block
REM   %~dp0                    directory where this .bat lives
REM
REM After this .bat exits, the three PowerShell windows spawned
REM by the .ps1 keep running. Close them when done.

powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0start-dev.ps1"
