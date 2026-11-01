package cmd

import "minimart/db"

// RunMigrate applies all pending schema migrations, then exits. Deploys run this
// as a one-shot step before starting the server.
func RunMigrate() error {
	m, err := db.New()
	if err != nil {
		return err
	}
	return m.Migrate()
}
