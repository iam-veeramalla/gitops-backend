package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/julienschmidt/httprouter"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/rhd-gitops-examples/gitops-backend/pkg/git"
	"github.com/rhd-gitops-examples/gitops-backend/pkg/httpapi/secrets"
)

// DefaultSecretRef is the name looked up if none is provided in the URL.
var DefaultSecretRef = types.NamespacedName{
	Name:      "pipelines-app-gitops",
	Namespace: "pipelines-app-delivery",
}

// APIRouter is an HTTP API for accessing app configurations.
type APIRouter struct {
	*httprouter.Router
	gitClientFactory git.ClientFactory
	secretGetter     secrets.SecretGetter
	secretRef        types.NamespacedName
}

// GePipelines fetches and returns the pipeline body.
func (a *APIRouter) GetPipelines(w http.ResponseWriter, r *http.Request) {
	urlToFetch := r.URL.Query().Get("url")
	if urlToFetch == "" {
		log.Println("ERROR: could not get url from request")
		http.Error(w, "missing parameter 'url'", http.StatusBadRequest)
		return
	}

	// TODO: replace this with logr or sugar.
	log.Printf("urlToFetch = %#v\n", urlToFetch)
	repo, err := parseURL(urlToFetch)
	if err != nil {
		log.Printf("ERROR: failed to parse the URL: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client, err := a.getAuthenticatedGitClient(r.Context(), r, urlToFetch)
	if err != nil {
		log.Println("ERROR: failed to get an authenticated client")
		http.Error(w, "unable to authenticate request", http.StatusBadRequest)
		return
	}

	// TODO: don't send back the error directly.
	//
	// Add a "not found" error that can be returned, otherwise it's a
	// StatusInternalServerError.
	log.Println("got an authenticated client")
	body, err := client.FileContents(r.Context(), repo, "pipelines.yaml", "master")
	if err != nil {
		log.Printf("ERROR: failed to get file contents for repo %#v: %s", repo, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pipelines := &config{}
	err = yaml.Unmarshal(body, &pipelines)
	if err != nil {
		log.Printf("ERROR: failed to unmarshal body %s", err)
		http.Error(w, fmt.Sprintf("failed to unmarshal pipelines.yaml: %s", err.Error()), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pipelinesToAppsResponse(pipelines))
}

func (a *APIRouter) getAuthenticatedGitClient(ctx context.Context, req *http.Request, fetchURL string) (git.SCM, error) {
	token := AuthToken(ctx)
	secret, ok := secretRefFromQuery(req.URL.Query())
	if !ok {
		secret = a.secretRef
	}
	token, err := a.secretGetter.SecretToken(ctx, token, secret)
	if err != nil {
		return nil, err
	}
	return a.gitClientFactory.Create(fetchURL, token)
}

// NewRouter creates and returns a new APIRouter.
func NewRouter(c git.ClientFactory, s secrets.SecretGetter) *APIRouter {
	api := &APIRouter{Router: httprouter.New(), gitClientFactory: c, secretGetter: s, secretRef: DefaultSecretRef}
	api.HandlerFunc(http.MethodGet, "/pipelines", api.GetPipelines)
	return api
}

func parseURL(s string) (string, error) {
	parsed, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("failed to parse %#v: %w", s, err)
	}
	return strings.TrimLeft(strings.Trim(parsed.Path, ".git"), "/"), nil
}

func secretRefFromQuery(v url.Values) (types.NamespacedName, bool) {
	ns := v.Get("secretNS")
	name := v.Get("secretName")
	if ns != "" && name != "" {
		return types.NamespacedName{
			Name:      name,
			Namespace: ns,
		}, true
	}
	return types.NamespacedName{}, false
}
