{{/*
  _helpers.tpl
  NOTE: Placeholder helper templates. To be expanded once the monolith is split
  and each service gets its own Deployment + Service + HPA manifests.
*/}}

{{/*
  Expand the name of the chart.
  Truncates at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "aleph-v2.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
  Create a default fully qualified app name.
  If release name already contains the chart name, use it as the full name.
*/}}
{{- define "aleph-v2.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
  Create chart name and version as used by the chart label.
*/}}
{{- define "aleph-v2.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
  Common labels shared across all resources.
*/}}
{{- define "aleph-v2.labels" -}}
helm.sh/chart: {{ include "aleph-v2.chart" . }}
{{ include "aleph-v2.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
  Selector labels — used by Deployments, Services, HPAs.
*/}}
{{- define "aleph-v2.selectorLabels" -}}
app.kubernetes.io/name: {{ include "aleph-v2.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
  Create the name of the service account to use.
*/}}
{{- define "aleph-v2.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "aleph-v2.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
  DuckDB PVC name helper.
  DuckDB requires a single attached volume, so this helper ensures stable naming.
*/}}
{{- define "aleph-v2.duckdb.pvcName" -}}
{{- printf "%s-duckdb-data" (include "aleph-v2.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
  Postgres secret name helper.
*/}}
{{- define "aleph-v2.postgres.secretName" -}}
{{- if .Values.postgres.auth.existingSecret }}
{{- .Values.postgres.auth.existingSecret }}
{{- else }}
{{- printf "%s-postgres" (include "aleph-v2.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
  Grafana secret name helper.
*/}}
{{- define "aleph-v2.grafana.secretName" -}}
{{- if .Values.grafana.admin.existingSecret }}
{{- .Values.grafana.admin.existingSecret }}
{{- else }}
{{- printf "%s-grafana" (include "aleph-v2.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
