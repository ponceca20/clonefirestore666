package mongodb

import (
	"context"
	"errors"
	"firestore-clone/internal/firestore/domain/model"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DocumentOperations handles basic CRUD operations for documents in Firestore clone.
type DocumentOperations struct {
	repo *DocumentRepository
}

// NewDocumentOperations creates a new DocumentOperations instance.
func NewDocumentOperations(repo *DocumentRepository) *DocumentOperations {
	return &DocumentOperations{repo: repo}
}

// NewDocumentOperationsWithStore creates a new DocumentOperations instance with shared store.
// This method is kept for compatibility but now uses MongoDB instead of in-memory store.
func NewDocumentOperationsWithStore(repo *DocumentRepository, mem map[string]*model.Document) *DocumentOperations {
	return &DocumentOperations{repo: repo}
}

// MongoDocument represents the MongoDB document structure
type MongoDocument struct {
	ID                primitive.ObjectID           `bson:"_id,omitempty"`
	ProjectID         string                       `bson:"projectID"`
	DatabaseID        string                       `bson:"databaseID"`
	CollectionID      string                       `bson:"collectionID"`
	DocumentID        string                       `bson:"documentID"`
	Path              string                       `bson:"path"`
	ParentPath        string                       `bson:"parentPath"`
	Fields            map[string]*model.FieldValue `bson:"fields"`
	CreateTime        time.Time                    `bson:"createTime"`
	UpdateTime        time.Time                    `bson:"updateTime"`
	ReadTime          time.Time                    `bson:"readTime"`
	Version           int64                        `bson:"version"`
	Exists            bool                         `bson:"exists"`
	HasSubcollections bool                         `bson:"hasSubcollections"`
}

// Helper: parse Firestore path
func parseFirestorePath(path string) (projectID, databaseID, collectionID, documentID string, err error) {
	// projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/{COLLECTION_ID}/{DOCUMENT_ID}
	parts := make([]string, 0)
	for _, p := range splitAndTrim(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) < 6 {
		return "", "", "", "", errors.New("invalid path")
	}
	return parts[1], parts[3], parts[5], parts[6], nil
}

func splitAndTrim(s, sep string) []string {
	var out []string
	for _, p := range strings.Split(s, sep) {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Helper: convert MongoDocument to model.Document
// mongoToModelDocument convierte un MongoDocument a model.Document (para compatibilidad)
func mongoToModelDocument(mongoDoc *MongoDocument) *model.Document {
	return &model.Document{
		ID:                mongoDoc.ID,
		ProjectID:         mongoDoc.ProjectID,
		DatabaseID:        mongoDoc.DatabaseID,
		CollectionID:      mongoDoc.CollectionID,
		DocumentID:        mongoDoc.DocumentID,
		Path:              mongoDoc.Path,
		ParentPath:        mongoDoc.ParentPath,
		Fields:            mongoDoc.Fields,
		CreateTime:        mongoDoc.CreateTime,
		UpdateTime:        mongoDoc.UpdateTime,
		ReadTime:          mongoDoc.ReadTime,
		Version:           mongoDoc.Version,
		Exists:            mongoDoc.Exists,
		HasSubcollections: mongoDoc.HasSubcollections,
	}
}

// flattenFieldsForMongoDB convierte los FieldValue a una estructura plana para MongoDB
func flattenFieldsForMongoDB(fields map[string]*model.FieldValue) map[string]interface{} {
	result := make(map[string]interface{})

	for key, fieldValue := range fields {
		if fieldValue == nil {
			continue
		}

		switch fieldValue.ValueType {
		case model.FieldTypeBool:
			if boolVal, ok := fieldValue.Value.(bool); ok {
				result[key] = map[string]interface{}{
					"booleanValue": boolVal,
				}
			}
		case model.FieldTypeString:
			if strVal, ok := fieldValue.Value.(string); ok {
				result[key] = map[string]interface{}{
					"stringValue": strVal,
				}
			}
		case model.FieldTypeInt:
			// Los enteros en Firestore se guardan como string
			if intVal, ok := fieldValue.Value.(string); ok {
				result[key] = map[string]interface{}{
					"integerValue": intVal,
				}
			} else if intVal, ok := fieldValue.Value.(int64); ok {
				result[key] = map[string]interface{}{
					"integerValue": fmt.Sprintf("%d", intVal),
				}
			}
		case model.FieldTypeDouble:
			if doubleVal, ok := fieldValue.Value.(float64); ok {
				result[key] = map[string]interface{}{
					"doubleValue": doubleVal,
				}
			}
		case model.FieldTypeTimestamp:
			// CRITICAL: Store timestamp as MongoDB Date for efficient queries
			if timeVal, ok := fieldValue.Value.(time.Time); ok {
				result[key] = map[string]interface{}{
					"timestampValue": timeVal, // Store as MongoDB Date, not string
				}
			} else if strVal, ok := fieldValue.Value.(string); ok {
				// Parse string and store as MongoDB Date
				if t, err := time.Parse(time.RFC3339, strVal); err == nil {
					result[key] = map[string]interface{}{
						"timestampValue": t, // Store as MongoDB Date, not string
					}
				} else {
					// If can't parse, store as string but log warning
					result[key] = map[string]interface{}{
						"stringValue": strVal,
					}
				}
			}
		case model.FieldTypeArray:
			if arrayVal, ok := fieldValue.Value.(*model.ArrayValue); ok {
				flattenedValues := make([]map[string]interface{}, 0, len(arrayVal.Values))
				for _, val := range arrayVal.Values {
					if val.ValueType == model.FieldTypeString {
						if strVal, ok := val.Value.(string); ok {
							flattenedValues = append(flattenedValues, map[string]interface{}{
								"stringValue": strVal,
							})
						}
					}
					// Add more types as needed including timestamps
					if val.ValueType == model.FieldTypeTimestamp {
						if timeVal, ok := val.Value.(time.Time); ok {
							flattenedValues = append(flattenedValues, map[string]interface{}{
								"timestampValue": timeVal.Format(time.RFC3339Nano),
							})
						}
					}
				}
				result[key] = map[string]interface{}{
					"arrayValue": map[string]interface{}{
						"values": flattenedValues,
					},
				}
			}
		case model.FieldTypeMap:
			if mapVal, ok := fieldValue.Value.(*model.MapValue); ok {
				result[key] = map[string]interface{}{
					"mapValue": map[string]interface{}{
						"fields": flattenFieldsForMongoDB(mapVal.Fields),
					},
				}
			}
		default:
			// Para tipos no manejados, guardar tal como está
			result[key] = fieldValue.Value
		}
	}

	return result
}

// expandFieldsFromMongoDB convierte la estructura plana de MongoDB de vuelta a FieldValue
func expandFieldsFromMongoDB(flatFields map[string]interface{}) map[string]*model.FieldValue {
	result := make(map[string]*model.FieldValue)

	for key, value := range flatFields {
		// Debug logging to understand what MongoDB is returning
		fmt.Printf("[DEBUG expandFieldsFromMongoDB] Processing field '%s' with value type: %T, value: %+v\n", key, value, value)

		if valueMap, ok := value.(map[string]interface{}); ok {
			if boolVal, exists := valueMap["booleanValue"]; exists {
				result[key] = &model.FieldValue{
					ValueType: model.FieldTypeBool,
					Value:     boolVal,
				}
			} else if strVal, exists := valueMap["stringValue"]; exists {
				result[key] = &model.FieldValue{
					ValueType: model.FieldTypeString,
					Value:     strVal,
				}
			} else if intVal, exists := valueMap["integerValue"]; exists {
				result[key] = &model.FieldValue{
					ValueType: model.FieldTypeInt,
					Value:     intVal,
				}
			} else if doubleVal, exists := valueMap["doubleValue"]; exists {
				result[key] = &model.FieldValue{
					ValueType: model.FieldTypeDouble,
					Value:     doubleVal,
				}
			} else if timestampVal, exists := valueMap["timestampValue"]; exists {
				// Handle timestamp values properly - MongoDB can return different types
				if timestampStr, ok := timestampVal.(string); ok {
					// String timestamp - parse it
					if t, err := time.Parse(time.RFC3339Nano, timestampStr); err == nil {
						result[key] = &model.FieldValue{
							ValueType: model.FieldTypeTimestamp,
							Value:     t,
						}
					} else if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
						result[key] = &model.FieldValue{
							ValueType: model.FieldTypeTimestamp,
							Value:     t,
						}
					} else {
						// Fallback to string if can't parse
						result[key] = &model.FieldValue{
							ValueType: model.FieldTypeString,
							Value:     timestampStr,
						}
					}
				} else if timeVal, ok := timestampVal.(time.Time); ok {
					// Direct time.Time value from MongoDB
					result[key] = &model.FieldValue{
						ValueType: model.FieldTypeTimestamp,
						Value:     timeVal,
					}
				} else if primitiveDateTime, ok := timestampVal.(primitive.DateTime); ok {
					// MongoDB primitive.DateTime - convert to time.Time
					result[key] = &model.FieldValue{
						ValueType: model.FieldTypeTimestamp,
						Value:     primitiveDateTime.Time(),
					}
				} else if dateMap, ok := timestampVal.(map[string]interface{}); ok {
					// Handle MongoDB $date format: {"$date": "2025-06-10T11:00:00.000Z"}
					if dateStr, exists := dateMap["$date"]; exists {
						if dateString, ok := dateStr.(string); ok {
							if t, err := time.Parse(time.RFC3339, dateString); err == nil {
								result[key] = &model.FieldValue{
									ValueType: model.FieldTypeTimestamp,
									Value:     t,
								}
							} else if t, err := time.Parse(time.RFC3339Nano, dateString); err == nil {
								result[key] = &model.FieldValue{
									ValueType: model.FieldTypeTimestamp,
									Value:     t,
								}
							}
						} else if dateTime, ok := dateStr.(time.Time); ok {
							result[key] = &model.FieldValue{
								ValueType: model.FieldTypeTimestamp,
								Value:     dateTime,
							}
						}
					}
				}
			} else if arrayVal, exists := valueMap["arrayValue"]; exists {
				// Debug: Log array processing
				fmt.Printf("[DEBUG expandFieldsFromMongoDB] Found arrayValue for field '%s': %+v\n", key, arrayVal)

				if arrayMap, ok := arrayVal.(map[string]interface{}); ok {
					fmt.Printf("[DEBUG expandFieldsFromMongoDB] arrayMap structure: %+v\n", arrayMap)

					// Handle array processing more robustly
					if valuesRaw, exists := arrayMap["values"]; exists {
						fieldValues := make([]*model.FieldValue, 0)

						// Handle different types of values array
						switch values := valuesRaw.(type) {
						case []interface{}:
							fmt.Printf("[DEBUG expandFieldsFromMongoDB] Processing %d array values ([]interface{})\n", len(values))
							for _, val := range values {
								if valMap, ok := val.(map[string]interface{}); ok {
									// Process each array element
									if boolVal, exists := valMap["booleanValue"]; exists {
										fieldValues = append(fieldValues, &model.FieldValue{
											ValueType: model.FieldTypeBool,
											Value:     boolVal,
										})
									} else if strVal, exists := valMap["stringValue"]; exists {
										fieldValues = append(fieldValues, &model.FieldValue{
											ValueType: model.FieldTypeString,
											Value:     strVal,
										})
									} else if intVal, exists := valMap["integerValue"]; exists {
										fieldValues = append(fieldValues, &model.FieldValue{
											ValueType: model.FieldTypeInt,
											Value:     intVal,
										})
									} else if doubleVal, exists := valMap["doubleValue"]; exists {
										fieldValues = append(fieldValues, &model.FieldValue{
											ValueType: model.FieldTypeDouble,
											Value:     doubleVal,
										})
									} else if timestampVal, exists := valMap["timestampValue"]; exists {
										// Handle timestamp in arrays
										if timestampStr, ok := timestampVal.(string); ok {
											if t, err := time.Parse(time.RFC3339Nano, timestampStr); err == nil {
												fieldValues = append(fieldValues, &model.FieldValue{
													ValueType: model.FieldTypeTimestamp,
													Value:     t,
												})
											} else if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
												fieldValues = append(fieldValues, &model.FieldValue{
													ValueType: model.FieldTypeTimestamp,
													Value:     t,
												})
											} else {
												fieldValues = append(fieldValues, &model.FieldValue{
													ValueType: model.FieldTypeString,
													Value:     timestampStr,
												})
											}
										} else if timeVal, ok := timestampVal.(time.Time); ok {
											fieldValues = append(fieldValues, &model.FieldValue{
												ValueType: model.FieldTypeTimestamp,
												Value:     timeVal,
											})
										} else if primitiveDateTime, ok := timestampVal.(primitive.DateTime); ok {
											fieldValues = append(fieldValues, &model.FieldValue{
												ValueType: model.FieldTypeTimestamp,
												Value:     primitiveDateTime.Time(),
											})
										} else if dateMap, ok := timestampVal.(map[string]interface{}); ok {
											if dateStr, exists := dateMap["$date"]; exists {
												if dateString, ok := dateStr.(string); ok {
													if t, err := time.Parse(time.RFC3339, dateString); err == nil {
														fieldValues = append(fieldValues, &model.FieldValue{
															ValueType: model.FieldTypeTimestamp,
															Value:     t,
														})
													} else if t, err := time.Parse(time.RFC3339Nano, dateString); err == nil {
														fieldValues = append(fieldValues, &model.FieldValue{
															ValueType: model.FieldTypeTimestamp,
															Value:     t,
														})
													}
												} else if dateTime, ok := dateStr.(time.Time); ok {
													fieldValues = append(fieldValues, &model.FieldValue{
														ValueType: model.FieldTypeTimestamp,
														Value:     dateTime,
													})
												}
											}
										}
									}
								}
							}

						case []map[string]interface{}:
							fmt.Printf("[DEBUG expandFieldsFromMongoDB] Processing %d array values ([]map[string]interface{})\n", len(values))
							for _, valMap := range values {
								// Process each array element directly
								if boolVal, exists := valMap["booleanValue"]; exists {
									fieldValues = append(fieldValues, &model.FieldValue{
										ValueType: model.FieldTypeBool,
										Value:     boolVal,
									})
								} else if strVal, exists := valMap["stringValue"]; exists {
									fieldValues = append(fieldValues, &model.FieldValue{
										ValueType: model.FieldTypeString,
										Value:     strVal,
									})
								} else if intVal, exists := valMap["integerValue"]; exists {
									fieldValues = append(fieldValues, &model.FieldValue{
										ValueType: model.FieldTypeInt,
										Value:     intVal,
									})
								} else if doubleVal, exists := valMap["doubleValue"]; exists {
									fieldValues = append(fieldValues, &model.FieldValue{
										ValueType: model.FieldTypeDouble,
										Value:     doubleVal,
									})
								} else if timestampVal, exists := valMap["timestampValue"]; exists {
									// Handle timestamp in arrays (second case)
									if timestampStr, ok := timestampVal.(string); ok {
										if t, err := time.Parse(time.RFC3339Nano, timestampStr); err == nil {
											fieldValues = append(fieldValues, &model.FieldValue{
												ValueType: model.FieldTypeTimestamp,
												Value:     t,
											})
										} else if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
											fieldValues = append(fieldValues, &model.FieldValue{
												ValueType: model.FieldTypeTimestamp,
												Value:     t,
											})
										} else {
											fieldValues = append(fieldValues, &model.FieldValue{
												ValueType: model.FieldTypeString,
												Value:     timestampStr,
											})
										}
									} else if timeVal, ok := timestampVal.(time.Time); ok {
										fieldValues = append(fieldValues, &model.FieldValue{
											ValueType: model.FieldTypeTimestamp,
											Value:     timeVal,
										})
									} else if primitiveDateTime, ok := timestampVal.(primitive.DateTime); ok {
										fieldValues = append(fieldValues, &model.FieldValue{
											ValueType: model.FieldTypeTimestamp,
											Value:     primitiveDateTime.Time(),
										})
									} else if dateMap, ok := timestampVal.(map[string]interface{}); ok {
										if dateStr, exists := dateMap["$date"]; exists {
											if dateString, ok := dateStr.(string); ok {
												if t, err := time.Parse(time.RFC3339, dateString); err == nil {
													fieldValues = append(fieldValues, &model.FieldValue{
														ValueType: model.FieldTypeTimestamp,
														Value:     t,
													})
												} else if t, err := time.Parse(time.RFC3339Nano, dateString); err == nil {
													fieldValues = append(fieldValues, &model.FieldValue{
														ValueType: model.FieldTypeTimestamp,
														Value:     t,
													})
												}
											} else if dateTime, ok := dateStr.(time.Time); ok {
												fieldValues = append(fieldValues, &model.FieldValue{
													ValueType: model.FieldTypeTimestamp,
													Value:     dateTime,
												})
											}
										}
									}
								}
							}

						case primitive.A:
							fmt.Printf("[DEBUG expandFieldsFromMongoDB] Processing %d array values (primitive.A)\n", len(values))
							for _, val := range values {
								if valMap, ok := val.(map[string]interface{}); ok {
									// Process each array element
									if boolVal, exists := valMap["booleanValue"]; exists {
										fieldValues = append(fieldValues, &model.FieldValue{
											ValueType: model.FieldTypeBool,
											Value:     boolVal,
										})
									} else if strVal, exists := valMap["stringValue"]; exists {
										fieldValues = append(fieldValues, &model.FieldValue{
											ValueType: model.FieldTypeString,
											Value:     strVal,
										})
									} else if intVal, exists := valMap["integerValue"]; exists {
										fieldValues = append(fieldValues, &model.FieldValue{
											ValueType: model.FieldTypeInt,
											Value:     intVal,
										})
									} else if doubleVal, exists := valMap["doubleValue"]; exists {
										fieldValues = append(fieldValues, &model.FieldValue{
											ValueType: model.FieldTypeDouble,
											Value:     doubleVal,
										})
									} else if timestampVal, exists := valMap["timestampValue"]; exists {
										// Handle timestamp in arrays
										if timestampStr, ok := timestampVal.(string); ok {
											if t, err := time.Parse(time.RFC3339Nano, timestampStr); err == nil {
												fieldValues = append(fieldValues, &model.FieldValue{
													ValueType: model.FieldTypeTimestamp,
													Value:     t,
												})
											} else if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
												fieldValues = append(fieldValues, &model.FieldValue{
													ValueType: model.FieldTypeTimestamp,
													Value:     t,
												})
											} else {
												fieldValues = append(fieldValues, &model.FieldValue{
													ValueType: model.FieldTypeString,
													Value:     timestampStr,
												})
											}
										} else if timeVal, ok := timestampVal.(time.Time); ok {
											fieldValues = append(fieldValues, &model.FieldValue{
												ValueType: model.FieldTypeTimestamp,
												Value:     timeVal,
											})
										} else if primitiveDateTime, ok := timestampVal.(primitive.DateTime); ok {
											fieldValues = append(fieldValues, &model.FieldValue{
												ValueType: model.FieldTypeTimestamp,
												Value:     primitiveDateTime.Time(),
											})
										} else if dateMap, ok := timestampVal.(map[string]interface{}); ok {
											if dateStr, exists := dateMap["$date"]; exists {
												if dateString, ok := dateStr.(string); ok {
													if t, err := time.Parse(time.RFC3339, dateString); err == nil {
														fieldValues = append(fieldValues, &model.FieldValue{
															ValueType: model.FieldTypeTimestamp,
															Value:     t,
														})
													} else if t, err := time.Parse(time.RFC3339Nano, dateString); err == nil {
														fieldValues = append(fieldValues, &model.FieldValue{
															ValueType: model.FieldTypeTimestamp,
															Value:     t,
														})
													}
												} else if dateTime, ok := dateStr.(time.Time); ok {
													fieldValues = append(fieldValues, &model.FieldValue{
														ValueType: model.FieldTypeTimestamp,
														Value:     dateTime,
													})
												}
											}
										}
									}
								}
							}

						default:
							fmt.Printf("[DEBUG expandFieldsFromMongoDB] Unexpected values type: %T\n", values)
						}

						// Create the array field
						fmt.Printf("[DEBUG expandFieldsFromMongoDB] Creating FieldValue for array field '%s' with %d values\n", key, len(fieldValues))
						result[key] = &model.FieldValue{
							ValueType: model.FieldTypeArray,
							Value: &model.ArrayValue{
								Values: fieldValues,
							},
						}
						fmt.Printf("[DEBUG expandFieldsFromMongoDB] Successfully created array field '%s'\n", key)
					} else {
						fmt.Printf("[DEBUG expandFieldsFromMongoDB] Array field '%s' missing 'values' key\n", key)
					}
				} else {
					fmt.Printf("[DEBUG expandFieldsFromMongoDB] arrayVal for field '%s' is not a map[string]interface{}\n", key)
				}
			} else if mapVal, exists := valueMap["mapValue"]; exists {
				if mapMap, ok := mapVal.(map[string]interface{}); ok {
					if fields, ok := mapMap["fields"].(map[string]interface{}); ok {
						result[key] = &model.FieldValue{
							ValueType: model.FieldTypeMap,
							Value: &model.MapValue{
								Fields: expandFieldsFromMongoDB(fields),
							},
						}
					}
				}
			}
		}
	}

	return result
}

