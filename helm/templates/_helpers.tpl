{{/*
Expand the name of the chart.
*/}}
{{- define "altc-chart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "altc-chart.fullname" -}}
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
{{- define "altc-chart.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "altc-chart.labels" -}}
helm.sh/chart: {{ include "altc-chart.chart" . }}
{{ include "altc-chart.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "altc-chart.selectorLabels" -}}
app.kubernetes.io/name: {{ include "altc-chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "altc-chart.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "altc-chart.name" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the cluster role for accessing kubernetes resources
*/}}
{{- define "altc-chart.resourcesClusterRoleName" -}}
{{- .Values.clusterRole.podResources.name }}
{{- end }}

{{/*
Create the name of the cluster role for accessing kubernetes secrets
*/}}
{{- define "altc-chart.secretsClusterRoleName" -}}
{{- .Values.clusterRole.secrets.name }}
{{- end }}

{{/*
Create the name of the altc configmap
*/}}
{{- define "altc-chart.configMapName" -}}
{{- default "altc-env-config-map" .Values.configMapName}}
{{- end }}

{{/*
Create the name of the altc secret
*/}}
{{- define "altc-chart.secretName" -}}
{{- default "altc-env-secret" .Values.secretName }}
{{- end }}
