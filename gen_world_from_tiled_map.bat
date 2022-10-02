setlocal
pushd %~dp0

echo package main> world_generated.go
echo var worldTiles = []int {>> world_generated.go
type assets\world.tmx | first 105 lines | last 100 lines >> world_generated.go
echo ,>> world_generated.go
echo }>> world_generated.go

go fmt world_generated.go

popd
