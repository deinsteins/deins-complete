// deinscomplete-admin provides narrowly scoped, server-side account operations.
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"deinscomplete/api/internal/account"
	"deinscomplete/api/internal/accountauth"
	"deinscomplete/api/internal/config"
	"deinscomplete/api/internal/database"
)

func main() {
	if len(os.Args) < 3 {
		usage()
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
	switch os.Args[1] {
	case "set-plan":
		if len(os.Args) != 4 || (os.Args[3] != "free" && os.Args[3] != "pro") {
			usage()
		}
		u, err := repo.FindUserByEmail(ctx, os.Args[2])
		if err == nil {
			err = repo.SetUserPlan(ctx, u.ID, os.Args[3])
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to set plan")
			os.Exit(1)
		}
		fmt.Println("plan updated")
	case "create-invite":
		if len(os.Args) != 3 && len(os.Args) != 4 {
			usage()
		}
		days := 7
		if len(os.Args) == 4 {
			value, parseErr := strconv.Atoi(os.Args[3])
			if parseErr != nil || value < 1 || value > 30 {
				fmt.Fprintln(os.Stderr, "expiry days must be between 1 and 30")
				os.Exit(2)
			}
			days = value
		}
		code, err := accountauth.NewOpaqueToken()
		if err == nil {
			_, err = repo.CreateInvite(ctx, accountauth.HashToken(code), os.Args[2], time.Now().Add(time.Duration(days)*24*time.Hour))
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to create invite")
			os.Exit(1)
		}
		fmt.Printf("Invite for %s (expires in %d days):\n%s\n", account.NormalizeEmail(os.Args[2]), days, code)
	default:
		usage()
	}
}
func usage() {
	fmt.Fprintln(os.Stderr, "usage: deinscomplete-admin set-plan <email> <free|pro>\n       deinscomplete-admin create-invite <email> [expiry-days]")
	os.Exit(2)
}
