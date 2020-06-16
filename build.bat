gox -osarch="windows/amd64 darwin/amd64 linux/amd64" -output="_builds/{{.Dir}}_{{.OS}}_{{.Arch}}"
copy mamd.css _builds
copy template.html _builds