// MongoDocumentFlat representa la estructura del documento en MongoDB con campos aplanados
type MongoDocumentFlat struct {
	ID                primitive.ObjectID     `bson:"_id,omitempty"`
	ProjectID         string                 `bson:"projectID"`
	DatabaseID        string                 `bson:"databaseID"`
	CollectionID      string                 `bson:"collectionID"`
	DocumentID        string                 `bson:"documentID"`
	Path              string                 `bson:"path"`
	ParentPath        string                 `bson:"parentPath"`
	Fields            map[string]interface{} `bson:"fields"`
	CreateTime        time.Time              `bson:"createTime"`
	UpdateTime        time.Time              `bson:"updateTime"`
	ReadTime          time.Time              `bson:"readTime"`
	Version           int64                  `bson:"version"`
	Exists            bool                   `bson:"exists"`
	HasSubcollections bool                   `bson:"hasSubcollections"`
}

// Helper: convert model.Document to MongoDocument
// modelToMongoDocumentFlat convierte un modelo de documento a la estructura plana de MongoDB
func modelToMongoDocumentFlat(doc *model.Document) *MongoDocumentFlat {
	return &MongoDocumentFlat{
		ID:                doc.ID,
		ProjectID:         doc.ProjectID,
		DatabaseID:        doc.DatabaseID,
		CollectionID:      doc.CollectionID,
		DocumentID:        doc.DocumentID,
		Path:              doc.Path,
		ParentPath:        doc.ParentPath,
		Fields:            flattenFieldsForMongoDB(doc.Fields),
		CreateTime:        doc.CreateTime,
		UpdateTime:        doc.UpdateTime,
		ReadTime:          doc.ReadTime,
		Version:           doc.Version,
		Exists:            doc.Exists,
		HasSubcollections: doc.HasSubcollections,
	}
}

