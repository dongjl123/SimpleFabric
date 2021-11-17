module github.com/SimpleFabric/main

go 1.16

require (
	github.com/SimpleFabric/fabric v0.0.0
	github.com/SimpleFabric/pool v0.0.0 // indirect
	golang.org/x/sys v0.0.0-20211116061358-0a5406a5449c // indirect
	golang.org/x/text v0.3.7 // indirect
)

replace github.com/SimpleFabric/fabric v0.0.0 => ../fabric

replace github.com/SimpleFabric/pool v0.0.0 => ../pool
