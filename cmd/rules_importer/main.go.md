package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"firestore-clone/internal/firestore/adapter/persistence/mongodb"
	"firestore-clone/internal/shared/logger"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"firestore-clone/internal/firestore/domain/repository"
	"firestore-clone/internal/rules_translator/adapter"
	"firestore-clone/internal/rules_translator/adapter/parser"
	"firestore-clone/internal/rules_translator/domain"
	"firestore-clone/internal/rules_translator/usecase"
)

const (
	version = "1.0.0"
	appName = "firestore-rules-importer"
)

// Config estructura de configuración de la aplicación
type Config struct {
	RulesFile    string
	ProjectID    string
	DatabaseID   string
	DryRun       bool
	Verbose      bool
	NoCache      bool
	Optimize     bool
	ValidateOnly bool
	OutputFormat string
	ConfigFile   string
}

// App estructura principal de la aplicación
type App struct {
	config     *Config
	parser     domain.RulesParser
	translator domain.RulesTranslator
	deployer   domain.RulesDeployer
	cache      domain.RulesCache
	optimizer  domain.RulesOptimizer
}

func main() {
	config := parseFlags()

	if config == nil {
		os.Exit(1)
	}

	app, err := NewApp(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing application: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := app.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// parseFlags parsea argumentos de línea de comandos
func parseFlags() *Config {
	config := &Config{}

	var useMock bool
	flag.BoolVar(&useMock, "mock", false, "Use mock security engine (for testing)")
	flag.StringVar(&config.RulesFile, "rules", "", "Path to firestore.rules file (required)")
	flag.StringVar(&config.ProjectID, "project", "default-project", "Project ID")
	flag.StringVar(&config.DatabaseID, "database", "default", "Database ID")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Validate and translate rules without deploying")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&config.NoCache, "no-cache", false, "Disable caching")
	flag.BoolVar(&config.Optimize, "optimize", true, "Enable rule optimization")
	flag.BoolVar(&config.ValidateOnly, "validate-only", false, "Only validate syntax without translation")
	flag.StringVar(&config.OutputFormat, "output", "text", "Output format: text, json")
	flag.StringVar(&config.ConfigFile, "config", "", "Configuration file path")

	// Custom usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", appName)
		fmt.Fprintf(os.Stderr, "%s v%s - Import Firestore security rules\n\n", appName, version)
		fmt.Fprintf(os.Stderr, "This tool translates Firestore security rules (.rules files) into\n")
		fmt.Fprintf(os.Stderr, "the internal format used by your Firestore clone system.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -rules=firestore.rules -project=myapp\n", appName)
		fmt.Fprintf(os.Stderr, "  %s -rules=firestore.rules -dry-run -verbose\n", appName)
		fmt.Fprintf(os.Stderr, "  %s -rules=firestore.rules -validate-only\n", appName)
		fmt.Fprintf(os.Stderr, "  %s -rules=firestore.rules -output=json\n", appName)
	}

	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version information")

	flag.Parse()

	if showVersion {
		fmt.Printf("%s v%s\n", appName, version)
		return nil
	}

	if config.RulesFile == "" {
		fmt.Fprintf(os.Stderr, "Error: -rules flag is required\n\n")
		flag.Usage()
		return nil
	}

	// Verificar que el archivo existe
	if _, err := os.Stat(config.RulesFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: rules file '%s' does not exist\n", config.RulesFile)
		return nil
	}

	return config
}

// NewApp crea una nueva instancia de la aplicación
func NewApp(config *Config) (*App, error) {
	// Cargar variables de entorno desde .env si existe
	_ = godotenv.Load()

	// Configurar componentes optimizados
	translatorConfig := usecase.DefaultTranslatorConfig()
	translatorConfig.EnableCache = !config.NoCache
	translatorConfig.EnableOptimization = config.Optimize
	translatorConfig.EnableMetrics = config.Verbose

	cacheConfig := adapter.DefaultCacheConfig()
	if config.NoCache {
		cacheConfig.MaxSize = 0
	}

	optimizerConfig := adapter.DefaultOptimizerConfig()
	optimizerConfig.EnableAggressiveOptim = config.Optimize
	// Inicializar componentes
	rParser := parser.NewModernParserInstance()
	cache := adapter.NewMemoryCache(cacheConfig)
	optimizer := adapter.NewRulesOptimizer(optimizerConfig)
	translator := usecase.NewFastTranslator(cache, optimizer, translatorConfig)

	// Determinar si usar mock o motor real
	useMock := false
	for _, arg := range os.Args {
		if arg == "--mock" {
			useMock = true
			break
		}
	}

	var securityEngine repository.SecurityRulesEngine
	if useMock {
		securityEngine = &MockSecurityRulesEngine{}
	} else {
		// Leer configuración de MongoDB desde variables de entorno
		mongoURI := os.Getenv("MONGODB_URI")
		dbName := os.Getenv("DATABASE_NAME")
		if mongoURI == "" || dbName == "" {
			return nil, fmt.Errorf("MONGODB_URI y DATABASE_NAME deben estar definidos en el entorno o .env")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
		if err != nil {
			return nil, fmt.Errorf("error conectando a MongoDB: %w", err)
		}
		db := client.Database(dbName)
		log := logger.NewLogger()
		securityEngine = mongodb.NewSecurityRulesEngine(db, log)
	}

	// Configurar deployer
	validator := adapter.NewSimpleValidator()
	historyStore := adapter.NewMemoryHistoryStore()
	deployerConfig := adapter.DefaultDeployerConfig()
	deployerConfig.EnableValidation = !config.DryRun

	deployer := adapter.NewRulesDeployer(securityEngine, validator, historyStore, deployerConfig)

	return &App{
		config:     config,
		parser:     rParser,
		translator: translator,
		deployer:   deployer,
		cache:      cache,
		optimizer:  optimizer,
	}, nil
}

// Run ejecuta la aplicación principal
func (a *App) Run(ctx context.Context) error {
	startTime := time.Now()

	if a.config.Verbose {
		fmt.Printf("🚀 Starting %s v%s\n", appName, version)
		fmt.Printf("📁 Rules file: %s\n", a.config.RulesFile)
		fmt.Printf("🎯 Project: %s, Database: %s\n", a.config.ProjectID, a.config.DatabaseID)
		if a.config.DryRun {
			fmt.Printf("🧪 Dry run mode enabled\n")
		}
		fmt.Println()
	}

	// 1. Leer archivo de reglas
	rulesContent, err := a.readRulesFile()
	if err != nil {
		return fmt.Errorf("failed to read rules file: %w", err)
	}

	if a.config.Verbose {
		fmt.Printf("📖 Read %d bytes from rules file\n", len(rulesContent))
	}

	// 2. Parsear reglas
	parseResult, err := a.parseRules(ctx, rulesContent)
	if err != nil {
		return fmt.Errorf("failed to parse rules: %w", err)
	}

	if a.config.Verbose {
		fmt.Printf("✅ Parsed %d rules in %v\n", parseResult.RuleCount, parseResult.ParseTime)
		if len(parseResult.Errors) > 0 {
			fmt.Printf("⚠️  %d parse errors found\n", len(parseResult.Errors))
		}
		if len(parseResult.Warnings) > 0 {
			fmt.Printf("⚠️  %d parse warnings found\n", len(parseResult.Warnings))
		}
	}

	// Mostrar errores si los hay
	if len(parseResult.Errors) > 0 {
		return a.displayParseErrors(parseResult.Errors)
	}

	// Si solo validación, terminar aquí
	if a.config.ValidateOnly {
		if a.config.Verbose {
			fmt.Println("✅ Validation completed successfully")
		}
		return a.outputValidationResult(parseResult)
	}

	// 3. Traducir reglas
	translationResult, err := a.translateRules(ctx, parseResult.Ruleset)
	if err != nil {
		return fmt.Errorf("failed to translate rules: %w", err)
	}

	if a.config.Verbose {
		fmt.Printf("🔄 Translated to %d security rules in %v\n",
			translationResult.RulesGenerated, translationResult.TranslationTime)
		if translationResult.OptimizedRules > 0 {
			fmt.Printf("⚡ Optimized %d rules\n", translationResult.OptimizedRules)
		}
	}

	// Si dry run, mostrar resultado sin desplegar
	if a.config.DryRun {
		return a.outputDryRunResult(parseResult, translationResult)
	}

	// 4. Desplegar reglas
	deployResult, err := a.deployRules(ctx, translationResult.Rules)
	if err != nil {
		return fmt.Errorf("failed to deploy rules: %w", err)
	}

	if a.config.Verbose {
		fmt.Printf("🚀 Deployed %d rules in %v\n", deployResult.RulesDeployed, deployResult.DeployTime)
		fmt.Printf("📋 Version: %s\n", deployResult.Version)
	}

	// Mostrar resultado final
	totalTime := time.Since(startTime)
	return a.outputFinalResult(parseResult, translationResult, deployResult, totalTime)
}

// readRulesFile lee el archivo de reglas
func (a *App) readRulesFile() (string, error) {
	file, err := os.Open(a.config.RulesFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// parseRules parsea el contenido de las reglas
func (a *App) parseRules(ctx context.Context, content string) (*domain.ParseResult, error) {
	return a.parser.ParseString(ctx, content)
}

// translateRules traduce las reglas parseadas
func (a *App) translateRules(ctx context.Context, ruleset *domain.FirestoreRuleset) (*domain.TranslationResult, error) {
	if a.config.NoCache {
		return a.translator.Translate(ctx, ruleset)
	}

	// Usar caché si está habilitado
	cacheKey := &domain.CacheKey{
		ProjectID:  a.config.ProjectID,
		DatabaseID: a.config.DatabaseID,
		Version:    ruleset.Version,
		Hash:       a.calculateRulesetHash(ruleset),
	}

	return a.translator.TranslateWithCache(ctx, ruleset, cacheKey)
}

// deployRules despliega las reglas traducidas
func (a *App) deployRules(ctx context.Context, rules interface{}) (*domain.DeployResult, error) {
	return a.deployer.DeployWithValidation(ctx, a.config.ProjectID, a.config.DatabaseID, rules)
}

// calculateRulesetHash calcula hash del ruleset para caché
func (a *App) calculateRulesetHash(ruleset *domain.FirestoreRuleset) string {
	// Hash simple basado en el service y número de matches
	content := fmt.Sprintf("%s:%d", ruleset.Service, len(ruleset.Matches))
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// Output methods

func (a *App) displayParseErrors(errors []domain.ParseError) error {
	fmt.Fprintf(os.Stderr, "❌ Parse errors found:\n\n")
	for _, err := range errors {
		fmt.Fprintf(os.Stderr, "  Line %d: %s\n", err.Line, err.Message)
	}
	return fmt.Errorf("parse errors prevent further processing")
}

func (a *App) outputValidationResult(result *domain.ParseResult) error {
	if a.config.OutputFormat == "json" {
		return a.outputJSON(map[string]interface{}{
			"valid":      len(result.Errors) == 0,
			"rules":      result.RuleCount,
			"parse_time": result.ParseTime.String(),
			"errors":     result.Errors,
			"warnings":   result.Warnings,
		})
	}

	fmt.Printf("✅ Rules file is valid\n")
	fmt.Printf("📊 Statistics:\n")
	fmt.Printf("   Rules found: %d\n", result.RuleCount)
	fmt.Printf("   Parse time: %v\n", result.ParseTime)
	if len(result.Warnings) > 0 {
		fmt.Printf("   Warnings: %d\n", len(result.Warnings))
	}

	return nil
}

func (a *App) outputDryRunResult(parseResult *domain.ParseResult, translationResult *domain.TranslationResult) error {
	if a.config.OutputFormat == "json" {
		return a.outputJSON(map[string]interface{}{
			"dry_run":            true,
			"parse_result":       parseResult,
			"translation_result": translationResult,
		})
	}

	fmt.Printf("🧪 Dry run completed successfully\n\n")
	fmt.Printf("📊 Summary:\n")
	fmt.Printf("   Original rules: %d\n", parseResult.RuleCount)
	fmt.Printf("   Translated rules: %d\n", translationResult.RulesGenerated)
	fmt.Printf("   Optimized rules: %d\n", translationResult.OptimizedRules)
	fmt.Printf("   Parse time: %v\n", parseResult.ParseTime)
	fmt.Printf("   Translation time: %v\n", translationResult.TranslationTime)

	if len(translationResult.Errors) > 0 {
		fmt.Printf("\n❌ Translation errors:\n")
		for _, err := range translationResult.Errors {
			fmt.Printf("   - %s\n", err)
		}
	}

	return nil
}

func (a *App) outputFinalResult(parseResult *domain.ParseResult, translationResult *domain.TranslationResult, deployResult *domain.DeployResult, totalTime time.Duration) error {
	if a.config.OutputFormat == "json" {
		return a.outputJSON(map[string]interface{}{
			"success":            deployResult.Success,
			"version":            deployResult.Version,
			"rules_deployed":     deployResult.RulesDeployed,
			"total_time":         totalTime.String(),
			"parse_result":       parseResult,
			"translation_result": translationResult,
			"deploy_result":      deployResult,
		})
	}

	if deployResult.Success {
		fmt.Printf("🎉 Rules deployed successfully!\n\n")
	} else {
		fmt.Printf("❌ Deployment failed!\n\n")
	}

	fmt.Printf("📊 Final Summary:\n")
	fmt.Printf("   Version: %s\n", deployResult.Version)
	fmt.Printf("   Rules deployed: %d\n", deployResult.RulesDeployed)
	fmt.Printf("   Total time: %v\n", totalTime)
	fmt.Printf("   Project: %s\n", a.config.ProjectID)
	fmt.Printf("   Database: %s\n", a.config.DatabaseID)

	if len(deployResult.Errors) > 0 {
		fmt.Printf("\n❌ Errors:\n")
		for _, err := range deployResult.Errors {
			fmt.Printf("   - %s\n", err)
		}
	}

	if len(deployResult.Warnings) > 0 {
		fmt.Printf("\n⚠️  Warnings:\n")
		for _, warning := range deployResult.Warnings {
			fmt.Printf("   - %s\n", warning)
		}
	}

	// Mostrar métricas detalladas en modo verbose
	if a.config.Verbose {
		fmt.Printf("\n📈 Performance Metrics:\n")
		if parserMetrics := a.parser.GetMetrics(); parserMetrics != nil {
			fmt.Printf("   Parser cache hit rate: %.1f%%\n", parserMetrics.CacheHitRate*100)
			fmt.Printf("   Parser average time: %v\n", parserMetrics.AverageParseTime)
		}

		if cacheStats := a.cache.GetStats(); cacheStats != nil {
			fmt.Printf("   Cache hit rate: %.1f%%\n", cacheStats.HitRate*100)
			fmt.Printf("   Cache size: %d entries\n", cacheStats.CacheSize)
		}
	}

	if !deployResult.Success {
		return fmt.Errorf("deployment failed")
	}

	return nil
}

func (a *App) outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// MockSecurityRulesEngine implementación mock para testing/demo
type MockSecurityRulesEngine struct{}

func (m *MockSecurityRulesEngine) EvaluateAccess(ctx context.Context, operation repository.OperationType, securityContext *repository.SecurityContext) (*repository.RuleEvaluationResult, error) {
	return &repository.RuleEvaluationResult{Allowed: true}, nil
}

func (m *MockSecurityRulesEngine) LoadRules(ctx context.Context, projectID, databaseID string) ([]*repository.SecurityRule, error) {
	return []*repository.SecurityRule{}, nil
}

func (m *MockSecurityRulesEngine) SaveRules(ctx context.Context, projectID, databaseID string, rules []*repository.SecurityRule) error {
	fmt.Printf("📝 Mock: Would save %d rules for project %s, database %s\n", len(rules), projectID, databaseID)
	return nil
}

func (m *MockSecurityRulesEngine) ValidateRules(rules []*repository.SecurityRule) error {
	return nil
}

func (m *MockSecurityRulesEngine) ClearCache(projectID, databaseID string) {
	// No-op for mock
}

func (m *MockSecurityRulesEngine) SetResourceAccessor(accessor repository.ResourceAccessor) {
	// No-op for mock
}

func (m *MockSecurityRulesEngine) DeleteRules(ctx context.Context, projectID, databaseID string) error {
	fmt.Printf("🗑️ Mock: Would delete rules for project %s, database %s\n", projectID, databaseID)
	return nil
}

func (m *MockSecurityRulesEngine) GetRawRules(ctx context.Context, projectID, databaseID string) (string, error) {
	fmt.Printf("📦 Mock: Would return raw rules for project %s, database %s\n", projectID, databaseID)
	return "{}", nil
}