// mongoFlatToModelDocument convierte un MongoDocumentFlat a model.Document
func mongoFlatToModelDocument(mongoDoc *MongoDocumentFlat) *model.Document {
	return &model.Document{
		ID:                mongoDoc.ID,
		ProjectID:         mongoDoc.ProjectID,
		DatabaseID:        mongoDoc.DatabaseID,
		CollectionID:      mongoDoc.CollectionID,
		DocumentID:        mongoDoc.DocumentID,
		Path:              mongoDoc.Path,
		ParentPath:        mongoDoc.ParentPath,
		Fields:            expandFieldsFromMongoDB(mongoDoc.Fields),
		CreateTime:        mongoDoc.CreateTime,
		UpdateTime:        mongoDoc.UpdateTime,
		ReadTime:          mongoDoc.ReadTime,
		Version:           mongoDoc.Version,
		Exists:            mongoDoc.Exists,
		HasSubcollections: mongoDoc.HasSubcollections,
	}
}

// CreateDocument creates a new document
func (ops *DocumentOperations) CreateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue) (*model.Document, error) {
	now := time.Now()

	// Generate a new document ID if not provided
	if documentID == "" {
		documentID = primitive.NewObjectID().Hex()
	}

	// Build the document path
	path := fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID)
	parentPath := fmt.Sprintf("projects/%s/databases/%s/documents/%s", projectID, databaseID, collectionID)

	doc := &model.Document{
		ID:           primitive.NewObjectID(),
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
		Path:         path,
		ParentPath:   parentPath,
		Fields:       data,
		CreateTime:   now,
		UpdateTime:   now,
		Version:      1,
		Exists:       true,
	}

	// Convert to MongoDB document with flattened fields
	mongoDoc := modelToMongoDocumentFlat(doc)

	fmt.Println("[CreateDocument] Flattened fields for MongoDB:", mongoDoc.Fields)

	// CORRECCIÓN CRÍTICA: Usar colección separada basada en collectionID
	targetCollection := ops.repo.db.Collection(collectionID)
	// Insert into MongoDB
	_, err := targetCollection.InsertOne(ctx, mongoDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	// Log the creation
	ops.repo.logger.Info(fmt.Sprintf("Document created successfully in MongoDB - documentID: %s", documentID))

	return doc, nil
}

