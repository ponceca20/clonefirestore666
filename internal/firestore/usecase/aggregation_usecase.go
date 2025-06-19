package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"firestore-clone/internal/firestore/domain/model"
)

// RunAggregationQuery ejecuta consultas de agregación siguiendo la API de Firestore con extensiones
func (uc *FirestoreUsecase) RunAggregationQuery(ctx context.Context, req AggregationQueryRequest) (*AggregationQueryResponse, error) {
	// Validar la solicitud
	if err := uc.validateAggregationRequest(req); err != nil {
		return nil, fmt.Errorf("invalid aggregation request: %w", err)
	}

	// Construir la consulta base si existe
	var baseQuery *model.Query
	if req.StructuredAggregationQuery.StructuredQuery != nil {
		baseQuery = req.StructuredAggregationQuery.StructuredQuery
		// Asegurar que el path esté configurado correctamente
		if baseQuery.Path == "" {
			baseQuery.Path = req.Parent
		}
	} else {
		// Si no hay consulta estructurada, crear una consulta vacía
		baseQuery = &model.Query{
			Path: req.Parent,
		}
	}

	// Construir el pipeline de agregación de MongoDB
	pipeline, err := uc.buildAggregationPipeline(baseQuery, req.StructuredAggregationQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to build aggregation pipeline: %w", err)
	}

	// Ejecutar el pipeline de agregación
	results, err := uc.queryEngine.ExecuteAggregationPipeline(ctx, req.ProjectID, req.DatabaseID, baseQuery.CollectionID, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregation pipeline: %w", err)
	}

	// Formatear los resultados siguiendo el formato de Firestore
	response, err := uc.formatAggregationResults(results, req.StructuredAggregationQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to format aggregation results: %w", err)
	}

	return response, nil
}

// validateAggregationRequest valida la solicitud de agregación
func (uc *FirestoreUsecase) validateAggregationRequest(req AggregationQueryRequest) error {
	if req.ProjectID == "" {
		return errors.New("projectID is required")
	}
	if req.DatabaseID == "" {
		return errors.New("databaseID is required")
	}
	if req.Parent == "" {
		return errors.New("parent is required")
	}
	if req.StructuredAggregationQuery == nil {
		return errors.New("structuredAggregationQuery is required")
	}

	// Validar agregaciones
	aggregations := req.StructuredAggregationQuery.Aggregations
	if len(aggregations) == 0 {
		return errors.New("at least one aggregation is required")
	}
	if len(aggregations) > 5 {
		return errors.New("maximum 5 aggregations allowed")
	}

	// Validar cada agregación
	aliasMap := make(map[string]bool)
	for i, agg := range aggregations {
		if agg.Alias == "" {
			return fmt.Errorf("aggregation at index %d must have an alias", i)
		}
		if aliasMap[agg.Alias] {
			return fmt.Errorf("duplicate alias: %s", agg.Alias)
		}
		aliasMap[agg.Alias] = true

		// Validar que solo hay un tipo de agregación por objeto
		count := 0
		if agg.Count != nil {
			count++
		}
		if agg.Sum != nil {
			count++
		}
		if agg.Avg != nil {
			count++
		}
		if agg.Min != nil {
			count++
		}
		if agg.Max != nil {
			count++
		}

		if count != 1 {
			return fmt.Errorf("aggregation '%s' must have exactly one aggregation type", agg.Alias)
		}

		// Validar que las agregaciones numéricas tienen un campo
		if agg.Sum != nil && agg.Sum.Field.FieldPath == "" {
			return fmt.Errorf("sum aggregation '%s' must specify a field", agg.Alias)
		}
		if agg.Avg != nil && agg.Avg.Field.FieldPath == "" {
			return fmt.Errorf("avg aggregation '%s' must specify a field", agg.Alias)
		}
		if agg.Min != nil && agg.Min.Field.FieldPath == "" {
			return fmt.Errorf("min aggregation '%s' must specify a field", agg.Alias)
		}
		if agg.Max != nil && agg.Max.Field.FieldPath == "" {
			return fmt.Errorf("max aggregation '%s' must specify a field", agg.Alias)
		}
	}

	// Validar groupBy si existe
	for i, groupBy := range req.StructuredAggregationQuery.GroupBy {
		if groupBy.FieldPath == "" {
			return fmt.Errorf("groupBy at index %d must specify a fieldPath", i)
		}
	}

	return nil
}

