package main

import (
	"fmt"
	"log"
	"net"
	"os"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"diaxel_zerde/database-service/models"
	"diaxel_zerde/database-service/proto"
	"diaxel_zerde/database-service/repository"
	"diaxel_zerde/database-service/server"
)

func main() {
	// Database connection
	dbHost := getEnv("POSTGRES_HOST", "database-postgres")
	dbPort := getEnv("POSTGRES_PORT", "5432")
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPassword := getEnv("POSTGRES_PASSWORD", "postgres")
	dbName := getEnv("POSTGRES_DB", "database_service")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Test database connection
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Database connected successfully")

	// Run migrations
	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	chatRepo := repository.NewChatRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	assistantRepo := repository.NewAssistantRepository(db)
	twilioRepo := repository.NewTwilioRepository(db)

	// Initialize gRPC server
	grpcServer := grpc.NewServer()
	databaseServer := server.NewDatabaseServer(userRepo, refreshTokenRepo, chatRepo, messageRepo, assistantRepo, twilioRepo)

	proto.RegisterDatabaseServiceServer(grpcServer, databaseServer)

	// Enable reflection for development
	reflection.Register(grpcServer)

	// Start gRPC server
	port := getEnv("GRPC_PORT", "50051")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Starting gRPC server on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func runMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// 1. Ensure schema_migrations table exists
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	_, err = sqlDB.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// 2. Read and run SQL migrations
	migrationFiles, err := os.ReadDir("./migrations")
	if err != nil {
		// Log error but continue (might be missing in development)
		log.Printf("Warning: failed to read migrations directory: %v", err)
	} else {
		for _, file := range migrationFiles {
			if file.IsDir() || len(file.Name()) < 4 || file.Name()[len(file.Name())-4:] != ".sql" {
				continue
			}

			version := file.Name()
			var count int64
			db.Table("schema_migrations").Where("version = ?", version).Count(&count)

			if count == 0 {
				log.Printf("Applying migration: %s", version)
				content, err := os.ReadFile("./migrations/" + version)
				if err != nil {
					return fmt.Errorf("failed to read migration file %s: %w", version, err)
				}

				if err := db.Exec(string(content)).Error; err != nil {
					return fmt.Errorf("failed to apply migration %s: %w", version, err)
				}

				if err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version).Error; err != nil {
					return fmt.Errorf("failed to record migration %s: %w", version, err)
				}
			}
		}
	}

	// 3. Run GORM AutoMigrate for model synchronization
	// Note: AutoMigrate will not drop columns!
	err = db.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
		&models.Analytics{},
		&models.Assistant{},
		&models.Chat{},
		&models.Message{},
		&models.TwilioConfig{},
	)

	if err != nil {
		return fmt.Errorf("failed to run AutoMigrate: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}
