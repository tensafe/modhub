@chcp 65001 >nul
@echo off

REM 设置 Go 编译环境
SET CGO_ENABLED=0

REM 创建输出目录
IF NOT EXIST build (
    mkdir build
)

REM 编译 Windows 版本
echo 编译 Windows 版本...
go build -o build\modhub.exe main.go

REM 编译 Linux 版本
echo 编译 Linux 版本...
SET GOOS=linux
SET GOARCH=amd64
go build -o build\modhub-linux main.go

REM 编译 macOS (amd64) 版本
echo 编译 macOS (amd64) 版本...
SET GOOS=darwin
SET GOARCH=amd64
go build -o build\modhub-mac main.go

echo 编译完成！
echo %date% %time%