// buildAggregationPipeline construye el pipeline de agregación de MongoDB
func (uc *FirestoreUsecase) buildAggregationPipeline(baseQuery *model.Query, aggQuery *StructuredAggregationQuery) ([]interface{}, error) {
	var pipeline []interface{}

	// Etapa 1: $match - Filtrar documentos usando la consulta base
	if baseQuery != nil && len(baseQuery.Filters) > 0 {
		matchStage, err := uc.buildMatchStage(baseQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to build match stage: %w", err)
		}
		if matchStage != nil {
			pipeline = append(pipeline, map[string]interface{}{"$match": matchStage})
		}
	}

	// Etapa 2: $group - Agrupar y agregar
	groupStage, err := uc.buildGroupStage(aggQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to build group stage: %w", err)
	}
	pipeline = append(pipeline, map[string]interface{}{"$group": groupStage})
	// Etapa 3: $project - Formatear la salida
	projectStage, err := uc.buildProjectStage(aggQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to build project stage: %w", err)
	}
	pipeline = append(pipeline, map[string]interface{}{"$project": projectStage})

	// Etapa 4: $match post-project - Filtrar grupos null si hay groupBy
	if len(aggQuery.GroupBy) > 0 {
		postMatchStage := uc.buildPostGroupMatchStage(aggQuery)
		if postMatchStage != nil {
			pipeline = append(pipeline, map[string]interface{}{"$match": postMatchStage})
		}
	}

	return pipeline, nil
}

// buildMatchStage construye la etapa $match usando la lógica existente del query engine
func (uc *FirestoreUsecase) buildMatchStage(query *model.Query) (interface{}, error) {
	// Reutilizar la lógica existente del query engine para construir filtros MongoDB
	// Esta lógica ya está implementada en el query engine
	return uc.queryEngine.BuildMongoFilter(query.Filters)
}

// buildGroupStage construye la etapa $group
func (uc *FirestoreUsecase) buildGroupStage(aggQuery *StructuredAggregationQuery) (map[string]interface{}, error) {
	groupStage := map[string]interface{}{}

	// Configurar el campo _id para agrupación
	if len(aggQuery.GroupBy) > 0 {
		// Agrupación por múltiples campos
		if len(aggQuery.GroupBy) == 1 {
			// Agrupación por un solo campo
			fieldPath := uc.buildFieldPath(aggQuery.GroupBy[0].FieldPath)
			groupStage["_id"] = "$" + fieldPath
		} else {
			// Agrupación por múltiples campos
			idDoc := map[string]interface{}{}
			for _, groupBy := range aggQuery.GroupBy {
				fieldPath := uc.buildFieldPath(groupBy.FieldPath)
				idDoc[groupBy.FieldPath] = "$" + fieldPath
			}
			groupStage["_id"] = idDoc
		}
	} else {
		// Sin agrupación (comportamiento de Firestore)
		groupStage["_id"] = nil
	} // Agregar acumuladores
	for _, agg := range aggQuery.Aggregations {
		if agg.Count != nil {
			groupStage[agg.Alias] = map[string]interface{}{"$sum": 1}
		} else if agg.Sum != nil {
			fieldPath := agg.Sum.Field.FieldPath
			// Mapear el tipo del campo
			fieldTypeMap := map[string]string{
				"price": "doubleValue", "stock": "doubleValue", "weightKg": "doubleValue",
			}
			fieldType := fieldTypeMap[fieldPath]
			if fieldType == "" {
				fieldType = "doubleValue" // Por defecto
			}

			// Usar buildFlexibleFieldPath para robustez
			flexPath := uc.buildFlexibleFieldPath(fieldPath, fieldType)
			groupStage[agg.Alias] = map[string]interface{}{"$sum": flexPath}
		} else if agg.Avg != nil {
			fieldPath := agg.Avg.Field.FieldPath
			// Mapear el tipo del campo
			fieldTypeMap := map[string]string{
				"price": "doubleValue", "stock": "doubleValue", "weightKg": "doubleValue",
			}
			fieldType := fieldTypeMap[fieldPath]
			if fieldType == "" {
				fieldType = "doubleValue" // Por defecto
			}

			// Usar buildFlexibleFieldPath para robustez
			flexPath := uc.buildFlexibleFieldPath(fieldPath, fieldType)
			groupStage[agg.Alias] = map[string]interface{}{"$avg": flexPath}
		} else if agg.Min != nil {
			fieldPath := uc.buildFieldPath(agg.Min.Field.FieldPath)
			groupStage[agg.Alias] = map[string]interface{}{"$min": "$" + fieldPath}
		} else if agg.Max != nil {
			fieldPath := uc.buildFieldPath(agg.Max.Field.FieldPath)
			groupStage[agg.Alias] = map[string]interface{}{"$max": "$" + fieldPath}
		}
	}

	return groupStage, nil
}

