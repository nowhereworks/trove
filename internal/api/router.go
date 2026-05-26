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

	"trove/internal/auth"
	"trove/internal/config"
	coreskills "trove/internal/core/skills"
	"trove/internal/db/sqlc"
	"trove/internal/packages"
	"trove/internal/reviews"
	"trove/internal/security"
	"trove/internal/ui"
	"trove/internal/updates"
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
	if store == nil {
		panic("api.NewRouter requires a PostgreSQL-backed package store")
	}
	writeStore, _ := store.(packages.WriteStore)

	var authenticator *auth.Authenticator
	var reviewService *reviews.Service
	scanner := security.NewScanner(cfg.Security)
	updateService := updates.NewService(store)

	if ps, ok := store.(interface{ Queries() *sqlc.Queries }); ok {
		queries := ps.Queries()
		if queries != nil {
			authenticator, _ = auth.NewAuthenticator(cfg, queries)
			reviewService = reviews.NewService(queries, cfg.Reviews)
		}
	}

	r := chi.NewRouter()
	r.Use(requestIDMiddleware)

	if authenticator != nil {
		r.Use(authenticator.Middleware)
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Route %s %s was not found.", r.Method, r.URL.Path))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", fmt.Sprintf("Method %s is not allowed for %s.", r.Method, r.URL.Path))
	})

	r.Get("/healthz", handleHealth)
	r.Get("/readyz", handleReady(readiness))
	r.Get("/api/v1/config", handleGetConfig(cfg))
	r.Get("/api/v1/core/skills/{skill}/SKILL.md", handleCoreSkill)
	r.Get("/api/v1/search/packages", handleSearchPackages(store))
	r.Get("/api/v1/packages", handleListPackages(store))
	r.Get("/api/v1/resolve/{org}/{namespace}/{packageSelector}", handleResolve(store))
	r.Post("/api/v1/packages/{org}/{namespace}/{package}/versions", auth.RequireAuth(auth.RequireScope("package:write")(handleCreateDraft(writeStore, cfg))))
	r.Post("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/archive", auth.RequireAuth(auth.RequireScope("package:write")(handleUploadArchive(writeStore, scanner, cfg))))
	r.Put("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/artifacts/*", auth.RequireAuth(auth.RequireScope("package:write")(handleUploadArtifact(writeStore, scanner, cfg))))
	r.Post("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/publish", auth.RequireAuth(auth.RequireScope("version:publish")(handlePublishVersion(writeStore, reviewService, cfg))))
	r.Post("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/reset-draft", auth.RequireAuth(auth.RequireScope("package:write")(handleResetUnpublishedVersion(writeStore))))
	r.Post("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/deprecate", auth.RequireAuth(auth.RequireScope("package:write")(handleDeprecateVersion(writeStore))))
	r.Post("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/yank", auth.RequireAuth(auth.RequireScope("package:write")(handleYankVersion(writeStore))))
	r.Post("/api/v1/orgs", auth.RequireAuth(auth.RequireScope("org:write")(handleCreateOrg(writeStore, cfg))))
	r.Post("/api/v1/orgs/{org}/namespaces", auth.RequireAuth(auth.RequireScope("namespace:write")(handleCreateNamespace(writeStore))))
	r.Post("/api/v1/packages", auth.RequireAuth(auth.RequireScope("package:write")(handleCreatePackage(writeStore))))
	r.Get("/api/v1/packages/{org}/{namespace}/{package}", handleGetPackage(store))
	r.Get("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}", handleGetPackageVersion(store, writeStore))
	r.Get("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/manifest", handleGetManifest(store))
	r.Get("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/archive.tar.gz", handleGetArchive(store, packages.ArchiveTarGz))
	r.Get("/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/archive.zip", handleGetArchive(store, packages.ArchiveZip))
	r.Get("/api/v1/packages/{org}/{namespace}/{package}/adoption", handleGetPackageAdoption(store))
	r.Get("/raw/{org}/{namespace}/{package}/*", handleRawArtifact(store, cfg))

	r.Post("/api/v1/updates/check", handleCheckUpdate(updateService))
	r.Post("/api/v1/compatibility/check", handleCheckCompatibility(updateService))
	r.Post("/api/v1/projects/report", auth.RequireAuth(handleReportProjectAdoption(writeStore)))
	r.Post("/api/v1/projects", auth.RequireAuth(handleCreateProject(writeStore)))

	r.Group(func(r chi.Router) {
		r.Post("/api/v1/reviews/{org}/{namespace}/{package}/versions/{version}/submit", auth.RequireAuth(auth.RequireScope("review:write")(handleSubmitReview(reviewService))))
		r.Post("/api/v1/reviews/{reviewId}/approve", auth.RequireAuth(auth.RequireScope("review:write")(handleApproveReview(reviewService))))
		r.Post("/api/v1/reviews/{reviewId}/request-changes", auth.RequireAuth(auth.RequireScope("review:write")(handleRequestChanges(reviewService))))
		r.Post("/api/v1/reviews/{reviewId}/comments", auth.RequireAuth(auth.RequireScope("review:write")(handleAddReviewComment(reviewService))))
		r.Get("/api/v1/reviews/{org}/{namespace}/{package}/versions/{version}", handleListReviews(reviewService))
		r.Get("/api/v1/reviews/{org}/{namespace}/{package}/versions/{version}/approval-status", handleApprovalStatus(reviewService))
	})

	r.Group(func(r chi.Router) {
		r.Post("/api/v1/tokens", handleCreateToken(authenticator, cfg))
		r.Post("/api/v1/tokens/{tokenId}/revoke", handleRevokeToken(authenticator))
	})

	r.Get("/auth/oidc/login", handleOIDCLogin(authenticator))
	r.Get("/auth/oidc/callback", handleOIDCCallback(authenticator))

	uiHandler := ui.Handler()
	r.Get("/", uiHandler.ServeHTTP)
	r.Get("/search", uiHandler.ServeHTTP)
	r.Get("/adoption", uiHandler.ServeHTTP)
	r.Get("/upload", uiHandler.ServeHTTP)
	r.Get("/reviews", uiHandler.ServeHTTP)
	r.Get("/orgs/new", uiHandler.ServeHTTP)
	r.Get("/packages/{org}/{namespace}/{name}", uiHandler.ServeHTTP)
	r.Get("/assets/*", uiHandler.ServeHTTP)

	return r
}

