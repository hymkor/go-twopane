@setlocal
@set "PROMPT=%0> "
@call :"%1"
@endlocal
@exit /b

:""
:"package"
    for %%I in (386 amd64) do call :package1 windows %%I .exe
    for %%I in (386 amd64) do call :package1 linux   %%I
    exit /b

:package1
    setlocal
    set "GOOS=%1"
    set "GOARCH=%2"
    set "SUFFIX=%3"
    for /F %%I in ('cd') do set "NAME=%%~nI"
    go build
    zip -9m "%NAME%-%GOOS%-%GOARCH%-%DATE:/=%.zip" %NAME%%SUFFIX%
    endlocal
    exit /b
