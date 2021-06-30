setlocal
for /F "skip=1" %%I in ('where svnview.exe') do set "TARGETDIR=%%~dpI"
if not "%TARGETDIR%" == "" copy svnview.exe "%TARGETDIR%"
endlocal
