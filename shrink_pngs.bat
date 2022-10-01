setlocal
pushd %~dp0

for %%x in (*.png) do zopflipng %%x %%x -y

popd
