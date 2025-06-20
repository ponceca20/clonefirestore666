import { BackendClient, RestBackendClient } from './backend_client';
import { CollectionReference as CollectionReferenceClass } from './collection'; // Actual class
import { DocumentReference as DocumentReferenceClass } from './document'; // Actual class
import { RealtimeManager } from './realtime'; // For onSnapshot integration later

// --- Keep existing interfaces and types from the previous firestore.ts version ---
// (FirebaseAppConfig, FirebaseApp, DocumentData, FieldPath, WhereFilterOp, OrderByDirection, etc.)
// Ensure they are all here or imported if they were moved.

/**
 * Configuration for initializing a FirebaseApp.
 */
export interface FirebaseAppConfig {
  apiKey?: string;
  authDomain?: string;
  projectId?: string;
  storageBucket?: string;
  messagingSenderId?: string;
  appId?: string;
  measurementId?: string;
  backendUrl: string; // URL of the custom backend
  // Add any other backend-specific config options here
}

/**
 * Represents a Firebase App.
 */
export interface FirebaseApp {
  name: string;
  options: FirebaseAppConfig;
}

export interface DocumentData {
  [field: string]: any;
}

export type WhereFilterOp =
  | '<'
  | '<='
  | '=='
  | '!='
  | '>='
  | '>'
  | 'array-contains'
  | 'in'
  | 'not-in'
  | 'array-contains-any';

export type OrderByDirection = 'desc' | 'asc';

export class FieldPath {
  private _internalPath: string[];
  constructor(...fieldNames: string[]) {
    if (fieldNames.length === 0) throw new Error('FieldPath constructor must receive at least one field name.');
    if (fieldNames.some(name => typeof name !== 'string' || name.length === 0)) {
      throw new Error('Field names must be non-empty strings.');
    }
    this._internalPath = fieldNames;
  }
  get_internalPath(): string[] { return this._internalPath; }
  toString(): string { return this._internalPath.join('.'); }
}

export interface DocumentChange<T = DocumentData> {
  readonly type: DocumentChangeType;
  readonly doc: QueryDocumentSnapshot<T>;
  readonly oldIndex: number;
  readonly newIndex: number;
}
export type DocumentChangeType = 'added' | 'modified' | 'removed';
export interface SnapshotListenOptions { readonly includeMetadataChanges?: boolean; }

// Forward declarations for interfaces used by Firestore
export interface DocumentReference<T = DocumentData> {
  readonly id: string;
  readonly path: string;
  readonly firestore: Firestore; // Should be this specific Firestore instance
  readonly parent: CollectionReference<T>; // or CollectionReferenceInterface
  collection(collectionPath: string): CollectionReference<DocumentData>; // or CollectionReferenceInterface
  get(): Promise<DocumentSnapshot<T>>;
  set(data: T): Promise<void>;
  update(data: Partial<T>): Promise<void>;
  delete(): Promise<void>;
  onSnapshot(observer: {
    next?: (snapshot: DocumentSnapshot<T>) => void;
    error?: (error: Error) => void;
  }): () => void;
}

export interface CollectionReference<T = DocumentData> extends Query<T> {
  readonly id: string;
  readonly path: string;
  readonly parent: DocumentReference<DocumentData> | null;
  doc(documentPath?: string): DocumentReference<T>;
  add(data: T): Promise<DocumentReference<T>>;
}

export interface Query<T = DocumentData> {
  readonly firestore: Firestore; // Should be this specific Firestore instance
  where(fieldPath: string | FieldPath, opStr: WhereFilterOp, value: any): Query<T>;
  orderBy(fieldPath: string | FieldPath, directionStr?: OrderByDirection): Query<T>;
  limit(limit: number): Query<T>;
  limitToLast(limitToLast: number): Query<T>;
  startAt(...fieldValues: any[]): Query<T>;
  startAt(snapshot: DocumentSnapshot<any>): Query<T>;
  startAfter(...fieldValues: any[]): Query<T>;
  startAfter(snapshot: DocumentSnapshot<any>): Query<T>;
  endBefore(...fieldValues: any[]): Query<T>;
  endBefore(snapshot: DocumentSnapshot<any>): Query<T>;
  endAt(...fieldValues: any[]): Query<T>;
  endAt(snapshot: DocumentSnapshot<any>): Query<T>;
  get(): Promise<QuerySnapshot<T>>;
  onSnapshot(observer: {
    next?: (snapshot: QuerySnapshot<T>) => void;
    error?: (error: Error) => void;
  }): () => void;
}