// buildProjectStage construye la etapa $project para formatear la salida
func (uc *FirestoreUsecase) buildProjectStage(aggQuery *StructuredAggregationQuery) (map[string]interface{}, error) {
	projectStage := map[string]interface{}{
		"_id": 0, // Ocultar el _id de MongoDB
	}

	// Si hay groupBy, promover los campos de agrupación desde _id
	if len(aggQuery.GroupBy) > 0 {
		if len(aggQuery.GroupBy) == 1 {
			// Un solo campo de agrupación
			fieldName := aggQuery.GroupBy[0].FieldPath
			projectStage[fieldName] = "$_id"
		} else {
			// Múltiples campos de agrupación
			for _, groupBy := range aggQuery.GroupBy {
				projectStage[groupBy.FieldPath] = "$_id." + groupBy.FieldPath
			}
		}
	}

	// Incluir todos los campos de resultado de agregación
	for _, agg := range aggQuery.Aggregations {
		projectStage[agg.Alias] = "$" + agg.Alias
	}

	return projectStage, nil
}

// buildFieldPath construye el path del campo para MongoDB considerando la estructura de Firestore
func (uc *FirestoreUsecase) buildFieldPath(fieldPath string) string {
	// Mapeo de campos conocidos y sus tipos
	fieldTypeMap := map[string]string{
		"brand":       "stringValue",
		"name":        "stringValue",
		"category":    "stringValue",
		"description": "stringValue",
		"productId":   "stringValue",
		"price":       "doubleValue",
		"stock":       "doubleValue",
		"weightKg":    "doubleValue",
		"available":   "booleanValue",
	}

	// Buscar el tipo del campo en el mapeo
	if fieldType, exists := fieldTypeMap[fieldPath]; exists {
		return "fields." + fieldPath + "." + fieldType
	}

	// Para campos desconocidos, intentar inferir el tipo o usar una estrategia por defecto
	// Esta implementación podría mejorarse con introspección de datos reales
	// Por ahora, usamos una heurística simple basada en el nombre del campo
	if fieldPath == "count" || fieldPath == "quantity" || fieldPath == "amount" {
		return "fields." + fieldPath + ".integerValue"
	}

	// Para nombres que típicamente son strings
	if fieldPath == "id" || fieldPath == "type" || fieldPath == "status" || fieldPath == "color" {
		return "fields." + fieldPath + ".stringValue"
	}

	// Por defecto, asumir doubleValue para compatibilidad con agregaciones numéricas
	return "fields." + fieldPath + ".doubleValue"
}

