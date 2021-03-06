package httpclient

import "sync"

// WebSocketRequestsTracker tracks in progress WebSocketRequest objects
type WebSocketRequestsTracker struct {
	requests map[uint64]*WebSocketRequest
	mutex    sync.Mutex
}

// InProgress adds a WebSocketRequest to the "in progress" list
func (t *WebSocketRequestsTracker) InProgress(request *WebSocketRequest) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if request == nil || request.Data == nil {
		return
	}

	if t.requests == nil {
		t.requests = make(map[uint64]*WebSocketRequest)
	}

	t.requests[request.Data.GetID()] = request
}

// Done deletes a previously stored WebSocketRequest and returns it
func (t *WebSocketRequestsTracker) Done(id uint64) *WebSocketRequest {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	req := t.requests[id]
	delete(t.requests, id)
	return req
}

// IsInProgress returns true if the request id is already in progress, false otherwise
func (t *WebSocketRequestsTracker) IsInProgress(id uint64) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	_, ok := t.requests[id]
	return ok
}
