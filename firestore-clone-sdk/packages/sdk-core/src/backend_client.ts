import { FirebaseAppConfig } from './firestore'; // For backendUrl
import { QueryConstraint } from './query'; // For serializing query constraints

// Define the structure for backend responses (these are examples)
export interface BackendDocumentResponse {
  id: string;
  path: string;
  exists: boolean;
  data?: { [key: string]: any };
  error?: string;
}

export interface BackendQueryResponse {
  queryPath: string;
  docs: Array<{ id: string; path: string; data: { [key: string]: any } }>; // Simplified doc structure
  error?: string;
}

export interface BackendError {
  code: string; // e.g., 'permission-denied', 'not-found', 'internal'
  message: string;
}

/**
 * Interface for the backend communication client.
 * This abstracts the actual HTTP/WebSocket calls.
 */
export interface BackendClient {
  getDocument(path: string): Promise<BackendDocumentResponse>;
  setDocument(path: string, data: { [key: string]: any } /*, options?: SetOptions */): Promise<void>;
  updateDocument(path: string, data: { [key: string]: any } /*, options?: UpdateOptions */): Promise<void>;
  deleteDocument(path: string): Promise<void>;
  executeQuery(collectionPath: string, constraints: QueryConstraint[]): Promise<BackendQueryResponse>;

  // WebSocket related methods (could be part of a separate RealtimeBackendClient)
  connectRealtime(
    onOpen: () => void,
    onMessage: (message: any) => void,
    onError: (error: any) => void,
    onClose: () => void
  ): void;
  sendRealtimeMessage(message: any): void;
  disconnectRealtime(): void;
}

/**
 * A basic REST-based backend client using fetch.
 * This needs to be configured with the actual backend API details.
 */
export class RestBackendClient implements BackendClient {
  private backendUrl: string;
  private appConfig: FirebaseAppConfig;
  private authToken: string | null = null; // Placeholder for auth token

  constructor(appConfig: FirebaseAppConfig) {
    this.appConfig = appConfig;
    if (!appConfig.backendUrl) {
      throw new Error("Backend URL is not configured in FirebaseAppConfig.");
    }
    this.backendUrl = appConfig.backendUrl.replace(/\/$/, ''); // Remove trailing slash if any
    console.log(`RestBackendClient initialized for URL: ${this.backendUrl}`);
  }

  // Placeholder for setting auth token if your backend uses it
  public setAuthToken(token: string | null): void {
    this.authToken = token;
    console.log(`Auth token ${token ? 'set' : 'cleared'} for RestBackendClient`);
  }

  private async _fetch(endpoint: string, options: RequestInit = {}): Promise<any> {
    const url = `${this.backendUrl}${endpoint}`;
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    } as Record<string, string>; // Cast to allow arbitrary string keys
    if (this.authToken) {
      headers['Authorization'] = `Bearer ${this.authToken}`;
    }

    console.log(`Fetching: ${options.method || 'GET'} ${url}`, options.body ? { body: options.body } : {});

