package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"trove/internal/config"
	"trove/internal/packages"
	"trove/internal/ui"
	"trove/internal/uploads"
)

const HeaderRequestID = "X-Request-Id"

type requestIDContextKey struct{}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

type ReadinessCheck func(context.Context) error

func NewRouter(cfg config.Config, store packages.Store, readiness ReadinessCheck) http.Handler {
	_ = cfg
	if store == nil {
		store = packages.NewSeedMemoryStore()
	}
	writeStore, _ := store.(packages.WriteStore)

	r := chi.NewRouter()
	r.Use(requestIDMiddleware)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Route %s %s was not found.", r.Method, r.URL.Path))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", fmt.Sprintf("Method %s is not allowed for %s.", r.Method, r.URL.Path))
	})

	r.Get("/healthz", handleHealth)
	r.Get("/readyz", handleReady(readiness))
	r.Get("/api/v1/search/packages", handleListPackages(store))
	r.Get("/api/v1/packages", handleListPackages(store))
	r.Get("/api/v1/resolve/{org}/{namespace}/{packageSelector}", handleResolve(store))
	r.Post("/api/v1/packages/{org}/{namespace}/{package}/versions", handleCreateDraft(writeStore))
	r.Post("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/archive", handleUploadArchive(writeStore))
	r.Put("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/artifacts/*", handleUploadArtifact(writeStore))
	r.Post("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/publish", handlePublishVersion(writeStore))
	r.Get("/api/v1/packages/{org}/{namespace}/{package}", handleGetPackage(store))
	r.Get("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/manifest", handleGetManifest(store))
	r.Get("/raw/{org}/{namespace}/{package}/{selector}/*", handleRawArtifact(store))
	uiHandler := ui.Handler()
	r.Get("/", uiHandler.ServeHTTP)
	r.Get("/packages", uiHandler.ServeHTTP)
	r.Get("/app.js", uiHandler.ServeHTTP)
	r.Get("/styles.css", uiHandler.ServeHTTP)

	return r
}

