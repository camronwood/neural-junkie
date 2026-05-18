package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	_ "github.com/lib/pq" // PostgreSQL driver
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DatabaseMCP provides MCP tools for database operations
type DatabaseMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
	db         *sql.DB
}

// NewDatabaseMCP creates a new Database MCP server
func NewDatabaseMCP() (*DatabaseMCP, error) {
	config := mcp.GetMCPServerConfig("DATABASE")

	mcpServer, httpServer, err := mcp.NewMCPServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	d := &DatabaseMCP{
		mcpServer:  mcpServer,
		httpServer: httpServer,
		config:     config,
	}

	// Initialize database connection
	if err := d.initDatabase(); err != nil {
		log.Printf("Warning: Failed to initialize database connection: %v", err)
		log.Printf("Database MCP tools will have limited functionality")
	}

	d.registerTools()

	return d, nil
}

// Start starts the Database MCP server
func (d *DatabaseMCP) Start() error {
	if d.httpServer == nil {
		return fmt.Errorf("MCP server not configured")
	}

	return mcp.StartMCPServer(d.httpServer, d.config.Port)
}

// GetMCPServer returns the underlying MCP server
func (d *DatabaseMCP) GetMCPServer() *server.MCPServer {
	return d.mcpServer
}

// initDatabase initializes database connection
func (d *DatabaseMCP) initDatabase() error {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Try to construct from individual components
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")

		if host == "" {
			host = "localhost"
		}
		if port == "" {
			port = "5432"
		}
		if user == "" {
			user = "postgres"
		}
		if dbname == "" {
			dbname = "postgres"
		}

		if password != "" {
			dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
		} else {
			dbURL = fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=disable", user, host, port, dbname)
		}
	}

	var err error
	d.db, err = sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := d.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Database connection established")
	return nil
}

// registerTools registers all Database MCP tools
func (d *DatabaseMCP) registerTools() {
	// Tool 1: explain_query
	d.mcpServer.AddTool(mcp.CreateTool(
		"explain_query",
		"Run EXPLAIN ANALYZE on SQL queries to analyze performance",
		mcp.CreateStringInputSchema("sql_query", "SQL query to analyze"),
		nil,
	), d.handleExplainQuery)

	// Tool 2: check_indexes
	d.mcpServer.AddTool(mcp.CreateTool(
		"check_indexes",
		"Analyze table indexes for optimization opportunities",
		mcp.CreateStringInputSchema("table_name", "Table name to analyze indexes for"),
		nil,
	), d.handleCheckIndexes)

	// Tool 3: validate_schema
	d.mcpServer.AddTool(mcp.CreateTool(
		"validate_schema",
		"Check database schema for consistency and best practices",
		mcp.CreateStringInputSchema("schema_name", "Schema name to validate (optional, defaults to public)"),
		nil,
	), d.handleValidateSchema)

	// Tool 4: suggest_optimizations
	d.mcpServer.AddTool(mcp.CreateTool(
		"suggest_optimizations",
		"Analyze query patterns and suggest database optimizations",
		mcp.CreateStringInputSchema("table_name", "Table name to analyze for optimization suggestions"),
		nil,
	), d.handleSuggestOptimizations)

	// Tool 5: check_table_stats
	d.mcpServer.AddTool(mcp.CreateTool(
		"check_table_stats",
		"Get table statistics including size, row count, and storage info",
		mcp.CreateStringInputSchema("table_name", "Table name to get statistics for"),
		nil,
	), d.handleCheckTableStats)

	// Tool 6: generate_migration
	d.mcpServer.AddTool(mcp.CreateTool(
		"generate_migration",
		"Generate database migration scripts based on schema changes",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"description": "Description of the migration",
			"changes":     "Description of schema changes to implement",
		}),
		nil,
	), d.handleGenerateMigration)

	log.Printf("Registered %d Database MCP tools", len(d.mcpServer.ListTools()))
}

