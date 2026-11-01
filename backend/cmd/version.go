package cmd

// version is the build's version string. It defaults to "dev" and is overridden
// at build time with:
//
//	go build -ldflags "-X minimart/cmd.version=$(VERSION)" -o minimart .
var version = "dev"