func handleGetConfig(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, struct {
			Org            string `json:"org"`
			AllowCreateOrg bool   `json:"allowCreateOrg"`
		}{
			Org:            cfg.Orgs.DefaultOrg,
			AllowCreateOrg: cfg.Orgs.AllowCreateOrg,
		})
	}
}

func handleCoreSkill(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "skill")
	content, ok := coreskills.Read(name)
	if !ok {
		writeError(w, r, http.StatusNotFound, "NOT_FOUND", "Core skill was not found.")
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func handleCreateDraft(writeStore packages.WriteStore, cfg config.Config) http.HandlerFunc {
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

		request := packages.CreateDraftVersionRequest{
			Org:        chi.URLParam(r, "org"),
			Namespace:  chi.URLParam(r, "namespace"),
			Package:    chi.URLParam(r, "package"),
			Version:    body.Version,
			Visibility: body.Visibility,
		}
		result, err := writeStore.CreateDraftVersion(r.Context(), request)
		if errors.Is(err, packages.ErrPackageNotFound) && cfg.Packages.CreatePackageOnPush {
			visibility := body.Visibility
			if visibility == "" {
				visibility = "private"
			}
			if cfg.Packages.CreateNamespaceOnPush {
				_, err = writeStore.EnsureNamespace(r.Context(), packages.CreateNamespaceRequest{Org: request.Org, Slug: request.Namespace, DisplayName: request.Namespace, Visibility: visibility})
				if err != nil {
					writeStoreError(w, r, err)
					return
				}
			}
			_, err = writeStore.EnsurePackage(r.Context(), packages.CreatePackageRequest{Org: request.Org, Namespace: request.Namespace, Name: request.Package, DisplayName: request.Package, Visibility: visibility})
			if err != nil {
				writeStoreError(w, r, err)
				return
			}
			result, err = writeStore.CreateDraftVersion(r.Context(), request)
		}
		if err != nil {
			writeStoreError(w, r, err)
			return
		}

		w.Header().Set("Location", "/api/v1/packages/"+result.Org+"/"+result.Namespace+"/"+result.Package+"/versions/"+result.Version)
		writeJSON(w, http.StatusCreated, result)
	}
}

