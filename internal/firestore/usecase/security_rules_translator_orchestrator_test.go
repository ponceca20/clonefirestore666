package usecase_test

import (
	"context"
	"testing"

	"firestore-clone/internal/firestore/adapter/persistence/mongodb"
	"firestore-clone/internal/firestore/usecase"
	rtadapter "firestore-clone/internal/rules_translator/adapter"
	rtadapterparser "firestore-clone/internal/rules_translator/adapter/parser"
	rtusecase "firestore-clone/internal/rules_translator/usecase"
	"firestore-clone/internal/shared/logger"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestImportAndDeployFirestoreRules_Integration(t *testing.T) {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(ctx)
	testDB := client.Database("firestore_translator_test")
	defer testDB.Drop(ctx)

	log := logger.NewTestLogger()
	securityEngine := mongodb.NewSecurityRulesEngine(testDB, log)
	resourceAccessor := mongodb.NewResourceAccessor(testDB, log)
	securityEngine.SetResourceAccessor(resourceAccessor)

	parser := rtadapterparser.NewModernParserInstance()
	optimizer := rtadapter.NewRulesOptimizer(nil)
	cache := rtadapter.NewMemoryCache(nil)
	translator := rtusecase.NewFastTranslator(cache, optimizer, nil)

	orchestrator := usecase.NewSecurityRulesTranslatorOrchestrator(parser, translator, optimizer, securityEngine)

	projectID := "test-project"
	databaseID := "test-db"
	rulesContent := `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /users/{userId} {
      allow read, write: if request.auth != null && request.auth.uid == userId;
    }
    match /public/{docId} {
      allow read: if true;
    }
  }
}`

	err = orchestrator.ImportAndDeployFirestoreRules(ctx, rulesContent, projectID, databaseID)
	require.NoError(t, err)

	// Verifica que las reglas fueron desplegadas y evaluadas correctamente
	securityUseCase := usecase.NewSecurityRulesUseCase(securityEngine, log)
	loadedRules, err := securityUseCase.LoadRules(ctx, projectID, databaseID)
	require.NoError(t, err)
	require.NotEmpty(t, loadedRules)
}