// UpdateDocument updates a document
func (ops *DocumentOperations) UpdateDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	filter := bson.M{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	// Flatten the fields for MongoDB storage
	flattenedFields := flattenFieldsForMongoDB(data)

	update := bson.M{
		"$set": bson.M{
			"fields":     flattenedFields,
			"updateTime": time.Now(),
		},
		"$inc": bson.M{
			"version": 1,
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var mongoDoc MongoDocumentFlat
	// CORRECCIÓN CRÍTICA: Usar colección separada basada en collectionID
	targetCollection := ops.repo.db.Collection(collectionID)
	err := targetCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&mongoDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return mongoFlatToModelDocument(&mongoDoc), nil
}

// SetDocument sets (creates or updates) a document
func (ops *DocumentOperations) SetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string, data map[string]*model.FieldValue, merge bool) (*model.Document, error) {
	now := time.Now()

	// Generate a new document ID if not provided
	if documentID == "" {
		documentID = primitive.NewObjectID().Hex()
	}

	// Build the document path
	path := fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID)
	parentPath := fmt.Sprintf("projects/%s/databases/%s/documents/%s", projectID, databaseID, collectionID)

	filter := bson.M{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	// Flatten the fields for MongoDB storage
	flattenedFields := flattenFieldsForMongoDB(data)

	update := bson.M{
		"$set": bson.M{
			"projectID":    projectID,
			"databaseID":   databaseID,
			"collectionID": collectionID,
			"documentID":   documentID,
			"path":         path,
			"parentPath":   parentPath,
			"fields":       flattenedFields,
			"updateTime":   now,
			"exists":       true,
		},
		"$setOnInsert": bson.M{
			"createTime": now,
			"version":    1,
		},
		"$inc": bson.M{
			"version": 1,
		},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var mongoDoc MongoDocumentFlat
	// CORRECCIÓN CRÍTICA: Usar colección separada basada en collectionID
	targetCollection := ops.repo.db.Collection(collectionID)
	err := targetCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&mongoDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to set document: %w", err)
	}

	return mongoFlatToModelDocument(&mongoDoc), nil
}

// DeleteDocument deletes a document by ID
func (ops *DocumentOperations) DeleteDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) error {
	filter := bson.M{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
	}

	// CORRECCIÓN CRÍTICA: Usar colección separada basada en collectionID
	targetCollection := ops.repo.db.Collection(collectionID)
	result, err := targetCollection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if result.Deleted() == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// GetDocument retrieves a document by ID
func (ops *DocumentOperations) GetDocument(ctx context.Context, projectID, databaseID, collectionID, documentID string) (*model.Document, error) {
	filter := bson.M{
		"projectID":    projectID,
		"databaseID":   databaseID,
		"collectionID": collectionID,
		"documentID":   documentID,
		"exists":       true,
	}

	// CORRECCIÓN CRÍTICA: Usar colección separada basada en collectionID
	targetCollection := ops.repo.db.Collection(collectionID)
	var mongoDoc MongoDocumentFlat
	err := targetCollection.FindOne(ctx, filter).Decode(&mongoDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Debug logging for GetDocument
	fmt.Printf("[DEBUG GetDocument] Raw mongoDoc.Fields: %+v\n", mongoDoc.Fields)

	modelDoc := mongoFlatToModelDocument(&mongoDoc)

	// Debug logging after conversion
	fmt.Printf("[DEBUG GetDocument] Converted modelDoc.Fields: %+v\n", modelDoc.Fields)

	return modelDoc, nil
}

// GetDocumentByPath retrieves a document by path
func (ops *DocumentOperations) GetDocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	projectID, databaseID, collectionID, documentID, err := parseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	return ops.GetDocument(ctx, projectID, databaseID, collectionID, documentID)
}

// CreateDocumentByPath creates a document by path
func (ops *DocumentOperations) CreateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue) (*model.Document, error) {
	projectID, databaseID, collectionID, documentID, err := parseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	return ops.CreateDocument(ctx, projectID, databaseID, collectionID, documentID, data)
}