export interface DocumentSnapshot<T = DocumentData> {
  readonly id: string;
  readonly ref: DocumentReference<T>;
  readonly exists: boolean;
  data(): T | undefined;
  get(fieldPath: string | FieldPath): any;
}

export interface QueryDocumentSnapshot<T = DocumentData> extends DocumentSnapshot<T> {
  data(): T;
}

export interface QuerySnapshot<T = DocumentData> {
  readonly query: Query<T>;
  readonly docs: Array<QueryDocumentSnapshot<T>>;
  readonly empty: boolean;
  readonly size: number;
  docChanges(options?: SnapshotListenOptions): Array<DocumentChange<T>>;
  forEach(callback: (result: QueryDocumentSnapshot<T>, index: number) => void, thisArg?: any): void;
}

export interface WriteBatch {
  set<T>(documentRef: DocumentReference<T>, data: T): WriteBatch;
  update<T>(documentRef: DocumentReference<T>, data: Partial<T>): WriteBatch;
  delete(documentRef: DocumentReference<any>): WriteBatch;
  commit(): Promise<void>;
}

export interface Transaction {
  get<T>(documentRef: DocumentReference<T>): Promise<DocumentSnapshot<T>>;
  set<T>(documentRef: DocumentReference<T>, data: T): Transaction;
  update<T>(documentRef: DocumentReference<T>, data: Partial<T>): Transaction;
  delete(documentRef: DocumentReference<any>): Transaction;
}
// --- End of existing interfaces ---


/**
 * Represents the main Firestore service.
 * This is the interface users will interact with.
 */
export interface Firestore {
  readonly app: FirebaseApp;
  readonly backendClient: BackendClient; // Expose backend client for direct use if necessary (or keep internal)
  readonly realtimeManager: RealtimeManager; // Expose realtime manager

  collection<T = DocumentData>(collectionPath: string): CollectionReference<T>;
  doc<T = DocumentData>(documentPath: string): DocumentReference<T>;
  batch(): WriteBatch;
  runTransaction<T>(updateFunction: (transaction: Transaction) => Promise<T>): Promise<T>;
  // Add other methods like terminate, etc.
  terminate?(): Promise<void>; // Optional terminate method
}

/**
 * FirestoreService class implements the Firestore interface.
 */
class FirestoreService implements Firestore {
  readonly app: FirebaseApp;
  readonly backendClient: BackendClient;
  readonly realtimeManager: RealtimeManager;

  constructor(app: FirebaseApp) {
    this.app = app;
    // Initialize the backend client (e.g., RestBackendClient)
    this.backendClient = new RestBackendClient(app.options);

    // Initialize the RealtimeManager
    // The RealtimeManager might need a way to make authenticated WebSocket connections,
    // potentially using info from backendClient or appConfig.
    this.realtimeManager = new RealtimeManager(this, { backendUrl: app.options.backendUrl });

    console.log(`FirestoreService initialized for app: ${app.name}`);
  }

  collection<T = DocumentData>(collectionPath: string): CollectionReference<T> {
    if (!collectionPath) {
      throw new Error("Collection path cannot be empty.");
    }
    const pathSegments = collectionPath.split('/').filter(s => s.length > 0);
    if (pathSegments.length % 2 === 0) {
      // This should ideally not happen if called from valid DocRef.collection() or for root.
      // But good to be defensive. Or rely on CollectionReference constructor to validate.
      throw new Error(`Invalid collection path: "${collectionPath}". Path must have an odd number of segments.`);
    }

    let parentDocRef = null;
    if (pathSegments.length > 1) {
      const parentDocPath = pathSegments.slice(0, -1).join('/');
      // To create the parent DocumentReference, we need its parent collection, which could be null if parentDocPath is a root doc.
      // This gets a bit recursive if we want to build the full chain.
      // For now, the DocumentReference constructor needs a parent *CollectionReference*.
      // The parent collection of parentDocPath would be pathSegments.slice(0, -2).join('/')
      let grandParentCollectionPath = null;
      if (pathSegments.length > 2) {
          grandParentCollectionPath = pathSegments.slice(0, -2).join('/');
      }
      const parentCollectionForParentDoc = new CollectionReferenceClass<DocumentData>(this, grandParentCollectionPath || '', null); // Simplified parent for now
      parentDocRef = new DocumentReferenceClass<DocumentData>(this, parentDocPath, parentCollectionForParentDoc);
    }
    return new CollectionReferenceClass<T>(this, collectionPath, parentDocRef);
  }

