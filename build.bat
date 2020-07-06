@echo off

echo building all versions ...
gox -osarch="windows/amd64 darwin/amd64 linux/amd64" -output="_builds/{{.Dir}}_{{.OS}}_{{.Arch}}"

echo cleaning examples ...
rmdir /s /q _builds\example_in
mkdir _builds\example_in
xcopy .\example_in _builds\example_in

rmdir /s /q _builds\example_out

echo copying template and CSS ...
copy mamd.css _builds
copy template.html _builds

echo rebuilding example output ...
cd _builds
mkdir example_out
MaMD_windows_amd64.exe -i .\example_in -o .\example_out

pause