func handleUploadArtifact(writeStore packages.WriteStore, scanner *security.Scanner, cfg config.Config) http.HandlerFunc {
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

		if int64(len(content)) > cfg.Storage.Limits.MaxArtifactFileBytes {
			writeError(w, r, http.StatusBadRequest, "ARTIFACT_TOO_LARGE", "Artifact exceeds maximum file size.")
			return
		}

		if scanner != nil {
			scanResult := scanner.ScanContent(path, content)
			if scanResult.Blocked {
				if len(scanResult.SecretsFound) > 0 {
					writeError(w, r, http.StatusBadRequest, "SECRET_DETECTED", "Potential secret detected in artifact.")
					return
				}
				if len(scanResult.UnsafeFound) > 0 {
					writeError(w, r, http.StatusBadRequest, "UNSAFE_INSTRUCTION", "High-risk unsafe instruction detected in artifact.")
					return
				}
			}
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

func handleUploadArchive(writeStore packages.WriteStore, scanner *security.Scanner, cfg config.Config) http.HandlerFunc {
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

		totalSize := int64(0)
		for _, file := range files {
			totalSize += int64(len(file.Content))
			if int64(len(file.Content)) > cfg.Storage.Limits.MaxArtifactFileBytes {
				writeError(w, r, http.StatusBadRequest, "ARTIFACT_TOO_LARGE", fmt.Sprintf("Artifact %s exceeds maximum file size.", file.Path))
				return
			}
		}
		if totalSize > cfg.Storage.Limits.MaxUnpackedPackageBytes {
			writeError(w, r, http.StatusBadRequest, "PACKAGE_TOO_LARGE", "Unpacked package exceeds maximum size.")
			return
		}
		if len(files) > cfg.Storage.Limits.MaxArtifactsPerVersion {
			writeError(w, r, http.StatusBadRequest, "TOO_MANY_ARTIFACTS", "Package exceeds maximum artifact count.")
			return
		}

		if scanner != nil {
			for _, file := range files {
				scanResult := scanner.ScanContent(file.Path, file.Content)
				if scanResult.Blocked {
					if len(scanResult.SecretsFound) > 0 {
						writeError(w, r, http.StatusBadRequest, "SECRET_DETECTED", fmt.Sprintf("Potential secret detected in %s.", file.Path))
						return
					}
					if len(scanResult.UnsafeFound) > 0 {
						writeError(w, r, http.StatusBadRequest, "UNSAFE_INSTRUCTION", fmt.Sprintf("High-risk unsafe instruction detected in %s.", file.Path))
						return
					}
				}
			}
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

func handlePublishVersion(writeStore packages.WriteStore, reviewService *reviews.Service, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeStore == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Write APIs are not configured.")
			return
		}

		if reviewService != nil && cfg.Reviews.RequireApproval {
			packageVersionID, err := reviewService.ResolvePackageVersionID(
				r.Context(),
				chi.URLParam(r, "org"),
				chi.URLParam(r, "namespace"),
				chi.URLParam(r, "package"),
				chi.URLParam(r, "version"),
			)
			if err != nil {
				writeStoreError(w, r, packages.ErrVersionNotFound)
				return
			}

			status := reviewService.GetApprovalStatus(r.Context(), packageVersionID)
			if !status.HasEnoughApprovals {
				writeError(w, r, http.StatusForbidden, "APPROVAL_REQUIRED", "Version requires review approval before publishing.")
				return
			}
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

func handleDeprecateVersion(writeStore packages.WriteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeStore == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Write APIs are not configured.")
			return
		}

		result, err := writeStore.DeprecateVersion(r.Context(), packages.LifecycleChangeRequest{
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

func handleYankVersion(writeStore packages.WriteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeStore == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Write APIs are not configured.")
			return
		}

		result, err := writeStore.YankVersion(r.Context(), packages.LifecycleChangeRequest{
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

func handleCreateOrg(writeStore packages.WriteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !cfg.Orgs.AllowCreateOrg {
			writeError(w, r, http.StatusForbidden, "ORG_CREATION_DISABLED", "Organization creation is disabled. Configure TROVE_ORG at startup or enable TROVE_ALLOW_CREATE_ORG.")
			return
		}
		if writeStore == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Write APIs are not configured.")
			return
		}

		var body struct {
			Slug        string `json:"slug"`
			DisplayName string `json:"displayName"`
			Visibility  string `json:"visibility"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			return
		}

		result, err := writeStore.CreateOrg(r.Context(), packages.CreateOrgRequest{
			Slug:        body.Slug,
			DisplayName: body.DisplayName,
			Visibility:  body.Visibility,
		})
		if err != nil {
			writeStoreError(w, r, err)
			return
		}

		w.Header().Set("Location", "/api/v1/orgs/"+result.Slug)
		writeJSON(w, http.StatusCreated, result)
	}
}

func handleCreateNamespace(writeStore packages.WriteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeStore == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Write APIs are not configured.")
			return
		}

		var body struct {
			Slug        string `json:"slug"`
			DisplayName string `json:"displayName"`
			Visibility  string `json:"visibility"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			return
		}

		result, err := writeStore.CreateNamespace(r.Context(), packages.CreateNamespaceRequest{
			Org:         chi.URLParam(r, "org"),
			Slug:        body.Slug,
			DisplayName: body.DisplayName,
			Visibility:  body.Visibility,
		})
		if err != nil {
			writeStoreError(w, r, err)
			return
		}

		w.Header().Set("Location", "/api/v1/orgs/"+chi.URLParam(r, "org")+"/namespaces/"+result.Slug)
		writeJSON(w, http.StatusCreated, result)
	}
}

func handleCreatePackage(writeStore packages.WriteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeStore == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Write APIs are not configured.")
			return
		}

		var body struct {
			Org         string `json:"org"`
			Namespace   string `json:"namespace"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			Description string `json:"description"`
			Visibility  string `json:"visibility"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			return
		}

		result, err := writeStore.CreatePackage(r.Context(), packages.CreatePackageRequest{
			Org:         body.Org,
			Namespace:   body.Namespace,
			Name:        body.Name,
			DisplayName: body.DisplayName,
			Description: body.Description,
			Visibility:  body.Visibility,
		})
		if err != nil {
			writeStoreError(w, r, err)
			return
		}

		w.Header().Set("Location", "/api/v1/packages/"+result.Org+"/"+result.Namespace+"/"+result.Name)
		writeJSON(w, http.StatusCreated, result)
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

		user, _ := auth.UserFromContext(r.Context())
		if !user.IsAuthenticated && !user.IsDev {
			visibility, verr := store.CheckVisibility(r.Context(), result.Org, result.Namespace, result.Package, result.ResolvedVersion)
			if verr != nil || !auth.CheckVisibility(visibility, user, true) {
				writeError(w, r, http.StatusNotFound, "NOT_FOUND", "Package version was not found.")
				return
			}
		}

		writeJSON(w, http.StatusOK, result)
	}
}

func handleGetManifest(store packages.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFromContext(r.Context())
		if !user.IsAuthenticated && !user.IsDev {
			visibility, verr := store.CheckVisibility(r.Context(), chi.URLParam(r, "org"), chi.URLParam(r, "namespace"), chi.URLParam(r, "package"), chi.URLParam(r, "version"))
			if verr != nil || !auth.CheckVisibility(visibility, user, true) {
				writeError(w, r, http.StatusNotFound, "NOT_FOUND", "Package version was not found.")
				return
			}
		}

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

func handleGetArchive(store packages.Store, format packages.ArchiveFormat) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version := chi.URLParam(r, "version")
		parsed, err := packages.ParseSelector(version)
		if err != nil {
			writeStoreError(w, r, err)
			return
		}

		org := chi.URLParam(r, "org")
		namespace := chi.URLParam(r, "namespace")
		pkg := chi.URLParam(r, "package")

		if parsed.Kind != packages.SelectorExact {
			resolved, err := store.Resolve(r.Context(), org, namespace, pkg, version)
			if err != nil {
				writeStoreError(w, r, err)
				return
			}

			user, _ := auth.UserFromContext(r.Context())
			if !user.IsAuthenticated && !user.IsDev {
				visibility, verr := store.CheckVisibility(r.Context(), resolved.Org, resolved.Namespace, resolved.Package, resolved.ResolvedVersion)
				if verr != nil || !auth.CheckVisibility(visibility, user, true) {
					writeError(w, r, http.StatusNotFound, "NOT_FOUND", "Package version was not found.")
					return
				}
			}

			extension := "archive.tar.gz"
			if format == packages.ArchiveZip {
				extension = "archive.zip"
			}
			w.Header().Set("Cache-Control", "no-cache")
			http.Redirect(w, r, "/api/v1/packages/"+resolved.Org+"/"+resolved.Namespace+"/"+resolved.Package+"/versions/"+resolved.ResolvedVersion+"/"+extension, http.StatusFound)
			return
		}

		user, _ := auth.UserFromContext(r.Context())
		if !user.IsAuthenticated && !user.IsDev {
			visibility, verr := store.CheckVisibility(r.Context(), org, namespace, pkg, parsed.Version)
			if verr != nil || !auth.CheckVisibility(visibility, user, true) {
				writeError(w, r, http.StatusNotFound, "NOT_FOUND", "Package version was not found.")
				return
			}
		}

		archive, err := store.GetArchive(r.Context(), org, namespace, pkg, parsed.Version, format)
		if err != nil {
			writeStoreError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", archive.ContentType)
		w.Header().Set("ETag", archive.ETag)
		w.Header().Set("Cache-Control", archive.CacheControl)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archive.Content)
	}
}

func handleRawArtifact(store packages.Store, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := chi.URLParam(r, "*")
		path, selector, err := splitRawArtifactPathSelector(path)
		if err != nil || path == "" || strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
			writeError(w, r, http.StatusBadRequest, "INVALID_ARTIFACT_PATH", "Artifact path is invalid.")
			return
		}

		user, _ := auth.UserFromContext(r.Context())

		parsed, err := packages.ParseSelector(selector)
		if err != nil {
			writeStoreError(w, r, err)
			return
		}
		version := parsed.Version
		if parsed.Kind != packages.SelectorExact {
			resolved, err := store.Resolve(r.Context(), chi.URLParam(r, "org"), chi.URLParam(r, "namespace"), chi.URLParam(r, "package"), selector)
			if err != nil {
				writeStoreError(w, r, err)
				return
			}
			if !authorizeRawArtifact(w, r, store, cfg, user, resolved.Org, resolved.Namespace, resolved.Package, resolved.ResolvedVersion) {
				return
			}
			w.Header().Set("Cache-Control", "no-cache")
			http.Redirect(w, r, "/raw/"+resolved.Org+"/"+resolved.Namespace+"/"+resolved.Package+"/"+path+"@"+resolved.ResolvedVersion, http.StatusFound)
			return
		}
		if selector != version {
			if !authorizeRawArtifact(w, r, store, cfg, user, chi.URLParam(r, "org"), chi.URLParam(r, "namespace"), chi.URLParam(r, "package"), version) {
				return
			}
			w.Header().Set("Cache-Control", "no-cache")
			http.Redirect(w, r, "/raw/"+chi.URLParam(r, "org")+"/"+chi.URLParam(r, "namespace")+"/"+chi.URLParam(r, "package")+"/"+path+"@"+version, http.StatusFound)
			return
		}

		if !authorizeRawArtifact(w, r, store, cfg, user, chi.URLParam(r, "org"), chi.URLParam(r, "namespace"), chi.URLParam(r, "package"), version) {
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

func authorizeRawArtifact(w http.ResponseWriter, r *http.Request, store packages.Store, cfg config.Config, user auth.User, org, namespace, name, version string) bool {
	if !cfg.Raw.RequireAuthByDefault || user.IsAuthenticated || user.IsDev {
		return true
	}
	visibility, err := store.CheckVisibility(r.Context(), org, namespace, name, version)
	if err != nil {
		writeStoreError(w, r, err)
		return false
	}
	if visibility == "public" && cfg.Raw.AllowPublicPackages {
		return true
	}
	writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required to access this artifact.")
	return false
}

func splitRawArtifactPathSelector(raw string) (string, string, error) {
	if raw == "" || strings.HasPrefix(raw, "/") {
		return "", "", packages.ErrInvalidSelector
	}
	if strings.Count(raw, "@") > 1 {
		return "", "", packages.ErrInvalidSelector
	}
	idx := strings.LastIndex(raw, "@")
	if idx == -1 {
		return raw, "stable", nil
	}
	if idx == 0 || idx == len(raw)-1 {
		return "", "", packages.ErrInvalidSelector
	}
	return raw[:idx], raw[idx+1:], nil
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

		user, _ := auth.UserFromContext(r.Context())
		if !user.IsAuthenticated && !user.IsDev {
			filtered := make([]packages.PackageSummary, 0, len(result.Items))
			for _, item := range result.Items {
				if item.Visibility == "public" {
					filtered = append(filtered, item)
				}
			}
			result.Items = filtered
		}

		writeJSON(w, http.StatusOK, result)
	}
}

func handleGetPackage(store packages.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFromContext(r.Context())
		if !user.IsAuthenticated && !user.IsDev {
			visibility, verr := store.CheckVisibility(r.Context(), chi.URLParam(r, "org"), chi.URLParam(r, "namespace"), chi.URLParam(r, "package"), "latest")
			if verr != nil || !auth.CheckVisibility(visibility, user, true) {
				writeError(w, r, http.StatusNotFound, "NOT_FOUND", "Package was not found.")
				return
			}
		}

		result, err := store.GetPackage(r.Context(), chi.URLParam(r, "org"), chi.URLParam(r, "namespace"), chi.URLParam(r, "package"))
		if err != nil {
			writeStoreError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func handleGetPackageVersion(store packages.Store, writeStore packages.WriteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFromContext(r.Context())
		org := chi.URLParam(r, "org")
		namespace := chi.URLParam(r, "namespace")
		pkg := chi.URLParam(r, "package")
		version := chi.URLParam(r, "version")
		if canReadUnpublishedVersion(user) {
			if managementStore, ok := writeStore.(packages.ManagementStore); ok {
				result, err := managementStore.GetPackageVersion(r.Context(), packages.LifecycleChangeRequest{Org: org, Namespace: namespace, Package: pkg, Version: version})
				if err != nil {
					writeStoreError(w, r, err)
					return
				}
				writeJSON(w, http.StatusOK, result)
				return
			}
		}
		if !user.IsAuthenticated && !user.IsDev {
			visibility, verr := store.CheckVisibility(r.Context(), org, namespace, pkg, version)
			if verr != nil || !auth.CheckVisibility(visibility, user, true) {
				writeError(w, r, http.StatusNotFound, "NOT_FOUND", "Package version was not found.")
				return
			}
		}

		detail, err := store.GetPackage(r.Context(), org, namespace, pkg)
		if err != nil {
			writeStoreError(w, r, err)
			return
		}
		for _, candidate := range detail.Versions {
			if candidate.Version == version {
				writeJSON(w, http.StatusOK, packages.VersionResource{Org: org, Namespace: namespace, Package: pkg, Version: candidate.Version, Lifecycle: candidate.Lifecycle, Visibility: detail.Visibility, Digest: candidate.Digest, PublishedAt: candidate.PublishedAt})
				return
			}
		}
		writeStoreError(w, r, packages.ErrVersionNotFound)
	}
}

func canReadUnpublishedVersion(user auth.User) bool {
	if user.IsDev {
		return true
	}
	for _, scope := range user.TokenScopes {
		switch scope {
		case "*:*", "package:write", "version:publish", "review:write":
			return true
		}
	}
	return false
}

func handleResetUnpublishedVersion(writeStore packages.WriteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		managementStore, ok := writeStore.(packages.ManagementStore)
		if !ok {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Version reset is not configured.")
			return
		}

		result, err := managementStore.ResetUnpublishedVersion(r.Context(), packages.LifecycleChangeRequest{
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

func handleSubmitReview(reviewService *reviews.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if reviewService == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Review workflow is not configured.")
			return
		}

		user, ok := auth.UserFromContext(r.Context())
		if !ok || !user.IsAuthenticated {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
			return
		}

		packageVersionID, err := reviewService.ResolvePackageVersionID(
			r.Context(),
			chi.URLParam(r, "org"),
			chi.URLParam(r, "namespace"),
			chi.URLParam(r, "package"),
			chi.URLParam(r, "version"),
		)
		if err != nil {
			writeStoreError(w, r, packages.ErrVersionNotFound)
			return
		}

		err = reviewService.SubmitForReview(r.Context(), packageVersionID, user.ID)
		if err != nil {
			switch {
			case errors.Is(err, reviews.ErrInvalidReviewer):
				writeError(w, r, http.StatusForbidden, "REVIEWER_USER_REQUIRED", "Review submission requires a user-backed token.")
			case errors.Is(err, reviews.ErrVersionNotSubmittable):
				writeError(w, r, http.StatusConflict, "VERSION_NOT_SUBMITTABLE", "Only draft or review versions can be submitted for review.")
			default:
				writeError(w, r, http.StatusInternalServerError, "REVIEW_SUBMIT_FAILED", "Failed to submit for review.")
			}
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "submitted"})
	}
}

func handleApproveReview(reviewService *reviews.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if reviewService == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Review workflow is not configured.")
			return
		}

		user, ok := auth.UserFromContext(r.Context())
		if !ok || !user.IsAuthenticated {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
			return
		}

		var body struct {
			Comment          string `json:"comment"`
			PackageVersionID string `json:"packageVersionId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			return
		}

		packageVersionID := body.PackageVersionID
		if packageVersionID == "" {
			resolvedID, err := reviewService.PackageVersionIDForReview(r.Context(), chi.URLParam(r, "reviewId"))
			if err != nil {
				writeError(w, r, http.StatusNotFound, "REVIEW_NOT_FOUND", "Review was not found.")
				return
			}
			packageVersionID = resolvedID
		}

		review, err := reviewService.Approve(r.Context(), chi.URLParam(r, "reviewId"), user.ID, packageVersionID, body.Comment)
		if err != nil {
			switch {
			case errors.Is(err, reviews.ErrSelfApproval):
				writeError(w, r, http.StatusForbidden, "SELF_APPROVAL_BLOCKED", "Self-approval is not allowed.")
			case errors.Is(err, reviews.ErrAlreadyApproved):
				writeError(w, r, http.StatusConflict, "ALREADY_APPROVED", "Already approved by this reviewer.")
			default:
				writeError(w, r, http.StatusInternalServerError, "APPROVE_FAILED", "Failed to approve review.")
			}
			return
		}

		writeJSON(w, http.StatusOK, review)
	}
}

func handleRequestChanges(reviewService *reviews.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if reviewService == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Review workflow is not configured.")
			return
		}

		user, ok := auth.UserFromContext(r.Context())
		if !ok || !user.IsAuthenticated {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
			return
		}

		var body struct {
			Comment string `json:"comment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			return
		}

		review, err := reviewService.RequestChanges(r.Context(), chi.URLParam(r, "reviewId"), body.Comment)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "REQUEST_CHANGES_FAILED", "Failed to request changes.")
			return
		}

		writeJSON(w, http.StatusOK, review)
	}
}

func handleAddReviewComment(reviewService *reviews.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if reviewService == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Review workflow is not configured.")
			return
		}

		user, ok := auth.UserFromContext(r.Context())
		if !ok || !user.IsAuthenticated {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
			return
		}

		var body struct {
			Body string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			return
		}

		comment, err := reviewService.AddComment(r.Context(), chi.URLParam(r, "reviewId"), user.ID, body.Body)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "COMMENT_FAILED", "Failed to add comment.")
			return
		}

		writeJSON(w, http.StatusCreated, comment)
	}
}

