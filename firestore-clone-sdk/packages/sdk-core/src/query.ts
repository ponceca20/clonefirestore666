import {
  Firestore, // Interface
  DocumentData,
  FieldPath,
  WhereFilterOp,
  OrderByDirection,
  DocumentSnapshot as DocumentSnapshotInterface, // Interface
  QueryDocumentSnapshot as QueryDocumentSnapshotInterface, // Interface
  QuerySnapshot as IQuerySnapshot, // Use a different alias for the interface
  DocumentChange,
  SnapshotListenOptions,
  CollectionReference as CollectionReferenceInterface, // Interface
  DocumentReference as DocumentReferenceInterface, // Interface
  Query as QueryInterface // Import Query interface with alias
} from './firestore';
import {
  DocumentSnapshot as ActualDocSnapshotClass,
  QueryDocumentSnapshot as ActualQueryDocSnapshotClass,
  DocumentReference as ActualDocRefClass
} from './document'; // Actual DocumentSnapshot and DocumentReference classes
import { BackendQueryResponse, BackendDocumentResponse } from './backend_client'; // For response types

// QueryConstraint interface remains the same
export interface QueryConstraint {
  type: 'where' | 'orderBy' | 'limit' | 'limitToLast' | 'startAt' | 'startAfter' | 'endAt' | 'endBefore';
  apply(query: Query<any>): Query<any>;
  // Add specific fields for each constraint type for better type safety if needed later
  fieldPath?: string | FieldPath;
  opStr?: WhereFilterOp;
  value?: any;
  directionStr?: OrderByDirection;
  limit?: number;
  values?: any[]; // For startAt, startAfter etc.
}

export class QuerySnapshot<T = DocumentData> implements IQuerySnapshot<T> { // Use new alias
  readonly query: Query<T>; // This will be QueryClass instance
  readonly docs: Array<ActualQueryDocSnapshotClass<T>>;
  readonly empty: boolean;
  readonly size: number;
  private _docChanges?: Array<DocumentChange<T>>;

  constructor(query: Query<T>, docs: Array<ActualQueryDocSnapshotClass<T>>, docChanges?: Array<DocumentChange<T>>) {
    this.query = query;
    this.docs = docs;
    this.empty = docs.length === 0;
    this.size = docs.length;
    this._docChanges = docChanges;
  }

  docChanges(options?: SnapshotListenOptions): Array<DocumentChange<T>> {
    if (this._docChanges) {
        return this._docChanges;
    }
    if (options && options.includeMetadataChanges) {
        return this.docs.map((doc, index) => ({
            type: 'added' as 'added', // Cast to literal type
            doc: doc,
            oldIndex: -1,
            newIndex: index,
        }));
    }
    return [];
  }

  forEach(callback: (result: ActualQueryDocSnapshotClass<T>, index: number) => void, thisArg?: any): void {
    this.docs.forEach(callback, thisArg);
  }
}

export class Query<T = DocumentData> implements QueryInterface<T> {
  readonly firestore: Firestore; // Instance of FirestoreService
  protected _constraints: QueryConstraint[] = [];
  protected _path: string; // Collection path

  constructor(firestore: Firestore, path: string) {
    this.firestore = firestore;
    this._path = path;
  }

  protected _addConstraint(constraintInput: Partial<QueryConstraint>): Query<T> {
    // Create a new Query instance for immutability
    // The constructor for Query needs firestore and path.
    // If CollectionReference extends Query, this new Query might need to be a CollectionReference
    // if the methods are meant to return the same type.
    // For now, Query methods return Query.
    const newQueryInstance = new Query<T>(this.firestore, this._path);
    newQueryInstance._constraints = [...this._constraints, constraintInput as QueryConstraint];
    return newQueryInstance;
  }

  where(fieldPath: string | FieldPath, opStr: WhereFilterOp, value: any): Query<T> {
    return this._addConstraint({ type: 'where', fieldPath, opStr, value });
  }

  orderBy(fieldPath: string | FieldPath, directionStr: OrderByDirection = 'asc'): Query<T> {
    return this._addConstraint({ type: 'orderBy', fieldPath, directionStr });
  }

  limit(limitCount: number): Query<T> {
    if (limitCount <= 0) throw new Error("Query limit must be a positive integer.");
    return this._addConstraint({ type: 'limit', limit: limitCount });
  }

  limitToLast(limitToLastCount: number): Query<T> {
    if (limitToLastCount <= 0) throw new Error("Query limitToLast must be a positive integer.");
    return this._addConstraint({ type: 'limitToLast', limit: limitToLastCount });
  }

  startAt(snapshot: DocumentSnapshotInterface<any>): Query<T>;
  startAt(...fieldValues: any[]): Query<T>;
  startAt(...args: any[]): Query<T> {
    return this._addConstraint({ type: 'startAt', values: args });
  }

  startAfter(snapshot: DocumentSnapshotInterface<any>): Query<T>;
  startAfter(...fieldValues: any[]): Query<T>;
  startAfter(...args: any[]): Query<T> {
    return this._addConstraint({ type: 'startAfter', values: args });
  }

  endBefore(snapshot: DocumentSnapshotInterface<any>): Query<T>;
  endBefore(...fieldValues: any[]): Query<T>;
  endBefore(...args: any[]): Query<T> {
    return this._addConstraint({ type: 'endBefore', values: args });
  }

  endAt(snapshot: DocumentSnapshotInterface<any>): Query<T>;
  endAt(...fieldValues: any[]): Query<T>;
  endAt(...args: any[]): Query<T> {
    return this._addConstraint({ type: 'endAt', values: args });
  }

  async get(): Promise<QuerySnapshot<T>> {
    console.log(`Query.get() for path: ${this._path} with constraints:`, this._constraints);
    try {
      const response: BackendQueryResponse = await this.firestore.backendClient.executeQuery(this._path, this._constraints);

      // Need to get the CollectionReference for the parent of these docs to pass to ActualDocRefClass constructor
      // This query's path IS the collection path.
      const parentCollectionRef = this.firestore.collection(this._path);

      const docs = response.docs.map(docData => {
        // Construct the full path for each document
        const docFullPath = `${this._path}/${docData.id}`;
        // Create a DocumentReference for each document
        const docRef = new ActualDocRefClass<T>(
            this.firestore,
            docFullPath,
            parentCollectionRef as any // Cast because parentCollectionRef is CollectionReference<DocumentData> by default from firestore.collection
                                       // but ActualDocRefClass might expect CollectionReference<T>
                                       // This relies on parentCollectionRef being compatible.
        );
        return new ActualQueryDocSnapshotClass<T>(
          this.firestore,
          docRef, // Pass the created DocumentReference
          docData.id,
          docData.data as T
        );
      });
      // TODO: Process docChanges if backend provides them and they are needed for get()
      return new QuerySnapshot<T>(this, docs);
    } catch (error) {
      console.error(`Error in Query.get() for ${this._path}:`, error);
      throw error;
    }
  }

  onSnapshot(observer: {
    next?: (snapshot: IQuerySnapshot<T>) => void; // Use new alias for interface for observer
    error?: (error: Error) => void;
  }): () => void {
    console.log(`Query.onSnapshot() for path: ${this._path} with constraints:`, this._constraints);
    // Delegate to RealtimeManager
    // The RealtimeManager will handle transforming backend messages to QuerySnapshots
    return this.firestore.realtimeManager.listenToQuery(this, observer);
  }
}

// QueryInterface is now imported at the top.

console.log('query.ts (with implemented get/onSnapshot) loaded');
