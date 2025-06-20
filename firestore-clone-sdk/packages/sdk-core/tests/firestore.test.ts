import { initializeApp, getFirestore, deleteApp, FirebaseApp, FirebaseAppConfig, Firestore } from '../src/index';

describe('Firestore Initialization', () => {
  const defaultConfig: FirebaseAppConfig = {
    backendUrl: 'http://localhost:8080/api',
    projectId: 'test-project',
  };

  afterEach(async () => {
    try {
      const defaultApp = initializeApp(defaultConfig);
      await deleteApp(defaultApp);
    } catch (e) { /* Ignore */ }
    try {
        const namedApp = initializeApp(defaultConfig, "myNamedApp");
        await deleteApp(namedApp);
    } catch(e) { /* Ignore */ }
    try {
        const refTestApp = initializeApp(defaultConfig, "refTestApp");
        await deleteApp(refTestApp);
    } catch(e) { /* Ignore */ }
  });

  it('should initialize the default app', () => {
    const app = initializeApp(defaultConfig);
    expect(app).toBeDefined();
    expect(app.name).toBe('[DEFAULT]');
    expect(app.options).toEqual(defaultConfig);
  });

  it('should return the same default app instance if initialized again with same config', () => {
    const app1 = initializeApp(defaultConfig);
    const app2 = initializeApp(defaultConfig);
    expect(app1).toBe(app2);
  });

  it('should initialize a named app', () => {
    const appName = 'myNamedApp';
    const app = initializeApp(defaultConfig, appName);
    expect(app).toBeDefined();
    expect(app.name).toBe(appName);
    expect(app.options).toEqual(defaultConfig);
  });

  it('should return the same named app instance if initialized again', () => {
    const appName = 'myNamedApp';
    const app1 = initializeApp(defaultConfig, appName);
    const app2 = initializeApp(defaultConfig, appName);
    expect(app1).toBe(app2);
  });

  it('should get a Firestore instance for the default app', () => {
    const app = initializeApp(defaultConfig);
    const firestore = getFirestore(app);
    expect(firestore).toBeDefined();
    expect(firestore.app).toBe(app);
    expect(firestore.collection).toBeInstanceOf(Function);
    expect(firestore.doc).toBeInstanceOf(Function);
  });

  it('should get the same Firestore instance for the same app', () => {
    const app = initializeApp(defaultConfig);
    const firestore1 = getFirestore(app);
    const firestore2 = getFirestore(app);
    expect(firestore1).toBe(firestore2);
  });

  it('should get a Firestore instance using the default app if no app is provided to getFirestore', () => {
    initializeApp(defaultConfig);
    const firestore = getFirestore();
    expect(firestore).toBeDefined();
    expect(firestore.app.name).toBe('[DEFAULT]');
  });

  it('should throw if getFirestore is called before initializeApp for default app', () => {
    // Relies on afterEach to clean up. More robust: ensure specific cleanup for this test.
    try {
        const app = initializeApp(defaultConfig); // Try to get default app
        if (app && app.name === '[DEFAULT]') { // Check if it's indeed the default app
             deleteApp(app); // Delete if found
        }
    } catch (e) { /* Fine if it wasn't initialized or already deleted */ }
    expect(() => getFirestore()).toThrow('Firebase app has not been initialized.');
  });

   it('deleteApp should remove the app and its firestore instance', async () => {
    const app = initializeApp(defaultConfig);
    const firestore = getFirestore(app);
    expect(firestore).toBeDefined();

    await deleteApp(app);

    const newAppInstance = initializeApp(defaultConfig);
    const newFirestore = getFirestore(newAppInstance);
    expect(newFirestore).toBeDefined();
    expect(newFirestore).not.toBe(firestore);
  });
});

describe('Firestore Reference Creation', () => {
  let app: FirebaseApp;
  let firestore: Firestore;
  const refTestConfig: FirebaseAppConfig = {
    backendUrl: 'http://localhost:8080/api',
    projectId: 'test-project-refs',
  };

  beforeEach(() => {
    app = initializeApp(refTestConfig, "refTestApp");
    firestore = getFirestore(app);
  });

  afterEach(async () => {
     if (app) await deleteApp(app);
  });

  it('should create a valid DocumentReference', () => {
    const docRef = firestore.doc('users/userId');
    expect(docRef).toBeDefined();
    expect(docRef.path).toBe('users/userId');
    expect(docRef.id).toBe('userId');
    expect(docRef.firestore).toBe(firestore);
  });

  it('should create a valid CollectionReference', () => {
    const collRef = firestore.collection('users');
    expect(collRef).toBeDefined();
    expect(collRef.path).toBe('users');
    expect(collRef.id).toBe('users');
    expect(collRef.firestore).toBe(firestore);
  });

  it('DocumentReference.collection() should create a valid sub-collection reference', () => {
    const docRef = firestore.doc('users/userId');
    const subCollRef = docRef.collection('posts');
    expect(subCollRef).toBeDefined();
    expect(subCollRef.path).toBe('users/userId/posts');
    expect(subCollRef.id).toBe('posts');
    expect(subCollRef.firestore).toBe(firestore);
    expect(subCollRef.parent?.path).toBe('users/userId');
  });

  it('CollectionReference.doc() should create a valid document reference', () => {
    const collRef = firestore.collection('users');
    const docRef = collRef.doc('aNewDoc');
    expect(docRef.path).toBe('users/aNewDoc');
    expect(docRef.id).toBe('aNewDoc');
    expect(docRef.parent?.path).toBe('users');

    const autoIdDocRef = collRef.doc();
    expect(autoIdDocRef.path).toMatch(/^users\/[A-Za-z0-9]{20}$/);
    expect(autoIdDocRef.id.length).toBe(20);
    expect(autoIdDocRef.parent?.path).toBe('users');
  });

  it('should throw error for invalid document path (odd segments)', () => {
    expect(() => firestore.doc('users')).toThrow('Invalid document path: "users". Path must have an even number of segments.');
    expect(() => firestore.doc('users/userId/posts')).toThrow('Invalid document path: "users/userId/posts". Path must have an even number of segments.');
  });

  it('should throw error for invalid collection path (empty path)', () => {
    expect(() => firestore.collection('')).toThrow("Collection path cannot be empty.");
  });
});
