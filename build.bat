@ECHO OFF
for /f %%i in ('git rev-list -1 HEAD') do set GIT_COMMIT=%%i
for /f %%i in ('git describe --always --abbrev^=0 --tags') do set VERSION=%%i
for /f %%i in ('date /T') do set BUILD_DATE=%%i
for /f %%i in ('time /T') do set BUILD_TIME=%%i
@ECHO ON
go generate .\src\
go build -ldflags="-X main.GitCommit=%GIT_COMMIT% -X 'main.BuildDate=%BUILD_DATE% %BUILD_TIME%' -X main.Version=%VERSION% -H=windowsgui -s -w" -o .\bin\scaleway-tray.exe .\src\
