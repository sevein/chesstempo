package http

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/filter/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

	"github.com/sevein/chesstempo/game"
	"github.com/sevein/chesstempo/http/assets"
)

type Server struct {
	ln     net.Listener
	server *http.Server
	router *mux.Router

	Addr           string
	TemporalClient client.Client
}

func NewServer() *Server {
	s := &Server{
		server: &http.Server{},
		router: mux.NewRouter(),
	}

	router := s.router.PathPrefix("/").Subrouter()

	// API namespace.
	{
		r := router.PathPrefix("/api").Subrouter()
		r.StrictSlash(true)

		r.Handle("/games", appHandler(s.handleGameList)).Methods("GET")
		r.HandleFunc("/games", s.handleGameCreate).Methods("POST")
		r.HandleFunc("/games/{id}", s.handleGameRead).Methods("GET")
		r.HandleFunc("/games/{id}/move/{move}", s.handleGameMove).Methods("POST")
		r.HandleFunc("/games/{id}/resign", s.handleGameResign).Methods("POST")
	}

	// Assets.
	{
		web := assets.SPAHandler()
		router.PathPrefix("/").HandlerFunc(web)
	}

	s.server.Handler = router

	return s
}

func (s *Server) Open() (err error) {
	if s.ln, err = net.Listen("tcp", s.Addr); err != nil {
		return err
	}

	// Begin serving requests on the listener. We use Serve() instead of
	// ListenAndServe() because it allows us to check for listen errors (such
	// as trying to use an already open port) synchronously.
	go s.server.Serve(s.ln)

	return nil
}

func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) handleGameList(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	opts := &workflowservice.ListOpenWorkflowExecutionsRequest{
		Filters: &workflowservice.ListOpenWorkflowExecutionsRequest_TypeFilter{
			TypeFilter: &filter.WorkflowTypeFilter{
				Name: "GameWorkflow",
			},
		},
	}
	resp, err := s.TemporalClient.ListOpenWorkflow(ctx, opts)
	if err != nil {
		return err
	}

	ret := make([]string, len(resp.Executions))
	for idx, exec := range resp.Executions {
		if exec.Execution == nil {
			continue
		}
		ret[idx] = exec.Execution.WorkflowId
	}

	err = json.NewEncoder(w).Encode(ret)
	if err != nil {
		return ResponseError{}
	}

	return nil
}

func (s *Server) handleGameCreate(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Read the payload.
	params := game.GameWorkflowParams{}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	opts := client.StartWorkflowOptions{
		ID:        uuid.New().String(),
		TaskQueue: "queue",
	}
	wr, err := s.TemporalClient.ExecuteWorkflow(ctx, opts, game.GameWorkflow, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ret := struct {
		ID string `json:"id"`
	}{
		ID: wr.GetID(),
	}
	err = json.NewEncoder(w).Encode(ret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleGameRead(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	vars := mux.Vars(r)
	workflowID := vars["id"]

	info := game.GameInfo{} // Response.

	// Query workflow.
	opts := client.QueryWorkflowWithOptionsRequest{
		WorkflowID:           workflowID,
		QueryType:            "info",
		QueryRejectCondition: enums.QUERY_REJECT_CONDITION_NOT_OPEN,
	}
	resp, err := s.TemporalClient.QueryWorkflowWithOptions(ctx, &opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If rejected, the workflow is completed. Return is value.
	if resp.QueryRejected != nil {
		err := s.TemporalClient.GetWorkflow(ctx, workflowID, "").Get(ctx, &info)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		goto ret
	}

	err = resp.QueryResult.Get(&info)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

ret:
	err = json.NewEncoder(w).Encode(info)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleGameMove(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	vars := mux.Vars(r)
	workflowID := vars["id"]

	err := s.TemporalClient.SignalWorkflow(ctx, workflowID, "", "move", vars["move"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := struct{ OK bool }{OK: true}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleGameResign(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	vars := mux.Vars(r)
	workflowID := vars["id"]

	err := s.TemporalClient.SignalWorkflow(ctx, workflowID, "", "resign", struct{}{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := struct{ OK bool }{OK: true}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ListenAndServeDebug() error {
	h := http.NewServeMux()
	h.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(":6060", h)
}
