package mongodb

import (
	"context"
	"fmt"
	"time"

	"firestore-clone/internal/firestore/domain/model"
)

// AtomicOperations maneja operaciones atómicas sobre documentos
// Ahora acepta un DatabaseProvider hexagonal para colecciones dinámicas
// DatabaseProvider debe retornar CollectionInterface

type AtomicOperations struct {
	db DatabaseProvider
}

// NewAtomicOperations crea una nueva instancia de AtomicOperations con DatabaseProvider
func NewAtomicOperations(db DatabaseProvider) *AtomicOperations {
	return &AtomicOperations{db: db}
}

// AtomicIncrement realiza un incremento atómico sobre un campo numérico
func (a *AtomicOperations) AtomicIncrement(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value int64) error {
	if projectID == "" {
		return fmt.Errorf("project ID cannot be empty")
	}
	if databaseID == "" {
		return fmt.Errorf("database ID cannot be empty")
	}
	if collectionID == "" {
		return fmt.Errorf("collection ID cannot be empty")
	}
	if documentID == "" {
		return fmt.Errorf("document ID cannot be empty")
	}
	if field == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	filter := map[string]interface{}{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	updateDoc := map[string]interface{}{
		"$inc": map[string]interface{}{
			fmt.Sprintf("fields.%s.value", field): value,
		},
		"$set": map[string]interface{}{
			"update_time": time.Now(),
		},
	}

	targetCollection := a.db.Collection(collectionID)
	result, err := targetCollection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic increment: %w", err)
	}
	if result.Matched() == 0 {
		return fmt.Errorf("document not found")
	}
	return nil
}

// AtomicArrayUnion realiza una operación atómica de unión de arreglos
func (a *AtomicOperations) AtomicArrayUnion(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	if elements == nil {
		return fmt.Errorf("elements array cannot be nil")
	}
	filter := map[string]interface{}{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	// Convertir elementos FieldValue a interface{} para MongoDB
	var values []interface{}
	for _, element := range elements {
		values = append(values, element.Value)
	}

	// Construir la operación de actualización
	updateDoc := map[string]interface{}{
		"$addToSet": map[string]interface{}{
			fmt.Sprintf("fields.%s.value", field): map[string]interface{}{
				"$each": values,
			},
		},
		"$set": map[string]interface{}{
			"update_time": time.Now(),
		},
	}
	// Ejecutar la unión atómica de arreglos
	targetCollection := a.db.Collection(collectionID)
	result, err := targetCollection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic array union: %w", err)
	}

	if result.Matched() == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// AtomicArrayRemove realiza una operación atómica de eliminación de arreglos
func (a *AtomicOperations) AtomicArrayRemove(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, elements []*model.FieldValue) error {
	filter := map[string]interface{}{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	// Convertir elementos FieldValue a interface{} para MongoDB
	var values []interface{}
	for _, element := range elements {
		values = append(values, element.Value)
	}

	// Construir la operación de actualización
	updateDoc := map[string]interface{}{
		"$pullAll": map[string]interface{}{
			fmt.Sprintf("fields.%s.value", field): values,
		},
		"$set": map[string]interface{}{
			"update_time": time.Now(),
		},
	}
	// Ejecutar la eliminación atómica de arreglos
	targetCollection := a.db.Collection(collectionID)
	result, err := targetCollection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic array remove: %w", err)
	}

	if result.Matched() == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// AtomicServerTimestamp establece un campo con la marca de tiempo actual del servidor
func (a *AtomicOperations) AtomicServerTimestamp(ctx context.Context, projectID, databaseID, collectionID, documentID, field string) error {
	filter := map[string]interface{}{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	// Construir la operación de actualización
	updateDoc := map[string]interface{}{
		"$set": map[string]interface{}{
			fmt.Sprintf("fields.%s.value", field):      time.Now(),
			fmt.Sprintf("fields.%s.value_type", field): model.FieldTypeTimestamp,
			"update_time": time.Now(),
		},
	}
	// Ejecutar la operación atómica de marca de tiempo del servidor
	targetCollection := a.db.Collection(collectionID)
	result, err := targetCollection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to set atomic server timestamp: %w", err)
	}

	if result.Matched() == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// AtomicDelete realiza una eliminación atómica de campos
func (a *AtomicOperations) AtomicDelete(ctx context.Context, projectID, databaseID, collectionID, documentID string, fields []string) error {
	if len(fields) == 0 {
		return fmt.Errorf("fields list cannot be empty")
	}
	filter := map[string]interface{}{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	// Construir la operación unset para cada campo
	unsetFields := map[string]interface{}{}
	for _, field := range fields {
		unsetFields[fmt.Sprintf("fields.%s", field)] = ""
	}

	updateDoc := map[string]interface{}{
		"$unset": unsetFields,
		"$set": map[string]interface{}{
			"update_time": time.Now(),
		},
	}
	// Ejecutar la eliminación atómica de campos
	targetCollection := a.db.Collection(collectionID)
	result, err := targetCollection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic field deletion: %w", err)
	}

	if result.Matched() == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// AtomicSetIfEmpty establece un campo solo si no existe o está vacío
func (a *AtomicOperations) AtomicSetIfEmpty(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value *model.FieldValue) error {
	if value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	filter := map[string]interface{}{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
		"$or": []map[string]interface{}{
			{fmt.Sprintf("fields.%s", field): map[string]interface{}{"$exists": false}},
			{fmt.Sprintf("fields.%s.value", field): nil},
			{fmt.Sprintf("fields.%s.value", field): ""},
		},
	}

	updateDoc := map[string]interface{}{
		"$set": map[string]interface{}{
			fmt.Sprintf("fields.%s.value", field):      value.Value,
			fmt.Sprintf("fields.%s.value_type", field): value.ValueType,
			"update_time": time.Now(),
		},
	}
	// Ejecutar la operación de establecimiento condicional
	targetCollection := a.db.Collection(collectionID)
	result, err := targetCollection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic set if empty: %w", err)
	}

	if result.Matched() == 0 {
		return fmt.Errorf("document not found or field already has value")
	}

	return nil
}

// AtomicMaximum establece un campo al máximo entre su valor actual y el valor proporcionado
func (a *AtomicOperations) AtomicMaximum(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value interface{}) error {
	filter := map[string]interface{}{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	updateDoc := map[string]interface{}{
		"$max": map[string]interface{}{
			fmt.Sprintf("fields.%s.value", field): value,
		},
		"$set": map[string]interface{}{
			"update_time": time.Now(),
		},
	}
	// Ejecutar la operación atómica de máximo
	targetCollection := a.db.Collection(collectionID)
	result, err := targetCollection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic maximum: %w", err)
	}

	if result.Matched() == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// AtomicMinimum establece un campo al mínimo entre su valor actual y el valor proporcionado
func (a *AtomicOperations) AtomicMinimum(ctx context.Context, projectID, databaseID, collectionID, documentID, field string, value interface{}) error {
	filter := map[string]interface{}{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	updateDoc := map[string]interface{}{
		"$min": map[string]interface{}{
			fmt.Sprintf("fields.%s.value", field): value,
		},
		"$set": map[string]interface{}{
			"update_time": time.Now(),
		},
	}
	// Ejecutar la operación atómica de mínimo
	targetCollection := a.db.Collection(collectionID)
	result, err := targetCollection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to perform atomic minimum: %w", err)
	}

	if result.Matched() == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}