func handleListReviews(reviewService *reviews.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if reviewService == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Review workflow is not configured.")
			return
		}

		packageVersionID, err := reviewService.ResolvePackageVersionID(
			r.Context(),
			chi.URLParam(r, "org"),
			chi.URLParam(r, "namespace"),
			chi.URLParam(r, "package"),
			chi.URLParam(r, "version"),
		)
		if err != nil {
			writeStoreError(w, r, packages.ErrVersionNotFound)
			return
		}

		reviews, err := reviewService.ListReviews(r.Context(), packageVersionID)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "LIST_REVIEWS_FAILED", "Failed to list reviews.")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"items": reviews})
	}
}

func handleApprovalStatus(reviewService *reviews.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if reviewService == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Review workflow is not configured.")
			return
		}

		packageVersionID, err := reviewService.ResolvePackageVersionID(
			r.Context(),
			chi.URLParam(r, "org"),
			chi.URLParam(r, "namespace"),
			chi.URLParam(r, "package"),
			chi.URLParam(r, "version"),
		)
		if err != nil {
			writeStoreError(w, r, packages.ErrVersionNotFound)
			return
		}

		status := reviewService.GetApprovalStatus(r.Context(), packageVersionID)
		writeJSON(w, http.StatusOK, status)
	}
}