// handleExplainQuery runs EXPLAIN ANALYZE on SQL queries
func (d *DatabaseMCP) handleExplainQuery(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"sql_query"}); err != nil {
		return mcp.HandleToolError(err, "explain_query"), nil
	}

	if d.db == nil {
		return mcp.HandleToolError(fmt.Errorf("database connection not available"), "explain_query"), nil
	}

	sqlQuery := request.GetString("sql_query", "")
	if sqlQuery == "" {
		return mcp.HandleToolError(fmt.Errorf("empty SQL query"), "explain_query"), nil
	}

	// Validate that it's a SELECT query (safety check)
	queryUpper := strings.ToUpper(strings.TrimSpace(sqlQuery))
	if !strings.HasPrefix(queryUpper, "SELECT") && !strings.HasPrefix(queryUpper, "WITH") {
		return mcp.HandleToolError(fmt.Errorf("only SELECT and WITH queries are allowed for EXPLAIN"), "explain_query"), nil
	}

	// Run EXPLAIN ANALYZE
	explainQuery := fmt.Sprintf("EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) %s", sqlQuery)

	rows, err := d.db.QueryContext(ctx, explainQuery)
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("EXPLAIN query failed: %w", err), "explain_query"), nil
	}
	defer rows.Close()

	var result strings.Builder
	result.WriteString("=== Query Execution Plan ===\n")
	result.WriteString(fmt.Sprintf("Query: %s\n\n", sqlQuery))

	for rows.Next() {
		var plan string
		if err := rows.Scan(&plan); err != nil {
			result.WriteString(fmt.Sprintf("Error reading plan: %v\n", err))
			continue
		}
		result.WriteString(plan)
	}

	if err := rows.Err(); err != nil {
		result.WriteString(fmt.Sprintf("\nError during execution: %v", err))
	}

	return mcp.HandleToolSuccess(result.String()), nil
}

// handleCheckIndexes analyzes table indexes
func (d *DatabaseMCP) handleCheckIndexes(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"table_name"}); err != nil {
		return mcp.HandleToolError(err, "check_indexes"), nil
	}

	if d.db == nil {
		return mcp.HandleToolError(fmt.Errorf("database connection not available"), "check_indexes"), nil
	}

	tableName := request.GetString("table_name", "")
	if tableName == "" {
		return mcp.HandleToolError(fmt.Errorf("empty table name"), "check_indexes"), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Index Analysis for Table: %s ===\n\n", tableName))

	// Get all indexes for the table
	indexQuery := `
		SELECT 
			indexname,
			indexdef,
			pg_size_pretty(pg_relation_size(indexname::regclass)) as size
		FROM pg_indexes 
		WHERE tablename = $1
		ORDER BY indexname
	`

	rows, err := d.db.QueryContext(ctx, indexQuery, tableName)
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("failed to query indexes: %w", err), "check_indexes"), nil
	}
	defer rows.Close()

	result.WriteString("Existing Indexes:\n")
	indexCount := 0
	for rows.Next() {
		var indexName, indexDef, size string
		if err := rows.Scan(&indexName, &indexDef, &size); err != nil {
			continue
		}
		result.WriteString(fmt.Sprintf("- %s (%s): %s\n", indexName, size, indexDef))
		indexCount++
	}

	if indexCount == 0 {
		result.WriteString("No indexes found for this table.\n")
	}

	// Get table statistics
	statsQuery := `
		SELECT 
			n_tup_ins as inserts,
			n_tup_upd as updates,
			n_tup_del as deletes,
			n_live_tup as live_tuples,
			n_dead_tup as dead_tuples
		FROM pg_stat_user_tables 
		WHERE relname = $1
	`

	rows, err = d.db.QueryContext(ctx, statsQuery, tableName)
	if err == nil {
		result.WriteString("\nTable Statistics:\n")
		for rows.Next() {
			var inserts, updates, deletes, liveTuples, deadTuples int64
			if err := rows.Scan(&inserts, &updates, &deletes, &liveTuples, &deadTuples); err != nil {
				continue
			}
			result.WriteString(fmt.Sprintf("- Live tuples: %d\n", liveTuples))
			result.WriteString(fmt.Sprintf("- Dead tuples: %d\n", deadTuples))
			result.WriteString(fmt.Sprintf("- Total inserts: %d\n", inserts))
			result.WriteString(fmt.Sprintf("- Total updates: %d\n", updates))
			result.WriteString(fmt.Sprintf("- Total deletes: %d\n", deletes))
		}
		rows.Close()
	}

	return mcp.HandleToolSuccess(result.String()), nil
}

