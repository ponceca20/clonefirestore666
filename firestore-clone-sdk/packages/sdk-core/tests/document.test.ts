import { initializeApp, getFirestore, deleteApp, FirebaseApp, FirebaseAppConfig, Firestore, DocumentReference, DocumentSnapshot } from '../src/index';
import { RestBackendClient, BackendDocumentResponse } from '../src/backend_client'; // To mock its methods

// Mock the RestBackendClient
jest.mock('../src/backend_client', () => {
  // Important: We need to get the original module and mock specific methods,
  // but also keep the class constructor and other parts intact if they are used.
  // Or, provide a full mock implementation.
  const originalModule = jest.requireActual('../src/backend_client');
  return {
    ...originalModule, // Keep other exports like BackendDocumentResponse etc.
    RestBackendClient: jest.fn().mockImplementation(() => ({
      getDocument: jest.fn(),
      setDocument: jest.fn(),
      updateDocument: jest.fn(),
      deleteDocument: jest.fn(),
      executeQuery: jest.fn(),
      // Mock WebSocket methods if they were to be called by tested logic
      connectRealtime: jest.fn(),
      sendRealtimeMessage: jest.fn(),
      disconnectRealtime: jest.fn(),
      setAuthToken: jest.fn(), // if used
    })),
  };
});


describe('DocumentReference Operations', () => {
  let app: FirebaseApp;
  let firestore: Firestore;
  let mockBackendClientInstance: jest.Mocked<RestBackendClient>;

  const defaultConfig: FirebaseAppConfig = {
    backendUrl: 'http://testhost/api',
    projectId: 'test-doc-ops',
  };

  beforeEach(() => {
    // Clear any previous mock implementations or call counts
    // This is important because RestBackendClient is a mocked constructor.
    // We need to get the instance that FirestoreService will create.

    app = initializeApp(defaultConfig);
    firestore = getFirestore(app);

    // The FirestoreService constructor creates an instance of RestBackendClient.
    // We need to access that specific mocked instance.
    // Since FirestoreService holds backendClient as a public readonly property:
    mockBackendClientInstance = firestore.backendClient as jest.Mocked<RestBackendClient>;

    // Reset mock calls for each test
    mockBackendClientInstance.getDocument.mockReset();
    mockBackendClientInstance.setDocument.mockReset();
    mockBackendClientInstance.updateDocument.mockReset();
    mockBackendClientInstance.deleteDocument.mockReset();
  });

  afterEach(async () => {
    if (app) {
      await deleteApp(app);
    }
  });

  it('set() should call backendClient.setDocument with correct path and data', async () => {
    const docRef = firestore.doc('users/testUser');
    const userData = { name: 'Test User', email: 'test@example.com' };

    // Mock backendClient.setDocument to resolve successfully
    mockBackendClientInstance.setDocument.mockResolvedValue(Promise.resolve(undefined));

    await docRef.set(userData);

    expect(mockBackendClientInstance.setDocument).toHaveBeenCalledTimes(1);
    expect(mockBackendClientInstance.setDocument).toHaveBeenCalledWith('users/testUser', userData);
  });

  it('get() should call backendClient.getDocument and return a DocumentSnapshot', async () => {
    const docRef = firestore.doc('users/existingUser');
    const backendResponse: BackendDocumentResponse = {
      id: 'existingUser',
      path: 'users/existingUser',
      exists: true,
      data: { name: 'Existing User', role: 'reader' },
    };

    mockBackendClientInstance.getDocument.mockResolvedValue(backendResponse);

    const snapshot = await docRef.get(); // Use type inference

    expect(mockBackendClientInstance.getDocument).toHaveBeenCalledTimes(1);
    expect(mockBackendClientInstance.getDocument).toHaveBeenCalledWith('users/existingUser');

    expect(snapshot.exists).toBe(true);
    expect(snapshot.id).toBe('existingUser');
    expect(snapshot.data()).toEqual({ name: 'Existing User', role: 'reader' });
    expect(snapshot.ref.path).toBe(docRef.path);
  });

  it('get() should return a DocumentSnapshot with exists=false for non-existent document', async () => {
    const docRef = firestore.doc('users/nonExistentUser');
    const backendResponse: BackendDocumentResponse = {
      id: 'nonExistentUser',
      path: 'users/nonExistentUser',
      exists: false,
      // data will be undefined
    };

    mockBackendClientInstance.getDocument.mockResolvedValue(backendResponse);

    const snapshot = await docRef.get();

    expect(mockBackendClientInstance.getDocument).toHaveBeenCalledWith('users/nonExistentUser');
    expect(snapshot.exists).toBe(false);
    expect(snapshot.data()).toBeUndefined();
  });

  it('update() should call backendClient.updateDocument with correct path and data', async () => {
    const docRef = firestore.doc('items/item123');
    const itemUpdate = { price: 99.99, status: 'in-stock' };

    mockBackendClientInstance.updateDocument.mockResolvedValue(Promise.resolve(undefined));

    await docRef.update(itemUpdate);

    expect(mockBackendClientInstance.updateDocument).toHaveBeenCalledTimes(1);
    expect(mockBackendClientInstance.updateDocument).toHaveBeenCalledWith('items/item123', itemUpdate);
  });

  it('delete() should call backendClient.deleteDocument with correct path', async () => {
    const docRef = firestore.doc('logs/logEntry789');

    mockBackendClientInstance.deleteDocument.mockResolvedValue(Promise.resolve(undefined));

    await docRef.delete();

    expect(mockBackendClientInstance.deleteDocument).toHaveBeenCalledTimes(1);
    expect(mockBackendClientInstance.deleteDocument).toHaveBeenCalledWith('logs/logEntry789');
  });

  // Test for error handling from backend client
  it('set() should throw error if backendClient.setDocument fails', async () => {
    const docRef = firestore.doc('users/failUser');
    const userData = { name: 'Fail User' };
    const backendError = new Error("Backend Set Error");

    mockBackendClientInstance.setDocument.mockRejectedValue(backendError);

    await expect(docRef.set(userData)).rejects.toThrow("Backend Set Error");
    expect(mockBackendClientInstance.setDocument).toHaveBeenCalledWith('users/failUser', userData);
  });

  it('get() should throw error if backendClient.getDocument fails', async () => {
    const docRef = firestore.doc('items/failItem');
    const backendError = new Error("Backend Get Error");

    mockBackendClientInstance.getDocument.mockRejectedValue(backendError);

    await expect(docRef.get()).rejects.toThrow("Backend Get Error");
    expect(mockBackendClientInstance.getDocument).toHaveBeenCalledWith('items/failItem');
  });

});

// To run: npm test
