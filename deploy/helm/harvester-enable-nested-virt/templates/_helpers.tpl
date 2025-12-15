{{/*
Expand the name of the chart.
*/}}
{{- define "enable-nested-virt.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "enable-nested-virt.fullname" -}}
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
{{- define "enable-nested-virt.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "enable-nested-virt.labels" -}}
helm.sh/chart: {{ include "enable-nested-virt.chart" . }}
{{ include "enable-nested-virt.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "enable-nested-virt.selectorLabels" -}}
app.kubernetes.io/name: {{ include "enable-nested-virt.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "enable-nested-virt.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "enable-nested-virt.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Webhook service name
*/}}
{{- define "enable-nested-virt.webhookServiceName" -}}
{{- include "enable-nested-virt.fullname" . }}-webhook
{{- end }}

{{/*
Certificate secret name
*/}}
{{- define "enable-nested-virt.certificateSecretName" -}}
{{- include "enable-nested-virt.fullname" . }}-tls
{{- end }}

{{/*
Certificate name
*/}}
{{- define "enable-nested-virt.certificateName" -}}
{{- include "enable-nested-virt.fullname" . }}-cert
{{- end }}

{{/*
Issuer name
*/}}
{{- define "enable-nested-virt.issuerName" -}}
{{- include "enable-nested-virt.fullname" . }}-issuer
{{- end }}
