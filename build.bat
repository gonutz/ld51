setlocal
pushd %~dp0

set GOOS=windows
set GOARCH=386
go build -ldflags="-H=windowsgui -s -w" -o "LD51 - Every 10 Seconds.exe"

popd
