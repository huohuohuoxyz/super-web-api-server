@chcp 65001 > nul
@echo off
echo 正在编译 Go 程序为 Linux 可执行文件...

REM 编译为 Linux amd64
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -o dist/super-web-api-server-linux-amd64 .
if %ERRORLEVEL% EQU 0 (
    echo Linux amd64 版本编译成功！
)

@REM REM 编译为 Linux arm64
@REM set GOOS=linux
@REM set GOARCH=arm64
@REM set CGO_ENABLED=0
@REM go build -o super-web-api-server-linux-arm64 .
@REM if %ERRORLEVEL% EQU 0 (
@REM     echo Linux arm64 版本编译成功！
@REM )


REM 编译为 Window amd64
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -o dist/super-web-api-server-windows-amd64.exe .
if %ERRORLEVEL% EQU 0 (
    echo windows amd64 版本编译成功！
)

REM 重置环境变量
set GOOS=
set GOARCH=
set CGO_ENABLED=

echo 所有版本编译完成！