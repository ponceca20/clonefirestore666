package model

import (
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Document represents a document in Firestore following the hierarchy:
// projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/{COLLECTION_ID}/{DOCUMENT_ID}
type Document struct {
	// MongoDB internal ID
	ID primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`

	// Firestore hierarchy identifiers
	ProjectID    string `json:"projectID" bson:"project_id"`
	DatabaseID   string `json:"databaseID" bson:"database_id"`
	CollectionID string `json:"collectionID" bson:"collection_id"`
	DocumentID   string `json:"documentID" bson:"document_id"`

	// Document path and parent information
	Path       string `json:"path" bson:"path"`              // Full path: projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/{COLLECTION_ID}/{DOCUMENT_ID}
	ParentPath string `json:"parentPath" bson:"parent_path"` // Parent collection path

	// Document data and metadata
	Fields map[string]*FieldValue `json:"fields,omitempty" bson:"fields,omitempty"`

	// Firestore timestamps
	CreateTime time.Time `json:"createTime" bson:"create_time"`
	UpdateTime time.Time `json:"updateTime" bson:"update_time"`
	ReadTime   time.Time `json:"readTime,omitempty" bson:"read_time,omitempty"`

	// Version and etag for optimistic concurrency
	Version int64  `json:"version" bson:"version"`
	ETag    string `json:"etag,omitempty" bson:"etag,omitempty"`

	// Document state
	Exists bool `json:"exists" bson:"exists"`

	// Subcollections
	HasSubcollections bool     `json:"hasSubcollections" bson:"has_subcollections"`
	Subcollections    []string `json:"subcollections,omitempty" bson:"subcollections,omitempty"`
}

// FieldValue represents a Firestore field value with type information
type FieldValue struct {
	ValueType FieldValueType `json:"valueType" bson:"value_type"`
	Value     interface{}    `json:"value" bson:"value"`
}

// MarshalJSON implements custom JSON marshaling for FieldValue to match Firestore API format
func (fv *FieldValue) MarshalJSON() ([]byte, error) {
	// Create a map that follows Firestore API format
	result := make(map[string]interface{})

	switch fv.ValueType {
	case FieldTypeBool:
		result["booleanValue"] = fv.Value
	case FieldTypeString:
		result["stringValue"] = fv.Value
	case FieldTypeInt:
		// Firestore API expects integers as strings
		if intVal, ok := fv.Value.(int64); ok {
			result["integerValue"] = fmt.Sprintf("%d", intVal)
		} else if strVal, ok := fv.Value.(string); ok {
			result["integerValue"] = strVal
		} else {
			result["integerValue"] = fmt.Sprintf("%v", fv.Value)
		}
	case FieldTypeDouble:
		result["doubleValue"] = fv.Value
	case FieldTypeTimestamp:
		// Convert timestamp to Firestore API format
		if timeVal, ok := fv.Value.(time.Time); ok {
			result["timestampValue"] = timeVal.Format(time.RFC3339Nano)
		} else if strVal, ok := fv.Value.(string); ok {
			result["timestampValue"] = strVal
		} else {
			result["timestampValue"] = fv.Value
		}
	case FieldTypeBytes:
		result["bytesValue"] = fv.Value
	case FieldTypeReference:
		result["referenceValue"] = fv.Value
	case FieldTypeGeoPoint:
		result["geoPointValue"] = fv.Value
	case FieldTypeArray:
		result["arrayValue"] = fv.Value
	case FieldTypeMap:
		result["mapValue"] = fv.Value
	case FieldTypeNull:
		result["nullValue"] = nil
	default:
		// Fallback - use the value type as key
		result[string(fv.ValueType)] = fv.Value
	}

	return json.Marshal(result)
}

// UnmarshalJSON implements custom JSON unmarshaling for FieldValue from Firestore API format
func (fv *FieldValue) UnmarshalJSON(data []byte) error {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	// Determine the value type and value from the map
	if val, exists := result["booleanValue"]; exists {
		fv.ValueType = FieldTypeBool
		fv.Value = val
	} else if val, exists := result["stringValue"]; exists {
		fv.ValueType = FieldTypeString
		fv.Value = val
	} else if val, exists := result["integerValue"]; exists {
		fv.ValueType = FieldTypeInt
		fv.Value = val
	} else if val, exists := result["doubleValue"]; exists {
		fv.ValueType = FieldTypeDouble
		fv.Value = val
	} else if val, exists := result["timestampValue"]; exists {
		fv.ValueType = FieldTypeTimestamp
		// Try to parse as time if it's a string
		if strVal, ok := val.(string); ok {
			if t, err := time.Parse(time.RFC3339, strVal); err == nil {
				fv.Value = t
			} else if t, err := time.Parse(time.RFC3339Nano, strVal); err == nil {
				fv.Value = t
			} else {
				fv.Value = strVal
			}
		} else {
			fv.Value = val
		}
	} else if val, exists := result["bytesValue"]; exists {
		fv.ValueType = FieldTypeBytes
		fv.Value = val
	} else if val, exists := result["referenceValue"]; exists {
		fv.ValueType = FieldTypeReference
		fv.Value = val
	} else if val, exists := result["geoPointValue"]; exists {
		fv.ValueType = FieldTypeGeoPoint
		fv.Value = val
	} else if val, exists := result["arrayValue"]; exists {
		fv.ValueType = FieldTypeArray
		fv.Value = val
	} else if val, exists := result["mapValue"]; exists {
		fv.ValueType = FieldTypeMap
		fv.Value = val
	} else if _, exists := result["nullValue"]; exists {
		fv.ValueType = FieldTypeNull
		fv.Value = nil
	} else {
		// Unknown type - use the first key as type
		for key, val := range result {
			fv.ValueType = FieldValueType(key)
			fv.Value = val
			break
		}
	}

	return nil
}

// FieldValueType represents the type of a Firestore field value
type FieldValueType string

const (
	// Primitive types
	FieldTypeNull      FieldValueType = "nullValue"
	FieldTypeBool      FieldValueType = "booleanValue"
	FieldTypeInt       FieldValueType = "integerValue"
	FieldTypeDouble    FieldValueType = "doubleValue"
	FieldTypeString    FieldValueType = "stringValue"
	FieldTypeBytes     FieldValueType = "bytesValue"
	FieldTypeTimestamp FieldValueType = "timestampValue"

	// Complex types
	FieldTypeReference FieldValueType = "referenceValue"
	FieldTypeGeoPoint  FieldValueType = "geoPointValue"
	FieldTypeArray     FieldValueType = "arrayValue"
	FieldTypeMap       FieldValueType = "mapValue"
)

// GeoPoint represents a geographic point
type GeoPoint struct {
	Latitude  float64 `json:"latitude" bson:"latitude"`
	Longitude float64 `json:"longitude" bson:"longitude"`
}

// ArrayValue represents an array of field values
type ArrayValue struct {
	Values []*FieldValue `json:"values" bson:"values"`
}

// MapValue represents a map of field values
type MapValue struct {
	Fields map[string]*FieldValue `json:"fields" bson:"fields"`
}

// WriteOperationType represents the type of write operation
type WriteOperationType string

// WriteOperationType constants updated to match usage in document_repo.go
const (
	WriteTypeCreate         WriteOperationType = "create"
	WriteTypeUpdate         WriteOperationType = "update"
	WriteTypeDelete         WriteOperationType = "delete"
	WriteTypeSet            WriteOperationType = "set"
	WriteOperationTransform WriteOperationType = "transform"
)

// WriteOperation represents a single operation in a batch write.
type WriteOperation struct {
	Type         WriteOperationType     `json:"type"`
	Path         string                 `json:"path"`                   // Full document path
	Data         map[string]interface{} `json:"data,omitempty"`         // Used for Create and Update
	Precondition *Precondition          `json:"precondition,omitempty"` // Conditional check for operation
}

// DocumentTransform represents field transformations
type DocumentTransform struct {
	Document        string            `json:"document" bson:"document"`
	FieldTransforms []*FieldTransform `json:"fieldTransforms" bson:"field_transforms"`
}

// FieldTransform represents a transformation on a field
type FieldTransform struct {
	FieldPath             string      `json:"fieldPath" bson:"field_path"`
	SetToServerValue      ServerValue `json:"setToServerValue,omitempty" bson:"set_to_server_value,omitempty"`
	Increment             *FieldValue `json:"increment,omitempty" bson:"increment,omitempty"`
	Maximum               *FieldValue `json:"maximum,omitempty" bson:"maximum,omitempty"`
	Minimum               *FieldValue `json:"minimum,omitempty" bson:"minimum,omitempty"`
	AppendMissingElements *ArrayValue `json:"appendMissingElements,omitempty" bson:"append_missing_elements,omitempty"`
	RemoveAllFromArray    *ArrayValue `json:"removeAllFromArray,omitempty" bson:"remove_all_from_array,omitempty"`
}

// ServerValue represents server-side values
type ServerValue string

const (
	ServerValueTimestamp ServerValue = "REQUEST_TIME"
)

// Precondition represents a precondition for write operations
type Precondition struct {
	Exists     *bool      `json:"exists,omitempty" bson:"exists,omitempty"`         // Document must exist (true) or not exist (false)
	UpdateTime *time.Time `json:"updateTime,omitempty" bson:"updateTime,omitempty"` // Document must have this update time
}

// DocumentMask represents which fields to return
type DocumentMask struct {
	FieldPaths []string `json:"fieldPaths" bson:"field_paths"`
}

// Reference represents a reference to another document.
type Reference string

// FieldTransformType represents the type of field transformation
type FieldTransformType string

const (
	TransformSetToServerValue      FieldTransformType = "SET_TO_SERVER_VALUE"
	TransformIncrement             FieldTransformType = "INCREMENT"
	TransformMaximum               FieldTransformType = "MAXIMUM"
	TransformMinimum               FieldTransformType = "MINIMUM"
	TransformAppendMissingElements FieldTransformType = "APPEND_MISSING_ELEMENTS"
	TransformRemoveAllFromArray    FieldTransformType = "REMOVE_ALL_FROM_ARRAY"
)

// AggregationResult represents the result of an aggregation query
type AggregationResult struct {
	Count    *int64                 `json:"count,omitempty" bson:"count,omitempty"`
	Sum      *FieldValue            `json:"sum,omitempty" bson:"sum,omitempty"`
	Average  *FieldValue            `json:"average,omitempty" bson:"average,omitempty"`
	Fields   map[string]*FieldValue `json:"fields,omitempty" bson:"fields,omitempty"`
	ReadTime time.Time              `json:"readTime" bson:"read_time"`
}

// GetResourceName returns the full resource name for this document
func (d *Document) GetResourceName() string {
	return d.Path
}

// GetCollectionPath returns the collection path for this document
func (d *Document) GetCollectionPath() string {
	return d.ParentPath
}

// GetCollectionGroupPath returns the collection group path
func (d *Document) GetCollectionGroupPath() string {
	return d.CollectionID
}

// IsSubcollectionDocument returns true if this document is in a subcollection
func (d *Document) IsSubcollectionDocument() bool {
	// Count the number of segments in the path
	// A subcollection document has more than 6 segments
	// projects/{PROJECT_ID}/databases/{DATABASE_ID}/documents/{COLLECTION_ID}/{DOCUMENT_ID}
	return len(d.ParentPath) > 6
}

// NewDocument creates a new document with the given parameters
func NewDocument(projectID, databaseID, collectionID, documentID string, fields map[string]*FieldValue) *Document {
	now := time.Now()
	path := fmt.Sprintf("projects/%s/databases/%s/documents/%s/%s", projectID, databaseID, collectionID, documentID)
	parentPath := fmt.Sprintf("projects/%s/databases/%s/documents/%s", projectID, databaseID, collectionID)

	return &Document{
		ProjectID:    projectID,
		DatabaseID:   databaseID,
		CollectionID: collectionID,
		DocumentID:   documentID,
		Path:         path,
		ParentPath:   parentPath,
		Fields:       fields,
		CreateTime:   now,
		UpdateTime:   now,
		Version:      1,
		Exists:       true,
	}
}

// NewFieldValue creates a new field value with the appropriate type
func NewFieldValue(value interface{}) *FieldValue {
	switch v := value.(type) {
	case nil:
		return &FieldValue{ValueType: FieldTypeNull, Value: nil}
	case bool:
		return &FieldValue{ValueType: FieldTypeBool, Value: v}
	case int, int32, int64:
		return &FieldValue{ValueType: FieldTypeInt, Value: v}
	case float32, float64:
		return &FieldValue{ValueType: FieldTypeDouble, Value: v}
	case string:
		// Efficient timestamp detection - only by pattern, not field name
		parser := NewTimestampParser()
		if timestamp, isTimestamp := parser.TryParseAsTimestamp(v); isTimestamp {
			return &FieldValue{ValueType: FieldTypeTimestamp, Value: timestamp}
		}
		return &FieldValue{ValueType: FieldTypeString, Value: v}
	case []byte:
		return &FieldValue{ValueType: FieldTypeBytes, Value: v}
	case time.Time:
		return &FieldValue{ValueType: FieldTypeTimestamp, Value: v}
	case *GeoPoint:
		return &FieldValue{ValueType: FieldTypeGeoPoint, Value: v}
	case []interface{}:
		arrayValues := make([]*FieldValue, len(v))
		for i, item := range v {
			arrayValues[i] = NewFieldValue(item)
		}
		return &FieldValue{ValueType: FieldTypeArray, Value: &ArrayValue{Values: arrayValues}}
	case map[string]interface{}:
		mapFields := make(map[string]*FieldValue)
		for k, item := range v {
			mapFields[k] = NewFieldValue(item)
		}
		return &FieldValue{ValueType: FieldTypeMap, Value: &MapValue{Fields: mapFields}}
	default:
		// Default to string representation
		return &FieldValue{ValueType: FieldTypeString, Value: fmt.Sprintf("%v", v)}
	}
}

// NewFieldValueWithHint creates a field value with explicit type hint from client
func NewFieldValueWithHint(value interface{}, typeHint string) *FieldValue {
	// If client explicitly provides type hint, trust it
	parser := NewTimestampParser()
	if timestamp, isTimestamp := parser.ParseWithHint(value, typeHint); isTimestamp {
		return &FieldValue{ValueType: FieldTypeTimestamp, Value: timestamp}
	}

	// Fallback to normal detection
	return NewFieldValue(value)
}

// NewFieldValueWithContext creates a new field value with context-aware type detection
// DEPRECATED: Use NewFieldValue instead - field name detection is unreliable
func NewFieldValueWithContext(fieldName string, value interface{}) *FieldValue {
	// Just use the normal NewFieldValue - no field name dependency
	return NewFieldValue(value)
}

// ToInterface converts a FieldValue back to a Go interface{}
func (fv *FieldValue) ToInterface() interface{} {
	switch fv.ValueType {
	case FieldTypeNull:
		return nil
	case FieldTypeBool:
		return fv.Value.(bool)
	case FieldTypeInt:
		return fv.Value
	case FieldTypeDouble:
		return fv.Value
	case FieldTypeString:
		return fv.Value.(string)
	case FieldTypeBytes:
		return fv.Value.([]byte)
	case FieldTypeTimestamp:
		return fv.Value.(time.Time)
	case FieldTypeGeoPoint:
		return fv.Value.(*GeoPoint)
	case FieldTypeArray:
		arrayValue := fv.Value.(*ArrayValue)
		result := make([]interface{}, len(arrayValue.Values))
		for i, val := range arrayValue.Values {
			result[i] = val.ToInterface()
		}
		return result
	case FieldTypeMap:
		mapValue := fv.Value.(*MapValue)
		result := make(map[string]interface{})
		for k, val := range mapValue.Fields {
			result[k] = val.ToInterface()
		}
		return result
	default:
		return fv.Value
	}
}
