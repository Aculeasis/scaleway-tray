@ECHO OFF
set INPUT="icon\icon.ico"
set OUTPUT="src\icon_win.go"

IF "%GOPATH%"=="" GOTO NOGO
IF NOT EXIST %GOPATH%\bin\2goarray.exe GOTO INSTALL
:POSTINSTALL
ECHO Creating %OUTPUT%
ECHO //+build windows > %OUTPUT%
ECHO. >> %OUTPUT%
TYPE %INPUT% | %GOPATH%\bin\2goarray iconData main >> %OUTPUT%
GOTO DONE

:CREATEFAIL
ECHO Unable to create output file
GOTO DONE

:INSTALL
ECHO Installing 2goarray...
go get github.com/cratonica/2goarray
IF ERRORLEVEL 1 GOTO GETFAIL
GOTO POSTINSTALL

:GETFAIL
ECHO Failure running go get github.com/cratonica/2goarray.  Ensure that go and git are in PATH
GOTO DONE

:NOGO
ECHO GOPATH environment variable not set
GOTO DONE

:DONE

