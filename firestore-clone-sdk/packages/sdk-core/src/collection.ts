import {
  Firestore, // Interface
  DocumentReference as DocumentReferenceInterface, // Interface
  CollectionReference as CollectionReferenceInterface, // Interface
  DocumentData
  // Query-related types are now primarily in query.ts or imported via Firestore interface
} from './firestore';
import { DocumentReference as ActualDocRefClass } from './document'; // Actual DocumentReference class
import { Query as QueryClass } from './query'; // Actual Query class

export class CollectionReference<T = DocumentData> extends QueryClass<T> implements CollectionReferenceInterface<T> {
  readonly id: string;
  // 'path' is inherited from QueryClass as protected _path.
  // We can add a public getter if CollectionReferenceInterface demands a public path.
  // For now, assume _path in QueryClass is accessible enough or add public getter.
  public get path(): string {
    return this._path;
  }
  readonly parent: DocumentReferenceInterface<DocumentData> | null;

  constructor(firestore: Firestore, path: string, parent: DocumentReferenceInterface<DocumentData> | null = null) {
    super(firestore, path); // Call QueryClass constructor
    this.parent = parent;
    const parts = path.split('/');
    this.id = parts[parts.length - 1];
  }

  doc(documentPath?: string): DocumentReferenceInterface<T> {
    if (documentPath === '') {
        throw new Error("Document path cannot be an empty string.");
    }

    let id = documentPath;
    if (!documentPath) {
      const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
      id = Array(20).fill(null).map(() => chars.charAt(Math.floor(Math.random() * chars.length))).join('');
    }

    const newPath = `${this.path}/${id}`;
    // 'this' is the CollectionReference instance, which serves as the parent for the new DocumentReference.
    return new ActualDocRefClass<T>(this.firestore, newPath, this);
  }

  async add(data: T): Promise<DocumentReferenceInterface<T>> {
    console.log(`CollectionReference.add() for: ${this.path} with data:`, data);
    // Create a new DocumentReference with an auto-generated ID.
    const newDocRef = this.doc(); // This now returns ActualDocRefClass instance due to above change

    // Call set() on the new DocumentReference.
    // The set() method in ActualDocRefClass is already implemented to use the backendClient.
    await (newDocRef as ActualDocRefClass<T>).set(data);

    return newDocRef;
  }

  // where, orderBy, limit, get, onSnapshot etc. are inherited from QueryClass<T>
}

console.log('collection.ts (with implemented add) loaded');
