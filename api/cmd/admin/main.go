// deinscomplete-admin provides narrowly scoped, server-side account operations.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"deinscomplete/api/internal/account"
	"deinscomplete/api/internal/config"
	"deinscomplete/api/internal/database"
)

func main() {
	if len(os.Args) != 4 || os.Args[1] != "set-plan" {
		fmt.Fprintln(os.Stderr, "usage: deinscomplete-admin set-plan <email> <free|pro>")
		os.Exit(2)
	}
	if os.Args[3] != "free" && os.Args[3] != "pro" {
		fmt.Fprintln(os.Stderr, "plan must be free or pro")
		os.Exit(2)
	}
	c, err := config.Load()
	if err != nil || !c.Database.Enabled {
		fmt.Fprintln(os.Stderr, "DATABASE_ENABLED=true is required")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, database.Config{URL: c.Database.URL, MaxOpenConns: c.Database.MaxOpenConns, MaxIdleConns: c.Database.MaxIdleConns, ConnMaxLifetime: c.Database.ConnMaxLifetime})
	if err != nil {
		fmt.Fprintln(os.Stderr, "database unavailable")
		os.Exit(1)
	}
	defer pool.Close()
	repo := account.NewRepository(pool)
	u, err := repo.FindUserByEmail(ctx, os.Args[2])
	if err == nil {
		err = repo.SetUserPlan(ctx, u.ID, os.Args[3])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to set plan")
		os.Exit(1)
	}
	fmt.Println("plan updated")
}
