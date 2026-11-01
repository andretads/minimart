package main

import "minimart/cmd"

//	@title			MiniMart API
//	@version		1.0
//	@description	Product catalog and order placement for the MiniMart modular monolith.
//	@BasePath		/
//
// The whole program is a set of subcommands (server, worker, migrate, gateway,
// dlq-replay) behind one binary. main just hands off to cmd.Execute, which
// dispatches on os.Args.
func main() {
	cmd.Execute()
}
