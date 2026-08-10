package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/bootstrap"
	"erp-system/backend/internal/permissions"
	"erp-system/backend/internal/roles"
	"erp-system/backend/internal/users"
	"erp-system/backend/pkg/database"

	"golang.org/x/term"
)

func main() {
	ctx := context.Background()

	// Prefer explicit DATABASE_URL. If not provided, require all POSTGRES_* vars.
	connStr := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if connStr == "" {
		host := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
		port := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
		user := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
		pass := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD"))
		dbname := strings.TrimSpace(os.Getenv("POSTGRES_DB"))

		var missing []string
		if host == "" {
			missing = append(missing, "POSTGRES_HOST")
		}
		if port == "" {
			missing = append(missing, "POSTGRES_PORT")
		}
		if user == "" {
			missing = append(missing, "POSTGRES_USER")
		}
		if pass == "" {
			missing = append(missing, "POSTGRES_PASSWORD")
		}
		if dbname == "" {
			missing = append(missing, "POSTGRES_DB")
		}

		if len(missing) > 0 {
			log.Fatalf("missing required database environment variables: %s", strings.Join(missing, ", "))
		}

		connStr = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host,
			port,
			user,
			pass,
			dbname,
		)
	}

	db, err := database.Connect(connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	userRepo := users.NewRepository(db)
	roleRepo := roles.NewRepository(db)
	permRepo := permissions.NewRepository(db)
	auditRepo := audit.NewRepository(db)
	auditService := audit.NewService(auditRepo)
	refreshRepo := auth.NewRefreshTokenRepository(db)
	authService := auth.NewService(userRepo, roleRepo, permRepo, refreshRepo, auditService)

	email := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL"))
	name := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_NAME"))
	password := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"))

	reader := bufio.NewReader(os.Stdin)
	if email == "" {
		email = prompt(reader, "SUPER_ADMIN email")
	}
	if name == "" {
		name = prompt(reader, "SUPER_ADMIN name")
	}
	if password == "" {
		password, err = promptPassword(reader, "SUPER_ADMIN password")
		if err != nil {
			log.Fatal(err)
		}
	}

	id, err := bootstrap.BootstrapAdmin(ctx, userRepo, authService, email, name, password)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("SUPER_ADMIN created with user ID %d\n", id)
}

func prompt(reader *bufio.Reader, label string) string {
	fmt.Printf("%s: ", label)
	value, _ := reader.ReadString('\n')
	return strings.TrimSpace(value)
}

func promptPassword(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(passwordBytes)), nil
}
