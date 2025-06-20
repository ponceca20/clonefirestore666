import {
  Firestore, // This is the Interface
  CollectionReference as CollectionReferenceInterface, // Interface
  DocumentReference as DocumentReferenceInterface, // Import the interface
  DocumentSnapshot as DocumentSnapshotInterface, // Import the interface for implements
  DocumentData,
  FieldPath
  // QuerySnapshot, Query are imported via Firestore interface if needed by methods here
} from './firestore';
// Import FirestoreService if needed for type casting, or rely on Firestore interface
// import { FirestoreService } from './firestore';
import { BackendDocumentResponse } from './backend_client';
import { CollectionReference as ActualCollectionRefClass } from './collection'; // Actual CollectionReference class for parent type

// DocumentSnapshot and QueryDocumentSnapshot classes remain largely the same,
// but their constructors will receive the actual FirestoreService instance.

export class DocumentSnapshot<T = DocumentData> implements DocumentSnapshotInterface<T> {
  readonly id: string;
  readonly ref: DocumentReference<T>; // This will be DocumentReferenceClass instance
  private readonly _firestore: Firestore; // Instance of FirestoreService
  private _documentData?: T;
  private _exists: boolean;

  constructor(firestore: Firestore, ref: DocumentReference<T>, id: string, data?: T, exists: boolean = false) {
    this._firestore = firestore;
    this.ref = ref;
    this.id = id;
    this._documentData = data;
    this._exists = exists;
  }

  get exists(): boolean {
    return this._exists;
  }

  data(): T | undefined {
    return this._exists ? this._documentData : undefined;
  }

  get(fieldPath: string | FieldPath): any {
    if (!this._exists || !this._documentData) {
      return undefined;
    }
    const path = typeof fieldPath === 'string' ? fieldPath.split('.') : fieldPath.get_internalPath();
    let current: any = this._documentData;
    for (const segment of path) {
      if (current && typeof current === 'object' && segment in current) {
        current = current[segment];
      } else {
        return undefined;
      }
    }
    return current;
  }
}

import { QueryDocumentSnapshot as QueryDocumentSnapshotInterface } from './firestore'; // Import for implements

export class QueryDocumentSnapshot<T = DocumentData> extends DocumentSnapshot<T> implements QueryDocumentSnapshotInterface<T> {
  constructor(firestore: Firestore, ref: DocumentReference<T>, id: string, data: T) {
    super(firestore, ref, id, data, true);
  }

  data(): T {
    return super.data() as T;
  }
}

export class DocumentReference<T = DocumentData> implements DocumentReferenceInterface<T> {
  readonly id: string;
  readonly firestore: Firestore; // This will be an instance of FirestoreService
  readonly path: string;
  // Ensure parent is typed to the actual CollectionReference class if methods are called on it,
  // or ensure CollectionReferenceInterface is sufficient.
  readonly parent: CollectionReferenceInterface<T>;


  constructor(firestore: Firestore, path: string, parent: CollectionReferenceInterface<T>) {
    this.firestore = firestore; // Instance of FirestoreService
    this.path = path;
    this.parent = parent;
    const parts = path.split('/');
    this.id = parts[parts.length - 1];
  }

  collection<U = DocumentData>(collectionPath: string): CollectionReferenceInterface<U> {
    if (!collectionPath) {
        throw new Error("A collection path cannot be empty.");
    }
    const newPath = `${this.path}/${collectionPath}`;
    // Delegate to Firestore instance's collection method, which uses the actual CollectionReferenceClass
    return this.firestore.collection(newPath) as CollectionReferenceInterface<U>;
  }

  async get(): Promise<DocumentSnapshot<T>> {
    console.log(`DocumentReference.get() for: ${this.path}`);
    try {
      const response: BackendDocumentResponse = await this.firestore.backendClient.getDocument(this.path);
      return new DocumentSnapshot<T>(
        this.firestore,
        this, // Pass this DocumentReference instance itself as the ref
        this.id, // or response.id if backend can change it (should not for get)
        response.data as T,
        response.exists
      );
    } catch (error) {
      // TODO: Map backend errors to Firestore-like errors
      console.error(`Error in DocumentReference.get() for ${this.path}:`, error);
      throw error; // Re-throw for now
    }
  }

  async set(data: T /*, options?: SetOptions*/): Promise<void> {
    console.log(`DocumentReference.set() for: ${this.path} with data:`, data);
    // TODO: Handle SetOptions (merge, mergeFields) if implemented.
    // This would involve sending options to backendClient or performing a get-then-merge locally (less ideal).
    try {
      await this.firestore.backendClient.setDocument(this.path, data as { [key: string]: any });
    } catch (error) {
      console.error(`Error in DocumentReference.set() for ${this.path}:`, error);
      throw error;
    }
  }

  async update(data: Partial<T> /* | UpdateData */): Promise<void> {
    console.log(`DocumentReference.update() for: ${this.path} with data:`, data);
    // TODO: Handle dot-notation for field paths if backend doesn't support it directly.
    // This might involve transforming the data object.
    // Firestore's update requires the document to exist.
    // The backendClient.updateDocument should ideally enforce this.
    try {
      await this.firestore.backendClient.updateDocument(this.path, data as { [key: string]: any });
    } catch (error) {
      console.error(`Error in DocumentReference.update() for ${this.path}:`, error);
      throw error;
    }
  }

  async delete(): Promise<void> {
    console.log(`DocumentReference.delete() for: ${this.path}`);
    try {
      await this.firestore.backendClient.deleteDocument(this.path);
    } catch (error) {
      console.error(`Error in DocumentReference.delete() for ${this.path}:`, error);
      throw error;
    }
  }

  onSnapshot(observer: {
    next?: (snapshot: DocumentSnapshot<T>) => void;
    error?: (error: Error) => void;
  }): () => void {
    console.log(`DocumentReference.onSnapshot() for: ${this.path}`);
    // Delegate to RealtimeManager via FirestoreService
    // The RealtimeManager will handle transforming backend messages to DocumentSnapshots
    return this.firestore.realtimeManager.listenToDocument(this, observer);
  }
}

console.log('document.ts (with implemented CRUD) loaded');
