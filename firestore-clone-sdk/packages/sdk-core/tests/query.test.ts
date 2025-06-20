import {
  initializeApp,
  getFirestore,
  deleteApp,
  FirebaseApp,
  FirebaseAppConfig,
  Firestore,
  CollectionReference,
  QuerySnapshot,
  Query
} from '../src/index';
import { RestBackendClient, BackendQueryResponse, BackendDocumentResponse } from '../src/backend_client';

// Mock RestBackendClient as done in document.test.ts
jest.mock('../src/backend_client', () => {
  const originalModule = jest.requireActual('../src/backend_client');
  return {
    ...originalModule,
    RestBackendClient: jest.fn().mockImplementation(() => ({
      getDocument: jest.fn(),
      setDocument: jest.fn(),
      updateDocument: jest.fn(),
      deleteDocument: jest.fn(),
      executeQuery: jest.fn(),
      connectRealtime: jest.fn(),
      sendRealtimeMessage: jest.fn(),
      disconnectRealtime: jest.fn(),
      setAuthToken: jest.fn(),
    })),
  };
});

describe('Query Operations', () => {
  let app: FirebaseApp;
  let firestore: Firestore;
  let mockBackendClientInstance: jest.Mocked<RestBackendClient>;

  const defaultConfig: FirebaseAppConfig = {
    backendUrl: 'http://testhost/api',
    projectId: 'test-query-ops',
  };

  beforeEach(() => {
    app = initializeApp(defaultConfig);
    firestore = getFirestore(app);
    mockBackendClientInstance = firestore.backendClient as jest.Mocked<RestBackendClient>;

    mockBackendClientInstance.executeQuery.mockReset();
  });

  afterEach(async () => {
    if (app) {
      await deleteApp(app);
    }
  });

  it('get() on a CollectionReference should call backendClient.executeQuery and return QuerySnapshot', async () => {
    const collRef = firestore.collection('products');
    const backendResponse: BackendQueryResponse = {
      queryPath: 'products',
      docs: [
        { id: 'prod1', path: 'products/prod1', data: { name: 'Laptop', price: 1200 } },
        { id: 'prod2', path: 'products/prod2', data: { name: 'Mouse', price: 25 } },
      ],
    };

    mockBackendClientInstance.executeQuery.mockResolvedValue(backendResponse);

    const snapshot = await collRef.get(); // Use type inference

    expect(mockBackendClientInstance.executeQuery).toHaveBeenCalledTimes(1);
    // First arg is path, second is constraints array. For a direct collection.get(), constraints is empty.
    expect(mockBackendClientInstance.executeQuery).toHaveBeenCalledWith('products', []);

    expect(snapshot.empty).toBe(false);
    expect(snapshot.size).toBe(2);
    expect(snapshot.docs.length).toBe(2);

    expect(snapshot.docs[0].id).toBe('prod1');
    expect(snapshot.docs[0].data()).toEqual({ name: 'Laptop', price: 1200 });
    expect(snapshot.docs[0].ref.path).toBe('products/prod1');

    expect(snapshot.docs[1].id).toBe('prod2');
    expect(snapshot.docs[1].data()).toEqual({ name: 'Mouse', price: 25 });
    expect(snapshot.docs[1].ref.path).toBe('products/prod2');

    expect(snapshot.query).toBe(collRef); // The query that generated this snapshot
  });

  it('get() on a Query with where clause should call backendClient.executeQuery with constraints', async () => {
    const collRef = firestore.collection('orders');
    const query = collRef.where('status', '==', 'pending');

    const backendResponse: BackendQueryResponse = {
      queryPath: 'orders',
      docs: [
        { id: 'order1', path: 'orders/order1', data: { item: 'Book', status: 'pending' } },
      ],
    };
    mockBackendClientInstance.executeQuery.mockResolvedValue(backendResponse);

    const snapshot = await query.get();

    expect(mockBackendClientInstance.executeQuery).toHaveBeenCalledTimes(1);
    const expectedConstraints = [
      { type: 'where', fieldPath: 'status', opStr: '==', value: 'pending' }
    ];
    expect(mockBackendClientInstance.executeQuery).toHaveBeenCalledWith('orders', expect.arrayContaining(expectedConstraints.map(c => expect.objectContaining(c))));

    expect(snapshot.size).toBe(1);
    expect(snapshot.docs[0].id).toBe('order1');
    expect(snapshot.docs[0].data()).toEqual({ item: 'Book', status: 'pending' });
  });

  it('get() on a Query with orderBy clause should call backendClient.executeQuery with constraints', async () => {
    const collRef = firestore.collection('scores');
    const query = collRef.orderBy('points', 'desc');

    const backendResponse: BackendQueryResponse = {
      queryPath: 'scores',
      docs: [
        { id: 'score1', path: 'scores/score1', data: { player: 'Alice', points: 100 } },
        { id: 'score2', path: 'scores/score2', data: { player: 'Bob', points: 90 } },
      ],
    };
    mockBackendClientInstance.executeQuery.mockResolvedValue(backendResponse);

    await query.get();

    const expectedConstraints = [
      { type: 'orderBy', fieldPath: 'points', directionStr: 'desc' }
    ];
    expect(mockBackendClientInstance.executeQuery).toHaveBeenCalledWith('scores', expect.arrayContaining(expectedConstraints.map(c => expect.objectContaining(c))));
  });

  it('get() on a Query with limit clause should call backendClient.executeQuery with constraints', async () => {
    const collRef = firestore.collection('items');
    const query = collRef.limit(5);

    mockBackendClientInstance.executeQuery.mockResolvedValue({ queryPath: 'items', docs: [] }); // Empty response is fine for this check

    await query.get();

    const expectedConstraints = [
      { type: 'limit', limit: 5 }
    ];
    expect(mockBackendClientInstance.executeQuery).toHaveBeenCalledWith('items', expect.arrayContaining(expectedConstraints.map(c => expect.objectContaining(c))));
  });

  it('get() should return an empty QuerySnapshot for no results', async () => {
    const collRef = firestore.collection('emptyCollection');
    const backendResponse: BackendQueryResponse = {
      queryPath: 'emptyCollection',
      docs: [], // No documents returned
    };

    mockBackendClientInstance.executeQuery.mockResolvedValue(backendResponse);

    const snapshot = await collRef.get();

    expect(snapshot.empty).toBe(true);
    expect(snapshot.size).toBe(0);
    expect(snapshot.docs.length).toBe(0);
  });

  it('Query methods should be chainable and build up constraints', async () => {
    const collRef = firestore.collection('events');
    const query = collRef.where('type', '==', 'conference')
                         .orderBy('date', 'asc')
                         .limit(10);

    mockBackendClientInstance.executeQuery.mockResolvedValue({ queryPath: 'events', docs: [] });
    await query.get();

    const expectedConstraints = [
      { type: 'where', fieldPath: 'type', opStr: '==', value: 'conference' },
      { type: 'orderBy', fieldPath: 'date', directionStr: 'asc' },
      { type: 'limit', limit: 10 }
    ];
    // Check if all expected constraints are present in the call to executeQuery
    // The order might not be guaranteed by expect.arrayContaining if the internal _constraints array order changes.
    // A more robust check might involve checking each constraint individually if order is not fixed.
    expect(mockBackendClientInstance.executeQuery).toHaveBeenCalledWith('events',
      expect.arrayContaining(expectedConstraints.map(c => expect.objectContaining(c)))
    );
    expect(mockBackendClientInstance.executeQuery.mock.calls[0][1].length).toBe(expectedConstraints.length);

  });

  it('get() should throw error if backendClient.executeQuery fails', async () => {
    const collRef = firestore.collection('failingQuery');
    const backendError = new Error("Backend Query Error");

    mockBackendClientInstance.executeQuery.mockRejectedValue(backendError);

    await expect(collRef.get()).rejects.toThrow("Backend Query Error");
  });

});