    try {
      const response = await fetch(url, { ...options, headers });
      if (!response.ok) {
        // Attempt to parse error from backend
        let errorData: BackendError = { code: 'unknown', message: response.statusText };
        try {
          errorData = await response.json();
        } catch (e) {
          // Ignore if error response is not JSON
        }
        console.error(`Backend error: ${response.status}`, errorData);
        // TODO: Map this to a Firestore-like error object
        throw new Error(`Backend request failed: ${response.status} ${errorData.message} (Code: ${errorData.code})`);
      }
      if (response.status === 204) { // No Content
        return undefined;
      }
      return await response.json();
    } catch (error) {
      console.error(`Fetch error for ${url}:`, error);
      // Ensure it's re-thrown so calling method can handle
      throw error;
    }
  }

  async getDocument(path: string): Promise<BackendDocumentResponse> {
    console.log(`RestBackendClient.getDocument: ${path}`);
    // **TODO: Adapt this endpoint to your backend's API**
    // Example: GET /api/docs/{collection}/{docId} or /api/docs/{path}
    // Ensure your backend can return a structure matching BackendDocumentResponse.
    // The path might need to be encoded if it contains special characters.
    const endpoint = `/documents/${encodeURIComponent(path)}`;
    return this._fetch(endpoint, { method: 'GET' });
  }

  async setDocument(path: string, data: { [key: string]: any }): Promise<void> {
    console.log(`RestBackendClient.setDocument: ${path}`, data);
    // **TODO: Adapt this endpoint to your backend's API**
    // Example: PUT /api/docs/{path} or POST /api/docs/{path}
    const endpoint = `/documents/${encodeURIComponent(path)}`;
    await this._fetch(endpoint, {
      method: 'PUT', // Or POST, depending on your backend (PUT is often idempotent)
      body: JSON.stringify(data),
    });
  }

  async updateDocument(path: string, data: { [key: string]: any }): Promise<void> {
    console.log(`RestBackendClient.updateDocument: ${path}`, data);
    // **TODO: Adapt this endpoint to your backend's API**
    // Example: PATCH /api/docs/{path} or POST /api/docs/{path}/update
    const endpoint = `/documents/${encodeURIComponent(path)}`;
    await this._fetch(endpoint, {
      method: 'PATCH', // PATCH is typically used for partial updates
      body: JSON.stringify(data),
    });
  }

  async deleteDocument(path: string): Promise<void> {
    console.log(`RestBackendClient.deleteDocument: ${path}`);
    // **TODO: Adapt this endpoint to your backend's API**
    // Example: DELETE /api/docs/{path}
    const endpoint = `/documents/${encodeURIComponent(path)}`;
    await this._fetch(endpoint, { method: 'DELETE' });
  }

  async executeQuery(collectionPath: string, constraints: QueryConstraint[]): Promise<BackendQueryResponse> {
    console.log(`RestBackendClient.executeQuery for collection: ${collectionPath}`, constraints);
    // **TODO: Adapt this endpoint and payload to your backend's API**
    // Example: POST /api/query/{collectionPath} or GET /api/docs/{collectionPath}?queryparams...
    // The constraints will need to be serialized into a format your backend understands.
    // This might be a JSON body for POST, or query parameters for GET.
    const endpoint = `/query/${encodeURIComponent(collectionPath)}`;
    const body = {
      // Example: structure your constraints
      // where: constraints.filter(c => c.type === 'where').map(c => ({...})),
      // orderBy: constraints.filter(c => c.type === 'orderBy').map(c => ({...})),
      // limit: constraints.find(c => c.type === 'limit')?.limit
      constraints: constraints // Send raw constraints for now, backend needs to parse
    };
    return this._fetch(endpoint, {
      method: 'POST', // POST is often better for complex query bodies
      body: JSON.stringify(body),
    });
  }

  // --- WebSocket Methods (Placeholders) ---
  // These would typically be handled by a dedicated WebSocket client instance,
  // possibly integrated with or managed by RealtimeManager.

  connectRealtime(
    onOpen: () => void,
    onMessage: (message: any) => void,
    onError: (error: any) => void,
    onClose: () => void
  ): void {
    console.warn("RestBackendClient.connectRealtime: WebSocket functionality is not implemented here. Should be handled by RealtimeManager or a dedicated WebSocket client.");
    // Placeholder: This client could instantiate and manage a WebSocket connection
    // using the backendUrl (e.g., transforming http to ws/wss).
    // Example:
    // const wsUrl = this.backendUrl.replace(/^http/, 'ws') + '/realtime';
    // this.websocket = new WebSocket(wsUrl);
    // this.websocket.onopen = onOpen;
    // this.websocket.onmessage = (event) => onMessage(JSON.parse(event.data as string));
    // this.websocket.onerror = onError;
    // this.websocket.onclose = onClose;
    throw new Error("Realtime connection not implemented in RestBackendClient.");
  }

  sendRealtimeMessage(message: any): void {
    console.warn("RestBackendClient.sendRealtimeMessage: Placeholder.");
    // this.websocket?.send(JSON.stringify(message));
    throw new Error("Realtime messaging not implemented in RestBackendClient.");
  }

  disconnectRealtime(): void {
    console.warn("RestBackendClient.disconnectRealtime: Placeholder.");
    // this.websocket?.close();
    throw new Error("Realtime disconnection not implemented in RestBackendClient.");
  }
}

console.log('backend_client.ts loaded');
