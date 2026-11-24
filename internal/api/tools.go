package api

import (
	"errors"
	"net/http"
	"time"

	"calango/internal/core"
	"calango/internal/tools"
)

// codeConnectorUnavailable reports that a registered connector did not answer. Its
// message is fixed: the underlying error can name a local command or a URL.
const codeConnectorUnavailable = "connector_unavailable"

// toolAccessNone removes an agent's grant on a connector.
const toolAccessNone = "none"

type connectorBody struct {
	Name      string   `json:"name"`
	Transport string   `json:"transport"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	URL       string   `json:"url"`
}

type connectorResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Transport string    `json:"transport"`
	Command   string    `json:"command"`
	Args      []string  `json:"args"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func connectorResponseOf(c core.Connector) connectorResponse {
	args := c.Args
	if args == nil {
		args = []string{}
	}
	return connectorResponse{
		ID: c.ID, Name: c.Name, Transport: string(c.Transport), Command: c.Command,
		Args: args, URL: c.URL, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

type agentToolResponse struct {
	ConnectorID string `json:"connector_id"`
	Access      string `json:"access"`
}

type agentToolBody struct {
	Access string `json:"access"`
}

func (h *handler) registerTools(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/connectors", h.need(core.PermRead, h.listConnectors))
	mux.HandleFunc("POST /v1/connectors", h.need(core.PermWrite, h.createConnector))
	mux.HandleFunc("DELETE /v1/connectors/{id}", h.need(core.PermWrite, h.deleteConnector))
	mux.HandleFunc("GET /v1/connectors/{id}/tools", h.need(core.PermRead, h.connectorTools))
	mux.HandleFunc("GET /v1/agents/{id}/tools", h.need(core.PermRead, h.agentTools))
	mux.HandleFunc("PUT /v1/agents/{id}/tools/{connectorID}", h.need(core.PermWrite, h.setAgentTool))
}

func (h *handler) listConnectors(w http.ResponseWriter, r *http.Request) {
	list, err := h.tools.List(r.Context(), h.ownerID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]connectorResponse, 0, len(list))
	for _, c := range list {
		out = append(out, connectorResponseOf(c))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) createConnector(w http.ResponseWriter, r *http.Request) {
	var body connectorBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	c, err := h.tools.Register(r.Context(), h.ownerID, tools.NewConnector{
		Name:      body.Name,
		Transport: core.ConnectorTransport(body.Transport),
		Command:   body.Command,
		Args:      body.Args,
		URL:       body.URL,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, connectorResponseOf(c))
}

func (h *handler) deleteConnector(w http.ResponseWriter, r *http.Request) {
	if err := h.tools.Delete(r.Context(), h.ownerID, r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) connectorTools(w http.ResponseWriter, r *http.Request) {
	infos, err := h.tools.Catalog(r.Context(), h.ownerID, r.PathValue("id"))
	if errors.Is(err, tools.ErrUnavailable) {
		writeError(w, http.StatusBadGateway, codeConnectorUnavailable, "the connector did not list its tools")
		return
	}
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, infos)
}

func (h *handler) agentTools(w http.ResponseWriter, r *http.Request) {
	grants, err := h.tools.Grants(r.Context(), h.ownerID, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]agentToolResponse, 0, len(grants))
	for _, g := range grants {
		out = append(out, agentToolResponse{ConnectorID: g.ConnectorID, Access: string(g.Access)})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) setAgentTool(w http.ResponseWriter, r *http.Request) {
	var body agentToolBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	agentID, connectorID := r.PathValue("id"), r.PathValue("connectorID")
	var err error
	if body.Access == toolAccessNone {
		err = h.tools.Revoke(r.Context(), h.ownerID, agentID, connectorID)
	} else {
		err = h.tools.Grant(r.Context(), h.ownerID, agentID, connectorID, core.ToolAccess(body.Access))
	}
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
