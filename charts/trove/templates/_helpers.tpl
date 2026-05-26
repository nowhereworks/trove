{{/*
Expand the name of the chart.
*/}}
{{- define "trove.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "trove.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "trove.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels.
*/}}
{{- define "trove.labels" -}}
helm.sh/chart: {{ include "trove.chart" . }}
{{ include "trove.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels.
*/}}
{{- define "trove.selectorLabels" -}}
app.kubernetes.io/name: {{ include "trove.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Create the name of the service account to use.
*/}}
{{- define "trove.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "trove.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
Name of the chart-managed secret.
*/}}
{{- define "trove.secretName" -}}
{{- printf "%s-config" (include "trove.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Whether the chart needs to create a secret.
*/}}
{{- define "trove.hasManagedSecret" -}}
{{- if or (and .Values.database.url (not .Values.database.existingSecret.name)) (and .Values.auth.devToken (not .Values.auth.devTokenExistingSecret.name)) (and .Values.oidc.clientSecret (not .Values.oidc.clientSecretExistingSecret.name)) -}}
true
{{- end -}}
{{- end -}}

{{/*
Validate required PostgreSQL configuration.
*/}}
{{- define "trove.validateDatabase" -}}
{{- if and (not .Values.database.url) (not .Values.database.existingSecret.name) -}}
{{- fail "database.url or database.existingSecret.name is required; Trove supports PostgreSQL only" -}}
{{- end -}}
{{- end -}}