func handleCreateDraft(writeStore packages.WriteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeStore == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Write APIs are not configured.")
			return
		}

		var body struct {
			Version    string `json:"version"`
			Visibility string `json:"visibility"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			return
		}

		result, err := writeStore.CreateDraftVersion(r.Context(), packages.CreateDraftVersionRequest{
			Org:        chi.URLParam(r, "org"),
			Namespace:  chi.URLParam(r, "namespace"),
			Package:    chi.URLParam(r, "package"),
			Version:    body.Version,
			Visibility: body.Visibility,
		})
		if err != nil {
			writeStoreError(w, r, err)
			return
		}

		w.Header().Set("Location", "/api/v1/packages/"+result.Org+"/"+result.Namespace+"/"+result.Package+"/versions/"+result.Version)
		writeJSON(w, http.StatusCreated, result)
	}
}

func handleUploadArtifact(writeStore packages.WriteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeStore == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Write APIs are not configured.")
			return
		}

		path := chi.URLParam(r, "*")
		if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "..") {
			writeError(w, r, http.StatusBadRequest, "INVALID_ARTIFACT_PATH", "Artifact path is invalid.")
			return
		}
		content, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_ARTIFACT", "Artifact body could not be read.")
			return
		}

		result, err := writeStore.UploadArtifact(r.Context(), packages.UploadArtifactRequest{
			Org:         chi.URLParam(r, "org"),
			Namespace:   chi.URLParam(r, "namespace"),
			Package:     chi.URLParam(r, "package"),
			Version:     chi.URLParam(r, "version"),
			Path:        path,
			ContentType: r.Header.Get("Content-Type"),
			Content:     content,
		})
		if err != nil {
			writeStoreError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func handleUploadArchive(writeStore packages.WriteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeStore == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Write APIs are not configured.")
			return
		}

		content, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_ARCHIVE", "Archive body could not be read.")
			return
		}

		files, err := extractUploadArchive(r.Header.Get("Content-Type"), content)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_ARCHIVE", err.Error())
			return
		}
		artifacts := make([]packages.UploadArchiveArtifact, 0, len(files))
		for _, file := range files {
			artifacts = append(artifacts, packages.UploadArchiveArtifact{Path: file.Path, ContentType: contentTypeForPath(file.Path), Content: file.Content})
		}

		result, err := writeStore.UploadArtifacts(r.Context(), packages.UploadArtifactsRequest{
			Org:       chi.URLParam(r, "org"),
			Namespace: chi.URLParam(r, "namespace"),
			Package:   chi.URLParam(r, "package"),
			Version:   chi.URLParam(r, "version"),
			Artifacts: artifacts,
		})
		if err != nil {
			writeStoreError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"items": result})
	}
}

func handlePublishVersion(writeStore packages.WriteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeStore == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Write APIs are not configured.")
			return
		}

		result, err := writeStore.PublishVersion(r.Context(), packages.PublishVersionRequest{
			Org:       chi.URLParam(r, "org"),
			Namespace: chi.URLParam(r, "namespace"),
			Package:   chi.URLParam(r, "package"),
			Version:   chi.URLParam(r, "version"),
		})
		if err != nil {
			writeStoreError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(HeaderRequestID)
		if requestID == "" {
			requestID = newRequestID()
		}

		w.Header().Set(HeaderRequestID, requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return "req_" + hex.EncodeToString(bytes[:])
	}
	return "req_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleReady(readiness ReadinessCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if readiness == nil {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		if err := readiness(r.Context()); err != nil {
			writeError(w, r, http.StatusServiceUnavailable, "NOT_READY", "Readiness check failed.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func handleResolve(store packages.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		packageName, selector, err := packages.SplitPackageSelector(chi.URLParam(r, "packageSelector"))
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_SELECTOR", "Package reference must use package@selector.")
			return
		}

		result, err := store.Resolve(r.Context(), chi.URLParam(r, "org"), chi.URLParam(r, "namespace"), packageName, selector)
		if err != nil {
			writeStoreError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func handleGetManifest(store packages.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := store.GetManifest(
			r.Context(),
			chi.URLParam(r, "org"),
			chi.URLParam(r, "namespace"),
			chi.URLParam(r, "package"),
			chi.URLParam(r, "version"),
		)
		if err != nil {
			writeStoreError(w, r, err)
			return
		}
		writeRawJSON(w, http.StatusOK, result.Manifest)
	}
}

func handleRawArtifact(store packages.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		selector := chi.URLParam(r, "selector")
		path := chi.URLParam(r, "*")
		if path == "" || strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
			writeError(w, r, http.StatusBadRequest, "INVALID_ARTIFACT_PATH", "Artifact path is invalid.")
			return
		}

		parsed, err := packages.ParseSelector(selector)
		if err != nil {
			writeStoreError(w, r, err)
			return
		}
		version := selector
		if parsed.Kind != packages.SelectorExact {
			resolved, err := store.Resolve(r.Context(), chi.URLParam(r, "org"), chi.URLParam(r, "namespace"), chi.URLParam(r, "package"), selector)
			if err != nil {
				writeStoreError(w, r, err)
				return
			}
			w.Header().Set("Cache-Control", "no-cache")
			http.Redirect(w, r, "/raw/"+resolved.Org+"/"+resolved.Namespace+"/"+resolved.Package+"/"+resolved.ResolvedVersion+"/"+path, http.StatusFound)
			return
		}

		artifact, err := store.GetRawArtifact(
			r.Context(),
			chi.URLParam(r, "org"),
			chi.URLParam(r, "namespace"),
			chi.URLParam(r, "package"),
			version,
			path,
		)
		if err != nil {
			writeStoreError(w, r, err)
			return
		}

		w.Header().Set("Content-Type", artifact.ContentType)
		w.Header().Set("ETag", artifact.BlobDigest)
		w.Header().Set("Cache-Control", artifact.CacheControl)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(artifact.Content)
	}
}

func handleListPackages(store packages.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 50
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil || parsed < 1 {
				writeError(w, r, http.StatusBadRequest, "INVALID_LIMIT", "limit must be a positive integer.")
				return
			}
			limit = parsed
		}

		result, err := store.ListPackages(r.Context(), packages.ListPackagesParams{Limit: limit, Cursor: r.URL.Query().Get("cursor")})
		if err != nil {
			writeStoreError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func handleGetPackage(store packages.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := store.GetPackage(r.Context(), chi.URLParam(r, "org"), chi.URLParam(r, "namespace"), chi.URLParam(r, "package"))
		if err != nil {
			writeStoreError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code string, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: APIError{
			Code:      code,
			Message:   message,
			RequestID: RequestIDFromContext(r.Context()),
		},
	})
}

func writeStoreError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, packages.ErrVersionExists):
		writeError(w, r, http.StatusConflict, "VERSION_ALREADY_EXISTS", "Package version already exists.")
	case errors.Is(err, packages.ErrVersionImmutable):
		writeError(w, r, http.StatusConflict, "VERSION_IMMUTABLE", "Published package versions are immutable.")
	case errors.Is(err, packages.ErrInvalidManifest):
		writeError(w, r, http.StatusBadRequest, "INVALID_MANIFEST", err.Error())
	case errors.Is(err, packages.ErrMissingArtifact):
		writeError(w, r, http.StatusBadRequest, "INVALID_MANIFEST", err.Error())
	case errors.Is(err, packages.ErrInvalidSelector):
		writeError(w, r, http.StatusBadRequest, "INVALID_SELECTOR", "Selector is invalid.")
	case errors.Is(err, packages.ErrArtifactNotFound):
		writeError(w, r, http.StatusNotFound, "ARTIFACT_NOT_FOUND", "Artifact was not found.")
	case errors.Is(err, packages.ErrVersionNotFound):
		writeError(w, r, http.StatusNotFound, "VERSION_NOT_FOUND", "Package version was not found.")
	case errors.Is(err, packages.ErrPackageNotFound):
		writeError(w, r, http.StatusNotFound, "PACKAGE_NOT_FOUND", "Package was not found.")
	default:
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred.")
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeRawJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func extractUploadArchive(contentType string, content []byte) ([]uploads.File, error) {
	contentType = strings.ToLower(strings.Split(contentType, ";")[0])
	switch contentType {
	case "application/zip", "application/x-zip-compressed":
		return uploads.ExtractZip(content)
	case "application/gzip", "application/x-gzip", "application/tar+gzip":
		return uploads.ExtractTarGz(content)
	default:
		return nil, fmt.Errorf("unsupported archive content type %q", contentType)
	}
}

func contentTypeForPath(path string) string {
	switch {
	case strings.HasSuffix(path, ".yaml"), strings.HasSuffix(path, ".yml"):
		return "application/yaml"
	case strings.HasSuffix(path, ".md"):
		return "text/markdown; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