  doc<T = DocumentData>(documentPath: string): DocumentReference<T> {
    if (!documentPath) {
      throw new Error("Document path cannot be empty.");
    }
    const pathSegments = documentPath.split('/').filter(s => s.length > 0);
    if (pathSegments.length % 2 !== 0) {
      throw new Error(`Invalid document path: "${documentPath}". Path must have an even number of segments.`);
    }

    // The DocumentReference constructor expects a parent CollectionReference.
    // We need to derive the parent collection path and create a parent ref.
    const docId = pathSegments.pop()!; // Last segment is doc ID
    const collectionPath = pathSegments.join('/');

    // Create a "dummy" parent collection reference. It's only used by DocumentReference
    // to know its parent collection's path and for its 'parent' property.
    const parentColRef = new CollectionReferenceClass<T>(this, collectionPath, null); // Parent of this collection is null or derived if deeper
                                                                                      // For simplicity, assuming top-level doc call here.
                                                                                      // A more robust solution would trace back the true parent if path is nested.
                                                                                      // However, Firestore's doc() can point to any doc directly.

    return new DocumentReferenceClass<T>(this, documentPath, parentColRef);
  }

  batch(): WriteBatch {
    console.warn('Firestore.batch() not yet implemented.');
    throw new Error('Firestore.batch() not yet implemented.');
    // return new WriteBatchImplementation(this);
  }

  async runTransaction<T>(updateFunction: (transaction: Transaction) => Promise<T>): Promise<T> {
    console.warn('Firestore.runTransaction() not yet implemented.');
    throw new Error('Firestore.runTransaction() not yet implemented.');
    // const transaction = new TransactionImplementation(this);
    // try {
    //   const result = await updateFunction(transaction);
    //   await transaction.commit(); // Internal commit for transaction operations
    //   return result;
    // } catch (error) {
    //   await transaction.rollback(); // Internal rollback
    //   throw error;
    // }
  }

  async terminate(): Promise<void> {
    console.log(`Terminating FirestoreService for app: ${this.app.name}`);
    this.realtimeManager.terminate();
    // Any other cleanup for backendClient if needed
  }
}

// --- Global stores and initialization functions ---
const apps = new Map<string, FirebaseApp>();
const defaultAppName = "[DEFAULT]";
const firestoreInstances = new Map<string, FirestoreService>(); // Store FirestoreService instances

export function initializeApp(config: FirebaseAppConfig, name: string = defaultAppName): FirebaseApp {
  if (apps.has(name)) {
    const existingApp = apps.get(name)!;
    // Firestore JS SDK v9 allows re-initialization with the same name and config.
    // For simplicity, let's stick to original behavior of throwing or returning existing.
    if (JSON.stringify(existingApp.options) !== JSON.stringify(config) && name === defaultAppName) {
         console.warn(`Firebase app named "${name}" already exists but with different options. Returning existing app.`);
    } else if (name !== defaultAppName && apps.has(name)) {
        return existingApp; // Return existing named app
    } else if (name === defaultAppName && apps.has(name)) {
        return existingApp; // Return existing default app
    }
     // If we allow re-initialization, we might need to update the app in the map
     // and potentially re-initialize services like Firestore if config changed.
  }

  const app: FirebaseApp = { name, options: config };
  apps.set(name, app);
  return app;
}

export function getFirestore(app?: FirebaseApp): Firestore {
  const appToCheck = app || apps.get(defaultAppName);

  if (!appToCheck) {
    throw new Error("Firebase app has not been initialized. Call initializeApp() first.");
  }

  if (firestoreInstances.has(appToCheck.name)) {
    return firestoreInstances.get(appToCheck.name)!;
  }

  const firestoreService = new FirestoreService(appToCheck);
  firestoreInstances.set(appToCheck.name, firestoreService);
  return firestoreService;
}

export function deleteApp(app: FirebaseApp): Promise<void> {
    return new Promise((resolve) => {
        const appName = app.name;
        if (apps.has(appName)) {
            const firestoreInstance = firestoreInstances.get(appName);
            if (firestoreInstance) {
                firestoreInstance.terminate?.(); // Call terminate if it exists
                firestoreInstances.delete(appName);
            }
            apps.delete(appName);
            console.log(`App ${appName} deleted.`);
        }
        resolve();
    });
}


console.log('firestore.ts (with FirestoreService) loaded');