// handleValidateSchema checks schema consistency
func (d *DatabaseMCP) handleValidateSchema(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"schema_name"}); err != nil {
		return mcp.HandleToolError(err, "validate_schema"), nil
	}

	if d.db == nil {
		return mcp.HandleToolError(fmt.Errorf("database connection not available"), "validate_schema"), nil
	}

	schemaName := request.GetString("schema_name", "")
	if schemaName == "" {
		schemaName = "public" // Default schema
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Schema Validation for: %s ===\n\n", schemaName))

	// Check if schema exists
	schemaQuery := "SELECT schema_name FROM information_schema.schemata WHERE schema_name = $1"
	var exists string
	err := d.db.QueryRowContext(ctx, schemaQuery, schemaName).Scan(&exists)
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("schema %s does not exist", schemaName), "validate_schema"), nil
	}

	// Get all tables in schema
	tablesQuery := `
		SELECT table_name, table_type
		FROM information_schema.tables 
		WHERE table_schema = $1
		ORDER BY table_name
	`

	rows, err := d.db.QueryContext(ctx, tablesQuery, schemaName)
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("failed to query tables: %w", err), "validate_schema"), nil
	}
	defer rows.Close()

	result.WriteString("Tables in schema:\n")
	tableCount := 0
	for rows.Next() {
		var tableName, tableType string
		if err := rows.Scan(&tableName, &tableType); err != nil {
			continue
		}
		result.WriteString(fmt.Sprintf("- %s (%s)\n", tableName, tableType))
		tableCount++
	}

	if tableCount == 0 {
		result.WriteString("No tables found in this schema.\n")
	}

	// Check for foreign key constraints
	fkQuery := `
		SELECT 
			tc.table_name,
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY' 
			AND tc.table_schema = $1
	`

	rows, err = d.db.QueryContext(ctx, fkQuery, schemaName)
	if err == nil {
		result.WriteString("\nForeign Key Constraints:\n")
		fkCount := 0
		for rows.Next() {
			var table, column, foreignTable, foreignColumn string
			if err := rows.Scan(&table, &column, &foreignTable, &foreignColumn); err != nil {
				continue
			}
			result.WriteString(fmt.Sprintf("- %s.%s -> %s.%s\n", table, column, foreignTable, foreignColumn))
			fkCount++
		}
		if fkCount == 0 {
			result.WriteString("No foreign key constraints found.\n")
		}
		rows.Close()
	}

	return mcp.HandleToolSuccess(result.String()), nil
}

// handleSuggestOptimizations analyzes query patterns and suggests optimizations
func (d *DatabaseMCP) handleSuggestOptimizations(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"table_name"}); err != nil {
		return mcp.HandleToolError(err, "suggest_optimizations"), nil
	}

	if d.db == nil {
		return mcp.HandleToolError(fmt.Errorf("database connection not available"), "suggest_optimizations"), nil
	}

	tableName := request.GetString("table_name", "")
	if tableName == "" {
		return mcp.HandleToolError(fmt.Errorf("empty table name"), "suggest_optimizations"), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Optimization Suggestions for Table: %s ===\n\n", tableName))

	// Analyze table size and row count
	sizeQuery := `
		SELECT 
			pg_size_pretty(pg_total_relation_size($1)) as total_size,
			pg_size_pretty(pg_relation_size($1)) as table_size,
			pg_size_pretty(pg_total_relation_size($1) - pg_relation_size($1)) as index_size,
			(SELECT COUNT(*) FROM $1) as row_count
	`

	rows, err := d.db.QueryContext(ctx, sizeQuery, tableName)
	if err != nil {
		result.WriteString(fmt.Sprintf("Could not analyze table size: %v\n", err))
	} else {
		for rows.Next() {
			var totalSize, tableSize, indexSize, rowCount string
			if err := rows.Scan(&totalSize, &tableSize, &indexSize, &rowCount); err != nil {
				continue
			}
			result.WriteString("Table Statistics:\n")
			result.WriteString(fmt.Sprintf("- Total size: %s\n", totalSize))
			result.WriteString(fmt.Sprintf("- Table size: %s\n", tableSize))
			result.WriteString(fmt.Sprintf("- Index size: %s\n", indexSize))
			result.WriteString(fmt.Sprintf("- Row count: %s\n", rowCount))
		}
		rows.Close()
	}

	// Check for missing indexes on foreign keys
	fkIndexQuery := `
		SELECT 
			kcu.column_name,
			'Missing index on foreign key column' as suggestion
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.table_constraints tc ON kcu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY' 
			AND tc.table_name = $1
			AND NOT EXISTS (
				SELECT 1 FROM pg_indexes 
				WHERE tablename = $1 
				AND indexdef LIKE '%' || kcu.column_name || '%'
			)
	`

	rows, err = d.db.QueryContext(ctx, fkIndexQuery, tableName)
	if err == nil {
		result.WriteString("\nOptimization Suggestions:\n")
		suggestionCount := 0
		for rows.Next() {
			var column, suggestion string
			if err := rows.Scan(&column, &suggestion); err != nil {
				continue
			}
			result.WriteString(fmt.Sprintf("- %s: %s\n", column, suggestion))
			suggestionCount++
		}
		if suggestionCount == 0 {
			result.WriteString("No obvious optimization opportunities found.\n")
		}
		rows.Close()
	}

	return mcp.HandleToolSuccess(result.String()), nil
}