func handleCreateToken(authenticator *auth.Authenticator, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authenticator == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Token management is not configured.")
			return
		}

		user, ok := auth.UserFromContext(r.Context())
		if !ok || !user.IsAuthenticated {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
			return
		}

		var body struct {
			DisplayName         string   `json:"displayName"`
			Scopes              []string `json:"scopes"`
			OrgID               string   `json:"orgId,omitempty"`
			NamespaceID         string   `json:"namespaceId,omitempty"`
			PackageID           string   `json:"packageId,omitempty"`
			ActorServiceAccount string   `json:"actorServiceAccount,omitempty"`
			ExpiresAt           string   `json:"expiresAt,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			return
		}

		if len(body.Scopes) == 0 {
			body.Scopes = []string{"package:read"}
		}

		var expiresAt time.Time
		if body.ExpiresAt != "" {
			var parseErr error
			expiresAt, parseErr = time.Parse(time.RFC3339, body.ExpiresAt)
			if parseErr != nil {
				writeError(w, r, http.StatusBadRequest, "INVALID_EXPIRES_AT", "expiresAt must be in RFC3339 format.")
				return
			}
		}

		token, rawToken, err := authenticator.CreateAPIToken(r.Context(), auth.CreateTokenRequest{
			DisplayName:         body.DisplayName,
			ActorUserID:         user.ID,
			ActorServiceAccount: body.ActorServiceAccount,
			Scopes:              body.Scopes,
			OrgID:               body.OrgID,
			NamespaceID:         body.NamespaceID,
			PackageID:           body.PackageID,
			ExpiresAt:           expiresAt,
		})
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "TOKEN_CREATE_FAILED", "Failed to create token.")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"id":          token.ID,
			"displayName": token.DisplayName,
			"scopes":      token.Scopes,
			"token":       rawToken,
			"createdAt":   token.CreatedAt.Time.UTC().Format(time.RFC3339),
		})
	}
}

func handleRevokeToken(authenticator *auth.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authenticator == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Token management is not configured.")
			return
		}

		user, ok := auth.UserFromContext(r.Context())
		if !ok || !user.IsAuthenticated {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
			return
		}

		if err := authenticator.RevokeAPIToken(r.Context(), chi.URLParam(r, "tokenId")); err != nil {
			writeError(w, r, http.StatusInternalServerError, "TOKEN_REVOKE_FAILED", "Failed to revoke token.")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
	}
}

func handleOIDCLogin(authenticator *auth.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authenticator == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "OIDC is not configured.")
			return
		}

		authURL, state, err := authenticator.OIDCAuthURL()
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "OIDC_CONFIG_ERROR", "OIDC provider is not configured.")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "oidc_state",
			Value:    state,
			Path:     "/auth/oidc/callback",
			HttpOnly: true,
			Secure:   true,
			MaxAge:   300,
		})

		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

func handleOIDCCallback(authenticator *auth.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authenticator == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "OIDC is not configured.")
			return
		}

		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		if code == "" || state == "" {
			writeError(w, r, http.StatusBadRequest, "INVALID_CALLBACK", "Missing code or state parameter.")
			return
		}

		cookie, err := r.Cookie("oidc_state")
		if err != nil || cookie.Value != state {
			writeError(w, r, http.StatusBadRequest, "INVALID_STATE", "State parameter mismatch.")
			return
		}

		user, err := authenticator.OIDCCallback(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "OIDC_CALLBACK_FAILED", "Failed to complete OIDC login.")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":          user.ID,
			"email":       user.Email,
			"displayName": user.DisplayName,
		})
	}
}

func handleCheckUpdate(service *updates.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req updates.UpdateCheckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			return
		}

		if req.Package == "" {
			writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "package is required.")
			return
		}
		if req.CurrentVersion == "" {
			writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "currentVersion is required.")
			return
		}

		result, err := service.CheckUpdate(r.Context(), req)
		if err != nil {
			writeStoreError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

func handleCheckCompatibility(service *updates.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req updates.CompatibilityCheckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			return
		}

		if req.Package == "" {
			writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "package is required.")
			return
		}
		if req.Version == "" {
			writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "version is required.")
			return
		}

		result, err := service.CheckCompatibility(r.Context(), req)
		if err != nil {
			writeStoreError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

func handleSearchPackages(store packages.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "q query parameter is required.")
			return
		}

		limit := 50
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil || parsed < 1 {
				writeError(w, r, http.StatusBadRequest, "INVALID_LIMIT", "limit must be a positive integer.")
				return
			}
			limit = parsed
		}

		result, err := store.SearchPackages(r.Context(), packages.SearchParams{
			Query:        query,
			Org:          r.URL.Query().Get("org"),
			Namespace:    r.URL.Query().Get("namespace"),
			ArtifactType: r.URL.Query().Get("artifactType"),
			Tool:         r.URL.Query().Get("tool"),
			Limit:        limit,
			Cursor:       r.URL.Query().Get("cursor"),
		})
		if err != nil {
			writeStoreError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

func handleGetPackageAdoption(store packages.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := store.GetPackageAdoption(r.Context(), chi.URLParam(r, "org"), chi.URLParam(r, "namespace"), chi.URLParam(r, "package"))
		if err != nil {
			writeStoreError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func handleReportProjectAdoption(writeStore packages.WriteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeStore == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Write APIs are not configured.")
			return
		}

		user, ok := auth.UserFromContext(r.Context())
		if !ok || !user.IsAuthenticated {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
			return
		}

		var body struct {
			Org       string `json:"org"`
			Name      string `json:"name"`
			RepoURL   string `json:"repoUrl"`
			Lockfile  string `json:"lockfile,omitempty"`
			Installed []struct {
				Package string `json:"package"`
				Version string `json:"version"`
				Digest  string `json:"digest"`
			} `json:"installed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			return
		}

		if body.Org == "" || body.Name == "" || body.RepoURL == "" {
			writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "org, name, and repoUrl are required.")
			return
		}

		installed := make([]packages.InstalledPackage, 0, len(body.Installed))
		for _, item := range body.Installed {
			installed = append(installed, packages.InstalledPackage{
				Package: item.Package,
				Version: item.Version,
				Digest:  item.Digest,
			})
		}

		if err := writeStore.ReportProjectAdoption(r.Context(), packages.ReportProjectAdoptionRequest{
			Org:       body.Org,
			Name:      body.Name,
			RepoURL:   body.RepoURL,
			Installed: installed,
		}); err != nil {
			writeStoreError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "reported"})
	}
}

func handleCreateProject(writeStore packages.WriteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeStore == nil {
			writeError(w, r, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Write APIs are not configured.")
			return
		}

		user, ok := auth.UserFromContext(r.Context())
		if !ok || !user.IsAuthenticated {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
			return
		}

		var body struct {
			Org     string `json:"org"`
			Name    string `json:"name"`
			RepoURL string `json:"repoUrl"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.")
			return
		}

		if body.Org == "" || body.Name == "" || body.RepoURL == "" {
			writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "org, name, and repoUrl are required.")
			return
		}

		result, err := writeStore.CreateProject(r.Context(), packages.CreateProjectRequest{
			Org:     body.Org,
			Name:    body.Name,
			RepoURL: body.RepoURL,
		})
		if err != nil {
			writeStoreError(w, r, err)
			return
		}

		w.Header().Set("Location", "/api/v1/projects/"+result.ID)
		writeJSON(w, http.StatusCreated, result)
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
	case errors.Is(err, packages.ErrInvalidArchive):
		writeError(w, r, http.StatusBadRequest, "INVALID_ARCHIVE", "Archive format is invalid.")
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