// UpdateDocumentByPath updates a document by path
func (ops *DocumentOperations) UpdateDocumentByPath(ctx context.Context, path string, data map[string]*model.FieldValue, updateMask []string) (*model.Document, error) {
	projectID, databaseID, collectionID, documentID, err := parseFirestorePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path format: %w", err)
	}

	return ops.UpdateDocument(ctx, projectID, databaseID, collectionID, documentID, data, updateMask)
}

// DeleteDocumentByPath deletes a document by path
func (ops *DocumentOperations) DeleteDocumentByPath(ctx context.Context, path string) error {
	projectID, databaseID, collectionID, documentID, err := parseFirestorePath(path)
	if err != nil {
		return fmt.Errorf("invalid path format: %w", err)
	}

	return ops.DeleteDocument(ctx, projectID, databaseID, collectionID, documentID)
}

// ListDocuments lists documents in a collection
func (ops *DocumentOperations) ListDocuments(ctx context.Context, projectID, databaseID, collectionID string, pageSize int32, pageToken string, orderBy string, showMissing bool) ([]*model.Document, string, error) {
	// When using separate collections, we need minimal filtering
	// Include projectID and databaseID for security, but not collectionID since we're in the specific collection
	// SOLUCIÓN: Para colecciones separadas, usar filtro mínimo sin metadata
	filter := bson.M{}

	// Handle pagination with page token
	if pageToken != "" {
		// For simplicity, use documentID as pagination token
		filter["documentID"] = bson.M{"$gt": pageToken}
	}

	// Set default page size if not provided
	if pageSize <= 0 {
		pageSize = 50 // Default page size
	}

	// Build find options
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "documentID", Value: 1}}) // Sort by documentID for consistent pagination
	findOptions.SetLimit(int64(pageSize + 1))                  // Request one extra to determine if there's a next page

	// CORRECCIÓN CRÍTICA: Usar colección separada basada en collectionID
	targetCollection := ops.repo.db.Collection(collectionID)

	cursor, err := targetCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list documents: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []*model.Document
	var mongoDocuments []MongoDocumentFlat

	for cursor.Next(ctx) {
		var mongoDoc MongoDocumentFlat
		if err := cursor.Decode(&mongoDoc); err != nil {
			return nil, "", fmt.Errorf("failed to decode document: %w", err)
		}
		mongoDocuments = append(mongoDocuments, mongoDoc)
	}

	if err := cursor.Err(); err != nil {
		return nil, "", fmt.Errorf("cursor error: %w", err)
	}

	// Determine next page token
	var nextPageToken string
	if len(mongoDocuments) > int(pageSize) {
		// There's a next page, use the last document in the current page as token
		nextPageToken = mongoDocuments[pageSize-1].DocumentID
		// Remove the extra document from results
		mongoDocuments = mongoDocuments[:pageSize]
	}

	// Convert to model documents
	for _, mongoDoc := range mongoDocuments {
		documents = append(documents, mongoFlatToModelDocument(&mongoDoc))
	}

	return documents, nextPageToken, nil
}
