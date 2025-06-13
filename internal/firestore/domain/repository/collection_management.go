package repository

// CollectionNamingStrategy define cómo mapear rutas Firestore a nombres de colección optimizados.
type CollectionNamingStrategy interface {
	CollectionName(projectID, databaseID, collectionPath string) string
}

// CollectionFactory crea o recupera colecciones según el contexto y la estrategia de nomenclatura.
type CollectionFactory interface {
	GetCollection(projectID, databaseID, collectionPath string) (CollectionReference, error)
}

// CollectionReference abstrae una referencia a una colección optimizada.
type CollectionReference interface {
	Name() string
	// Métodos CRUD, índices, etc. pueden agregarse aquí según la arquitectura optimizada.
}

// CollectionManager maneja el ciclo de vida y caching de colecciones optimizadas.
type CollectionManager interface {
	GetOrCreateCollection(projectID, databaseID, collectionPath string) (CollectionReference, error)
}

// Value object para identificador de colección, alineado a la arquitectura optimizada.
type CollectionIdentifier struct {
	ProjectID      string
	DatabaseID     string
	CollectionPath string
}

func (c CollectionIdentifier) String() string {
	return c.ProjectID + "." + c.DatabaseID + "." + c.CollectionPath
}