// handleCheckTableStats gets table statistics
func (d *DatabaseMCP) handleCheckTableStats(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"table_name"}); err != nil {
		return mcp.HandleToolError(err, "check_table_stats"), nil
	}

	if d.db == nil {
		return mcp.HandleToolError(fmt.Errorf("database connection not available"), "check_table_stats"), nil
	}

	tableName := request.GetString("table_name", "")
	if tableName == "" {
		return mcp.HandleToolError(fmt.Errorf("empty table name"), "check_table_stats"), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Table Statistics for: %s ===\n\n", tableName))

	// Get comprehensive table statistics
	statsQuery := `
		SELECT 
			pg_size_pretty(pg_total_relation_size($1)) as total_size,
			pg_size_pretty(pg_relation_size($1)) as table_size,
			pg_size_pretty(pg_total_relation_size($1) - pg_relation_size($1)) as index_size,
			(SELECT COUNT(*) FROM $1) as row_count,
			(SELECT COUNT(*) FROM information_schema.columns WHERE table_name = $1) as column_count
	`

	rows, err := d.db.QueryContext(ctx, statsQuery, tableName)
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("failed to get table statistics: %w", err), "check_table_stats"), nil
	}
	defer rows.Close()

	for rows.Next() {
		var totalSize, tableSize, indexSize, rowCount, columnCount string
		if err := rows.Scan(&totalSize, &tableSize, &indexSize, &rowCount, &columnCount); err != nil {
			continue
		}
		result.WriteString("Storage Information:\n")
		result.WriteString(fmt.Sprintf("- Total size: %s\n", totalSize))
		result.WriteString(fmt.Sprintf("- Table size: %s\n", tableSize))
		result.WriteString(fmt.Sprintf("- Index size: %s\n", indexSize))
		result.WriteString(fmt.Sprintf("- Row count: %s\n", rowCount))
		result.WriteString(fmt.Sprintf("- Column count: %s\n", columnCount))
	}

	return mcp.HandleToolSuccess(result.String()), nil
}

// handleGenerateMigration generates database migration scripts
func (d *DatabaseMCP) handleGenerateMigration(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"description", "changes"}); err != nil {
		return mcp.HandleToolError(err, "generate_migration"), nil
	}

	description := request.GetString("description", "")
	changes := request.GetString("changes", "")

	var result strings.Builder
	result.WriteString("=== Generated Migration Script ===\n\n")
	result.WriteString(fmt.Sprintf("-- Migration: %s\n", description))
	result.WriteString(fmt.Sprintf("-- Changes: %s\n\n", changes))
	result.WriteString("-- This is a template migration script.\n")
	result.WriteString("-- Please review and modify as needed before applying.\n\n")

	result.WriteString("BEGIN;\n\n")
	result.WriteString("-- Add your migration SQL here\n")
	result.WriteString("-- Example:\n")
	result.WriteString("-- ALTER TABLE users ADD COLUMN email_verified BOOLEAN DEFAULT FALSE;\n")
	result.WriteString("-- CREATE INDEX idx_users_email ON users(email);\n\n")
	result.WriteString("COMMIT;\n")

	return mcp.HandleToolSuccess(result.String()), nil
}
