// Main entry point for the SDK Core

// Firestore main functions and types
export {
  initializeApp,
  getFirestore,
  deleteApp, // Add deleteApp here
  // Interfaces
  FirebaseAppConfig,
  FirebaseApp,
  Firestore,
  WriteBatch,
  Transaction,
  // Helper Types
  DocumentData,
  FieldPath, // Export the class for type checking and advanced usage
  WhereFilterOp,
  OrderByDirection,
  DocumentChangeType,
  SnapshotListenOptions,
  DocumentChange // Export DocumentChange directly from firestore.ts
} from './firestore';

// Document related
export {
  DocumentReference,
  DocumentSnapshot,
  QueryDocumentSnapshot
} from './document'; // DocumentReference is now a class

// Collection related
export {
  CollectionReference // CollectionReference is now a class
} from './collection';

// Query related
export {
  Query, // Query is now a class
  QuerySnapshot // QuerySnapshot is now a class
  // DocumentChange is now exported from firestore.ts
  // QueryConstraint is an internal detail for now, not typically part of public API
} from './query';

// Realtime (optional, if RealtimeManager is meant to be directly used, or internal)
// For now, onSnapshot methods on DocRef and Query will use it internally.
// export { RealtimeManager } from './realtime';

console.log('index.ts loaded, SDK Core exports defined.');
