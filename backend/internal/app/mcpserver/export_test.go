package mcpserver

import "net/http"

func (h *Handler) SetMetadataTransportForTest(transport http.RoundTripper) {
	h.metadata.Transport = transport
}