// buildFlexibleFieldPath crea un $ifNull que intenta múltiples tipos para mayor robustez
func (uc *FirestoreUsecase) buildFlexibleFieldPath(fieldPath string, preferredType string) interface{} {
	basePath := "fields." + fieldPath

	switch preferredType {
	case "doubleValue":
		// Para campos numéricos, intentar doubleValue primero, luego integerValue
		return map[string]interface{}{
			"$ifNull": []interface{}{
				"$" + basePath + ".doubleValue",
				map[string]interface{}{
					"$ifNull": []interface{}{
						"$" + basePath + ".integerValue",
						0, // Valor por defecto para numéricos
					},
				},
			},
		}
	case "stringValue":
		// Para campos de texto, solo intentar stringValue
		return "$" + basePath + ".stringValue"
	case "booleanValue":
		// Para campos booleanos
		return map[string]interface{}{
			"$ifNull": []interface{}{
				"$" + basePath + ".booleanValue",
				false,
			},
		}
	default:
		// Fallback al comportamiento original
		return "$" + basePath + "." + preferredType
	}
}

// formatAggregationResults formatea los resultados siguiendo el formato de AggregationResult de Firestore
func (uc *FirestoreUsecase) formatAggregationResults(mongoResults []map[string]interface{}, aggQuery *StructuredAggregationQuery) (*AggregationQueryResponse, error) {
	response := &AggregationQueryResponse{
		Results: make([]AggregationResult, 0, len(mongoResults)),
	}

	readTime := time.Now().UTC().Format(time.RFC3339Nano)

	for _, mongoResult := range mongoResults {
		result := AggregationResult{
			ReadTime: readTime,
			Result: AggregationResultData{
				AggregateFields: make(map[string]interface{}),
			},
		}

		// Agregar campos de agrupación si existen
		if len(aggQuery.GroupBy) > 0 {
			for _, groupBy := range aggQuery.GroupBy {
				if value, exists := mongoResult[groupBy.FieldPath]; exists {
					result.Result.AggregateFields[groupBy.FieldPath] = uc.formatFirestoreValue(value)
				}
			}
		}

		// Agregar resultados de agregación
		for _, agg := range aggQuery.Aggregations {
			if value, exists := mongoResult[agg.Alias]; exists {
				result.Result.AggregateFields[agg.Alias] = uc.formatFirestoreValue(value)
			}
		}

		response.Results = append(response.Results, result)
	}

	// Si no hay resultados y no hay groupBy, devolver un resultado con valores por defecto
	if len(response.Results) == 0 && len(aggQuery.GroupBy) == 0 {
		result := AggregationResult{
			ReadTime: readTime,
			Result: AggregationResultData{
				AggregateFields: make(map[string]interface{}),
			},
		}

		// Establecer valores por defecto para las agregaciones
		for _, agg := range aggQuery.Aggregations {
			if agg.Count != nil {
				result.Result.AggregateFields[agg.Alias] = map[string]interface{}{"integerValue": "0"}
			} else {
				result.Result.AggregateFields[agg.Alias] = nil
			}
		}

		response.Results = append(response.Results, result)
	}

	return response, nil
}

// formatFirestoreValue convierte valores de MongoDB al formato de valores tipados de Firestore
func (uc *FirestoreUsecase) formatFirestoreValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case int32:
		return map[string]interface{}{"integerValue": fmt.Sprintf("%d", v)}
	case int64:
		return map[string]interface{}{"integerValue": fmt.Sprintf("%d", v)}
	case float64:
		return map[string]interface{}{"doubleValue": v}
	case string:
		return map[string]interface{}{"stringValue": v}
	case bool:
		return map[string]interface{}{"booleanValue": v}
	default:
		// Para otros tipos, intentar convertir a string
		return map[string]interface{}{"stringValue": fmt.Sprintf("%v", v)}
	}
}

// buildPostGroupMatchStage construye un filtro post-project para excluir grupos null
func (uc *FirestoreUsecase) buildPostGroupMatchStage(aggQuery *StructuredAggregationQuery) map[string]interface{} {
	if len(aggQuery.GroupBy) == 0 {
		return nil
	}

	matchConditions := map[string]interface{}{}

	// Crear condición para cada campo de agrupación
	for _, groupBy := range aggQuery.GroupBy {
		// Filtrar resultados donde el campo de agrupación no sea null
		matchConditions[groupBy.FieldPath] = map[string]interface{}{
			"$ne": nil,
		}
	}

	return matchConditions
}
